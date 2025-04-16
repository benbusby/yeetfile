package admin

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"yeetfile/backend/db"
	"yeetfile/backend/utils"
	"yeetfile/shared"
)

func UserActionHandler(w http.ResponseWriter, req *http.Request, id string) {
	segments := strings.Split(req.URL.Path, "/")
	userID := segments[len(segments)-1]

	switch req.Method {
	case http.MethodDelete:
		if userID == id {
			http.Error(w, "Cannot delete yourself", http.StatusBadRequest)
			return
		}

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
			ID:               user.ID,
			Email:            user.Email,
			StorageUsed:      user.StorageUsed,
			StorageAvailable: user.StorageAvailable,
			SendUsed:         user.SendUsed,
			SendAvailable:    user.SendAvailable,

			Files: files,
		}

		_ = json.NewEncoder(w).Encode(userResponse)
	case http.MethodPut:
		var action shared.AdminUserAction
		err := utils.LimitedJSONReader(w, req.Body).Decode(&action)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		user, err := getUserInfo(userID)
		if err != nil {
			http.Error(w, "No match found", http.StatusNotFound)
			return
		}

		sendErr := db.OverrideUserSend(user.ID, action.SendAvailable)
		storageErr := db.OverrideUserStorage(user.ID, action.StorageAvailable)
		if sendErr != nil || storageErr != nil {
			http.Error(w, "Error updating user storage/send", http.StatusInternalServerError)
			return
		}
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

func InviteActionsHandler(w http.ResponseWriter, req *http.Request, _ string) {
	var inviteAction shared.AdminInviteAction
	err := utils.LimitedJSONReader(w, req.Body).Decode(&inviteAction)
	if err != nil || len(inviteAction.Emails) == 0 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	switch req.Method {
	case http.MethodPost:
		err = createInvites(inviteAction.Emails)
		if err != nil {
			http.Error(w, "Error generating invites", http.StatusInternalServerError)
			return
		}
	case http.MethodDelete:
		err = deleteInvites(inviteAction.Emails)
		if err != nil {
			http.Error(w, "Error deleting invites", http.StatusInternalServerError)
			return
		}
	}
}
