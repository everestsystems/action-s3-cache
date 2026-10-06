// Package main implements an S3-backed cache for GitHub Actions.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

//nolint:funlen // Keep the existing action dispatch flow; refactor in a follow-up.
func main() {
	action := Action{
		Action:    os.Getenv("ACTION"),
		Bucket:    os.Getenv("BUCKET"),
		S3Class:   os.Getenv("S3_CLASS"),
		Key:       fmt.Sprintf("%s-%s.tgz", os.Getenv("KEY"), os.Getenv("ARCH")),
		Artifacts: strings.Split(strings.TrimSpace(os.Getenv("ARTIFACTS")), "\n"),
	}

	log.Printf("starting the caching process with:")
	log.Printf("Key=%s\n", action.Key)
	log.Printf("Artifacts=%v\n", action.Artifacts)
	log.Printf("Bucket=%s\n", action.Bucket)

	switch act := action.Action; act {
	case PutAction:
		if len(action.Artifacts[0]) == 0 {
			log.Fatal("No artifacts patterns provided")
		}

		log.Printf("starting the tar process")
		if err := Tar(action.Key, action.Artifacts); err != nil {
			log.Fatal(err)
		}

		log.Printf("uploading file to s3")
		if err := PutObject(action.Key, action.Bucket, action.S3Class); err != nil {
			log.Fatal(err)
		}
	case GetAction:
		exists, err := ObjectExists(action.Key, action.Bucket)
		if err != nil {
			log.Fatal(err)
		}

		// Get and unzip if object exists
		if exists {
			log.Printf("reading from s3")
			if err := GetObject(action.Key, action.Bucket); err != nil {
				log.Fatal(err)
			}

			log.Printf("starting the untar process")
			if err := Untar(action.Key); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Printf("No caches found for the following key: %s", action.Key)
		}
	case DeleteAction:
		log.Printf("deleting from s3")
		if err := DeleteObject(action.Key, action.Bucket); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf(
			"Action \"%s\" is not allowed. Valid options are: [%s, %s, %s]",
			act,
			PutAction,
			DeleteAction,
			GetAction,
		)
	}
	log.Printf("caching process finished!")
}
