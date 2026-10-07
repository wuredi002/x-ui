package config

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
)

//go:embed version
var version string

//go:embed name
var name string

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

func GetVersion() string {
	return strings.TrimSpace(version)
}

func GetName() string {
	return strings.TrimSpace(name)
}

func GetLogLevel() LogLevel {
	if IsDebug() {
		return Debug
	}
	logLevel := os.Getenv("XRAY_LOG_LEVEL")
	if logLevel == "" {
		logLevel = os.Getenv("XUI_LOG_LEVEL")
	}
	if logLevel == "" {
		return Info
	}
	return LogLevel(logLevel)
}

func IsDebug() bool {
	return os.Getenv("XRAY_DEBUG") == "true" || os.Getenv("XUI_DEBUG") == "true"
}

func GetDBPath() string {
	path := fmt.Sprintf("/etc/%s/%s.db", GetName(), GetName())
	if _, err := os.Stat(path); os.IsNotExist(err) {
		legacyPath := "/etc/x-ui/x-ui.db"
		if _, legacyErr := os.Stat(legacyPath); legacyErr == nil {
			return legacyPath
		}
	}
	return path
}
