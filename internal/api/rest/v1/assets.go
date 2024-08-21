package v1

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

var regExpBoundaryString = regexp.MustCompile("^.+boundary=(.+)$")

// swagger:route GET /api/assets assets LoadAssetsList
//
// # Get users's assets list
//
// ---
// Produces:
//   - application/json
//
// Security:
// - Bearer: []
//
// responses:
//   - 200: AssetsListResponse
//   - 500: ErrorResponse
func LoadAssetsList(w http.ResponseWriter, r *http.Request, t *services.AccessToken) error {
	slog.Info(fmt.Sprintf("attempt to load assets list for user '%v'\n", t.UserUUID))
	list, err := services.Instance().AssetsService.GetAssetList(t.UserUUID)
	if err != nil {
		return WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError)
	}

	return WriteJSON(w, http.StatusOK, AssetsListResponse{list})
}

// swagger:route GET /api/asset/{name} assets LoadAsset
//
// # Get users's asset by name
//
// ---
// Produces:
//   - application/json
//
// Security:
// - Bearer: []
//
// responses:
//   - 200: StatusResponse
//   - 404: ErrorResponse
//   - 500: ErrorResponse
func LoadAsset(w http.ResponseWriter, r *http.Request, t *services.AccessToken) error {
	assetName := r.PathValue("name")
	slog.Info(fmt.Sprintf("attempt to load asset '%v'\n", assetName))

	var startStreaming services.StartStreamingFunc = func(content io.ReadSeeker) {
		http.ServeContent(w, r, assetName, time.Now(), content)
	}
	err := services.Instance().AssetsService.GetAsset(assetName, t.UserUUID, startStreaming)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFoundAsset):
			return WithStatus(err, AssetNotFoundMsg, http.StatusNotFound)
		default:
			return WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError)
		}
	}
	// correct status code will be returned by http.ServeContent
	return nil
}

// swagger:route DELETE /api/asset/{name} assets DeleteAsset
//
// # Delete users's asset by name
//
// ---
// Produces:
//   - application/json
//
// Security:
// - Bearer: []
//
// responses:
//   - 200: StatusResponse
//   - 404: ErrorResponse
//   - 500: ErrorResponse
func DeleteAsset(w http.ResponseWriter, r *http.Request, t *services.AccessToken) error {
	assetName := r.PathValue("name")
	slog.Info(fmt.Sprintf("attempt to delete asset '%v'\n", assetName))

	err := services.Instance().AssetsService.DeleteAsset(assetName, t.UserUUID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFoundAsset):
			return WithStatus(err, AssetNotFoundMsg, http.StatusNotFound)
		default:
			return WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError)
		}
	}

	return WriteJSON(w, http.StatusOK, StatusResponse{"ok"})
}

// swagger:route POST /api/upload-asset/{name} assets StoreAsset
//
// # Store asset
//
// ---
// Produces:
//   - application/json
//
// Consumes:
//   - any
//
// Security:
// - Bearer: []
//
// responses:
//   - 201: StatusResponse
//   - 400: ErrorResponse
//   - 500: ErrorResponse
func StoreAsset(w http.ResponseWriter, r *http.Request, t *services.AccessToken) error {
	assetName := r.PathValue("name")
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		slog.Info("special case of mime type: multipart/form-data")
		err := storeMultipartedAssets(contentType, r, t)
		if err != nil {
			return processStoreAsserError(err)
		}
	} else {
		slog.Info("default case for others mime types")
		slog.Info(fmt.Sprintf("attempt to store asset '%v'\n", assetName))
		err := storeOneAsset(assetName, r.Body, t)
		if err != nil {
			return processStoreAsserError(err)
		}
	}

	return WriteJSON(w, http.StatusCreated, StatusResponse{"ok"})
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
		slog.Info(fmt.Sprintf("attempt to store mutliparted asset '%v'\n", miltipartedAssetName))
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

func processStoreAsserError(err error) error {
	switch {
	case errors.Is(err, services.ErrDuplicateAsset):
		return WithStatus(err, err.Error(), http.StatusBadRequest)
	default:
		return WithStatus(err, InternalServerErrorMsg, http.StatusInternalServerError)
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

// swagger:parameters LoadAsset DeleteAsset
type AssetsRequest struct {
	// asset name
	//
	// in: path
	// required: true
	AssetName string `json:"name"`
}

// swagger:parameters StoreAsset
type CreateAssetParams struct {
	// asset name
	//
	// in: path
	// required: true
	AssetName string `json:"name"`

	// asset data
	//
	// in: formData
	// required: true
	// swagger:file
	Data *bytes.Buffer `json:"data"`
}
