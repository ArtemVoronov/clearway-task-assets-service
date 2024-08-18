package v1

import (
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

// TODO: clean
const uuid_test = "1F615C1D-6BAE-4D8F-EF0B-2FCDC247EF69"

var boundaryStringRegExp = regexp.MustCompile("^.+boundary=(.+)$")

func loadAsset(w http.ResponseWriter, r *http.Request) {
	assetName := getField(r, 0)
	log.Printf("attempt to load asset '%v'\n", assetName)

	var startStreaming services.StartStreamingFunc = func(content io.ReadSeeker) {
		http.ServeContent(w, r, assetName, time.Now(), content)
	}
	err := services.Instance().AssetsService.GetAsset(assetName, uuid_test, startStreaming)

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

// TODO: check error cases, add appopriate tests
// 1. user not found
// 2. token is expired
// 3. too large file
// 4. file is not belonds to the user
// 5. user exceed the limits (3 types of limit: max 100 files, for each file 4GB, for total space 15GB)
// 6. delete user with files
// 7. delete file without user

func storeAsset(w http.ResponseWriter, r *http.Request) {
	assetName := getField(r, 0)
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		log.Println("special case of mime type: multipart/form-data")
		err := storeMultipartedAssets(contentType, r)
		if err != nil {
			processStoreAsserError(err, w)
			return
		}
	} else if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		log.Println("special case of mime type: application/x-www-form-urlencoded")
		log.Printf("attempt to store urlencoded assets '%v'\n", assetName) // log each parameter
		// TODO: parse body and store each asset separately
	} else {
		log.Println("default case for others mime types")
		log.Printf("attempt to store asset '%v'\n", assetName)
		err := storeOneAsset(assetName, r.Body)
		if err != nil {
			processStoreAsserError(err, w)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
	w.WriteHeader(http.StatusCreated)
}

func storeOneAsset(assetName string, reader io.Reader) error {
	// TODO: get real uuid after finishing the debugging
	return services.Instance().AssetsService.CreateAsset(assetName, uuid_test, reader)
}

func storeMultipartedAssets(contentType string, r *http.Request) error {
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
		miltipartedAssetName := p.FormName()
		log.Printf("attempt to store mutliparted asset '%v'\n", miltipartedAssetName)
		err = storeOneAsset(miltipartedAssetName, p)
		if err != nil {
			return err
		}
	}
	return nil
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
	matches := boundaryStringRegExp.FindStringSubmatch(contentTypeString)

	actualMathchesCount := len(matches)
	if actualMathchesCount != 2 {
		return "", fmt.Errorf("wrong len of matches")
	}
	result := matches[1]
	return result, nil
}
