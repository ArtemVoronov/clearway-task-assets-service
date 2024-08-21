// Assets Service
//
//	Schemes: https
//	BasePath: /
//	Version: 1.0
//	Host: localhost:3005
//
//  SecurityDefinitions:
//    Bearer:
//      type: apiKey
//      name: Authorization
//      in: header
//
// swagger:meta
package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/app"
	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

// TODO: add limits for users, i.e. max 100 files, max file size 4GB, for total files 15GB

type LoggerHandler struct {
	handler http.Handler
}

func (h *LoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.handler.ServeHTTP(w, r)
	slog.Info(fmt.Sprintf("%s %s %v", r.Method, r.URL.Path, time.Since(start)))
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
			return nil, WithStatus(services.ErrNotFoundAccessToken, UnauthorizedMsg, http.StatusUnauthorized)
		default:
			return nil, WithStatus(fmt.Errorf("unexpected error during getting access token: %w", err), InternalServerErrorMsg, http.StatusInternalServerError)
		}
	}
	if result.IsExpired() {
		return nil, WithStatus(ErrAccessTokenExpired, UnauthorizedMsg, http.StatusUnauthorized)
	}
	return &result, nil
}

func AuthRequired(handlerToWrap AuthenticateHandlerFunc) *AuthenicateHandler {
	return &AuthenicateHandler{handlerToWrap}
}

type BodySizeLimitHandler struct {
	handler     http.Handler
	bodyMaxSize int
}

func (h *BodySizeLimitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	isExceeded, err := h.isBodyLimitExceeded(r)
	if err != nil {
		processHttpError(w, WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError))
		return
	}
	if isExceeded {
		processHttpError(w, WithStatus(err, fmt.Sprintf("body size exceeds the limit in %v bytes", h.bodyMaxSize), http.StatusBadRequest))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(h.bodyMaxSize))
	h.handler.ServeHTTP(w, r)
}

func (h *BodySizeLimitHandler) isBodyLimitExceeded(r *http.Request) (bool, error) {
	contentLength := r.Header.Get("Content-Length")
	if len(contentLength) == 0 {
		return false, nil
	}

	bodyLength, err := strconv.Atoi(contentLength)
	if err != nil {
		return false, fmt.Errorf("unable to parse 'Content-Length' header: %w", err)
	}

	return bodyLength > h.bodyMaxSize, nil
}

func NewBodySizeLimitHandler(handlerToWrap http.Handler, bodyMaxSize int) *BodySizeLimitHandler {
	return &BodySizeLimitHandler{
		handler:     handlerToWrap,
		bodyMaxSize: bodyMaxSize,
	}
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

func NewProcessOptionsRequestsFunc() http.HandlerFunc {
	allowedHeaders, ok := os.LookupEnv("CORS_ALLOWED_HEADERS")
	if !ok {
		allowedHeaders = app.DefaultCORSAllowedHeaders
	}
	allowedOrigin, ok := os.LookupEnv("CORS_ALLOWED_ORIGIN")
	if !ok {
		allowedOrigin = app.DefaultCORSAllowedOrigin
	}
	allowedMethods, ok := os.LookupEnv("CORS_ALLOWED_METHODS")
	if !ok {
		allowedMethods = app.DefaultCORSAllowedMethods
	}
	return func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Headers", allowedHeaders)
		h.Set("Access-Control-Allow-Origin", allowedOrigin)
		h.Set("Access-Control-Allow-Methods", allowedMethods)
		h.Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}

// message for REST API users
const InternalServerErrorMsg = "Internal Server Error"
const AssetNotFoundMsg = "Asset not found"
const UserDuplicateMsg = "User exists already"
const InvalidCredentialsMsg = "Invalid credentials"
const DuplicateAccessTokenMsg = "Duplicate access token generation"
const UnauthorizedMsg = "Unauthorized"

// Common success response
// swagger:response StatusResponse
type StatusResponse struct {
	// status
	// example: "ok"
	Status string `json:"status"`
}

// Success authenication response
// swagger:response TokenResponse
type TokenResponse struct {
	// token
	// example: "b5a302e740d0d84bbdc2254c97f1427b"
	Token string `json:"token"`
}

// Success get assets list response
// swagger:response AssetsListResponse
type AssetsListResponse struct {
	// assets
	// example: "[file1.txt, file2.txt, file3.txt]"
	AssetsList []string `json:"assets"`
}

// Common error response
// swagger:response ErrorResponse
type ErrorResponse struct {
	// error
	// example: "Internal server error"
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, code int, obj any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8application/json")
	w.WriteHeader(code)
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError)
	}
	_, err = w.Write(jsonBytes)
	if err != nil {
		return WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError)
	}
	return nil
}
