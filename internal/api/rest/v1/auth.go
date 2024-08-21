package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

var ErrAccessTokenExpired = errors.New("access token is expired")
var regExpIpAddr = regexp.MustCompile("^(.+):.+$")
var regExpAuthHeaderBearerToken = regexp.MustCompile("^Bearer (.+)$")

const maxAttempts = 10

// swagger:route POST /api/auth auth Authenicate
//
// # Authenicates user
//
// ---
// Produces:
//   - application/json
//
// Consumes:
//   - application/json
//
// responses:
//   - 200: TokenResponse
//   - 400: ErrorResponse
//   - 500: ErrorResponse
func Authenicate(w http.ResponseWriter, r *http.Request) error {
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError)
	}

	var user Credentials
	err = json.Unmarshal(b, &user)
	if err != nil {
		return WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError)
	}

	ipAddr, err := parseIpAddr(r.RemoteAddr)
	if err != nil {
		return WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError)
	}

	token, err := createOrUpdateToken(user.Login, user.Password, ipAddr)
	if err != nil {
		return processAuthError(err)
	}

	slog.Info(fmt.Sprintf("user '%v' authenicated\n", user.Login))
	return WriteJSON(w, http.StatusOK, TokenResponse{token})
}

func createOrUpdateToken(login, password, ipAddr string) (string, error) {
	token, err := services.Instance().AuthService.CreateOrUpdateToken(login, password, ipAddr)
	// if duplicate error occurrs then try to repeat the tx
	if err != nil && errors.Is(err, services.ErrDuplicateAccessToken) {
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			token, err = services.Instance().AuthService.CreateOrUpdateToken(login, password, ipAddr)
			if err != nil && errors.Is(err, services.ErrDuplicateAccessToken) {
				continue
			}
			if err != nil {
				return "", err
			}
		}
	}
	return token, err
}

func processAuthError(err error) error {
	switch {
	case errors.Is(err, services.ErrDuplicateAccessToken):
		return WithStatus(err, DuplicateAccessTokenMsg, http.StatusInternalServerError)
	case errors.Is(err, services.ErrInvalidPassword):
		return WithStatus(err, InvalidCredentialsMsg, http.StatusBadRequest)
	case errors.Is(err, services.ErrUserNotFound):
		return WithStatus(err, InvalidCredentialsMsg, http.StatusBadRequest)
	default:
		return WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError)
	}
}

var ErrMissedAuthorizationHeader = errors.New("missed 'Authorization' header")
var ErrWrongAuthorizationHeader = errors.New("'Authorization' header must have the following format 'Bearer {token}'")

func parseIpAddr(remoteAddr string) (string, error) {
	matches := regExpIpAddr.FindStringSubmatch(remoteAddr)

	actualMathchesCount := len(matches)
	if actualMathchesCount != 2 {
		return "", fmt.Errorf("wrong len of matches")
	}
	result := matches[1]
	return result, nil
}

func parseAuthorizationHeader(authorizationHeader string) (string, error) {
	if len(authorizationHeader) <= 0 {
		return "", WithStatus(ErrMissedAuthorizationHeader, ErrMissedAuthorizationHeader.Error(), http.StatusUnauthorized)
	}

	if !strings.HasPrefix(authorizationHeader, "Bearer") {

		return "", WithStatus(ErrWrongAuthorizationHeader, ErrWrongAuthorizationHeader.Error(), http.StatusUnauthorized)
	}

	matches := regExpAuthHeaderBearerToken.FindStringSubmatch(authorizationHeader)

	actualMathchesCount := len(matches)
	if actualMathchesCount != 2 {
		return "", WithStatus(fmt.Errorf("unable to parse 'Authorization' header"), InternalServerErrorMsg, http.StatusInternalServerError)
	}
	result := matches[1]
	return result, nil
}
