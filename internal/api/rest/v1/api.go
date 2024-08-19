package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"sync"

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

// TODO: try to put authorization header reading into
// TODO: add configuration of body max
// TODO: add tech endpoints
// TODO: add parameters for pool configuration
// TODO: add configuration for shards count (DEFAULT_BUCKET_FACTOR)

func HandleRoute(w http.ResponseWriter, r *http.Request) {
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

	var h http.HandlerFunc
	var assetName string
	p := r.URL.Path
	switch {
	case matchAndAssignVars(p, "/api/assets"):
		h = allowMethod(loadAssetsList, "GET")
	case matchAndAssignVars(p, "/api/asset/([^/]+)", &assetName):
		r = withPathParams(r, []string{assetName})
		switch r.Method {
		case "GET":
			h = func(w http.ResponseWriter, r *http.Request) { loadAsset(w, r) }
		case "DELETE":
			h = func(w http.ResponseWriter, r *http.Request) { deleteAsset(w, r) }
		default:
			w.Header().Set("Allow", "[GET, DELETE]")
			http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
			return
		}
	case matchAndAssignVars(p, "/api/upload-asset/([^/]+)", &assetName):
		r = withPathParams(r, []string{assetName})
		h = allowMethod(storeAsset, "POST")
	case matchAndAssignVars(p, "/api/auth"):
		h = allowMethod(authenicate, "POST")
	case matchAndAssignVars(p, "/api/users"):
		h = allowMethod(createUser, "POST")
	default:
		http.NotFound(w, r)
		return
	}
	h.ServeHTTP(w, r)
}

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

func withPathParams(r *http.Request, params []string) *http.Request {
	ctx := context.WithValue(r.Context(), ctxKey{}, params)
	return r.WithContext(ctx)
}

type ctxKey struct{}

func getField(r *http.Request, index int) string {
	fields := r.Context().Value(ctxKey{}).([]string)
	return fields[index]
}

var (
	routeRegExps = make(map[string]*regexp.Regexp)
	m            sync.Mutex
)

func mustCompileAndCache(pattern string) *regexp.Regexp {
	m.Lock()
	defer m.Unlock()

	regex := routeRegExps[pattern]
	if regex == nil {
		regex = regexp.MustCompile("^" + pattern + "$")
		routeRegExps[pattern] = regex
	}
	return regex
}

func matchAndAssignVars(path, pattern string, vars ...*string) bool {
	regex := mustCompileAndCache(pattern)
	matches := regex.FindStringSubmatch(path)
	if len(matches) <= 0 {
		return false
	}
	for i, match := range matches[1:] {
		*vars[i] = match
	}
	return true
}

func allowMethod(h http.HandlerFunc, method string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if method != r.Method {
			w.Header().Set("Allow", method)
			http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
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
