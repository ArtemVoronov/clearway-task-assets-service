package v1

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

type UserDTO struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
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

	err = services.Instance().UsersService.CreateUser(user.Login, user.Password)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDuplicateUser):
			http.Error(w, "User exists already", http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
	w.WriteHeader(http.StatusOK)
}
