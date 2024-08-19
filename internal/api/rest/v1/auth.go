package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

var regExpIpAddr = regexp.MustCompile("^(.+):.+$")

func authenicate(w http.ResponseWriter, r *http.Request) {
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

	ipAddr, err := parseIpAddr(r.RemoteAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := services.Instance().AuthService.CreateToken(user.Login, user.Password, ipAddr)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidPasswod):
			http.Error(w, "Invalid credentials", http.StatusBadRequest)
		case errors.Is(err, services.ErrUserNotFound):
			http.Error(w, "User not found", http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"token\":\"%v\"}", token)))
	w.WriteHeader(http.StatusOK)
}

func parseIpAddr(remoteAddr string) (string, error) {
	matches := regExpIpAddr.FindStringSubmatch(remoteAddr)

	actualMathchesCount := len(matches)
	if actualMathchesCount != 2 {
		return "", fmt.Errorf("wrong len of matches")
	}
	result := matches[1]
	return result, nil
}
