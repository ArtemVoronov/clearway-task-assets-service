package users

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

type UserDTO struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func ProcessUsersRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		createUser(w, r)
	default:
		http.Error(w, "Not Implemented", 501)
	}
}

func createUser(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var user UserDTO
	err = json.Unmarshal(b, &user)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = services.Instance().UsersService.CreateUser(user.Login, user.Password)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDuplicateUser):
			http.Error(w, "User exists already", 400)
		default:
			http.Error(w, err.Error(), 500)
		}
	}

	w.Write([]byte(fmt.Sprintf("Done\n")))
	w.WriteHeader(200)
}
