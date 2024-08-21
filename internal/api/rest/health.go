package rest

import (
	"encoding/json"
	"net/http"
	"os"
)

func ApiSpec(w http.ResponseWriter, r *http.Request) error {
	spec, err := os.ReadFile("./api/swagger/swagger.json")
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(spec)
	return nil
}

// Cumulative information about the readiness and performance of the service
// swagger:response AppInfo
type AppInfo struct {
	// version
	// example: "1.0"
	Version string `json:"version"`

	// state
	// example: "running"
	State string `json:"state"`
}

// swagger:route GET /health application health
//
// # Cumulative information about the readiness and performance of the service
//
// ---
// Produces:
//   - application/json
//
// responses:
//   - 200: OkResponse
//   - 500: ErrorResponse
func Health(w http.ResponseWriter, r *http.Request) error {
	appInfo := AppInfo{"1.0", "running"}
	result, err := json.Marshal(appInfo)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(result)
	return nil
}
