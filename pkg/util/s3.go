package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type S3Client struct {
	s3            *s3.Client
	bucket        *string
	pathDelimiter *string
	logRequests   bool
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
		logRequests:   config.LogRequests,
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
	var response *GetKeyResponse
	headers := make(map[string]string)

	start := time.Now()

	output, err := c.s3.GetObject(req.Ctx, &s3.GetObjectInput{
		Bucket: c.bucket,
		Key:    req.Key,
	})

	duration := time.Since(start)

	if err != nil {
		response = &GetKeyResponse{
			Err:         err,
			Body:        make([]byte, 0),
			ContentType: "",
			Headers:     headers,
		}
	} else {
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

		response = &GetKeyResponse{
			Err:         err,
			Body:        body,
			ContentType: *output.ContentType,
			Headers:     headers,
		}
	}

	if c.logRequests {
		var evt *zerolog.Event
		switch {
		case response.Err != nil:
			{
				evt = log.Err(response.Err)
			}
		default:
			{
				evt = log.Info()
			}
		}

		LogWithHostname(evt).
			Str("service", "S3").
			Str("operation", "GetKey").
			Str("key", *req.Key).
			Dur("responseTime", duration).
			Send()
	}

	return response
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

	start := time.Now()

	output, err := c.s3.ListObjectsV2(req.Ctx, &s3.ListObjectsV2Input{
		Bucket:     c.bucket,
		MaxKeys:    1000000,
		Delimiter:  c.pathDelimiter,
		Prefix:     &path,
		StartAfter: &path,
	})

	populateResults(output, err)

	duration := time.Since(start)

	sort.Strings(files)
	sort.Strings(folders)

	if c.logRequests {
		var evt *zerolog.Event
		switch {
		case finalError != nil:
			{
				evt = log.Err(finalError)
			}
		default:
			{
				evt = log.Info()
			}
		}

		LogWithHostname(evt).
			Str("service", "S3").
			Str("operation", "ListBucketPath").
			Str("path", path).
			Dur("responseTime", duration).
			Int("totalFiles", len(files)).
			Int("totalFolders", len(folders)).
			Send()
	}

	return &ListBucketPathResponse{
		Err:     finalError,
		Files:   files,
		Folders: folders,
	}
}
