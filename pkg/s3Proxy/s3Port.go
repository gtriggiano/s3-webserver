package s3Proxy

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Port struct {
	client *s3.Client
	bucket string
}

type GetKeyResponse struct {
	Err         error
	Body        []byte
	ContentType string
	Headers     map[string]string
}

type ListBucketPathResponse struct {
	Err     error
	Files   []string
	Folders []string
}

func newS3Port() *s3Port {
	config := newS3Config()

	awsConfig := aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider(
			config.AccessKeyID,
			config.SecretAccessKey,
			"",
		),
		Region: config.Region,
	}

	return &s3Port{
		client: s3.NewFromConfig(awsConfig, func(o *s3.Options) {
			o.UsePathStyle = config.ForcePathStyle
			if config.Endpoint != "" {
				o.BaseEndpoint = &config.Endpoint
			}
		}),
		bucket: config.Bucket,
	}
}

func (port *s3Port) GetKey(key string) *GetKeyResponse {
	var response *GetKeyResponse
	headers := make(map[string]string)

	output, err := port.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &port.bucket,
		Key:    &key,
	})

	if err != nil {
		response = &GetKeyResponse{
			Err:         err,
			Body:        make([]byte, 0),
			ContentType: "",
			Headers:     headers,
		}
	} else {
		body, err := io.ReadAll(output.Body)

		headers["content-length"] = strconv.FormatInt(*output.ContentLength, 10)
		if output.LastModified != nil {
			headers["last-modified"] = output.LastModified.UTC().String()
		}
		if output.Expiration != nil {
			headers["expiration"] = *output.Expiration
		}
		if output.ETag != nil {
			headers["etag"] = *output.ETag
		}
		if output.ContentEncoding != nil {
			headers["content-encoding"] = *output.ContentEncoding
		}
		if output.ContentType != nil {
			headers["content-type"] = *output.ContentType
		}

		response = &GetKeyResponse{
			Err:         err,
			Body:        body,
			ContentType: *output.ContentType,
			Headers:     headers,
		}
	}

	return response
}

func (port *s3Port) ListBucketPath(key string) *ListBucketPathResponse {
	if !strings.HasSuffix(key, "/") {
		key = fmt.Sprintf("%s/", key)
	}

	var maxKeys = int32(10000)
	var pathDelimiter = "/"

	var finalError error
	files := make([]string, 0)
	folders := make([]string, 0)

	var populateResults func(output *s3.ListObjectsV2Output, err error)

	populateResults = func(output *s3.ListObjectsV2Output, err error) {
		if err != nil {
			finalError = err
			return
		}
		for _, file := range output.Contents {
			files = append(files, *file.Key)
		}
		for _, prefix := range output.CommonPrefixes {
			folders = append(folders, *prefix.Prefix)
		}

		if *output.IsTruncated {
			output, err := port.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
				Bucket:            &port.bucket,
				MaxKeys:           &maxKeys,
				Delimiter:         &pathDelimiter,
				Prefix:            &key,
				ContinuationToken: output.ContinuationToken,
			})
			populateResults(output, err)
		}
	}

	output, err := port.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket:     &port.bucket,
		MaxKeys:    &maxKeys,
		Delimiter:  &pathDelimiter,
		Prefix:     &key,
		StartAfter: &key,
	})

	populateResults(output, err)

	sort.Strings(files)
	sort.Strings(folders)

	return &ListBucketPathResponse{
		Err:     finalError,
		Files:   files,
		Folders: folders,
	}
}
