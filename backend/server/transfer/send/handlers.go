package send

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"yeetfile/backend/cache"
	"yeetfile/backend/config"
	"yeetfile/backend/db"
	"yeetfile/backend/server/transfer"
	"yeetfile/backend/storage"
	"yeetfile/backend/utils"
	"yeetfile/shared"
	"yeetfile/shared/constants"
)

// UploadMetadataHandler handles a POST request to /u with the metadata required to set
// up a file for uploading. This is defined in the UploadMetadata struct.
func UploadMetadataHandler(w http.ResponseWriter, req *http.Request, userID string) {
	var meta shared.UploadMetadata
	data, _ := utils.LimitedReader(w, req.Body)
	err := json.Unmarshal(data, &meta)
	if err != nil {
		log.Printf("%v\n", req.Body)
		log.Printf("Error: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if meta.Chunks == 0 {
		http.Error(w, "# of chunks cannot be 0", http.StatusBadRequest)
		return
	}

	downloadsErr := validateSendDownloads(meta.Downloads)
	if downloadsErr != nil {
		http.Error(w, downloadsErr.Error(), http.StatusBadRequest)
		return
	}

	exp := utils.StrToDuration(meta.Expiration, config.IsDebugMode)
	expiryErr := validateSendExpiry(exp)
	if expiryErr != nil {
		http.Error(w, expiryErr.Error(), http.StatusBadRequest)
		return
	}

	_, err = UserCanSend(meta.Size, req)
	if err == OutOfSpaceError {
		http.Error(w, "Not enough space available", http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	id, _ := db.InsertMetadata(meta.Chunks, userID, meta.Name, false)
	err = db.CreateNewUpload(id, meta.Name)
	if err != nil {
		log.Printf("Error initializing new upload: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	err = db.SetFileExpiry(id, meta.Downloads, time.Now().Add(exp).UTC())
	if err != nil {
		log.Printf("Error setting file expiry: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if meta.Chunks == 1 {
		err = storage.Interface.InitUpload(id)
	} else {
		err = storage.Interface.InitLargeUpload(meta.Name, id)
	}

	if err != nil {
		log.Println("Error initializing storage", err)
		http.Error(w, "Error initializing storage", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(shared.MetadataUploadResponse{ID: id})
	if err != nil {
		http.Error(w, "Error sending response", http.StatusInternalServerError)
		return
	}
}

// UploadDataHandler handles the process of uploading file chunks to the server,
// after having already initialized the file metadata beforehand.
func UploadDataHandler(w http.ResponseWriter, req *http.Request, userID string) {
	segments := strings.Split(req.URL.Path, "/")
	id := segments[len(segments)-2]
	chunkNum, err := strconv.Atoi(segments[len(segments)-1])
	if err != nil {
		http.Error(w, "Invalid upload URL", http.StatusBadRequest)
		return
	}

	data, err := utils.LimitedChunkReader(w, req.Body)
	if err != nil {
		log.Printf("[YF Send] Chunk reader err: %v\n", err)
		http.Error(w, "Error", http.StatusBadRequest)
		return
	}

	metadata, err := db.RetrieveMetadata(id)
	if err != nil || metadata.Expiration.Before(time.Now().UTC()) {
		log.Printf("[YF Send] Metadata err: %v\n", err)
		http.Error(w, "No metadata found for file", http.StatusBadRequest)
		return
	}

	fileChunk, uploadValues, err := transfer.PrepareUpload(metadata, chunkNum, data)
	metadata.B2ID = uploadValues.UploadID

	// Update user meter
	meterAmount := len(data) - constants.TotalOverhead
	err = UpdateUserMeter(meterAmount, userID)
	if err == db.UserSendExceeded {
		http.Error(w, "Upload failed", http.StatusInternalServerError)
		abortUpload(metadata, userID, meterAmount, chunkNum)
		return
	} else if err != nil {
		log.Printf("[YF Send] Error updating meter: %v\n", err)
	}

	// Upload content
	var finishedUploading bool
	if metadata.Chunks == 1 {
		finishedUploading = true
		err = storage.Interface.UploadSingleChunk(fileChunk, uploadValues)
	} else {
		finishedUploading, err = storage.Interface.UploadMultiChunk(
			fileChunk,
			uploadValues)
	}

	if err != nil {
		log.Printf("[YF Send] Chunk upload err: %v\n", err)
		http.Error(w, "Upload error", http.StatusBadRequest)
		abortUpload(metadata, userID, meterAmount, chunkNum)
		return
	}

	if finishedUploading {
		_, _ = io.WriteString(w, id)
	}
}

// UploadTextHandler handles uploading encrypted text with a max size of
// shared.MaxTextLen characters (constants.go).
func UploadTextHandler(w http.ResponseWriter, req *http.Request, _ string) {
	var upload shared.TextUpload
	err := utils.LimitedJSONReader(w, req.Body).Decode(&upload)
	if err != nil {
		log.Printf("Error: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(upload.Text) > constants.MaxTextLen+constants.TotalOverhead {
		http.Error(w, "Invalid upload size", http.StatusBadRequest)
		return
	}

	downloadsErr := validateSendDownloads(upload.Downloads)
	if downloadsErr != nil {
		http.Error(w, downloadsErr.Error(), http.StatusBadRequest)
		return
	}

	exp := utils.StrToDuration(upload.Expiration, config.IsDebugMode)
	expiryErr := validateSendExpiry(exp)
	if expiryErr != nil {
		http.Error(w, expiryErr.Error(), http.StatusBadRequest)
		return
	}

	id, err := db.InsertMetadata(1, "", upload.Name, true)
	if err != nil {
		log.Printf("Error inserting new text-only upload metadata: %v\n", err)
		http.Error(w, "Unable to init metadata", http.StatusInternalServerError)
		return
	}

	err = db.CreateNewUpload(id, upload.Name)
	if err != nil {
		log.Printf("Error initializing new upload: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	err = db.SetFileExpiry(id, upload.Downloads, time.Now().UTC().Add(exp))
	if err != nil {
		log.Printf("Error setting file expiry: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	err = storage.Interface.InitUpload(id)
	if err != nil {
		http.Error(w, "Unable to init file", http.StatusBadRequest)
		return
	}

	metadata, err := db.RetrieveMetadata(id)
	if err != nil || metadata.Expiration.Before(time.Now().UTC()) {
		http.Error(w, "No metadata found", http.StatusBadRequest)
		return
	}

	fileChunk, uploadValues, err := transfer.PrepareUpload(metadata, 1, upload.Text)
	err = storage.Interface.UploadSingleChunk(fileChunk, uploadValues)

	if err != nil {
		http.Error(w, "Upload error", http.StatusBadRequest)
		return
	}

	err = json.NewEncoder(w).Encode(shared.MetadataUploadResponse{ID: id})
	if err != nil {
		http.Error(w, "Error sending response", http.StatusInternalServerError)
		return
	}
}

// DownloadHandler fetches metadata for downloading a file, such as the name of
// the file, the number of chunks, expiration, etc.
func DownloadHandler(w http.ResponseWriter, req *http.Request) {
	segments := strings.Split(req.URL.Path, "/")
	id := segments[len(segments)-1]

	metadata, err := db.RetrieveMetadata(id)
	if err != nil || metadata.Expiration.Before(time.Now().UTC()) {
		http.Error(w, "File expired", http.StatusBadRequest)
		return
	}

	expiry := db.GetFileExpiry(id)

	response := shared.DownloadResponse{
		Name:       metadata.Name,
		ID:         metadata.ID,
		Chunks:     metadata.Chunks,
		Size:       metadata.Length,
		Downloads:  expiry.Downloads,
		Expiration: expiry.Date,
	}

	jsonData, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonData)
}

// DownloadChunkHandler downloads individual chunks of a file using the chunk
// num from the file path and the decryption key in the header.
// Ex: /d/abc123/2 -- download the second chunk of file with id "abc123"
func DownloadChunkHandler(w http.ResponseWriter, req *http.Request) {
	segments := strings.Split(req.URL.Path, "/")

	if len(segments) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	id := segments[len(segments)-2]
	chunk, _ := strconv.Atoi(segments[len(segments)-1])
	if chunk <= 0 {
		chunk = 1 // Downloads begin with chunk #1
	}

	metadata, err := db.RetrieveMetadata(id)
	if err != nil || metadata.Expiration.Before(time.Now().UTC()) {
		http.Error(w, "No metadata found", http.StatusBadRequest)
		return
	}

	var (
		eof   bool
		bytes []byte
	)

	if cache.HasFile(id, metadata.Length) {
		eof, bytes = transfer.DownloadFileFromCache(id, metadata.Length, chunk)
	} else {
		cache.PrepCache(id, metadata.Length)
		eof, bytes = transfer.DownloadFile(
			metadata.B2ID,
			metadata.Name,
			metadata.Length,
			chunk)
		_ = cache.Write(id, bytes)
	}

	// If the file is finished downloading, decrease the download counter
	// for that file, and delete if 0 are remaining
	rem := -1
	if eof {
		exp := db.GetFileExpiry(metadata.ID)
		rem = db.DecrementDownloads(metadata.ID)

		if rem == 0 {
			storage.DeleteFileByMetadata(metadata)
		}

		if rem >= 0 {
			w.Header().Set("Downloads", strconv.Itoa(rem))
		}
		w.Header().Set("Date", fmt.Sprintf("%s", exp.Date.String()))
	}

	_, _ = w.Write(bytes)
}
