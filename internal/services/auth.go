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
	createTokenQuery        = `INSERT INTO access_tokens (access_token, user_uuid, ip_addr) VALUES ($1, $2, $3) RETURNING access_token`
	updateTokenQuery        = `UPDATE access_tokens SET access_token=$1, ip_addr=$2, create_date=NOW() WHERE user_uuid=$3`
	getTokenByUserUUIDQuery = `SELECT access_token FROM access_tokens WHERE user_uuid = $1`
)

var ErrNotFoundAccessToken = errors.New("access token not found")
var ErrInvalidPassword = errors.New("invalid password")
var ErrDuplicateAccessToken = errors.New("duplciate access_token")

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

func (s *AuthService) CreateOrUpdateToken(login string, password string, ipAddr string) (string, error) {
	result, err := s.client.Tx(
		func(tx pgx.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
			// 1. find user
			var u User
			internalErr := tx.QueryRow(ctx, getUserByLoginQuery, login).Scan(&u.UUID, &u.Login, &u.PasswordHash)
			if internalErr != nil {
				if errors.Is(internalErr, pgx.ErrNoRows) {
					return "", ErrUserNotFound
				}
				return "", internalErr
			}

			// 2. validate credentials
			expectedPassword := u.PasswordHash
			actualPasswrod := utils.MD5Hash(password)
			if actualPasswrod != expectedPassword {
				return "", ErrInvalidPassword
			}

			// 3. find token
			oldToken := ""
			newToken := ""
			internalErr = tx.QueryRow(ctx, getTokenByUserUUIDQuery, u.UUID).Scan(&oldToken)
			if internalErr != nil && !errors.Is(internalErr, pgx.ErrNoRows) {
				return "", internalErr
			}

			// 4. if there is no token then need to create it
			if internalErr != nil && errors.Is(internalErr, pgx.ErrNoRows) {
				internalErr = tx.QueryRow(ctx, createTokenQuery, utils.GenerateToken(), u.UUID, ipAddr).Scan(&newToken)
				if internalErr != nil {
					var pgErr *pgconn.PgError
					switch {
					case errors.As(internalErr, &pgErr):
						if pgErr.Code == DuplicateErrorCode {
							return "", ErrDuplicateAccessToken
						} else {
							return "", fmt.Errorf("unable to create token for user with login '%v': %w", login, internalErr)
						}
					default:
						return "", fmt.Errorf("unable to create token for user with login '%v': %w", login, internalErr)
					}
				}
				return newToken, nil
			}

			// 5. else need to update it
			newToken = utils.GenerateToken()
			_, internalErr = tx.Exec(ctx, updateTokenQuery, newToken, ipAddr, u.UUID)
			if internalErr != nil && !errors.Is(internalErr, pgx.ErrNoRows) {
				var pgErr *pgconn.PgError
				switch {
				case errors.As(internalErr, &pgErr):
					if pgErr.Code == DuplicateErrorCode {
						return "", ErrDuplicateAccessToken
					} else {
						return "", fmt.Errorf("unable to update token for user with login '%v': %w", login, internalErr)
					}
				default:
					return "", fmt.Errorf("unable to update token for user with login '%v': %w", login, internalErr)
				}
			}

			return newToken, nil
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
