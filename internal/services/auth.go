package services

import (
	"errors"
	"fmt"
)

const (
	createTokenQuery    = `INSERT INTO access_tokens (token, user_uuid, ip_addr) VALUES ($1, $2, $3)`
	updateTokenQuery    = `UPDATE access_tokens SET token=$1, ip_addr=$2, create_date=NOW()`
	getTokenQuery       = `SELECT token, user_uuid, ip_addr, create_date FROM assets WHERE user_uuid = $1`
	getTokenByUserQuery = `SELECT token, user_uuid, ip_addr, create_date FROM assets WHERE user_uuid = $1`
)

var ErrNotFoundToken = errors.New("token not found")

type AuthService struct {
	client *PostgreSQLService
}

func CreateAuthService(client *PostgreSQLService) *AuthService {
	return &AuthService{
		client: client,
	}
}

func (s *AuthService) Shutdown() error {
	return s.client.Shutdown()
}

func (s *AuthService) CreateToken(token string, userUuid string, ipAddr string) error {
	return fmt.Errorf("not implemented")
}

func (s *AuthService) UpdateToken(token string, userUuid string, ipAddr string) error {
	return fmt.Errorf("not implemented")
}

func (s *AuthService) GetToken(token string, userUuid string, ipAddr string) error {
	return fmt.Errorf("not implemented")
}
