//go:build server_test

package api

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"
	"yeetfile/backend/config"
	"yeetfile/cli/crypto"
	"yeetfile/shared"
	"yeetfile/shared/endpoints"
)

var fileContent = "test"

// createRandomFolder creates a new folder within a parent folder and returns
// that folder's key, id, and any errors.
func createRandomFolder(
	user TestUser,
	parentFolderID string,
	parentKey []byte,
) ([]byte, string, error) {
	folderKey, _ := crypto.GenerateRandomKey()
	encName, _ := crypto.EncryptChunk(folderKey, []byte("My Folder"))
	hexName := hex.EncodeToString(encName)

	var protectedKey []byte
	if len(parentKey) == 0 {
		protectedKey, _ = crypto.EncryptRSA(UserA.pubKey, folderKey)
	} else {
		protectedKey, _ = crypto.EncryptChunk(parentKey, folderKey)
	}

	resp, err := user.context.CreateVaultFolder(shared.NewVaultFolder{
		Name:         hexName,
		ProtectedKey: protectedKey,
		ParentID:     parentFolderID,
	}, false)

	return folderKey, resp.ID, err
}

func uploadRandomFile(user TestUser, folderID string, folderKey []byte) (string, error) {
	upload, err := generateRandomUpload(user, folderID, folderKey)
	if err != nil {
		return "", err
	}

	meta, err := user.context.InitVaultFile(upload)

	if err != nil {
		return "", err
	}

	var key []byte
	if len(folderKey) == 0 {
		key, err = crypto.DecryptRSA(user.privKey, upload.ProtectedKey)
	} else {
		key, err = crypto.DecryptChunk(folderKey, upload.ProtectedKey)
	}

	if err != nil {
		return "", err
	}

	encData, err := crypto.EncryptChunk(key, []byte(fileContent))
	if err != nil {
		return "", err
	}

	url := endpoints.UploadVaultFileData.Format(server, meta.ID, "1")
	id, err := user.context.UploadFileChunk(url, encData)
	if err != nil {
		return "", err
	}

	if len(id) == 0 {
		return "", errors.New("file uploaded, but empty server response")
	}

	return id, nil
}

func generateRandomUpload(user TestUser, folderID string, folderKey []byte) (shared.VaultUpload, error) {
	name := shared.GenRandomString(12)
	key, err := crypto.GenerateRandomKey()
	if err != nil {
		return shared.VaultUpload{}, err
	}

	encName, err := crypto.EncryptChunk(key, []byte(name))
	if err != nil {
		return shared.VaultUpload{}, err
	}

	hexEncName := hex.EncodeToString(encName)

	var encKey []byte
	if len(folderKey) == 0 {
		encKey, err = crypto.EncryptRSA(user.pubKey, key)
	} else {
		encKey, err = crypto.EncryptChunk(folderKey, key)
	}

	if err != nil {
		return shared.VaultUpload{}, err
	}

	return shared.VaultUpload{
		Name:         hexEncName,
		Length:       int64(len(fileContent)),
		Chunks:       1,
		FolderID:     folderID,
		ProtectedKey: encKey,
	}, nil
}

func TestInitVaultFile(t *testing.T) {
	upload, err := generateRandomUpload(UserA, "", nil)
	if err != nil {
		t.Fatalf("Error generating upload: %v\n", err)
	}

	_, err = UserA.context.InitVaultFile(upload)

	if err != nil {
		t.Fatalf("Error initializing vault file: %v\n", err)
	}
}

func TestUploadVaultFile(t *testing.T) {
	upload, _ := generateRandomUpload(UserA, "", nil)
	meta, _ := UserA.context.InitVaultFile(upload)

	key, _ := crypto.DecryptRSA(UserA.privKey, upload.ProtectedKey)
	encData, _ := crypto.EncryptChunk(key, []byte(fileContent))

	url := endpoints.UploadVaultFileData.Format(server, meta.ID, "1")

	// Attempt uploading with a different user
	_, err := UserB.context.UploadFileChunk(url, encData)
	if err == nil {
		t.Fatalf("A different user was able to upload content for " +
			"a file that another user initiated!")
	}

	// Correct user
	id, err := UserA.context.UploadFileChunk(url, encData)
	if err != nil {
		t.Fatalf("Failed to upload file content")
	} else if len(id) == 0 {
		t.Fatal("File content was uploaded, but server response was empty")
	}
}

func TestDownloadFile(t *testing.T) {
	id, err := uploadRandomFile(UserA, "", nil)
	if err != nil {
		t.Fatal("Failed to upload random file")
	}

	// Test downloading metadata with another user
	_, err = UserB.context.GetVaultItemMetadata(id)
	if err == nil {
		t.Fatal("UserB was able to download metadata for a file UserA uploaded")
	}

	meta, err := UserA.context.GetVaultItemMetadata(id)
	if err != nil {
		t.Fatalf("Failed to download vault file metadata: %v\n", err)
	}

	// Test downloading file data with another user
	url := endpoints.DownloadVaultFileData.Format(server, meta.ID, "1")
	_, err = UserB.context.DownloadFileChunk(url)
	if err == nil {
		t.Fatal("UserB was able to download data for a file UserA uploaded")
	}

	encData, err := UserA.context.DownloadFileChunk(url)
	if err != nil {
		t.Fatalf("Failed to download vault file data: %v\n", err)
	}

	// Attempt decrypting key with UserB's key
	_, err = crypto.DecryptRSA(UserB.privKey, meta.ProtectedKey)
	if err == nil {
		t.Fatal("UserB was able to decrypt the file key for a file UserA uploaded")
	}

	key, err := crypto.DecryptRSA(UserA.privKey, meta.ProtectedKey)
	if err != nil {
		t.Fatalf("Error decrypting file key: %v\n", err)
	}

	data, err := crypto.DecryptChunk(key, encData)
	if err != nil {
		t.Fatalf("Error decrypting file data: %v\n", err)
	} else if string(data) != fileContent {
		t.Fatalf("Decrypted content doesn't match original: "+
			"'%s' (original) vs '%s' (decrypted)",
			string(data),
			fileContent)
	}
}

func TestVaultFolders(t *testing.T) {
	folderKey, _ := crypto.GenerateRandomKey()
	encName, _ := crypto.EncryptChunk(folderKey, []byte("My Folder"))
	hexName := hex.EncodeToString(encName)
	protectedKey, _ := crypto.EncryptRSA(UserA.pubKey, folderKey)

	resp, err := UserA.context.CreateVaultFolder(shared.NewVaultFolder{
		Name:         hexName,
		ProtectedKey: protectedKey,
		ParentID:     "",
	}, false)

	if err != nil {
		t.Fatalf("Error creating vault folder: %v\n", err)
	}

	fileID, err := uploadRandomFile(UserA, resp.ID, folderKey)
	if err != nil {
		t.Fatalf("Error uploading file to folder: %v\n", err)
	}

	_, err = uploadRandomFile(UserB, resp.ID, folderKey)
	if err == nil {
		t.Fatalf("UserB was able to upload a file to UserA's folder")
	}

	folder, err := UserA.context.FetchFolderContents(resp.ID, false)
	if err != nil {
		t.Fatalf("Error fetching folder contents: %v\n", err)
	}

	keyPair := crypto.IngestKeys(UserA.privKey, UserB.pubKey)
	decFolderKey, err := keyPair.UnwindKeySequence(folder.KeySequence)
	if err != nil {
		t.Fatalf("Error unwinding folder key sequence: %v\n", err)
	}

	assert.Equal(t, 1, len(folder.Items))
	assert.Equal(t, fileID, folder.Items[0].ID)
	assert.Equal(t, len(folderKey), len(decFolderKey))

	for i, b := range decFolderKey {
		assert.Equal(t, b, folderKey[i])
	}

	_, err = UserB.context.FetchFolderContents(resp.ID, false)
	if err == nil {
		t.Fatalf("UserB was able to fetch contents of UserA's folder")
	}

	file := folder.Items[0]
	fileKey, err := crypto.DecryptChunk(decFolderKey, file.ProtectedKey)
	if err != nil {
		t.Fatalf("Error decrypting file key within folder: %v\n", err)
	}

	fileNameBytes, _ := hex.DecodeString(file.Name)
	_, err = crypto.DecryptChunk(fileKey, fileNameBytes)
	if err != nil {
		t.Fatalf("Error decrypting file name: %v\n", err)
	}
}

func TestUploadPastLimit(t *testing.T) {
	account, err := UserA.context.GetAccountInfo()
	assert.Nil(t, err)

	used := account.StorageUsed
	fakeSize := account.StorageAvailable - account.StorageUsed
	realSize := fakeSize + 1

	name := shared.GenRandomString(12)
	key, err := crypto.GenerateRandomKey()
	assert.Nil(t, err)

	encName, err := crypto.EncryptChunk(key, []byte(name))
	assert.Nil(t, err)

	encData, err := crypto.EncryptChunk(key, []byte(strings.Repeat(".", int(realSize))))
	assert.Nil(t, err)

	hexEncName := hex.EncodeToString(encName)
	encKey, err := crypto.EncryptRSA(UserA.pubKey, key)
	assert.Nil(t, err)

	upload := shared.VaultUpload{
		Name:         hexEncName,
		Length:       realSize,
		Chunks:       1,
		FolderID:     "",
		ProtectedKey: encKey,
	}

	_, err = UserA.context.InitVaultFile(upload)
	assert.NotNil(t, err)

	upload.Length = fakeSize
	meta, err := UserA.context.InitVaultFile(upload)
	assert.Nil(t, err)

	url := endpoints.UploadVaultFileData.Format(server, meta.ID, "1")
	_, err = UserA.context.UploadFileChunk(url, encData)
	assert.NotNil(t, err)

	account, err = UserA.context.GetAccountInfo()
	assert.Equal(t, used, account.StorageUsed)
}

func TestDownloadLimiter(t *testing.T) {
	metaOnlyFileID, err := uploadRandomFile(UserB, "", nil)
	meta1, err := UserB.context.GetVaultItemMetadata(metaOnlyFileID)
	assert.Nil(t, err)
	meta2, err := UserB.context.GetVaultItemMetadata(metaOnlyFileID)
	assert.Nil(t, err)

	assert.Equal(t, meta1.ID, meta2.ID)

	fileID, err := uploadRandomFile(UserB, "", nil)
	assert.Nil(t, err)

	var downloadID string
	downloadFunc := func() error {
		var downloadErr error
		meta, downloadErr := UserB.context.GetVaultItemMetadata(fileID)
		if downloadErr != nil {
			return downloadErr
		}

		if len(downloadID) == 0 {
			assert.NotEqual(t, downloadID, meta.ID)
		}

		downloadID = meta.ID

		url := endpoints.DownloadVaultFileData.Format(server, meta.ID, "1")
		_, downloadErr = UserB.context.DownloadFileChunk(url)
		_, intentionalErr := UserB.context.DownloadFileChunk(url)
		assert.NotNil(t, intentionalErr)

		return downloadErr
	}

	attempt := 1
	for attempt <= config.YeetFileConfig.LimiterAttempts {
		err = downloadFunc()
		assert.Nil(t, err)
		attempt += 1
	}

	err = downloadFunc()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), fmt.Sprint(http.StatusTooManyRequests))
}

func TestPassVaultItem(t *testing.T) {
	username := "username"
	password := "password"
	url := "https://testing.com"

	passKey, _ := crypto.GenerateRandomKey()
	encName, _ := crypto.EncryptChunk(passKey, []byte("test"))
	hexName := hex.EncodeToString(encName)

	passEntry := shared.PassEntry{
		Username:        username,
		Password:        password,
		PasswordHistory: nil,
		URLs:            []string{url},
		Notes:           "",
	}

	jsonData, err := json.Marshal(passEntry)
	assert.Nil(t, err)

	encData, err := crypto.EncryptChunk(passKey, jsonData)
	assert.Nil(t, err)

	encKey, err := crypto.EncryptRSA(UserA.pubKey, passKey)
	assert.Nil(t, err)

	upload := shared.VaultUpload{
		Name:         hexName,
		Length:       1,
		Chunks:       1,
		FolderID:     "",
		ProtectedKey: encKey,
		PasswordData: encData,
	}

	meta, err := UserA.context.InitVaultFile(upload)
	assert.Nil(t, err)

	response, err := UserA.context.GetVaultItemMetadata(meta.ID)
	assert.Nil(t, err)
	assert.NotNil(t, response.PasswordData)
	assert.NotEmpty(t, response.PasswordData)

	decData, err := crypto.DecryptChunk(passKey, response.PasswordData)
	assert.Nil(t, err)

	var decPassEntry shared.PassEntry
	err = json.Unmarshal(decData, &decPassEntry)
	assert.Nil(t, err)

	assert.Equal(t, decPassEntry, passEntry)
}
