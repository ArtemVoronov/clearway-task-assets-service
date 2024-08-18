package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/app/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	createUserQuery          = `INSERT INTO users (uuid, login, password_hash) VALUES ($1, $2, $3)`
	findUserUUIDByLoginQuery = `SELECT uuid FROM users WHERE login = $1`
)

var ErrDuplicateUser = errors.New("duplicate user")

type UsersService struct {
	clientShards []*PostgreSQLService
	ShardsNum    int
	shardService *ShardService
}

func CreateUsersService(clients []*PostgreSQLService) *UsersService {
	return &UsersService{
		clientShards: clients,
		ShardsNum:    len(clients),
		shardService: CreateShardService(len(clients)),
	}
}

func (s *UsersService) Shutdown() error {
	return nil
}

func (s *UsersService) client(userUuid string) *PostgreSQLService {
	bucketIndex := s.shardService.GetBucketIndex(userUuid)
	bucket := s.shardService.GetBucketByIndex(bucketIndex)
	return s.clientShards[bucket]
}

func (s *UsersService) CreateUser(login string, password string) error {
	ok, err := s.CheckUserExistence(login)
	if err != nil {
		return fmt.Errorf("unable to check user existence: %w", err)
	}

	if !ok {
		return ErrDuplicateUser
	}

	userUuid, err := utils.PseudoUUID()
	if err != nil {
		return fmt.Errorf("unable to create uuid for user: %w", err)
	}

	err = s.client(userUuid).TxVoid(
		func(tx pgx.Tx, ctx context.Context, cancel context.CancelFunc) error {
			_, internalErr := tx.Exec(ctx, createUserQuery, userUuid, login, utils.MD5Hash(password))
			if internalErr != nil {
				var pgErr *pgconn.PgError
				switch {
				case errors.As(internalErr, &pgErr):
					if pgErr.Code == DuplicateErrorCode {
						return ErrDuplicateUser
					} else {
						return fmt.Errorf("unable to insert user with login '%v': %w", login, internalErr)
					}
				default:
					return fmt.Errorf("unable to insert user with login '%v': %w", login, internalErr)
				}
			}
			return nil
		},
		pgx.TxOptions{
			IsoLevel: pgx.ReadCommitted,
		})()

	return err
}

func (s *UsersService) CheckUserExistence(login string) (bool, error) {
	for shard := range s.clientShards {
		userUuid, err := s.clientShards[shard].Tx(
			func(tx pgx.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
				var uuid string
				internalErr := tx.QueryRow(ctx, "SELECT uuid from users where login = $1", login).Scan(&uuid)
				if internalErr != nil && !errors.Is(internalErr, pgx.ErrNoRows) {
					return "", fmt.Errorf("unable to check user existence with login '%v': %w", login, internalErr)
				}
				return uuid, internalErr
			},
			pgx.TxOptions{
				IsoLevel: pgx.ReadCommitted,
			})()
		if err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return false, err
		}
		result, ok := userUuid.(string)
		if !ok {
			return false, fmt.Errorf("unable to convert result into string")
		}
		if len(result) > 0 {
			fmt.Printf("attempt to create duplicate user with login '%v', existed uuid: %v", login, userUuid)
			return false, nil
		}
	}

	return true, nil
}
