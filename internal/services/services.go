package services

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Services struct {
	AuthService   *AuthService
	UsersService  *UsersService
	AssetsService *AssetsService
}

var once sync.Once
var instance *Services

func Instance() *Services {
	once.Do(func() {
		if instance == nil {
			var err error
			instance, err = createServices()
			if err != nil {
				log.Fatalf("error during app services creating: %s", err)
			}
		}
	})
	return instance
}

func createServices() (*Services, error) {
	pgForAssets, err := initPostgreServicesForAssets()
	if err != nil {
		return nil, fmt.Errorf("unable to init postgresql services for assets: %w", err)
	}
	pgForAuth, err := initPostgreServiceBySuffix("auth")
	if err != nil {
		return nil, fmt.Errorf("unable to init postgresql services for auth: %w", err)
	}
	pgForUsers, err := initPostgreServiceBySuffix("users")
	if err != nil {
		return nil, fmt.Errorf("unable to init postgresql services for users: %w", err)
	}

	return &Services{
		AuthService:   CreateAuthService(pgForAuth),
		UsersService:  CreateUsersService(pgForUsers),
		AssetsService: CreateAssetsService(pgForAssets),
	}, nil
}

func initPostgreServicesForAssets() ([]*PostgreSQLService, error) {
	pgServices := make([]*PostgreSQLService, 0, DEFAULT_BUCKET_FACTOR)
	for i := 1; i <= DEFAULT_BUCKET_FACTOR; i++ {
		pgService, err := initPostgreServiceBySuffix(strconv.Itoa(i))
		if err != nil {
			return nil, fmt.Errorf("unable to create postgresql service for assets shard '%v': %w", i, err)
		}
		pgServices = append(pgServices, pgService)
	}
	return pgServices, nil
}

func initPostgreServiceBySuffix(databaseSuffix string) (*PostgreSQLService, error) {
	pgHost, ok := os.LookupEnv("DATABASE_HOST")
	if !ok || len(strings.Trim(pgHost, " ")) == 0 {
		return nil, fmt.Errorf("missed 'DATABASE_HOST' parameter")
	}
	pgPort, ok := os.LookupEnv("DATABASE_PORT")
	if !ok || len(strings.Trim(pgPort, " ")) == 0 {
		return nil, fmt.Errorf("missed 'DATABASE_PORT' parameter")
	}
	pgUser, ok := os.LookupEnv("DATABASE_USER")
	if !ok || len(strings.Trim(pgUser, " ")) == 0 {
		return nil, fmt.Errorf("missed 'DATABASE_USER' parameter")
	}
	pgPassword, ok := os.LookupEnv("DATABASE_PASSWORD")
	if !ok || len(strings.Trim(pgPassword, " ")) == 0 {
		return nil, fmt.Errorf("missed 'DATABASE_PASSWORD' parameter")
	}
	pgDatabasePrefix, ok := os.LookupEnv("DATABASE_NAME_PREFIX")
	if !ok || len(strings.Trim(pgDatabasePrefix, " ")) == 0 {
		return nil, fmt.Errorf("missed 'DATABASE_NAME_PREFIX' parameter")
	}

	// TODO: add parameters for pool configuration
	pgPoolParams := "?pool_max_conns=100"

	pgDatabase := fmt.Sprintf("%v_%v", pgDatabasePrefix, databaseSuffix)
	connString := fmt.Sprintf("postgres://%v:%v@%v:%v/%v%v", pgUser, pgPassword, pgHost, pgPort, pgDatabase, pgPoolParams)
	result, err := CreatePostgreSQLService(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create postgresql service for '%v': %w", pgDatabase, err)
	}
	return result, nil
}

func (s *Services) Shutdown() error {
	result := []error{}
	err := s.AuthService.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.UsersService.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.AssetsService.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	if len(result) > 0 {
		errors.Join(result...)
	}
	return nil
}
