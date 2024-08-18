package app

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	DefaultConfigFilePath                    = ".env"
	DefaultHttpServerCertificateFilePath     = "server.crt"
	DefaultHttpServerKeyFilePath             = "server.key"
	DefaultHttpServerReadTimeout             = "15m"
	DefaultHttpServerWriteTimeout            = "15m"
	DefaultHttpServerGracefulShutdownTimeout = "2m"
	DefaultAppRestApiPort                    = "3005"
	DefaultDatabaseQueryTimeout              = "30s"

	// Current implementation of assets storing is based on large objects (see for details https://www.postgresql.org/docs/current/largeobjects.html).
	// A large object cannot exceed 4TB for PostgreSQL 9.3 or newer or 2GB for older versions.
	// Body max size should be configurable for different environments with appropriate system resources.
	// Body could contain multiple files which in sum should not exceed above limits.
	// TODO: add configuration of body max size
	DefaultBodyMaxSize = 1024 * 1024 * 1024 * 10 // 10 GB
)

var configRowRegExp = regexp.MustCompile(`(.+)=(.+)`)

func SetUpEnvVarsFromConfig() error {
	configFilePath, ok := os.LookupEnv("CONFIG_FILE_PATH")
	if !ok {
		configFilePath = DefaultConfigFilePath
	}

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("unable to read file '%v': %w", configFilePath, err)
	}

	input := string(data)
	rows := strings.Split(input, "\n")
	for _, row := range rows {
		submatches := configRowRegExp.FindStringSubmatch(row)
		if len(submatches) == 3 {
			paramName := strings.Trim(submatches[1], " ")
			paramValue := strings.Trim(submatches[2], " ")
			err := os.Setenv(paramName, paramValue)
			if err != nil {
				return fmt.Errorf("unable to set environment variable '%v': %w", paramName, err)
			}
		}
	}
	return nil
}

func NewHttpServerConfig() (*HttpServerConfig, error) {
	host, ok := os.LookupEnv("APP_REST_API_PORT")
	if !ok {
		host = DefaultAppRestApiPort
	}
	certPath, ok := os.LookupEnv("APP_TLS_CERT_PATH")
	if !ok {
		certPath = DefaultHttpServerCertificateFilePath
	}
	keyPath, ok := os.LookupEnv("APP_TLS_KEY_PATH")
	if !ok {
		keyPath = DefaultHttpServerKeyFilePath
	}
	readTimeoutStr, ok := os.LookupEnv("APP_SERVER_READ_TIMEOUT")
	if !ok {
		readTimeoutStr = DefaultHttpServerReadTimeout
	}
	readTimeout, err := time.ParseDuration(readTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("unable to configure server read timeout: %w", err)
	}
	writeTimeoutStr, ok := os.LookupEnv("APP_SERVER_WRITE_TIMEOUT")
	if !ok {
		writeTimeoutStr = DefaultHttpServerWriteTimeout
	}
	writeTimeout, err := time.ParseDuration(writeTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("unable to configure server write timeout: %w", err)
	}
	gracefulShutdownTimeoutStr, ok := os.LookupEnv("APP_SERVER_GRACEFUL_SHUTDOWN_TIMEOUT")
	if !ok {
		gracefulShutdownTimeoutStr = DefaultHttpServerGracefulShutdownTimeout
	}
	gracefulShutdownTimeout, err := time.ParseDuration(gracefulShutdownTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("unable to configure server graceful shutdown timeout: %w", err)
	}
	return &HttpServerConfig{
		Host:                    fmt.Sprintf(":%s", host),
		CertificateFilePath:     certPath,
		KeyFilePath:             keyPath,
		ReadTimeout:             readTimeout,
		WriteTimeout:            writeTimeout,
		GracefulShutdownTimeout: gracefulShutdownTimeout,
	}, nil
}
