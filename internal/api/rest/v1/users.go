// Package routes Users API.
//
//	Schemes: https
//	BasePath: /
//	Version: 1.0
//	Host: https://localhost:3005
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
// swagger:meta
package v1

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

// Credentails represents the login and password pair
//
// # Used for authenication and creating users
//
// swagger:model
type Credentials struct {
	// the name for the user
	// required: true
	// example: alice
	Login string `json:"login"`
	// the password for the user
	// required: true
	// example: alice
	Password string `json:"password"`
}

// swagger:operation POST /api/users createUser
//
// # Creates an user
//
// ---
// produces:
// - application/json
// consumes:
// - application/json
//
// responses:
//
//	'200':
//	'400':
//	'500':
func CreateUser(w http.ResponseWriter, r *http.Request) error {
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

	err = services.Instance().UsersService.CreateUser(user.Login, user.Password)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDuplicateUser):
			return WithStatus(services.ErrDuplicateUser, UserDuplicateMsg, http.StatusBadRequest)
		default:
			return WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
	w.WriteHeader(http.StatusOK)
	return nil
}
