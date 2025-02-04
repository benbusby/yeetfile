package admin

import (
	"strings"
	"yeetfile/backend/db"
	"yeetfile/backend/server/auth"
	"yeetfile/shared"
)

func deleteUser(userID string) error {
	return auth.DeleteUser(userID, shared.DeleteAccount{Identifier: userID})
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
