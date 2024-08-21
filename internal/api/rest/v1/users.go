package v1

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

// Credentails represents the login and password pair. Used for authenication and creating users
//
// swagger:model Credentials
type Credentials struct {
	// the name for the user
	//
	// required: true
	// example: "alice"
	Login string `json:"login"`

	// the password for the user
	//
	// required: true
	// example: "secret"
	Password string `json:"password"`
}

// swagger:route POST /api/users users CreateUser
//
// # Creates user
//
// ---
// Produces:
//   - application/json
//
// Consumes:
//   - application/json
//
// responses:
//   - 201: OkResponse
//   - 400: ErrorResponse
//   - 500: ErrorResponse
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

	if len(strings.Trim(user.Login, " ")) == 0 {
		return WithStatus(err, "Missed 'login' parameter. Expected json body with it", http.StatusBadRequest)
	}

	if len(strings.Trim(user.Password, " ")) == 0 {
		return WithStatus(err, "Missed 'password' parameter. Expected json body with it", http.StatusBadRequest)
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
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"ok"}`))
	return nil
}

// swagger:parameters CreateUser Authenicate
type CreateUserParams struct {
	// credentials of the user
	//
	// in: body
	// required: true
	// example: {"login": "alice", "password": "secret"}
	Credentials *Credentials `json:"Credentials"`
}
