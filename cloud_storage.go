package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
)

type cloudStorageConfig struct {
	client      *storage.Client
	bucket      *storage.BucketHandle
	bucketName  string
	clientEmail string
}

func newGoogleCloudStorage(projectID, bucketName, clientEmail string) (*cloudStorageConfig, error) {
	ctx := context.Background()

	client, err := storage.NewClient(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to create client: %v", err)
	}

	return &cloudStorageConfig{client: client, bucket: client.Bucket(bucketName), bucketName: bucketName, clientEmail: clientEmail}, nil
}

func (gc *cloudStorageConfig) uploadFile(id string, filename string, file multipart.File) (err error) {
	ctx := context.Background()

	uploadObject := gc.bucket.Object(id + "/" + filename)
	bucketWriter := uploadObject.NewWriter(ctx)

	if _, err = io.Copy(bucketWriter, file); err != nil {
		return
	}
	bucketWriter.Close()

	if err = uploadObject.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return
	}

	return
}

func (gc *cloudStorageConfig) getSignedURL(id, filename string) (url string, err error) {
	pkey, err := ioutil.ReadFile("key.pem")

	if err != nil {
		fmt.Println(err)
		return
	}

	url, err = storage.SignedURL(gc.bucketName, id+"/"+filename, &storage.SignedURLOptions{
		GoogleAccessID: gc.clientEmail,
		PrivateKey:     pkey,
		Method:         "GET",
		Expires:        time.Now().Add(1 * time.Hour),
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(url)
	return
}
