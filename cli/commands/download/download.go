package download

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"yeetfile/cli/crypto"
	"yeetfile/cli/globals"
	"yeetfile/cli/utils"
	"yeetfile/shared"
	"yeetfile/shared/constants"
)

var decryptError = errors.New("decryption error")

type DownloadResource struct {
	Server string
	ItemID string
	Secret []byte
}

type PreparedDownload struct {
	ID         string
	Server     string
	Name       string
	Size       int64
	Chunks     int
	Key        []byte
	Expiration time.Time
	Downloads  int
	IsText     bool
}

func parseLink(link string) DownloadResource {
	link = strings.Replace(link, "/send/", "/", 1)
	linkSegments := strings.Split(link, "/")
	server := strings.Join(linkSegments[0:len(linkSegments)-1], "/")
	resource := strings.Split(linkSegments[len(linkSegments)-1], "#")

	var id string
	var secret string
	if len(resource) == 1 {
		id = resource[0]
	} else if len(resource) > 1 {
		id = resource[0]
		secret = resource[1]
	}

	return DownloadResource{
		Server: server,
		ItemID: id,
		Secret: utils.B64Decode(secret),
	}
}

func prepDownload(link string) (PreparedDownload, error) {
	d := parseLink(link)
	metadata, err := d.fetchMetadata()
	if err != nil {
		return PreparedDownload{}, err
	}

	name, key, err := d.decryptResponse(metadata, "")
	if err != nil {
		return PreparedDownload{}, err
	}

	prep := PreparedDownload{
		ID:         metadata.ID,
		Name:       name,
		Key:        key,
		Size:       metadata.Size,
		Chunks:     metadata.Chunks,
		Expiration: metadata.Expiration,
		Downloads:  metadata.Downloads,
		Server:     d.Server,
		IsText:     strings.HasPrefix(metadata.ID, constants.TextIDPrefix),
	}

	return prep, nil
}

func (d DownloadResource) fetchMetadata() (shared.DownloadResponse, error) {
	return globals.API.FetchSendFileMetadata(d.Server, d.ItemID)
}

// decryptResponse decrypts the name from the shared.DownloadResponse returned
// by the server, and returns the decrypted name and the key that was successfully
// used to decrypt the name.
func (d DownloadResource) decryptResponse(
	response shared.DownloadResponse,
	password string,
) (string, []byte, error) {
	var key []byte
	var err error
	if len(password) == 0 {
		key = d.Secret
	} else {
		key, _, err = crypto.DeriveSendingKey([]byte(password), d.Secret)
	}

	if err != nil {
		return "", nil, err
	}

	encName, err := hex.DecodeString(response.Name)
	if err != nil {
		return "", nil, err
	}

	decName, err := crypto.DecryptChunk(key, encName)
	if err != nil {
		// Decryption error, likely means the file is password protected
		var newPassword string
		if password == "" {
			newPassword, err = showPasswordPromptModel(nil)
		} else {
			newPassword, err = showPasswordPromptModel(decryptError)
		}

		if err != nil {
			return "", nil, err
		}

		return d.decryptResponse(response, newPassword)
	}

	return string(decName), key, nil
}

func generateDescription(download PreparedDownload) string {
	name := download.Name
	if strings.HasPrefix(download.ID, constants.TextIDPrefix) {
		name = "N/A (text-only)"
	}

	timeDiff := download.Expiration.Sub(time.Now())

	return fmt.Sprintf(""+
		"- Name: %s\n"+
		"- Size: %s\n"+
		"- Expiration: %s (%s)\n"+
		"- Downloads Remaining: %d\n",
		name,
		shared.ReadableFileSize(download.Size),
		utils.LocalTimeFromUTC(download.Expiration),
		timeDiff,
		download.Downloads,
	)
}
