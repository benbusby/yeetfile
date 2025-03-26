//go:build server_test

package api

import (
	"encoding/hex"
	"testing"
	"yeetfile/cli/crypto"
	"yeetfile/shared"
	"yeetfile/shared/endpoints"
)

func TestUploadText(t *testing.T) {
	text := "top secret text"
	password := "topsecret"
	key, salt, err := crypto.DeriveSendingKey([]byte(password), nil)
	if err != nil {
		t.Fatalf("Error deriving sending keys: %v\n", err)
	}

	encText, err := crypto.EncryptChunk(key, []byte(text))
	if err != nil {
		t.Fatalf("Error encrypting text: %v\n", err)
	}

	name := shared.GenRandomString(12)
	encName, _ := crypto.EncryptChunk(key, []byte(name))
	hexName := hex.EncodeToString(encName)

	id, err := UserA.context.UploadText(shared.TextUpload{
		Name:       hexName,
		Salt:       salt,
		Downloads:  1,
		Expiration: "5m",
		Text:       encText,
	})

	if err != nil {
		t.Fatalf("Error uploading encrypted text to server: %v\n", err)
	}

	// Download as UserB (sent files/text are accessible to whoever has the url)
	url := endpoints.DownloadSendFileData.Format(server, id, "1")
	data, err := UserB.context.DownloadFileChunk(url)
	if err != nil {
		t.Fatalf("Error downloading encrypted text: %v\n", err)
	}

	newKey, _, err := crypto.DeriveSendingKey([]byte(password), salt)
	if err != nil {
		t.Fatalf("Error deriving new key from previous salt")
	}

	decData, err := crypto.DecryptChunk(newKey, data)
	if err != nil {
		t.Fatalf("Error decrypting encrypted text: %v\n", err)
	} else if string(decData) != text {
		t.Fatalf("Decrypted text content does not match original\n"+
			"original: '%s', decrypted: '%s'",
			text,
			string(decData))
	}

	// Downloaded file should now not exist (only set to 1 download)
	_, err = UserA.context.DownloadFileChunk(url)
	if err == nil {
		t.Fatalf("Uploaded text was set to 1 download, but data can be " +
			"downloaded more than once.")
	}

	_, err = UserA.context.FetchSendFileMetadata(server, id)
	if err == nil {
		t.Fatalf("Uploaded text was set to 1 download, but metadata " +
			"can be downloaded more than once.")
	}
}
