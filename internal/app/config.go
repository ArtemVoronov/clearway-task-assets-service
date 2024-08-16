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
)

var configRowRegExp = regexp.MustCompile(`(.+)=(.+)`)

func InitAppConfig() (*ServerConfig, error) {
	configFilePath, ok := os.LookupEnv("CONFIG_FILE_PATH")
	if !ok {
		configFilePath = DefaultConfigFilePath
	}

	envConfig, err := ReadEnvConfig(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read env file: %w", err)
	}

	config, err := ParseEnvConfiguration(envConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to parse env file: %w", err)
	}

	return config, nil
}

func ReadEnvConfig(pathToEnvFile string) (map[string]string, error) {
	data, err := os.ReadFile(pathToEnvFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read file '%v': %w", pathToEnvFile, err)
	}
	input := string(data)

	rows := strings.Split(input, "\n")

	result := make(map[string]string, len(rows))
	for _, row := range rows {
		submatches := configRowRegExp.FindStringSubmatch(row)
		if len(submatches) == 3 {
			paramName := strings.Trim(submatches[1], " ")
			paramValue := strings.Trim(submatches[2], " ")
			result[paramName] = paramValue
		}
	}

	return result, nil
}

func ParseEnvConfiguration(envConfig map[string]string) (*ServerConfig, error) {
	host, ok := envConfig["APP_REST_API_PORT"]
	if !ok {
		host = DefaultAppRestApiPort
	}
	certPath, ok := envConfig["APP_TLS_CERT_PATH"]
	if !ok {
		certPath = DefaultHttpServerCertificateFilePath
	}
	keyPath, ok := envConfig["APP_TLS_KEY_PATH"]
	if !ok {
		keyPath = DefaultHttpServerKeyFilePath
	}
	readTimeoutStr, ok := envConfig["APP_SERVER_READ_TIMEOUT"]
	if !ok {
		readTimeoutStr = DefaultHttpServerReadTimeout
	}
	readTimeout, err := time.ParseDuration(readTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("unable to configure server read timeout: %w", err)
	}
	writeTimeoutStr, ok := envConfig["APP_SERVER_WRITE_TIMEOUT"]
	if !ok {
		writeTimeoutStr = DefaultHttpServerWriteTimeout
	}
	writeTimeout, err := time.ParseDuration(writeTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("unable to configure server write timeout: %w", err)
	}
	gracefulShutdownTimeoutStr, ok := envConfig["APP_SERVER_GRACEFUL_SHUTDOWN_TIMEOUT"]
	if !ok {
		gracefulShutdownTimeoutStr = DefaultHttpServerGracefulShutdownTimeout
	}
	gracefulShutdownTimeout, err := time.ParseDuration(gracefulShutdownTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("unable to configure server graceful shutdown timeout: %w", err)
	}
	return &ServerConfig{
		Host:                    fmt.Sprintf(":%s", host),
		CertificateFilePath:     certPath,
		KeyFilePath:             keyPath,
		ReadTimeout:             readTimeout,
		WriteTimeout:            writeTimeout,
		GracefulShutdownTimeout: gracefulShutdownTimeout,
	}, nil
}
