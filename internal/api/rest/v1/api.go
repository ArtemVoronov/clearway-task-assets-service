package v1

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/app"
	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

// TODO: unify errors formatting

// TODO: check error cases, add appopriate tests
// 1. user not found
// 2. token is expired
// 3. too large file
// 4. file is not belonds to the user
// 5. user exceed the limits (3 types of limit: max 100 files, for each file 4GB, for total space 15GB)
// 6. delete user with files
// 7. delete file without user

// TODO: add configuration of body max
// TODO: add tech endpoints
// TODO: add parameters for pool configuration
// TODO: add configuration for shards count (DEFAULT_BUCKET_FACTOR)

func isBodyLimitExceeded(r *http.Request) (bool, error) {
	contentLength := r.Header.Get("Content-Length")
	if len(contentLength) == 0 {
		return false, nil
	}

	bodyLength, err := strconv.Atoi(contentLength)
	if err != nil {
		return false, fmt.Errorf("unable to parse 'Content-Length' header: %w", err)
	}

	return bodyLength > app.DefaultBodyMaxSize, nil
}

func CheckAuthorization(r *http.Request) (*services.AccessToken, error) {
	authorizationHeader := r.Header.Get("Authorization")
	t, err := parseAuthorizationHeader(authorizationHeader)
	if err != nil {
		return nil, err
	}
	result, err := services.Instance().AuthService.GetToken(t)
	if err != nil {
		return nil, fmt.Errorf("unexpected error during getting access token: %w", err)
	}
	if result.IsExpired() {
		return nil, ErrAccessTokenExpired
	}
	return &result, nil
}

func ProcessCheckAuthroizationError(w http.ResponseWriter, err error) {
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFoundAccessToken):
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		case errors.Is(err, ErrAccessTokenExpired):
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		default:
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}

type LoggerHandler struct {
	handler http.Handler
}

func (h *LoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.handler.ServeHTTP(w, r)
	log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
}

func NewLoggerHandler(handlerToWrap http.Handler) *LoggerHandler {
	return &LoggerHandler{handlerToWrap}
}

type AuthenticatedHandler func(http.ResponseWriter, *http.Request, *services.AccessToken)

type AuthenicateHandler struct {
	handler AuthenticatedHandler
}

func (h *AuthenicateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accessToken, err := CheckAuthorization(r)
	if err != nil {
		ProcessCheckAuthroizationError(w, err)
		return
	}

	h.handler(w, r, accessToken)
}

func AuthRequired(handlerToWrap AuthenticatedHandler) *AuthenicateHandler {
	return &AuthenicateHandler{handlerToWrap}
}

type BodySizeLimitHandler struct {
	handler http.Handler
}

func (h *BodySizeLimitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	isExceeded, err := isBodyLimitExceeded(r)
	if err != nil {
		http.Error(w, "Unexpected error during verifying body size", http.StatusInternalServerError)
		return
	}
	if isExceeded {
		http.Error(w, fmt.Sprintf("Body size exceeds the limit in %v bytes", app.DefaultBodyMaxSize), http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, app.DefaultBodyMaxSize)
	h.handler.ServeHTTP(w, r)
}

func NewBodySizeLimitHandler(handlerToWrap http.Handler) *BodySizeLimitHandler {
	return &BodySizeLimitHandler{handlerToWrap}
}
