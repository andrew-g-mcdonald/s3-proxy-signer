package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const presignDuration = 60 * time.Minute

func base(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	queryStringMap := r.URL.Query()

	if _, ok := queryStringMap["region"]; !ok {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if _, ok := queryStringMap["object"]; !ok {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	region := queryStringMap["region"][0]
	path := queryStringMap["object"][0]

	s := strings.Split(path, "/")
	bucket := s[0]
	key := strings.Join(s[1:], "/")

	// Create AWS config using env vars and provide the url configured region
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		fmt.Println(err)
	}

	// Create an Amazon S3 service client
	client := s3.NewFromConfig(cfg)

	// Use head object to check if the object asctually exists
	_, err = client.HeadObject(context.TODO(), &s3.HeadObjectInput{Bucket: &bucket, Key: &key})
	if err != nil {
		fmt.Printf("Bucket: %s Key: %s not found", bucket, key)
		http.NotFound(w, r)
		return
	}

	// Create Presign client and generate the URL with expiry
	presignClient := s3.NewPresignClient(client)
	presignObject, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{Bucket: &bucket, Key: &key}, s3.WithPresignExpires(presignDuration))
	if err != nil {
		fmt.Println("Error signing")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, presignObject.URL, 302)
	return
}
