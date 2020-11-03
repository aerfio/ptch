package image

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/aerfio/ptch/docker"
)

const tmpImageName = "tmpimage.tar"

type Image struct {
	repo string
	tag  string
}

func New(img string) (Image, error) {
	if !strings.Contains(img, ":") {
		return Image{}, errors.New("no tag provided")
	}
	// let's assume we have 2 elements here
	parts := strings.Split(img, ":")
	return Image{
		repo: parts[0],
		tag:  parts[1],
	}, nil
}

func (i Image) String() string {
	return fmt.Sprintf("%s:%s", i.repo, i.tag)
}

func (i Image) SaveToTmpDir() (string, error) {
	name, err := ioutil.TempDir("", "docker-image-*")
	if err != nil {
		return "", errors.Wrap(err, "while creating tmp dir for docker image tarball")
	}

	imageFullPath := path.Join(name, tmpImageName)
	err = docker.SaveImage(i.String(), imageFullPath)
	return imageFullPath, errors.Wrapf(err, "while saving docker image to tmp dir %s", name)
}
