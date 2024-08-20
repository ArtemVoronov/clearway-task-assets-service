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
	createUserQuery     = `INSERT INTO users (uuid, login, password_hash) VALUES ($1, $2, $3)`
	getUserByLoginQuery = `SELECT uuid, login, password_hash FROM users WHERE login = $1`
)

var ErrDuplicateUser = errors.New("duplicate user")
var ErrUserNotFound = errors.New("user not found")

type UsersService struct {
	client *PostgreSQLService
}

type User struct {
	UUID         string
	Login        string
	PasswordHash string
}

func CreateUsersService(client *PostgreSQLService) *UsersService {
	return &UsersService{
		client: client,
	}
}

func (s *UsersService) Shutdown() error {
	return nil
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
	_, err := s.GetUser(login)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func (s *UsersService) GetUser(login string) (User, error) {
	var user User
	result, err := s.client.Tx(
		func(tx pgx.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
			var u User
			internalErr := tx.QueryRow(ctx, getUserByLoginQuery, login).Scan(&u.UUID, &u.Login, &u.PasswordHash)
			if internalErr != nil && !errors.Is(internalErr, pgx.ErrNoRows) {
				return "", fmt.Errorf("unable to get user with login '%v': %w", login, internalErr)
			}
			return user, internalErr
		},
		pgx.TxOptions{
			IsoLevel: pgx.ReadCommitted,
		})()
	if err != nil {
		if err == pgx.ErrNoRows {
			return user, ErrUserNotFound
		}
		return user, err
	}
	user, ok := result.(User)
	if !ok {
		return user, fmt.Errorf("unable to convert result into User")
	}

	return user, nil
}
