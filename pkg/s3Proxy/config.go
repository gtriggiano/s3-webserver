package s3Proxy

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gtriggiano/s3-webserver/pkg/utils"
	"gopkg.in/yaml.v2"
)

type s3ProxyConfig struct {
	CachedPathsRegex              []string   `yaml:"cachedPathsRegex"`
	CacheFiles                    bool       `yaml:"cacheFiles"`
	CacheTTL                      int        `yaml:"cacheTTL"`
	Default404FilePath            string     `yaml:"default404FilePath"`
	EnableDirectoryListing        bool       `yaml:"enableDirectoryListing"`
	EscapePathSegments            bool       `yaml:"escapePathSegments"`
	FolderIndexFileName           string     `yaml:"folderIndexFileName"`
	HandleWindowLocationRedirects bool       `yaml:"handleWindowLocationRedirects"`
	ImmutableTree                 bool       `yaml:"immutableTree"`
	LogS3Requests                 bool       `yaml:"logS3Requests"`
	NeverCachedPathsRegex         []string   `yaml:"neverCachedPathsRegex"`
	Redirects                     []redirect `yaml:"redirects"`
}

type redirect struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type parsedS3ProxyConfig struct {
	CachedPathsRegex                    []*regexp.Regexp
	CacheFiles                          bool
	CacheTTL                            time.Duration
	CacheCleanupInterval                time.Duration
	Default404FilePath                  string
	EnableDirectoryListing              bool
	EscapePathSegments                  bool
	FolderIndexFileName                 string
	HandleWindowLocationRedirects       bool
	CacheControlHeaderForCachedFiles    string
	CacheControlHeaderForNonCachedFiles string
	ImmutableTree                       bool
	LogS3Requests                       bool
	NeverCachedPathsRegex               []*regexp.Regexp
	Redirects                           map[*url.URL]*url.URL
	S3Folder                            string
}

type s3Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Endpoint        string
	ForcePathStyle  bool
	Region          string
}

func newS3Config() *s3Config {
	return &s3Config{
		AccessKeyID:     utils.GetEnv("AWS_ACCESS_KEY_ID", true),
		SecretAccessKey: utils.GetEnv("AWS_SECRET_ACCESS_KEY", true),
		Bucket:          utils.GetEnv("S3_BUCKET", true),
		Endpoint:        utils.GetEnv("S3_ENDPOINT", false),
		ForcePathStyle:  utils.GetEnvAsBool("S3_FORCE_PATH_STYLE", false),
		Region:          utils.GetEnv("S3_REGION", false),
	}
}

func newS3ProxyConfig(configFile string) (*s3ProxyConfig, error) {
	s3ProxyConfig := &s3ProxyConfig{
		CachedPathsRegex:              make([]string, 0),
		CacheFiles:                    false,
		CacheTTL:                      0,
		Default404FilePath:            "",
		EnableDirectoryListing:        false,
		EscapePathSegments:            false,
		FolderIndexFileName:           "index.html",
		HandleWindowLocationRedirects: false,
		ImmutableTree:                 false,
		LogS3Requests:                 false,
		NeverCachedPathsRegex:         make([]string, 0),
		Redirects:                     make([]redirect, 0),
	}

	if configFile == "" {
		return s3ProxyConfig, nil
	}

	configFileData, err := os.ReadFile(filepath.Clean(configFile))

	if err != nil {
		return nil, utils.ExitError{Code: utils.EX_FAIL, Err: err}
	}

	err = yaml.Unmarshal(configFileData, s3ProxyConfig)
	if err != nil {
		return nil, utils.ExitError{Code: utils.EX_FAIL, Err: err}
	}

	return s3ProxyConfig, nil
}

func (config *s3ProxyConfig) Parsed() *parsedS3ProxyConfig {
	cachedPathsRegex := make([]*regexp.Regexp, 0)
	for i := 0; i < len(config.CachedPathsRegex); i++ {
		if r, err := regexp.Compile(config.CachedPathsRegex[i]); err == nil {
			cachedPathsRegex = append(cachedPathsRegex, r)
		}
	}

	neverCachedPathsRegex := make([]*regexp.Regexp, 0)
	for i := 0; i < len(config.NeverCachedPathsRegex); i++ {
		if r, err := regexp.Compile(config.NeverCachedPathsRegex[i]); err == nil {
			neverCachedPathsRegex = append(neverCachedPathsRegex, r)
		}
	}

	redirects := make(map[*url.URL]*url.URL)

	for i := 0; i < len(config.Redirects); i++ {
		if from, err := url.Parse(config.Redirects[i].From); err == nil {
			if to, err := url.Parse(config.Redirects[i].To); err == nil {
				redirects[from] = to
			}

		}
	}

	cacheTTL := config.CacheTTL
	cacheCleanupInterval := 60 // One minute

	if config.ImmutableTree {
		cacheTTL = 60 * 60 * 24 * 365 * 10 // Ten years
		cacheCleanupInterval = cacheTTL
	}

	cacheControlHeaderForCachedFiles := fmt.Sprintf("public, max-age=%d, immutable", config.CacheTTL)
	cacheControlHeaderForNonCachedFiles := "public, no-cache"

	return &parsedS3ProxyConfig{
		CachedPathsRegex:                    cachedPathsRegex,
		CacheFiles:                          config.CacheFiles,
		CacheTTL:                            time.Duration(cacheTTL) * time.Second,
		CacheCleanupInterval:                time.Duration(cacheCleanupInterval) * time.Second,
		Default404FilePath:                  config.Default404FilePath,
		EnableDirectoryListing:              config.EnableDirectoryListing,
		EscapePathSegments:                  config.EscapePathSegments,
		FolderIndexFileName:                 config.FolderIndexFileName,
		HandleWindowLocationRedirects:       config.HandleWindowLocationRedirects,
		CacheControlHeaderForCachedFiles:    cacheControlHeaderForCachedFiles,
		CacheControlHeaderForNonCachedFiles: cacheControlHeaderForNonCachedFiles,
		ImmutableTree:                       config.ImmutableTree,
		LogS3Requests:                       config.LogS3Requests,
		NeverCachedPathsRegex:               neverCachedPathsRegex,
		Redirects:                           redirects,
		S3Folder:                            strings.TrimPrefix(path.Clean(fmt.Sprintf("/%s", utils.GetEnv("S3_FOLDER", false))), "/"),
	}
}
