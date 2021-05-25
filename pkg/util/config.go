package util

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	Required string = "@@RequiredVariable"
)

func init() {
	godotenv.Load()
}

type Config struct {
	App AppConfig
	S3  S3Config
}

type AppConfig struct {
	CacheControlMaxAge          int
	CacheControlRegexpBlacklist []*regexp.Regexp
	CacheControlRegexpList      []*regexp.Regexp
	Default403File              string
	Default404File              string
	EnableDirectoryListing      bool
	FolderIndexFileName         string
	HandleGatsbyRedirects       bool
	HTTPPort                    int
	LogHTTPRequests             bool
	TrustProxy                  bool
}

type S3Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	CacheResponses  bool
	CacheTTL        int
	Endpoint        string
	Folder          string
	ForcePathStyle  bool
	ImmutableTree   bool
	Region          string
}

func LoadConfig() Config {
	appConfig := AppConfig{
		CacheControlMaxAge:          getEnvAsInt("CACHE_CONTROL_MAX_AGE", 63072000),
		CacheControlRegexpBlacklist: getEnvAsRegexpList("CACHE_CONTROL_REGEXP_BLACKLIST", "|||"),
		CacheControlRegexpList:      getEnvAsRegexpList("CACHE_CONTROL_REGEXP_BLACKLIST", "|||"),
		Default403File:              getEnv("DEFAULT_404_FILE", ""),
		Default404File:              getEnv("DEFAULT_404_FILE", ""),
		EnableDirectoryListing:      getEnvAsBool("ENABLE_DIRECTORY_LISTING", false),
		FolderIndexFileName:         getEnv("FOLDER_INDEX_FILE_NAME", "index.html"),
		HandleGatsbyRedirects:       getEnvAsBool("HANDLE_GATSBY_REDIRECTS", false),
		HTTPPort:                    getEnvAsInt("HTTP_PORT", 80),
		LogHTTPRequests:             getEnvAsBool("LOG_HTTP_REQUESTS", true),
		TrustProxy:                  getEnvAsBool("TRUST_PROXY", false),
	}

	s3Folder := strings.TrimPrefix(
		strings.TrimSuffix(getEnv("S3_FOLDER", Required), "/"),
		"/",
	)

	if s3Folder != "" {
		s3Folder = fmt.Sprintf("%s/", s3Folder)
	}

	s3Config := S3Config{
		AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", Required),
		SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", Required),
		Bucket:          getEnv("S3_BUCKET", Required),
		CacheResponses:  getEnvAsBool("S3_CACHE_RESPONSES", true),
		CacheTTL:        getEnvAsInt("S3_CACHE_TTL", 60),
		Endpoint:        getEnv("S3_ENDPOINT", Required),
		Folder:          s3Folder,
		ForcePathStyle:  getEnvAsBool("S3_FORCE_PATH_STYLE", false),
		ImmutableTree:   getEnvAsBool("S3_FORCE_PATH_STYLE", false),
		Region:          getEnv("S3_REGION", ""),
	}

	return Config{
		App: appConfig,
		S3:  s3Config,
	}
}

func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	if defaultValue == Required {
		log.Fatalf("%s environment variable is required", key)
	}
	return defaultValue
}
func getEnvAsSlice(name string, defaultVal []string, sep string) []string {
	valueStr := getEnv(name, "")
	if valueStr == "" {
		return defaultVal
	}
	val := strings.Split(valueStr, sep)
	return val
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsRegexpList(key string, sep string) []*regexp.Regexp {
	valueSlice := getEnvAsSlice(key, make([]string, 0), sep)
	regexpList := make([]*regexp.Regexp, 0, len(valueSlice))
	for _, expr := range valueSlice {
		if regex, err := regexp.Compile(expr); err != nil {
			regexpList = append(regexpList, regex)
		}
	}
	return regexpList
}
