package v1

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/app"
	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

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

type AuthenticateHandlerFunc func(http.ResponseWriter, *http.Request, *services.AccessToken) error

type AuthenicateHandler struct {
	handler AuthenticateHandlerFunc
}

func (h *AuthenicateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accessToken, err := CheckAuthorization(r)
	if err != nil {
		processHttpError(w, err)
		return
	}

	err = h.handler(w, r, accessToken)
	if err != nil {
		processHttpError(w, err)
	}
}

func CheckAuthorization(r *http.Request) (*services.AccessToken, error) {
	authorizationHeader := r.Header.Get("Authorization")
	t, err := parseAuthorizationHeader(authorizationHeader)
	if err != nil {
		return nil, err
	}
	result, err := services.Instance().AuthService.GetToken(t)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFoundAccessToken):
			return nil, WithStatus(services.ErrNotFoundAccessToken, "Unauthorized", http.StatusUnauthorized)
		default:
			return nil, WithStatus(fmt.Errorf("unexpected error during getting access token: %w", err), "Internal error", http.StatusInternalServerError)
		}
	}
	if result.IsExpired() {
		return nil, WithStatus(ErrAccessTokenExpired, "Unauthorized", http.StatusUnauthorized)
	}
	return &result, nil
}

func AuthRequired(handlerToWrap AuthenticateHandlerFunc) *AuthenicateHandler {
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

type ErrorsHandler struct {
	handler ErrorProcessedHandler
}

type ErrorProcessedHandler func(http.ResponseWriter, *http.Request) error

func (h *ErrorsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.handler(w, r)
	if err != nil {
		processHttpError(w, err)
	}
}

func processHttpError(w http.ResponseWriter, err error) {
	var statusError StatusError
	var status int
	var errorMsg string
	if errors.As(err, &statusError) {
		if statusError.Status() == http.StatusInternalServerError {
			slog.Error(err.Error())
		}
		status = statusError.Status()
		errorMsg = statusError.Error()
	} else {
		slog.Error(err.Error())
		status = UnexpectedError.Status()
		errorMsg = UnexpectedError.Error()
	}

	h := w.Header()
	h.Del("Content-Length")
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	fmt.Fprintln(w, errorMsg)
}

func ErrorHandleRequired(handlerToWrap ErrorProcessedHandler) *ErrorsHandler {
	return &ErrorsHandler{handlerToWrap}
}

var UnexpectedError = statusError{error: fmt.Errorf("unexpected error"), message: "Unexpected error", status: http.StatusInternalServerError}

type StatusError interface {
	error
	Status() int
}

type statusError struct {
	error
	message string
	status  int
}

func (e statusError) Unwrap() error { return e.error }
func (e statusError) Status() int   { return e.status }
func (e statusError) Error() string { return fmt.Sprintf("{\"error\":\"%s\"}", e.message) }

func WithStatus(err error, message string, status int) error {
	return statusError{
		error:   err,
		status:  status,
		message: message,
	}
}

// message for REST API users
const InternalServerErrorMsg = "Internal Server Error"
const AssetNotFoundMsg = "Asset not found"
const UserDuplicateMsg = "User exists already"
const InvalidCredentialsMsg = "Invalid credentials"
const DuplicateAccessTokenMsg = "Duplicate access token generation"
