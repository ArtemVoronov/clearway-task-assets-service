package v1

import (
	"context"
	"net/http"
	"regexp"
	"sync"
)

func HandleRoute(w http.ResponseWriter, r *http.Request) {
	var h http.Handler
	var assetName string
	p := r.URL.Path
	switch {
	case matchAndAssignVars(p, "/api/asset/([^/]+)", &assetName):
		ctx := context.WithValue(r.Context(), ctxKey{}, []string{assetName})
		r = r.WithContext(ctx)
		h = allowMethod(loadAsset, "GET")
	case matchAndAssignVars(p, "/api/upload-asset/([^/]+)", &assetName):
		ctx := context.WithValue(r.Context(), ctxKey{}, []string{assetName})
		r = r.WithContext(ctx)
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
