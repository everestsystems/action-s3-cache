package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pkg/errors"
)

// PutObject - Upload object to s3 bucket
func PutObject(key, bucket, s3Class string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	session := s3.NewFromConfig(cfg)

	file, err := os.Open(key)
	if err != nil {
		return err
	}
	// lol
	defer file.Close()

	i := &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		Body:         file,
		StorageClass: types.StorageClass(s3Class),
	}

	_, err = session.PutObject(context.TODO(), i)
	if err == nil {
		log.Print("Cache saved successfully")
	}

	return err
}

// GetObject - Get object from s3 bucket
func GetObject(key, bucket string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	session := s3.NewFromConfig(cfg)

	result, err := session.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("Couldn't get object %v:%v: %v\n", bucket, key, err)
		return err
	}
	defer result.Body.Close()
	file, err := os.Create(key)
	if err != nil {
		log.Printf("Couldn't create file %v: %v\n", key, err)
		return err
	}
	defer file.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		log.Printf("Couldn't read object body from %v: %v\n", key, err)
	}

	_, err = file.Write(body)
	if err == nil {
		log.Printf("Cache downloaded successfully, containing %d bytes", result.ContentLength)
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
