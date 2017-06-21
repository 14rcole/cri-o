package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	cp "github.com/containers/image/copy"
	is "github.com/containers/image/storage"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/urfave/cli"
)

type imageMetadata struct {
	Tag            string              `json:"tag"`
	CreatedTime    time.Time           `json:"created-time"`
	ID             string              `json:"id"`
	Blobs          []types.BlobInfo    `json:"blob-list"`
	Layers         map[string][]string `json:"layers"`
	SignatureSizes []string            `json:"signature-sizes"`
}

var needToShutdownStore = false

func getStore(c *cli.Context) (storage.Store, error) {
	options := storage.DefaultStoreOptions
	if c.GlobalIsSet("root") || c.GlobalIsSet("runroot") {
		options.GraphRoot = c.GlobalString("root")
		options.RunRoot = c.GlobalString("runroot")
	}

	if c.GlobalIsSet("storage-driver") {
		options.GraphDriverName = c.GlobalString("storage-driver")
	}
	if c.GlobalIsSet("storage-opt") {
		opts := c.GlobalStringSlice("storage-opt")
		if len(opts) > 0 {
			options.GraphDriverOptions = opts
		}
	}
	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	needToShutdownStore = true
	return store, err
}

func parseMetadata(image storage.Image) (imageMetadata, error) {
	var im imageMetadata

	dec := json.NewDecoder(strings.NewReader(image.Metadata))
	if err := dec.Decode(&im); err != nil {
		return imageMetadata{}, err
	}
	return im, nil
}

func getSize(image storage.Image, store storage.Store) (int64, error) {

	is.Transport.SetStore(store)
	storeRef, err := is.Transport.ParseStoreReference(store, "@"+image.ID)
	if err != nil {
		fmt.Println(err)
		return -1, err
	}
	img, err := storeRef.NewImage(nil)
	if err != nil {
		fmt.Println("Error with NewImage")
		return -1, err
	}
	imgSize, err := img.Size()
	if err != nil {
		fmt.Println("Error getting size")
		return -1, err
	}
	return imgSize, nil
}

func getCopyOptions(reportWriter io.Writer) *cp.Options {
	return &cp.Options{
		ReportWriter: reportWriter,
	}
}
