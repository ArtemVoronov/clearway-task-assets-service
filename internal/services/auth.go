package services

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/app/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	createTokenQuery    = `INSERT INTO access_tokens (access_token, user_uuid, ip_addr) VALUES ($1, $2, $3) RETURNING access_token`
	updateTokenQuery    = `UPDATE access_tokens SET access_token=$1, ip_addr=$2, create_date=NOW()`
	getTokenQuery       = `SELECT access_token, user_uuid, ip_addr, create_date FROM access_tokens WHERE access_token = $1`
	getTokenByUserQuery = `SELECT access_token, user_uuid, ip_addr, create_date FROM access_tokens WHERE user_uuid = $1`
)

const maxAttemptsForAccessTokenGeneration = 10

var ErrNotFoundToken = errors.New("access token not found")
var ErrInvalidPasswod = errors.New("invalid password")

type AuthService struct {
	client *PostgreSQLService
}

func CreateAuthService(client *PostgreSQLService) *AuthService {
	return &AuthService{
		client: client,
	}
}

func (s *AuthService) Shutdown() error {
	return nil
}

// TODO: rename to CreateOrUpdateToken
// TODO: add checking of existence token (create if it is not or to update in other case)
func (s *AuthService) CreateToken(login string, password string, ipAddr string) (string, error) {
	result, err := s.client.Tx(
		func(tx pgx.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {

			var u User
			internalErr := tx.QueryRow(ctx, getUserByLoginQuery, login).Scan(&u.UUID, &u.Login, &u.PasswordHash)
			if internalErr != nil && !errors.Is(internalErr, pgx.ErrNoRows) {
				if errors.Is(internalErr, pgx.ErrNoRows) {
					return "", ErrUserNotFound
				}
				return "", internalErr
			}

			expectedPassword := u.PasswordHash
			actualPasswrod := utils.MD5Hash(password)

			if actualPasswrod != expectedPassword {
				return "", ErrInvalidPasswod
			}

			token := ""
			for attempt := 1; attempt <= maxAttemptsForAccessTokenGeneration; attempt++ {
				internalErr = tx.QueryRow(ctx, createTokenQuery, utils.GenerateToken(), u.UUID, ipAddr).Scan(&token)
				if internalErr != nil {
					var pgErr *pgconn.PgError
					switch {
					case errors.As(internalErr, &pgErr):
						if pgErr.Code == DuplicateErrorCode {
							token = ""
							log.Printf("duplicated access_token, attempt %v to recreate it", attempt)
							continue
						} else {
							return "", fmt.Errorf("unable to create token for user with login '%v': %w", login, internalErr)
						}
					default:
						return "", fmt.Errorf("unable to create token for user with login '%v': %w", login, internalErr)
					}
				}

				return token, nil
			}

			if len(token) == 0 || internalErr != nil {
				return "", fmt.Errorf("unable to create token for user with login '%v' after max repeat attempts: %w", login, internalErr)
			}
			return token, nil
		},
		pgx.TxOptions{
			IsoLevel: pgx.ReadCommitted,
		})()

	if err != nil {
		return "", err
	}

	token, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unable to convert result into string")
	}

	return token, err
}

func (s *AuthService) GetToken(token string, userUuid string, ipAddr string) error {
	return fmt.Errorf("not implemented")
}
