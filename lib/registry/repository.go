package registry

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/docker/image"
)

type Repository interface {
	distribution.Repository
	AllTags() ([]string, error)
	Image(tag string) (img *Image, err error)
	ImageFromManifest(tagDigest digest.Digest, tag string) (img *Image, err error)
	DeleteImage(tag string) error
	WalkImages(images chan<- *Image) error
}

type RegistryRepository struct {
	distribution.Repository
	client Client
	mfs    distribution.ManifestService
	bfs    distribution.BlobService
}

// AllTags returns all existing tag for this repository instance
func (r *RegistryRepository) AllTags() ([]string, error) {
	return r.tagService().All(ctx)
}

func (r *RegistryRepository) tagService() distribution.TagService {
	return r.Repository.Tags(ctx)
}

func (r *RegistryRepository) manifestService() (distribution.ManifestService, error) {
	if r.mfs == nil {
		if mService, err := r.Manifests(ctx); err != nil {
			logrus.WithError(err).Errorln("Failed to instantiate manifestService")
			return nil, err
		} else {
			r.mfs = mService
		}
	}
	return r.mfs, nil
}

func (r *RegistryRepository) blobService() distribution.BlobService {
	if r.bfs == nil {
		r.bfs = r.Blobs(ctx)
	}
	return r.bfs
}

// Image return image info for a given tag
func (r *RegistryRepository) Image(tag string) (img *Image, err error) {

	var tagDigest digest.Digest
	if tagDigest, err = r.getTagDigest(tag); err != nil {
		return
	}

	if img, err = r.ImageFromManifest(tagDigest, tag); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name()}).WithError(err).Errorln("Failed to get image")
	}
	return
}

// ImageFromManifest returns image inforamtion from its manifest digest
func (r *RegistryRepository) ImageFromManifest(tagDigest digest.Digest, tag string) (image *Image, err error) {
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

	image = &Image{Tag: tag, Digest: string(tagDigest)}
	if err = json.Unmarshal(payload, image); err != nil {
		logrus.WithField("Digest", manif.Config.Digest).WithError(err).Errorln("Failed to read image")
		return
	}

	return
}

func (r *RegistryRepository) getTagDigest(tag string) (digest.Digest, error) {
	var err error
	var tDescriptor distribution.Descriptor
	if tDescriptor, err = r.tagService().Get(ctx, tag); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name(), "tag": tag}).WithError(err).Errorln("Failed to get Tag")
		return "", err
	}
	return tDescriptor.Digest, nil

}

// DeleteImage sends a delete request on tag
func (r *RegistryRepository) DeleteImage(tag string) error {
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

type ImageManifest struct {
	Name    string              `json:"name,omitempty"`
	Tag     string              `json:"tag,omitempty"`
	History []map[string]string `json:"history,omitempty"`
}

func (r *RegistryRepository) WalkImages(images chan<- *Image) error {
	return WalkImages(r, images)
}

func WalkImages(r Repository, images chan<- *Image) error {
	defer close(images)
	var err error
	var tags []string
	l := logrus.WithField("repository", r.Named().Name())

	l.Debugln("Walking through repository images")

	if tags, err = r.AllTags(); err != nil {
		l.WithError(err).Errorln("Failed to get tags ")
		return err
	}

	for _, tag := range tags {
		l = l.WithField("tag", tag)
		l.Debugln("Getting image details")

		var img *Image
		if img, err = r.Image(tag); err != nil {
			logrus.WithError(err).Errorln("Failed to get image")
			return err
		}

		l.WithField("image", img).Debugln("Walking on image")
		images <- img
	}
	return nil

}

type Image struct {
	*image.Image
	Tag    string
	Digest string
}
