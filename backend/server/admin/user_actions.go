package admin

import (
	"log"
	"strings"
	"yeetfile/backend/db"
	"yeetfile/backend/server/auth"
	"yeetfile/shared"
)

func deleteUser(userID string) error {
	return auth.DeleteUser(userID, shared.DeleteAccount{Identifier: userID})
}

func fetchAllFiles(userID string) []shared.AdminFileInfoResponse {
	if strings.Contains(userID, "@") {
		userID, _ = db.GetUserIDByEmail(userID)
	}

	files := []shared.AdminFileInfoResponse{}
	vaultFiles, err := db.AdminFetchVaultFiles(userID)
	if err != nil {
		log.Printf("Error fetching user files: %v\n", err)
	}

	files = append(files, vaultFiles...)

	sendFiles, err := db.AdminFetchSentFiles(userID)
	if err != nil {
		log.Printf("Error fetching user send files: %v\n", err)
	}

	files = append(files, sendFiles...)
	return files
}

func getUserInfo(userID string) (db.User, error) {
	var err error
	if strings.Contains(userID, "@") {
		userID, err = db.GetUserIDByEmail(userID)
		if err != nil {
			return db.User{}, err
		}
	}

	user, err := db.GetUserByID(userID)
	if err != nil {
		return db.User{}, err
	}

	return user, nil
}
