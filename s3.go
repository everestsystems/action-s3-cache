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

// newS3Client creates an S3 client with optional custom endpoint support
func newS3Client() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	endpoint := os.Getenv("ENDPOINT")
	if endpoint != "" {
		return s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}), nil
	}

	return s3.NewFromConfig(cfg), nil
}

// PutObject - Upload object to s3 bucket
func PutObject(key, bucket, s3Class string) error {
	client, err := newS3Client()
	if err != nil {
		return err
	}

	file, err := os.Open(key)
	if err != nil {
		return err
	}
	defer file.Close()

	i := &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		Body:         file,
		StorageClass: types.StorageClass(s3Class),
	}

	_, err = client.PutObject(context.TODO(), i)
	if err == nil {
		log.Print("Cache saved successfully")
	}

	return err
}

// GetObject - Get object from s3 bucket
func GetObject(key, bucket string) error {
	client, err := newS3Client()
	if err != nil {
		return err
	}

	result, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
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
	written, err := io.Copy(file, result.Body)
	if err != nil {
		log.Printf("Couldn't write object body to %v: %v\n", key, err)
		return err
	}

	log.Printf("Cache downloaded successfully, containing %d bytes", written)
	return nil
}

// DeleteObject - Delete object from s3 bucket
func DeleteObject(key, bucket string) error {
	client, err := newS3Client()
	if err != nil {
		return err
	}

	i := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err = client.DeleteObject(context.TODO(), i)
	if err == nil {
		log.Print("Cache purged successfully")
	}

	return err
}

// ObjectExists - Verify if object exists in s3
func ObjectExists(key, bucket string) (bool, error) {
	client, err := newS3Client()
	if err != nil {
		return false, err
	}

	i := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	if _, err = client.HeadObject(context.TODO(), i); err != nil {
		var nsk *types.NotFound
		if errors.As(err, &nsk) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
