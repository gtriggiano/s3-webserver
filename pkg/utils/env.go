package utils

import (
	"log"
	"os"
	"strconv"
)

func GetEnv(key string, required bool) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	if required {
		log.Fatalf("%s environment variable is required", key)
	}
	return ""
}

func GetEnvAsBool(key string, defaultValue bool) bool {
	valueStr := GetEnv(key, false)
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}
