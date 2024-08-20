package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

var regExpBoundaryString = regexp.MustCompile("^.+boundary=(.+)$")

func LoadAssetsList(w http.ResponseWriter, r *http.Request, t *services.AccessToken) {
	log.Printf("attempt to load assets list for user '%v'\n", t.UserUUID)
	list, err := services.Instance().AssetsService.GetAssetList(t.UserUUID)
	if err != nil {
		switch {
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	result, err := json.Marshal(list)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
	w.WriteHeader(http.StatusOK)
}

func LoadAsset(w http.ResponseWriter, r *http.Request, t *services.AccessToken) {
	assetName := r.PathValue("name")
	log.Printf("attempt to load asset '%v'\n", assetName)

	var startStreaming services.StartStreamingFunc = func(content io.ReadSeeker) {
		http.ServeContent(w, r, assetName, time.Now(), content)
	}
	err := services.Instance().AssetsService.GetAsset(assetName, t.UserUUID, startStreaming)

	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFoundAsset):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	// correct status code will be returned by http.ServeContent
}

func DeleteAsset(w http.ResponseWriter, r *http.Request, t *services.AccessToken) {
	assetName := r.PathValue("name")
	log.Printf("attempt to delete asset '%v'\n", assetName)

	err := services.Instance().AssetsService.DeleteAsset(assetName, t.UserUUID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFoundAsset):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
	w.WriteHeader(http.StatusCreated)
}

func StoreAsset(w http.ResponseWriter, r *http.Request, t *services.AccessToken) {
	assetName := r.PathValue("name")
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		log.Println("special case of mime type: multipart/form-data")
		err := storeMultipartedAssets(contentType, r, t)
		if err != nil {
			processStoreAsserError(err, w)
			return
		}
	} else {
		log.Println("default case for others mime types")
		log.Printf("attempt to store asset '%v'\n", assetName)
		err := storeOneAsset(assetName, r.Body, t)
		if err != nil {
			processStoreAsserError(err, w)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
	w.WriteHeader(http.StatusCreated)
}

func storeOneAsset(assetName string, reader io.Reader, t *services.AccessToken) error {
	return services.Instance().AssetsService.CreateAsset(assetName, t.UserUUID, reader)
}

func storeMultipartedAssets(contentType string, r *http.Request, t *services.AccessToken) error {
	boundaryString, err := parseBoundaryString(contentType)
	if err != nil {
		return err
	}

	partReader := multipart.NewReader(r.Body, boundaryString)
	for {
		p, err := partReader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		miltipartedAssetName := parseMultipartAssetName(p)
		log.Printf("attempt to store mutliparted asset '%v'\n", miltipartedAssetName)
		err = storeOneAsset(miltipartedAssetName, p, t)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseMultipartAssetName(p *multipart.Part) string {
	fileName := p.FileName()
	if len(fileName) > 0 {
		return fileName
	} else {
		return p.FormName()
	}
}

func processStoreAsserError(err error, w http.ResponseWriter) {
	switch {
	case errors.Is(err, services.ErrDuplicateAsset):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseBoundaryString(contentTypeString string) (string, error) {
	matches := regExpBoundaryString.FindStringSubmatch(contentTypeString)

	actualMathchesCount := len(matches)
	if actualMathchesCount != 2 {
		return "", fmt.Errorf("wrong len of matches")
	}
	result := matches[1]
	return result, nil
}
