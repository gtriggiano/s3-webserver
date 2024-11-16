package s3Proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var (
	windowLocationRedirectRegex = regexp.MustCompile(`^<script>window\.location\.href="(.+)"<\/script>$`)
)

// S3Proxy implements the capability to answer to an HTTP request serving
// content from S3
type S3Proxy struct {
	s3Client                *s3Port
	config                  *parsedS3ProxyConfig
	logger                  *zap.SugaredLogger
	getKeyResponses         *cache.Cache
	listBucketPathResponses *cache.Cache
	s3BytesCounter          *prometheus.CounterVec
	s3ResponsesCounter      *prometheus.CounterVec
}

func NewS3Proxy(configFile string, logger *zap.SugaredLogger, registry *prometheus.Registry) (*S3Proxy, error) {
	config, err := newS3ProxyConfig(configFile)

	if err != nil {
		return nil, err
	}

	parsedConfig := config.Parsed()

	s3BytesCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "s3_webserver_s3_bytes",
			Help: "Total bytes downloaded from S3",
		},
		[]string{"host"},
	)
	s3ResponsesCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "s3_webserver_s3_responses",
			Help: "Total number of times that the webserver needed a response from S3",
		},
		[]string{"fromCache", "host", "type"},
	)

	registry.MustRegister(s3BytesCounter)
	registry.MustRegister(s3ResponsesCounter)

	return &S3Proxy{
		s3Client:                newS3Port(),
		config:                  parsedConfig,
		logger:                  logger,
		getKeyResponses:         cache.New(parsedConfig.CacheTTL, parsedConfig.CacheCleanupInterval),
		listBucketPathResponses: cache.New(parsedConfig.CacheTTL, parsedConfig.CacheCleanupInterval),
		s3BytesCounter:          s3BytesCounter,
		s3ResponsesCounter:      s3ResponsesCounter,
	}, nil
}

func (proxy *S3Proxy) Answer(ctx *gin.Context) {
	if redirectUrl := proxy.getRedirectUrl(ctx); redirectUrl != nil {
		ctx.Redirect(http.StatusMovedPermanently, redirectUrl.String())
		return
	}

	urlPath := html.UnescapeString(ctx.Request.URL.Path)
	urlPathHasTrailingSlash := strings.HasSuffix(urlPath, "/")
	cleanUrlPath := path.Clean(urlPath)

	if shouldRedirectToCleanPath(urlPath, cleanUrlPath) {
		if urlPathHasTrailingSlash {
			ctx.Redirect(http.StatusMovedPermanently, fmt.Sprintf("%s/", cleanUrlPath))
		} else {
			ctx.Redirect(http.StatusMovedPermanently, cleanUrlPath)
		}
		return
	}

	if urlPathHasTrailingSlash {
		proxy.handleUrlPathAsDirectory(ctx, urlPath)
	} else {
		proxy.handleUrlPathAsAsFile(ctx, urlPath, true)
	}
}

func (proxy *S3Proxy) CheckHealth() error {
	maxKeys := int32(1)
	pathDelimiter := "/"
	prefix := "puppa"

	_, err := proxy.s3Client.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket:    &proxy.s3Client.bucket,
		MaxKeys:   &maxKeys,
		Delimiter: &pathDelimiter,
		Prefix:    &prefix,
	})

	return err
}

func (proxy *S3Proxy) getRedirectUrl(ctx *gin.Context) *url.URL {
	for from, to := range proxy.config.Redirects {
		if haveSamePathAndQueryVariables(ctx.Request.URL, from) {
			return to
		}
	}

	return nil
}

func (proxy *S3Proxy) serve404(ctx *gin.Context) {
	if proxy.config.Default404FilePath != "" {
		proxy.handleUrlPathAsAsFile(ctx, proxy.config.Default404FilePath, false)
	} else {
		ctx.Status(http.StatusNotFound)
	}
}

func (proxy *S3Proxy) serve404or500(ctx *gin.Context) {
	if proxy.config.Default404FilePath != "" {
		proxy.handleUrlPathAsAsFile(ctx, proxy.config.Default404FilePath, false)
	} else {
		ctx.Status(http.StatusInternalServerError)
	}
}

func (proxy *S3Proxy) isTheDefault404File(key string) bool {
	return key == proxy.config.Default404FilePath
}

func (proxy *S3Proxy) handleUrlPathAsDirectory(ctx *gin.Context, urlPath string) {
	shouldServeCacheHeader := proxy.shouldServeCacheHeader(urlPath)
	listBucketPathResponse, fromCache := proxy.getListBucketPathResponse(urlPath)

	proxy.s3ResponsesCounter.With(prometheus.Labels{
		"fromCache": strconv.FormatBool(fromCache),
		"host":      ctx.Request.Host,
		"type":      "ListBucket",
	}).Inc()

	if listBucketPathResponse.Err != nil {
		proxy.serve404or500(ctx)
		return
	}

	for _, file := range listBucketPathResponse.Files {
		if proxy.isFileKeyAFolderIndex(file) {
			proxy.handleUrlPathAsAsFile(ctx, path.Join(urlPath, proxy.config.FolderIndexFileName), false)
			return
		}
	}

	totalFoundItems := len(listBucketPathResponse.Files) + len(listBucketPathResponse.Folders)

	responseHeaders := make(map[string]string)
	if shouldServeCacheHeader {
		responseHeaders["cache-control"] = proxy.config.CacheControlHeaderForCachedFiles
	} else {
		responseHeaders["cache-control"] = proxy.config.CacheControlHeaderForNonCachedFiles
	}

	if proxy.config.EnableDirectoryListing {
		ctx.JSON(http.StatusOK, gin.H{
			"files":   listBucketPathResponse.Files,
			"folders": listBucketPathResponse.Folders,
		})
	} else {
		if totalFoundItems == 0 {
			proxy.serve404(ctx)
			return
		}
		ctx.Status(http.StatusForbidden)
	}
}

func (proxy *S3Proxy) handleUrlPathAsAsFile(ctx *gin.Context, urlPath string, fallbackToDirectoryServing bool) {
	shouldServeCacheHeader := proxy.shouldServeCacheHeader(urlPath)
	getKeyResponse, fromCache := proxy.getGetKeyResponse(urlPath)

	if !fromCache {
		proxy.s3BytesCounter.With(prometheus.Labels{
			"host": ctx.Request.Host,
		}).Add(float64(len(getKeyResponse.Body)))
	}
	proxy.s3ResponsesCounter.With(prometheus.Labels{
		"fromCache": strconv.FormatBool(fromCache),
		"host":      ctx.Request.Host,
		"type":      "GetKey",
	}).Inc()

	var noSuchKey *types.NoSuchKey
	if errors.As(getKeyResponse.Err, &noSuchKey) {
		if fallbackToDirectoryServing {
			proxy.handleUrlPathAsDirectory(ctx, urlPath)
		} else if !proxy.isTheDefault404File(urlPath) {
			proxy.serve404(ctx)
		} else {
			ctx.Status(http.StatusNotFound)
		}
		return
	}

	if getKeyResponse.Err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	if proxy.config.HandleWindowLocationRedirects && proxy.isFileKeyAFolderIndex(urlPath) {
		match := windowLocationRedirectRegex.FindStringSubmatch(string(getKeyResponse.Body))
		if len(match) > 1 {
			ctx.Redirect(http.StatusMovedPermanently, match[1])
			return
		}
	}

	responseHeaders := make(map[string]string)
	for k, v := range getKeyResponse.Headers {
		responseHeaders[k] = v
	}

	if shouldServeCacheHeader {
		responseHeaders["cache-control"] = proxy.config.CacheControlHeaderForCachedFiles
	} else {
		responseHeaders["cache-control"] = proxy.config.CacheControlHeaderForNonCachedFiles
	}

	if proxy.isTheDefault404File(urlPath) {
		ctx.DataFromReader(
			http.StatusNotFound,
			int64(len(getKeyResponse.Body)),
			getKeyResponse.ContentType,
			bytes.NewReader(getKeyResponse.Body),
			responseHeaders,
		)
	} else {
		ctx.DataFromReader(
			http.StatusOK,
			int64(len(getKeyResponse.Body)),
			getKeyResponse.ContentType,
			bytes.NewReader(getKeyResponse.Body),
			responseHeaders,
		)
	}
}

func (proxy *S3Proxy) getListBucketPathResponse(urlPath string) (*ListBucketPathResponse, bool) {
	bucketKey := path.Join(proxy.config.S3Folder, urlPath)

	if cachedResult, found := proxy.listBucketPathResponses.Get(bucketKey); found {
		return cachedResult.(*ListBucketPathResponse), true
	}

	start := time.Now()
	response := proxy.s3Client.ListBucketPath(bucketKey)
	elapsed := time.Since(start).Milliseconds()

	if proxy.config.LogS3Requests {
		l := proxy.logger.With(
			"key", bucketKey,
			"ms", elapsed,
		)

		if response.Err != nil {
			l.With("error", response.Err).Error("could not list directory")
		} else {
			l.Info("directory listed")
		}
	}

	var noSuchKey *types.NoSuchKey
	shouldCacheResult := (response.Err == nil || errors.As(response.Err, &noSuchKey)) && proxy.shouldCacheS3Result(urlPath)

	if shouldCacheResult {
		proxy.listBucketPathResponses.Set(bucketKey, response, proxy.config.CacheTTL)
	}

	return response, false
}

func (proxy *S3Proxy) getGetKeyResponse(urlPath string) (*GetKeyResponse, bool) {
	bucketKey := path.Join(proxy.config.S3Folder, urlPath)

	if cachedResult, found := proxy.getKeyResponses.Get(bucketKey); found {
		return cachedResult.(*GetKeyResponse), true
	}

	start := time.Now()
	response := proxy.s3Client.GetKey(bucketKey)
	elapsed := time.Since(start).Milliseconds()

	if proxy.config.LogS3Requests {
		l := proxy.logger.With(
			"key", bucketKey,
			"ms", elapsed,
		)

		if response.Err != nil {
			l.With("error", response.Err).Error("could not download file")
		} else {
			l.Info("file downloaded")
		}
	}

	var noSuchKey *types.NoSuchKey
	shouldCacheResult := (response.Err == nil || errors.As(response.Err, &noSuchKey)) && proxy.shouldCacheS3Result(urlPath)

	if shouldCacheResult {
		proxy.getKeyResponses.Set(bucketKey, response, proxy.config.CacheTTL)
	}

	return response, false
}

func (proxy *S3Proxy) isFileKeyAFolderIndex(key string) bool {
	return strings.HasSuffix(key, fmt.Sprintf("/%s", proxy.config.FolderIndexFileName))
}

func (proxy *S3Proxy) shouldCacheS3Result(urlPath string) bool {
	if proxy.config.ImmutableTree {
		return true
	}

	return proxy.shouldServeCacheHeader(urlPath)
}

func (proxy *S3Proxy) shouldServeCacheHeader(urlPath string) bool {
	if proxy.config.CacheFiles {
		for i := 0; i < len(proxy.config.NeverCachedPathsRegex); i++ {
			if proxy.config.NeverCachedPathsRegex[i].MatchString(urlPath) {
				return false
			}
		}

		for i := 0; i < len(proxy.config.CachedPathsRegex); i++ {
			if proxy.config.CachedPathsRegex[i].MatchString(urlPath) {
				return true
			}
		}
	}

	return false
}

func shouldRedirectToCleanPath(path, cleanPath string) bool {
	cleanPathWithTrailinSlash := fmt.Sprintf("%s/", cleanPath)
	if path != cleanPath && path != cleanPathWithTrailinSlash {
		return true
	}
	return false
}

func haveSamePathAndQueryVariables(url1, url2 *url.URL) bool {
	if url1.Path != url2.Path {
		return false
	}
	// Compare paths
	if url1.Path != url2.Path {
		return false
	}

	// Parse and compare query parameters
	query1 := url1.Query()
	query2 := url2.Query()

	return reflect.DeepEqual(query1, query2)
}
