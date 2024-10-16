package main

import (
	"flag"
	"fmt"
	"log/slog"
	"maps"
	"os"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	imageNames := flag.Args()

	allBlobs := map[string]int64{}

	for _, imageName := range imageNames {
		// we parse the image reference with a empty default tag that
		// it does not get defaulted to latest
		ref, err := name.ParseReference(imageName, name.WithDefaultTag(""))
		if err != nil {
			return err
		}

		if ref.Identifier() == "" {
			// get blobs for all tags
			blobs, err := getBlobsByNameForAllTags(imageName)
			if err != nil {
				return err
			}
			maps.Insert(allBlobs, maps.All(blobs))
		} else {
			// get blobs for specific tag
			blobs, err := getBlobsByImageTag(ref.String())
			if err != nil {
				return err
			}
			maps.Insert(allBlobs, maps.All(blobs))
		}

	}

	var total int64
	for blob, size := range allBlobs {
		fmt.Println(blob, size)
		total += size
	}
	fmt.Println("total:", total)
	return nil
}

// getBlobsByNameForAllTags gets blobs for all tags (e.g. busybox)
func getBlobsByNameForAllTags(name string) (map[string]int64, error) {
	tags, err := crane.ListTags(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags for '%s'", name)
	}

	allBlobs := map[string]int64{}

	for _, tag := range tags {
		fullName := name + ":" + tag
		blobs, err := getBlobsByImageTag(fullName)
		if err != nil {
			return nil, err
		}
		maps.Insert(allBlobs, maps.All(blobs))
	}

	return allBlobs, nil
}

// getBlobsByName gets blobs for a single tag (e.g. busybox:latest)
func getBlobsByImageTag(name string) (map[string]int64, error) {
	ref, err := crane.Get(name)
	if err != nil {
		return nil, err
	}

	if index, err := ref.ImageIndex(); err == nil {
		// handle image index

		allBlobs := map[string]int64{}

		//
		// add size of index manifest itself
		//
		indexDigest, err := index.Digest()
		if err != nil {
			return nil, err
		}
		indexSize, err := index.Size()
		if err != nil {
			return nil, err
		}

		allBlobs[indexDigest.Hex] = indexSize

		// get size of all images in index
		slog.Info("get images from image index", "name", name)
		indexManifest, err := index.IndexManifest()
		if err != nil {
			return nil, err
		}

		for _, ref := range indexManifest.Manifests {
			slog.Info("get blobs from image", "name", name, "platform", ref.Platform, "media", ref.MediaType)
			img, err := index.Image(ref.Digest)
			if err != nil {
				return nil, err
			}
			blobs, err := getBlobsByImage(img)
			if err != nil {
				return nil, err
			}
			maps.Insert(allBlobs, maps.All(blobs))
		}
		return allBlobs, nil
	} else if image, err := ref.Image(); err == nil {
		// single image image

		slog.Info("get blobs from image", "name", name)
		return getBlobsByImage(image)
	} else {
		return nil, fmt.Errorf("manifest is not image and image index but '%s'", ref.MediaType)
	}
}

func getBlobsByImage(image v1.Image) (map[string]int64, error) {

	blobs := map[string]int64{}

	image.RawManifest()

	// add size of image manifest itself
	imageManifestDigest, err := image.Digest()
	if err != nil {
		return nil, err
	}

	sizeManifestSize, err := image.Size()
	if err != nil {
		return nil, err
	}

	blobs[imageManifestDigest.Hex] = sizeManifestSize

	// add size of layers
	imageLayers, err := image.Layers()
	if err != nil {
		return nil, err
	}

	for _, blob := range imageLayers {
		digest, err := blob.Digest()
		if err != nil {
			return nil, err
		}
		size, err := blob.Size()
		if err != nil {
			return nil, err
		}
		blobs[digest.Hex] = size
	}

	return blobs, nil
}
