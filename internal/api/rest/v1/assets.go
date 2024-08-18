package v1

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

// TODO: clean
const uuid_test = "1F615C1D-6BAE-4D8F-EF0B-2FCDC247EF69"

// TODO: raise to 4GB?
const MaxSizeBody = 1024*1024*1024 - 1024 // 1 GB - 1 KB
// TODO: clean
// const MaxSizeBody = 1024 * 1024 * 2 // 2 MB

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
			http.Error(w, "Asset not found", http.StatusNotFound)
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
	log.Printf("attempt to store asset '%v'\n", assetName)
	// TODO: wrap MaxBytesReader for all request
	r.Body = http.MaxBytesReader(w, r.Body, MaxSizeBody)

	// TODO: need additional processing for the following mimes
	// mulipart/form-data
	// application/x-www-form-urlencoded

	// TODO: clean
	contentType := r.Header.Get("Content-Type")
	fmt.Printf("Content-Type: %v\n", contentType)
	// Parse our multipart form, 10 << 20 specifies a maximum upload of 10 MB files.
	// r.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	// fmt.Println("upload file stage")
	// file, _, err := r.FormFile("myFile")
	// // file, handler, err := r.FormFile("myFile")
	// if err != nil {
	// 	http.Error(w, fmt.Sprintf("Error Retrieving the File: %v", err.Error()), http.StatusInternalServerError)
	// 	return
	// }
	// defer file.Close()
	// fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	// fmt.Printf("File Size: %+v\n", handler.Size)
	// fmt.Printf("MIME Header: %+v\n", handler.Header)

	// TODO: get real uuid after finishing the debugging
	err := services.Instance().AssetsService.CreateAsset(assetName, uuid_test, r.Body)

	if err != nil {
		switch {
		case errors.Is(err, services.ErrDuplicateAsset):
			http.Error(w, "Asset exists already", http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
	w.WriteHeader(http.StatusCreated)
}
