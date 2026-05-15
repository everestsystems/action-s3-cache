package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// extensionFor returns the file extension for the given compression algorithm.
func extensionFor(compression string) string {
	switch compression {
	case CompressTarGzip:
		return ".tar.gz"
	case CompressTarZstd:
		return ".tar.zst"
	default: // CompressZip and anything unrecognised
		return ".zip"
	}
}

func main() {
	compression := os.Getenv("COMPRESSION")
	if compression == "" {
		compression = CompressTarGzip
	}

	action := Action{
		Action:      os.Getenv("ACTION"),
		Bucket:      os.Getenv("BUCKET"),
		S3Class:     os.Getenv("S3_CLASS"),
		Key:         fmt.Sprintf("%s%s", os.Getenv("KEY"), extensionFor(compression)),
		Artifacts:   strings.Split(strings.TrimSpace(os.Getenv("ARTIFACTS")), "\n"),
		Compression: compression,
	}

	log.Printf("starting the caching process with:")
	log.Printf("Key=%s\n", action.Key)
	log.Printf("Artifacts=%v\n", action.Artifacts)
	log.Printf("Bucket=%s\n", action.Bucket)
	log.Printf("Compression=%s\n", action.Compression)

	switch act := action.Action; act {
	case PutAction:
		if len(action.Artifacts[0]) <= 0 {
			log.Fatal("No artifacts patterns provided")
		}

		log.Printf("archiving artifacts")
		if err := archive(action.Key, action.Artifacts, action.Compression); err != nil {
			log.Fatal(err)
		}

		log.Printf("uploading to s3")
		if err := PutObject(action.Key, action.Bucket, action.S3Class); err != nil {
			log.Fatal(err)
		}

	case GetAction:
		exists, err := ObjectExists(action.Key, action.Bucket)
		if err != nil {
			log.Fatal(err)
		}

		if exists {
			log.Printf("downloading from s3")
			if err := GetObject(action.Key, action.Bucket); err != nil {
				log.Fatal(err)
			}

			log.Printf("extracting archive")
			if err := extract(action.Key, action.Compression); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Printf("No cache found for key: %s", action.Key)
		}

	case DeleteAction:
		log.Printf("deleting from s3")
		if err := DeleteObject(action.Key, action.Bucket); err != nil {
			log.Fatal(err)
		}

	default:
		log.Fatalf("Action %q is not allowed. Valid options: [%s, %s, %s]", act, PutAction, GetAction, DeleteAction)
	}
	log.Printf("caching process finished!")
}

// archive compresses artifacts to filename using the given compression type.
func archive(filename string, artifacts []string, compression string) error {
	switch compression {
	case CompressTarGzip:
		return TarGzip(filename, artifacts)
	case CompressTarZstd:
		return TarZstd(filename, artifacts)
	default: // CompressZip
		return Zip(filename, artifacts)
	}
}

// extract decompresses filename using the given compression type.
func extract(filename string, compression string) error {
	switch compression {
	case CompressTarGzip:
		return UntarGzip(filename)
	case CompressTarZstd:
		return UntarZstd(filename)
	default: // CompressZip
		return Unzip(filename)
	}
}
