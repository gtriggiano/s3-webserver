package util

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

func S3Handler(config Config) gin.HandlerFunc {
	var getCachedListBucketPathResponse func(path string, ginCtx *gin.Context) *ListBucketPathResponse
	var getCachedGetKeyResponse func(key string, ginCtx *gin.Context) *GetKeyResponse
	var serveKeyAsDirectory func(key string, ginCtx *gin.Context)
	var serveKeyAsFile func(key string, ginCtx *gin.Context, directoryFallback bool)

	GATSBY_REDIRECT_REGEX, _ := regexp.Compile(`^<script>window\.location\.href="(.+)"<\/script>$`)
	s3Client := NewS3Client(config.S3)
	default404FileKey := getDefault404FileKey(config)
	cacheControlHeaderForImmutableFiles := fmt.Sprintf("public, max-age=%d, immutable", config.App.CacheControlMaxAge)
	cacheControlHeaderForMutableFiles := "public, no-cache"

	isFileKeyAFolderIndex := func(fileKey string) bool {
		return strings.HasSuffix(fileKey, fmt.Sprintf("/%s", config.App.FolderIndexFileName))
	}
	isFileKeyTheDefault404File := func(fileKey string) bool {
		return default404FileKey != nil && fileKey == *default404FileKey
	}

	cacheDefaultExpiration := time.Duration(config.S3.CacheTTL) * time.Second
	cacheCleanupInteval := time.Duration(config.S3.CacheTTL+300) * time.Second
	cacheKeysExpiration := cacheDefaultExpiration
	if config.S3.ImmutableTree {
		cacheKeysExpiration = cache.NoExpiration
	}
	listBucketPathResponsesCache := cache.New(cacheDefaultExpiration, cacheCleanupInteval)
	getKeyResponsesCache := cache.New(cacheDefaultExpiration, cacheCleanupInteval)

	getCachedListBucketPathResponse = func(path string, ginCtx *gin.Context) *ListBucketPathResponse {
		if config.S3.CacheResponses || config.S3.ImmutableTree {
			cachedResult, found := listBucketPathResponsesCache.Get(path)
			if found {
				return cachedResult.(*ListBucketPathResponse)
			}
		}

		response := s3Client.ListBucketPath(path)

		if config.S3.CacheResponses || config.S3.ImmutableTree {
			if response.Err == nil {
				listBucketPathResponsesCache.Set(path, response, cacheKeysExpiration)
			}
		}

		return response
	}

	getCachedGetKeyResponse = func(key string, ginCtx *gin.Context) *GetKeyResponse {
		if config.S3.CacheResponses || config.S3.ImmutableTree {
			cachedResult, found := getKeyResponsesCache.Get(key)
			if found {
				return cachedResult.(*GetKeyResponse)
			}
		}

		response := s3Client.GetKey(key)

		if config.S3.CacheResponses || config.S3.ImmutableTree {
			if response.Err != nil {
				var noSuchKeyError *types.NoSuchKey
				if errors.As(response.Err, &noSuchKeyError) && config.S3.ImmutableTree {
					getKeyResponsesCache.Set(key, response, cacheKeysExpiration)
				}
			} else {
				getKeyResponsesCache.Set(key, response, cacheKeysExpiration)
			}
		}

		return response
	}

	serveKeyAsDirectory = func(key string, ginCtx *gin.Context) {
		response := getCachedListBucketPathResponse(key, ginCtx)

		if response.Err != nil {
			if default404FileKey != nil {
				serveKeyAsFile(*default404FileKey, ginCtx, false)
			} else {
				ginCtx.Status(http.StatusInternalServerError)
			}
			return
		}

		for _, file := range response.Files {
			if isFileKeyAFolderIndex(file) {
				serveKeyAsFile(file, ginCtx, false)
				return
			}
		}

		totalFoundItems := len(response.Files) + len(response.Folders)

		if totalFoundItems == 0 {
			if default404FileKey != nil {
				serveKeyAsFile(*default404FileKey, ginCtx, false)
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

	serveKeyAsFile = func(key string, ginCtx *gin.Context, directoryFallback bool) {
		requestPath := ginCtx.Request.URL.Path
		response := getCachedGetKeyResponse(key, ginCtx)

		var noSuchKey *types.NoSuchKey
		if errors.As(response.Err, &noSuchKey) {
			if directoryFallback {
				serveKeyAsDirectory(key, ginCtx)
			} else if default404FileKey != nil && !isFileKeyTheDefault404File(key) {
				serveKeyAsFile(*default404FileKey, ginCtx, false)
			} else {
				ginCtx.Status(http.StatusNotFound)
			}
			return
		}

		if response.Err != nil {
			ginCtx.Status(http.StatusInternalServerError)
			return
		}

		if config.App.HandleGatsbyRedirects && isFileKeyAFolderIndex(key) {
			match := GATSBY_REDIRECT_REGEX.FindStringSubmatch(string(response.Body))
			if len(match) > 1 {
				ginCtx.Redirect(http.StatusMovedPermanently, match[1])
				return
			}
		}

		responseHeaders := make(map[string]string)
		for k, v := range response.Headers {
			responseHeaders[k] = v
		}

		if !isFileKeyTheDefault404File(key) &&
			isRequestPathEligibleForImmutableCaching(requestPath, config.App) &&
			!isRequestPathBlacklistedFromImmutableCaching(requestPath, config.App) {

			responseHeaders["cache-control"] = cacheControlHeaderForImmutableFiles
		} else {
			responseHeaders["cache-control"] = cacheControlHeaderForMutableFiles
		}

		if isFileKeyTheDefault404File(key) {
			ginCtx.DataFromReader(
				http.StatusNotFound,
				int64(len(response.Body)),
				response.ContentType,
				bytes.NewReader(response.Body),
				responseHeaders,
			)
		} else {
			ginCtx.DataFromReader(
				http.StatusOK,
				int64(len(response.Body)),
				response.ContentType,
				bytes.NewReader(response.Body),
				responseHeaders,
			)
		}
	}

	return func(c *gin.Context) {
		originalPath := html.UnescapeString(c.Request.URL.Path)
		originalPathHasTrailingSlash := strings.HasSuffix(originalPath, "/")
		cleanPath := path.Clean(originalPath)

		if shouldRedirectToCleanPath(originalPath, cleanPath) {
			if originalPathHasTrailingSlash {
				c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("%s/", cleanPath))
			} else {
				c.Redirect(http.StatusMovedPermanently, cleanPath)
			}

			return
		}

		pathInBucket := path.Join(config.S3.Folder, originalPath)

		if originalPathHasTrailingSlash {
			serveKeyAsDirectory(pathInBucket, c)
		} else {
			serveKeyAsFile(pathInBucket, c, true)
		}
	}
}

func shouldRedirectToCleanPath(originalPath, cleanPath string) bool {
	cleanPathWithTrailinSlash := fmt.Sprintf("%s/", cleanPath)
	if originalPath != cleanPath && originalPath != cleanPathWithTrailinSlash {
		return true
	}
	return false
}

func requestAcceptsJSON(c *gin.Context) bool {
	return true
}

func getDefault404FileKey(config Config) *string {
	if config.App.Default404File != "" {
		key := path.Join(config.S3.Folder, config.App.Default404File)
		return &key
	}
	return nil
}

func isRequestPathEligibleForImmutableCaching(path string, config AppConfig) bool {
	for _, regexp := range config.CacheControlRegexpList {
		if regexp.Match([]byte(path)) {
			return true
		}
	}
	return false
}

func isRequestPathBlacklistedFromImmutableCaching(path string, config AppConfig) bool {
	for _, regexp := range config.CacheControlRegexpBlacklist {
		if regexp.Match([]byte(path)) {
			return true
		}
	}
	return false
}
