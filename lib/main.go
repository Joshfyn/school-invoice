package lib

import (
	"fmt"
	"os"
)

func IsProduction() bool {
	return os.Getenv("ENV") == "production"
}

func GetServiceHostWithPort(svcName string) string {
	if !IsProduction() {
		return fmt.Sprintf("0.0.0.0:%s", getEnv("PORT", "8080"))
	}
	return "0.0.0.0:80"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}