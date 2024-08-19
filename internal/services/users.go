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
	client *PostgreSQLService
}

func CreateUsersService(client *PostgreSQLService) *UsersService {
	return &UsersService{
		client: client,
	}
}

func (s *UsersService) Shutdown() error {
	return s.client.Shutdown()
}

func (s *UsersService) CreateUser(login string, password string) error {
	exists, err := s.CheckUserExistence(login)
	if err != nil {
		return fmt.Errorf("unable to check user existence: %w", err)
	}

	if exists {
		return ErrDuplicateUser
	}

	userUuid, err := utils.PseudoUUID()
	if err != nil {
		return fmt.Errorf("unable to create uuid for user: %w", err)
	}

	err = s.client.TxVoid(
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
	result, err := s.client.Tx(
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
			return false, nil
		}
		return false, err
	}
	userUuid, ok := result.(string)
	if !ok {
		return false, fmt.Errorf("unable to convert result into string")
	}

	fmt.Printf("CheckUserExistence: %v\n", userUuid)

	return true, nil
}
