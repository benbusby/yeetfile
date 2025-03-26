package send

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"yeetfile/cli/utils"

	"yeetfile/cli/crypto"
	"yeetfile/cli/globals"
	"yeetfile/cli/transfer"
	"yeetfile/shared"
)

type fileUpload struct {
	FilePath     string
	MaxDownloads int
	ExpUnits     string
	ExpValue     int
	Password     string
}

type textUpload struct {
	Text         string
	MaxDownloads int
	ExpUnits     string
	ExpValue     int
	Password     string
}

const (
	expMinutes = "minutes"
	expHours   = "hours"
	expDays    = "days"
)

func getDuration(value int64, units string) time.Duration {
	var duration time.Duration
	switch units {
	case expMinutes:
		duration = time.Duration(value * int64(time.Minute))
	case expHours:
		duration = time.Duration(value * int64(time.Hour))
	case expDays:
		duration = time.Duration(value * int64(time.Hour*24))
	}

	return duration
}

func getExpString(value int64, units string) string {
	duration := getDuration(value, units)
	return time.Now().Add(duration).Format("02 Jan 2006 15:04 MST")
}

func createTextLink(upload textUpload) (string, string, error) {
	key, salt, err := crypto.DeriveSendingKey(
		[]byte(upload.Password), nil)
	if err != nil {
		return "", "", err
	}

	encName, err := crypto.EncryptChunk(key, []byte(shared.GenRandomString(8)))
	if err != nil {
		return "", "", err
	}
	hexEncName := hex.EncodeToString(encName)
	encText, err := crypto.EncryptChunk(key, []byte(upload.Text))
	if err != nil {
		return "", "", err
	}

	encTextUpload := shared.TextUpload{
		Name:       hexEncName,
		Salt:       salt,
		Downloads:  upload.MaxDownloads,
		Expiration: createExpString(upload.ExpValue, upload.ExpUnits),
		Text:       encText,
	}

	id, err := globals.API.UploadText(encTextUpload)
	if err != nil {
		return "", "", err
	}

	if len(upload.Password) > 0 {
		return id, utils.B64Encode(salt), nil
	} else {
		return id, utils.B64Encode(key), nil
	}
}

func createFileLink(upload fileUpload, progress func(int, int)) (string, string, error) {
	key, salt, err := crypto.DeriveSendingKey(
		[]byte(upload.Password), nil)
	if err != nil {
		return "", "", err
	}

	file, stat, err := shared.GetFileInfo(upload.FilePath)

	encName, err := crypto.EncryptChunk(key, []byte(stat.Name()))
	hexEncName := hex.EncodeToString(encName)
	size := stat.Size()
	numChunks := transfer.GetNumChunks(stat.Size())

	metadata := shared.UploadMetadata{
		Name:       hexEncName,
		Chunks:     numChunks,
		Size:       size,
		Downloads:  upload.MaxDownloads,
		Expiration: createExpString(upload.ExpValue, upload.ExpUnits),
	}

	pending, err := transfer.InitSendFile(file, metadata, key)
	if err != nil {
		return "", "", err
	}

	chunk := 0
	result, err := pending.UploadData(func() {
		chunk += 1
		progress(chunk, pending.NumChunks)
	})
	if err != nil {
		return "", "", err
	}

	if len(upload.Password) > 0 {
		return result, utils.B64Encode(salt), nil
	} else {
		return result, utils.B64Encode(key), nil
	}
}

func createExpString(expValue int, expUnits string) string {
	return fmt.Sprintf("%d%s", expValue, strings.ToLower(string(expUnits[0])))
}

func validateSendDownloads(val int) error {
	maxDownloads := globals.ServerInfo.MaxSendDownloads
	if maxDownloads == -1 && (val < -1 || val == 0) {
		return errors.New("downloads must be either -1 or > 0")
	}

	if val <= 0 && maxDownloads != -1 {
		return errors.New("downloads must be >= 1")
	} else if val > maxDownloads && maxDownloads != -1 {
		msg := fmt.Sprintf("downloads must be <= %d", maxDownloads)
		return errors.New(msg)
	}

	return nil
}

func validateSendExpiry(val int, units string) error {
	maxExpiryDays := globals.ServerInfo.MaxSendExpiry
	if maxExpiryDays == -1 && (val < -1 || val == 0) {
		return errors.New("expiration value must be -1 or > 0")
	}

	if val <= 0 && maxExpiryDays != -1 {
		return errors.New("expiration must be >= 1")
	} else if maxExpiryDays != -1 {
		duration := getDuration(int64(val), units)
		maxDuration := getDuration(int64(maxExpiryDays), "d")

		if duration > maxDuration {
			msg := fmt.Sprintf("expiration must be < %d days", maxExpiryDays)
			return errors.New(msg)
		}
	}

	return nil
}
