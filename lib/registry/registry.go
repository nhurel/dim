package registry

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/image"
	"github.com/docker/docker/reference"
	"github.com/docker/docker/registry"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
	"net/http"
)

type Client struct {
	client.Registry
	transport   http.RoundTripper
	registryUrl string
}

type Repository struct {
	distribution.Repository
	client *Client
	mfs    distribution.ManifestService
	bfs    distribution.BlobService
}

var ctx = context.Background()

func New(registryAuth *types.AuthConfig, registryUrl string) (*Client, error) {
	var err error
	var reg client.Registry

	transport := http.DefaultTransport

	if registryAuth != nil {
		transport = registry.AuthTransport(transport, registryAuth, true)
	}

	if reg, err = client.NewRegistry(ctx, registryUrl, transport); err != nil {
		return nil, err
	}

	return &Client{reg, transport, registryUrl}, nil
}

func (c *Client) NewRepository(parsedName reference.Named) (Repository, error) {
	logrus.WithField("name", parsedName).Debugln("Creating new repository")
	if repo, err := client.NewRepository(ctx, parsedName, c.registryUrl, c.transport); err != nil {
		return Repository{}, err
	} else {
		return Repository{Repository: repo, client: c}, nil
	}
}

func (r *Repository) AllTags() ([]string, error) {
	return r.tagService().All(ctx)
}

func (r *Repository) tagService() distribution.TagService {
	return r.Repository.Tags(ctx)
}

func (r *Repository) manifestService() (distribution.ManifestService, error) {
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

func (r *Repository) blobService() distribution.BlobService {
	if r.bfs == nil {
		r.bfs = r.Blobs(ctx)
	}
	return r.bfs
}

func (r *Repository) Image(tag string) (dg string, img *image.Image, err error) {

	var tagDigest digest.Digest
	if tagDigest, err = r.getTagDigest(tag); err != nil {
		return
	}

	var mService distribution.ManifestService
	if mService, err = r.manifestService(); err != nil {
		return
	}

	var mf distribution.Manifest
	l := logrus.WithField("tagDigest", tagDigest)
	l.Debugln("Getting manifest")
	if mf, err = mService.Get(ctx, tagDigest, distribution.WithTag(tag)); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name(), "tag": tag}).WithError(err).Errorln("Failed to get manifest")
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
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name(), "tag": tag}).WithError(err).Errorln("Failed to read image manifest")
		return
	}

	dg = string(manif.Config.Digest)
	if payload, err = r.blobService().Get(ctx, manif.Config.Digest); err != nil {
		logrus.WithError(err).Errorln("Failed to get image config")
		return
	}

	l.Debugln("Unmarshalling V2Image")

	img = &image.Image{}
	if err = json.Unmarshal(payload, img); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name(), "tag": tag}).WithError(err).Errorln("Failed to read image")
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

type ImageManifest struct {
	Name    string              `json:"name,omitempty"`
	Tag     string              `json:"tag,omitempty"`
	History []map[string]string `json:"history,omitempty"`
}
