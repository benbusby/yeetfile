package admin

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"yeetfile/shared"
)

func UserActionHandler(w http.ResponseWriter, req *http.Request, id string) {
	segments := strings.Split(req.URL.Path, "/")
	userID := segments[len(segments)-1]

	if userID == id {
		http.Error(w, "Cannot fetch yourself", http.StatusBadRequest)
		return
	}

	switch req.Method {
	case http.MethodDelete:
		err := deleteUser(userID)
		if err != nil {
			log.Printf("Error deleting user: %v\n", err)
			http.Error(w, "Failed to delete user", http.StatusInternalServerError)
			return
		}
	case http.MethodGet:
		user, err := getUserInfo(userID)
		if err != nil {
			log.Printf("Error fetching user: %v\n", err)
			if err == sql.ErrNoRows {
				http.Error(w, "No match found", http.StatusNotFound)
				return
			}

			http.Error(w, "Failed to fetch user info", http.StatusInternalServerError)
			return
		}

		files := fetchAllFiles(userID)
		userResponse := shared.AdminUserInfoResponse{
			ID:          user.ID,
			Email:       user.Email,
			StorageUsed: shared.ReadableFileSize(user.StorageUsed),
			SendUsed:    shared.ReadableFileSize(user.SendUsed),

			Files: files,
		}

		_ = json.NewEncoder(w).Encode(userResponse)
	}
}

func FileActionHandler(w http.ResponseWriter, req *http.Request, _ string) {
	segments := strings.Split(req.URL.Path, "/")
	fileID := segments[len(segments)-1]

	switch req.Method {
	case http.MethodDelete:
		err := deleteFile(fileID)
		if err != nil {
			http.Error(w, "Error deleting file", http.StatusInternalServerError)
			return
		}
	case http.MethodGet:
		fileInfo, err := fetchFileMetadata(fileID)
		if err == sql.ErrNoRows {
			http.Error(w, "No match found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("Error fetching file metadata: %v\n", err)
			http.Error(w, "Error fetching file metadata", http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(fileInfo)
	}

}
