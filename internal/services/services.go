package services

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/app"
)

type Services struct {
	AuthService    *AuthService
	UsersService   *UsersService
	AssetsService  *AssetsService
	pgForAssets    []*PostgreSQLService
	pgForUnsharded *PostgreSQLService
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
	pgForUnsharded, err := initPostgreServiceBySuffix("unsharded")
	if err != nil {
		return nil, fmt.Errorf("unable to init postgresql services for auth: %w", err)
	}
	accessTokenTTL, err := parseAccessTokenTTL()
	if err != nil {
		return nil, fmt.Errorf("unable to init access token TTL: %w", err)
	}

	return &Services{
		AuthService:    CreateAuthService(pgForUnsharded, accessTokenTTL),
		UsersService:   CreateUsersService(pgForUnsharded),
		AssetsService:  CreateAssetsService(pgForAssets),
		pgForAssets:    pgForAssets,
		pgForUnsharded: pgForUnsharded,
	}, nil
}

func (s *Services) Shutdown() error {
	result := []error{}
	l := len(s.pgForAssets)
	for i := 0; i < l; i++ {
		err := s.pgForAssets[i].Shutdown()
		if err != nil {
			result = append(result, err)
		}
	}
	err := s.pgForUnsharded.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.AuthService.Shutdown()
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

func initPostgreServicesForAssets() ([]*PostgreSQLService, error) {
	shardsCount, err := parseDatabaseShardsCount()
	if err != nil {
		return nil, err
	}
	pgServices := make([]*PostgreSQLService, 0, shardsCount)
	for i := 1; i <= shardsCount; i++ {
		pgService, err := initPostgreServiceBySuffix(fmt.Sprintf("assets_shard_%v", i))
		if err != nil {
			return nil, fmt.Errorf("unable to create postgresql service for assets: %w", err)
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
	pgConnectTimeout, ok := os.LookupEnv("DATABASE_CONNECT_TIMEOUT_IN_SECONDS")
	if !ok || len(strings.Trim(pgConnectTimeout, " ")) == 0 {
		return nil, fmt.Errorf("missed 'DATABASE_CONNECT_TIMEOUT_IN_SECONDS' parameter")
	}
	pgPoolMaxConnections, ok := os.LookupEnv("DATABASE_CONNECTIONS_POOL_MAX_CONNS")
	if !ok || len(strings.Trim(pgPoolMaxConnections, " ")) == 0 {
		return nil, fmt.Errorf("missed 'DATABASE_CONNECTIONS_POOL_MAX_CONNS' parameter")
	}
	pgPoolMinConnections, ok := os.LookupEnv("DATABASE_CONNECTIONS_POOL_MIN_CONNS")
	if !ok || len(strings.Trim(pgPoolMinConnections, " ")) == 0 {
		return nil, fmt.Errorf("missed 'DATABASE_CONNECTIONS_POOL_MIN_CONNS' parameter")
	}
	pgPoolMaxConnLifeTime, ok := os.LookupEnv("DATABASE_CONNECTIONS_POOL_MAX_CONN_LIFE_TIME")
	if !ok || len(strings.Trim(pgPoolMaxConnLifeTime, " ")) == 0 {
		return nil, fmt.Errorf("missed 'DATABASE_CONNECTIONS_POOL_MAX_CONN_LIFE_TIME' parameter")
	}
	pgPoolMaxConnIdleTime, ok := os.LookupEnv("DATABASE_CONNECTIONS_POOL_MAX_CONN_IDLE_TIME")
	if !ok || len(strings.Trim(pgPoolMaxConnIdleTime, " ")) == 0 {
		return nil, fmt.Errorf("missed 'DATABASE_CONNECTIONS_POOL_MAX_CONN_IDLE_TIME' parameter")
	}
	pgPoolMinConnHealthcheckPeriod, ok := os.LookupEnv("DATABASE_CONNECTIONS_POOL_HEALTH_CHECK_PERIOD")
	if !ok || len(strings.Trim(pgPoolMinConnHealthcheckPeriod, " ")) == 0 {
		return nil, fmt.Errorf("missed 'DATABASE_CONNECTIONS_POOL_HEALTH_CHECK_PERIOD' parameter")
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("?connect_timeout=%v", pgConnectTimeout))
	b.WriteString(fmt.Sprintf("&pool_max_conns=%v", pgPoolMaxConnections))
	b.WriteString(fmt.Sprintf("&pool_min_conns=%v", pgPoolMinConnections))
	b.WriteString(fmt.Sprintf("&pool_max_conn_lifetime=%v", pgPoolMaxConnLifeTime))
	b.WriteString(fmt.Sprintf("&pool_max_conn_idle_time=%v", pgPoolMaxConnIdleTime))
	b.WriteString(fmt.Sprintf("&pool_health_check_period=%v", pgPoolMinConnHealthcheckPeriod))
	pgPoolParams := b.String()

	pgDatabase := fmt.Sprintf("%v_%v", pgDatabasePrefix, databaseSuffix)
	connString := fmt.Sprintf("postgres://%v:%v@%v:%v/%v%v", pgUser, pgPassword, pgHost, pgPort, pgDatabase, pgPoolParams)

	result, err := CreatePostgreSQLService(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create postgresql service for '%v': %w", pgDatabase, err)
	}
	return result, nil
}

func parseAccessTokenTTL() (time.Duration, error) {
	accessTokenTTLStr, ok := os.LookupEnv("AUTH_ACCESS_TOKEN_TTL")
	if !ok {
		accessTokenTTLStr = app.DefaultAccessTokenTTL
	}
	return time.ParseDuration(accessTokenTTLStr)
}

func parseDatabaseShardsCount() (int, error) {
	result := app.DefaultShardsCount
	shardsStr, ok := os.LookupEnv("DATABASE_SHARDS_COUNT")
	if ok {
		converted, err := strconv.Atoi(shardsStr)
		if err != nil {
			return 0, fmt.Errorf("unable to parse 'DATABASE_SHARDS_COUNT' parameter: %w", err)
		}
		result = converted
	}

	if result < 1 {
		return 0, fmt.Errorf("parameter 'DATABASE_SHARDS_COUNT' should by greater than or equal to 1")
	}

	return result, nil
}
