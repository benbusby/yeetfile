package send

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
	"yeetfile/backend/config"
	"yeetfile/backend/db"
	"yeetfile/backend/server/session"
)

var OutOfSpaceError = errors.New("not enough space to upload")

// UserCanSend fetches the user ID associated with the request and checks to
// see if they have enough remaining send space to send a file
func UserCanSend(size int64, req *http.Request) (bool, error) {
	// Skip if send limits aren't configured
	if config.YeetFileConfig.DefaultUserSend < 0 {
		return true, nil
	}

	// Validate that the user has enough space to upload this file
	s, err := session.GetSession(req)
	if err != nil {
		return false, err
	}

	id := session.GetSessionUserID(s)
	usedSend, availableSend, err := db.GetUserSendLimits(id)
	if err != nil {
		log.Printf("Error validating ability to upload: %v\n", err)
		return false, err
	} else if availableSend-usedSend < size {
		log.Printf("[Send] Out of space: %d - %d > %d", availableSend, usedSend, size)
		return false, OutOfSpaceError
	}

	return true, nil
}

// UpdateUserMeter receives the size of an uploaded chunk and subtracts that
// value from the user's available storage meter
func UpdateUserMeter(size int, id string) error {
	err := db.UpdateUserSendUsed(id, size)
	if err != nil {
		log.Printf("Error while updating user storage: %v\n", err)
		return err
	}

	return nil
}

func validateSendDownloads(downloads int) error {
	maxDownloads := config.YeetFileConfig.MaxSendDownloads
	if downloads == 0 {
		return errors.New("downloads cannot be set to 0")
	} else if downloads > maxDownloads && maxDownloads != -1 {
		msg := fmt.Sprintf("downloads must be between 1-%d", maxDownloads)
		return errors.New(msg)
	} else if downloads == -1 && maxDownloads != -1 {
		return errors.New("downloads cannot be set to -1")
	}

	return nil
}

func validateSendExpiry(expiration time.Duration) error {
	if config.YeetFileConfig.MaxSendExpiry == -1 {
		return nil
	}

	maxDuration := config.YeetFileConfig.MaxSendExpiry
	maxDays := time.Duration(maxDuration) * time.Hour * 24

	if expiration > maxDays || expiration < 1 {
		msg := fmt.Sprintf("expiration must be between 1-%d days", maxDuration)
		return errors.New(msg)
	}

	return nil
}
