package util

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
)

func MainHandler(config Config) gin.HandlerFunc {
	var getCachedListBucketPathResponse func(path *string, ginCtx *gin.Context) *ListBucketPathResponse
	var getCachedGetKeyResponse func(key *string, ginCtx *gin.Context) *GetKeyResponse
	var serveKeyAsDirectory func(key *string, ginCtx *gin.Context)
	var serveKeyAsFile func(key *string, ginCtx *gin.Context, directoryFallback bool)

	s3Client := NewS3Client(config.S3)
	// default403FileKey := getDefault403FileKey(config)
	default404FileKey := getDefault404FileKey(config)

	isFileKeyAFolderIndex := func(fileKey string) bool {
		return strings.HasSuffix(fileKey, fmt.Sprintf("/%s", config.App.FolderIndexFileName))
	}

	getCachedListBucketPathResponse = func(path *string, ginCtx *gin.Context) *ListBucketPathResponse {
		response := s3Client.ListBucketPath(ListBucketPathRequest{
			Ctx:  ginCtx.Request.Context(),
			Path: path,
		})

		return response
	}

	getCachedGetKeyResponse = func(key *string, ginCtx *gin.Context) *GetKeyResponse {
		response := s3Client.GetKey(GetKeyRequest{
			Ctx: ginCtx.Request.Context(),
			Key: key,
		})

		return response
	}

	serveKeyAsDirectory = func(key *string, ginCtx *gin.Context) {
		response := getCachedListBucketPathResponse(key, ginCtx)

		if response.Err != nil {
			if default404FileKey != nil {
				serveKeyAsFile(default404FileKey, ginCtx, false)
			} else {
				ginCtx.Status(http.StatusInternalServerError)
			}
			return
		}

		for _, file := range response.Files {
			if isFileKeyAFolderIndex(file) {
				serveKeyAsFile(&file, ginCtx, false)
				return
			}
		}

		totalFoundItems := len(response.Files) + len(response.Folders)

		if totalFoundItems == 0 {
			if default404FileKey != nil {
				serveKeyAsFile(default404FileKey, ginCtx, false)
			} else {
				ginCtx.Status(http.StatusNotFound)
			}
			return
		}

		if config.App.EnableDirectoryListing {
			if requestAcceptsJSON(ginCtx) {
				ginCtx.JSON(200, gin.H{
					"files":   response.Files,
					"folders": response.Folders,
				})
			} else {
				ginCtx.Status(http.StatusNotFound)
			}
			return
		}

		ginCtx.Status(http.StatusForbidden)
	}

	serveKeyAsFile = func(key *string, ginCtx *gin.Context, directoryFallback bool) {
		response := getCachedGetKeyResponse(key, ginCtx)

		var noSuchKey *types.NoSuchKey
		if errors.As(response.Err, &noSuchKey) {
			if directoryFallback {
				serveKeyAsDirectory(key, ginCtx)
			} else if default404FileKey != nil && *default404FileKey != *key {
				serveKeyAsFile(default404FileKey, ginCtx, false)
			} else {
				ginCtx.Status(http.StatusNotFound)
			}
			return
		}

		if response.Err != nil {
			ginCtx.Status(http.StatusInternalServerError)
			return
		}

		for header, value := range response.Headers {
			ginCtx.Header(header, value)
		}

		ginCtx.Data(200, response.ContentType, response.Body)
	}

	return func(c *gin.Context) {
		path := strings.TrimPrefix(c.Param("path"), "/")
		pathInBucket := html.UnescapeString(fmt.Sprintf("%s%s", config.S3.Folder, path))

		if path == "" || strings.HasSuffix(pathInBucket, "/") {
			serveKeyAsDirectory(&pathInBucket, c)
		} else {
			serveKeyAsFile(&pathInBucket, c, true)
		}
	}
}

func requestAcceptsJSON(c *gin.Context) bool {
	return true
}

func getDefault403FileKey(config Config) *string {
	if config.App.Default403File != "" {
		key := fmt.Sprintf("%s%s", config.S3.Folder, config.App.Default403File)
		return &key
	}
	return nil
}

func getDefault404FileKey(config Config) *string {
	if config.App.Default404File != "" {
		key := fmt.Sprintf("%s%s", config.S3.Folder, config.App.Default404File)
		return &key
	}
	return nil
}
