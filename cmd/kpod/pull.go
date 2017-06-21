package main

import (
	"os"

	cp "github.com/containers/image/copy"
	"github.com/containers/image/docker/reference"
	"github.com/containers/image/signature"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	// DefaultRegistry is a prefix that we apply to an image name if we
	// can't find one in the local Store, in order to generate a source
	// reference for the image that we can then copy to the local Store
	DefaultRegistry = "docker://"
)

var (
	pullFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "all-tags, a",
			Usage: "Download all tagged images in the repository",
		},
		cli.BoolFlag{
			Name:  "disable-content-trust",
			Usage: "Skip image verification",
		},
		cli.StringFlag{
			Name:  "registry",
			Usage: "`prefix` to prepend the image name in order to pull the image",
		},
	}

	pullDescription = "Pull an image from a registry"
	pullCommand     = cli.Command{
		Name:        "pull",
		Usage:       "pull an image from a registry",
		Description: pullDescription,
		Flags:       pullFlags,
		Action:      pullCmd,
		ArgsUsage:   "",
	}
)

func pullCmd(c *cli.Context) error {
	store, err := getStore(c)
	if err != nil {
		return err
	}

	allTags := false
	if c.IsSet("all-tags") {
		allTags = c.Bool("all-tags")
	}
	disableContentTrust := false
	if c.IsSet("disable-content-trust") {
		disableContentTrust = c.Bool("disable-content-trust")
	}
	imgName := ""
	if len(c.Args()) != 1 {
		return errors.New("'kpod pull' requires exactly 1 argument")
	}
	imgName = c.Args().Get(0)
	registry := DefaultRegistry
	if c.IsSet("registry") {
		registry = c.String("registry")
	}

	return pullImage(imgName, registry, store, allTags, disableContentTrust)
}

func pullImage(imgName, registry string, store storage.Store, allTags, disableContentTrust bool) error {
	srcRef, err := getSrcRef(imgName, registry)
	if err != nil {
		return errors.Wrap(err, "error getting source reference")
	}
	if ref := srcRef.DockerReference(); ref != nil {
		imgName = srcRef.DockerReference().Name()
		if tagged, ok := srcRef.DockerReference().(reference.NamedTagged); ok {
			imgName = imgName + ":" + tagged.Tag()
		}
	}

	destRef, err := is.Transport.ParseStoreReference(store, imgName)
	if err != nil {
		return errors.Wrapf(err, "error parsing full image name %q", imgName)
	}

	policyContext, err := getPolicyContext()
	if err != nil {
		return errors.Wrap(err, "error getting policy context")
	}
	err = cp.Image(policyContext, destRef, srcRef, getCopyOptions(os.Stderr))
	if err != nil {
		return err
	}
	return nil
}

func getSrcRef(imgName, registry string) (types.ImageReference, error) {
	spec := registry + imgName
	srcRef, err := alltransports.ParseImageName(imgName)
	if err != nil {
		srcRef2, err2 := alltransports.ParseImageName(spec)
		if err2 != nil {
			return nil, errors.Wrapf(err2, "error parsing image name %q", spec)
		}
		srcRef = srcRef2
	}
	return srcRef, nil
}

func getPolicyContext() (*signature.PolicyContext, error) {
	var systemContext types.SystemContext
	policy, err := signature.DefaultPolicy(&systemContext)
	if err != nil {
		return &signature.PolicyContext{}, err
	}
	return signature.NewPolicyContext(policy)
}
