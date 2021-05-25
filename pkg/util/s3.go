package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	s3            *s3.Client
	bucket        *string
	pathDelimiter *string
}

func NewS3Client(config S3Config) *S3Client {
	awsConfig := aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider(
			config.AccessKeyID,
			config.SecretAccessKey,
			"",
		),
		Region: config.Region,
	}

	if config.Endpoint != "" {
		awsConfig.EndpointResolver = aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				HostnameImmutable: config.ForcePathStyle,
				SigningRegion:     region,
				URL:               config.Endpoint,
			}, nil
		})
	}

	pathDelimiter := "/"

	return &S3Client{
		s3:            s3.NewFromConfig(awsConfig),
		bucket:        &config.Bucket,
		pathDelimiter: &pathDelimiter,
	}
}

type GetKeyRequest struct {
	Ctx context.Context
	Key *string
}

type GetKeyResponse struct {
	Err         error
	Body        []byte
	ContentType string
	Headers     map[string]string
}

func (c *S3Client) GetKey(req GetKeyRequest) *GetKeyResponse {
	headers := make(map[string]string)

	output, err := c.s3.GetObject(req.Ctx, &s3.GetObjectInput{
		Bucket: c.bucket,
		Key:    req.Key,
	})

	if err != nil {
		return &GetKeyResponse{
			Err:         err,
			Body:        make([]byte, 0),
			ContentType: "",
			Headers:     headers,
		}
	}

	body, err := ioutil.ReadAll(output.Body)

	headers["content-length"] = strconv.Itoa(int(output.ContentLength))
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

	return &GetKeyResponse{
		Err:         err,
		Body:        body,
		ContentType: *output.ContentType,
		Headers:     headers,
	}
}

type ListBucketPathRequest struct {
	Ctx  context.Context
	Path *string
}

type ListBucketPathResponse struct {
	Err     error
	Files   []string
	Folders []string
}

func (c *S3Client) ListBucketPath(req ListBucketPathRequest) *ListBucketPathResponse {
	path := *req.Path
	if strings.HasSuffix(path, "/") == false {
		path = fmt.Sprintf("%s/", path)
	}

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

		if output.IsTruncated {
			output, err := c.s3.ListObjectsV2(req.Ctx, &s3.ListObjectsV2Input{
				Bucket:            c.bucket,
				MaxKeys:           1000000,
				Delimiter:         c.pathDelimiter,
				Prefix:            &path,
				ContinuationToken: output.ContinuationToken,
			})
			populateResults(output, err)
		}
	}

	output, err := c.s3.ListObjectsV2(req.Ctx, &s3.ListObjectsV2Input{
		Bucket:     c.bucket,
		MaxKeys:    1000000,
		Delimiter:  c.pathDelimiter,
		Prefix:     &path,
		StartAfter: &path,
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
