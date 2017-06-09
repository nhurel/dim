// Copyright 2016
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"encoding/json"

	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/nhurel/dim/lib"
)

// Repository implements Repository interface
type Repository struct {
	distribution.Repository
	client dim.RegistryClient
	mfs    distribution.ManifestService
	bfs    distribution.BlobService
}

// AllTags returns all existing tag for this repository instance
func (r *Repository) AllTags() ([]string, error) {
	return r.tagService().All(ctx)
}

func (r *Repository) tagService() distribution.TagService {
	return r.Repository.Tags(ctx)
}

func (r *Repository) manifestService() (distribution.ManifestService, error) {
	if r.mfs == nil {
		var mService distribution.ManifestService
		var err error
		if mService, err = r.Manifests(ctx); err != nil {
			logrus.WithError(err).Errorln("Failed to instantiate manifestService")
			return nil, err
		}

		r.mfs = mService
	}
	return r.mfs, nil
}

func (r *Repository) blobService() distribution.BlobService {
	if r.bfs == nil {
		r.bfs = r.Blobs(ctx)
	}
	return r.bfs
}

// Image return image info for a given tag
func (r *Repository) Image(tag string) (img *dim.RegistryImage, err error) {

	var tagDigest digest.Digest
	if tagDigest, err = r.getTagDigest(tag); err != nil {
		return
	}

	if img, err = r.ImageFromManifest(tagDigest, tag); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name()}).WithError(err).Errorln("Failed to get image")
	}
	return
}

// ImageFromManifest returns image information from its manifest digest
func (r *Repository) ImageFromManifest(tagDigest digest.Digest, tag string) (image *dim.RegistryImage, err error) {
	var mService distribution.ManifestService
	if mService, err = r.manifestService(); err != nil {
		return
	}

	var mf distribution.Manifest
	l := logrus.WithField("tagDigest", tagDigest)
	l.Debugln("Getting manifest")
	if mf, err = mService.Get(ctx, tagDigest); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name()}).WithError(err).Errorln("Failed to get manifest")
		return
	}

	l.Debugln("Reading manifest")
	var payload []byte
	if _, payload, err = mf.Payload(); err != nil {
		logrus.WithError(err).Errorln("Failed to read manifest")
		return
	}

	l.Debugln("Unmarshalling manifest")
	manif := &schema2.Manifest{}
	if err = json.Unmarshal(payload, manif); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name()}).WithError(err).Errorln("Failed to read image manifest")
		return
	}

	if payload, err = r.blobService().Get(ctx, manif.Config.Digest); err != nil {
		logrus.WithError(err).Errorln("Failed to get image config")
		return
	}

	logrus.WithField("Digest", manif.Config.Digest).Debugln("Unmarshalling V2Image")

	image = &dim.RegistryImage{Tag: tag, Digest: string(tagDigest)}
	if err = json.Unmarshal(payload, image); err != nil {
		logrus.WithField("Digest", manif.Config.Digest).WithError(err).Errorln("Failed to read image")
		return
	}

	return
}

func (r *Repository) getTagDigest(tag string) (digest.Digest, error) {
	var err error
	var tDescriptor distribution.Descriptor
	if tDescriptor, err = r.tagService().Get(ctx, tag); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name(), "tag": tag}).WithError(err).Errorln("Failed to get Tag")
		return "", err
	}
	return tDescriptor.Digest, nil

}

// DeleteImage sends a delete request on tag
func (r *Repository) DeleteImage(tag string) error {
	logrus.WithField("tag", tag).Debugln("Entering DeleteImage")
	var err error
	var mfService distribution.ManifestService

	var tagDigest digest.Digest
	if tagDigest, err = r.getTagDigest(tag); err != nil {
		return err
	}

	if mfService, err = r.manifestService(); err != nil {
		return err
	}

	logrus.WithField("tagDigest", tagDigest).Debugln("Calling delete on manifestService")
	return mfService.Delete(ctx, tagDigest)
}

// WalkImages walks through all images of the repository and writes them in the given channel
func (r *Repository) WalkImages() <-chan *dim.RegistryImage {
	return WalkImages(r)
}

// WalkImages walks through all images of the repository and writes them in the given channel
func WalkImages(r dim.Repository) <-chan *dim.RegistryImage {
	images := make(chan *dim.RegistryImage, 3)

	go func() {
		defer close(images)
		var err error
		var tags []string
		l := logrus.WithField("repository", r.Named().Name())

		l.Debugln("Walking through repository images")

		if tags, err = r.AllTags(); err != nil {
			l.WithError(err).Errorln("Failed to get tags ")
			return
		}

		wg := sync.WaitGroup{}
		wg.Add(10)
		ch := make(chan string, 10)
		for i := 0; i < 10; i++ {
			go func() {
				for tag := range ch {
					l = l.WithField("tag", tag)
					l.Debugln("Getting image details")

					var img *dim.RegistryImage
					if img, err = r.Image(tag); err != nil {
						logrus.WithError(err).Errorln("Failed to get image")
						return
					}

					l.WithField("image", img).Debugln("Walking on image")
					images <- img
				}
				wg.Done()
			}()
		}

		for _, tag := range tags {
			ch <- tag
		}
		close(ch)
		wg.Wait()

	}()

	return images

}
