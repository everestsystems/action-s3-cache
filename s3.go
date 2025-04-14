package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pkg/errors"
)

const PART_MiBs int64 = 10 * 1024 * 1024

// PutObject - Upload object to s3 bucket
func PutObject(key, bucket, s3Class string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	session := s3.NewFromConfig(cfg)

	file, err := os.Open(key)
	if err != nil {
		return err
	}
	defer file.Close()

	uploader := manager.NewUploader(session, func(u *manager.Uploader) {
		u.PartSize = PART_MiBs
	})

	i := &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		Body:         file,
		StorageClass: types.StorageClass(s3Class),
	}

	_, err = uploader.Upload(context.TODO(), i)
	if err == nil {
		log.Print("Cache saved successfully")
	}

	return err
}

// GetObject - Get object from s3 bucket
func GetObject(key, bucket string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	session := s3.NewFromConfig(cfg)

	downloader := manager.NewDownloader(session, func(d *manager.Downloader) {
		d.PartSize = PART_MiBs
	})
	buffer := manager.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(context.TODO(), buffer, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		log.Printf("Couldn't get object %v:%v: %v\n", bucket, key, err)
		return err
	}
	file, err := os.Create(key)
	if err != nil {
		log.Printf("Couldn't create file %v: %v\n", key, err)
		return err
	}
	defer file.Close()

	_, err = file.Write(buffer.Bytes())
	if err == nil {
		log.Printf("Cache downloaded successfully, containing %d bytes", len(buffer.Bytes()))
	}
	return err
}

// DeleteObject - Delete object from s3 bucket
func DeleteObject(key, bucket string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	session := s3.NewFromConfig(cfg)

	i := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err = session.DeleteObject(context.TODO(), i)
	if err == nil {
		log.Print("Cache purged successfully")
	}

	return err
}

// ObjectExists - Verify if object exists in s3
func ObjectExists(key, bucket string) (bool, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	session := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	i := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	if _, err = session.HeadObject(context.TODO(), i); err != nil {
		var nsk *types.NotFound
		if errors.As(err, &nsk) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
