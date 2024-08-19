package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

var ErrAccessTokenExpired = errors.New("access token is expired")
var regExpIpAddr = regexp.MustCompile("^(.+):.+$")
var regExpAuthHeaderBearerToken = regexp.MustCompile("^Bearer (.+)$")

const maxAttempts = 10

func authenicate(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var user UserDTO
	err = json.Unmarshal(b, &user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ipAddr, err := parseIpAddr(r.RemoteAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := createOrUpdateToken(user.Login, user.Password, ipAddr)
	if err != nil {
		processAuthError(err, w)
		return
	}

	log.Printf("user '%v' authenicated\n", user.Login)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"token\":\"%v\"}", token)))
	w.WriteHeader(http.StatusOK)
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

func processAuthError(err error, w http.ResponseWriter) {
	switch {
	case errors.Is(err, services.ErrDuplicateAccessToken):
		http.Error(w, "Duplicate access token generation", http.StatusInternalServerError)
	case errors.Is(err, services.ErrInvalidPassword):
		http.Error(w, "Invalid credentials", http.StatusBadRequest)
	case errors.Is(err, services.ErrUserNotFound):
		http.Error(w, "User not found", http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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
		return "", fmt.Errorf("missed 'Authorization' header")
	}

	if !strings.HasPrefix(authorizationHeader, "Bearer") {
		return "", fmt.Errorf("supported only 'Authorization' header with Bearer token")
	}

	matches := regExpAuthHeaderBearerToken.FindStringSubmatch(authorizationHeader)

	actualMathchesCount := len(matches)
	if actualMathchesCount != 2 {
		return "", fmt.Errorf("wrong len of matches")
	}
	result := matches[1]
	return result, nil
}
