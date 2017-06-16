package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var (
	rmiDescription = "removes one or more locally stored images."
	rmiFlags       = []cli.Flag{
		cli.BoolFlag{
			Name:  "force, f",
			Usage: "force removal of the image",
		},
	}
	rmiCommand = cli.Command{
		Name:        "rmi",
		Usage:       "removes one or more images from local storage",
		Description: rmiDescription,
		Action:      rmiCmd,
		ArgsUsage:   "IMAGE-NAME-OR-ID [...]",
		Flags:       rmiFlags,
	}
)

func rmiCmd(c *cli.Context) error {

	force := false
	if c.IsSet("force") {
		force = c.Bool("force")
	}

	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("image name or ID must be specified")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	var e error
	for _, id := range args {
		// If it's an exact name or ID match with the underlying
		// storage library's information about the image, then it's
		// enough.
		_, err = store.DeleteImage(id, true)
		if err != nil {
			var ref types.ImageReference
			ref, err2 := properImageRef(id)
			if err2 != nil {
				logrus.Debug(err2)
			}
			if ref == nil {
				if ref, err2 = storageImageRef(store, id); err2 != nil {
					logrus.Debug(err2)
				}
			}
			if ref == nil {
				if ref, err2 = storageImageID(store, id); err2 != nil {
					logrus.Debug(err2)
				}
			}
			if ref != nil {
				if force {
					// Remove all running containers matching ref
					image, err2 := is.Transport.GetImage(ref)
					if err2 != nil {
						logrus.Debugf("Error parsing image ID: %v\n", err2)
					}
					containers, err2 := store.Containers()
					if err2 != nil {
						logrus.Debugf("Error getting associated containers: %v\n", err)
					}
					for i := 0; i < len(containers); i++ {
						if containers[i].ImageID == image.ID {
							fmt.Printf("Found the container")
							store.DeleteContainer(containers[i].ID)
						}
					}
				}
				err = ref.DeleteImage(nil)
			}
		}
		if e == nil {
			e = err
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error removing image %q: %v\n", id, err)
			continue
		}
		fmt.Printf("%s\n", id)
	}

	return e
}

// If it's looks like a proper image reference, parse it and check if it
// corresponds to an image that actually exists.
func properImageRef(id string) (types.ImageReference, error) {
	var err error
	if ref, err := alltransports.ParseImageName(id); err == nil {
		if img, err2 := ref.NewImage(nil); err2 == nil {
			img.Close()
			return ref, nil
		}
		return nil, fmt.Errorf("error confirming presence of image %q: %v", transports.ImageName(ref), err)
	}
	return nil, fmt.Errorf("error parsing %q as a store reference: %v", id, err)
}

// If it's looks like an image reference that's relative to our storage, parse
// it and check if it corresponds to an image that actually exists.
func storageImageRef(store storage.Store, id string) (types.ImageReference, error) {
	var err error
	if ref, err := is.Transport.ParseStoreReference(store, id); err == nil {
		if img, err2 := ref.NewImage(nil); err2 == nil {
			img.Close()
			return ref, nil
		}
		return nil, fmt.Errorf("error confirming presence of image %q: %v", transports.ImageName(ref), err)
	}
	return nil, fmt.Errorf("error parsing %q as a store reference: %v", id, err)
}

// If it might be an ID that's relative to our storage, parse it and check if it
// corresponds to an image that actually exists.  This _should_ be redundant,
// since we already tried deleting the image using the ID directly above, but it
// can't hurt either.
func storageImageID(store storage.Store, id string) (types.ImageReference, error) {
	var err error
	if ref, err := is.Transport.ParseStoreReference(store, "@"+id); err == nil {
		if img, err2 := ref.NewImage(nil); err2 == nil {
			img.Close()
			return ref, nil
		}
		return nil, fmt.Errorf("error confirming presence of image %q: %v", transports.ImageName(ref), err)
	}
	return nil, fmt.Errorf("error parsing %q as an image reference: %v", "@"+id, err)
}
