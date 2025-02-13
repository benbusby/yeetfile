package vault

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"yeetfile/backend/cache"
	"yeetfile/backend/config"
	"yeetfile/backend/db"
	"yeetfile/backend/server/session"
	"yeetfile/backend/server/transfer"
	"yeetfile/backend/storage"
	"yeetfile/backend/utils"
	"yeetfile/shared"
	"yeetfile/shared/constants"
	"yeetfile/shared/endpoints"
)

type vaultType int

const (
	FileVault vaultType = iota
	PassVault
)

type vaultFn func(w http.ResponseWriter, req *http.Request, userID string, passVault bool)

// FileHandler directs all requests to the appropriate handler for interacting
// with YeetFile Vault files
func FileHandler(w http.ResponseWriter, req *http.Request, userID string) {

	var fn session.HandlerFunc
	switch req.Method {
	case http.MethodGet:
		fn = GetFileHandler
	case http.MethodPut, http.MethodDelete:
		fn = ModifyFileHandler
	}

	fn(w, req, userID)
}

// FolderHandler directs all requests to the appropriate handler for
// interacting with file or password folders in YeetFile Vault
func FolderHandler(vType vaultType) session.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request, userID string) {
		var fn vaultFn
		switch req.Method {
		case http.MethodPut, http.MethodDelete:
			fn = modifyFolderHandler
		case http.MethodPost:
			fn = newFolderHandler
		case http.MethodGet:
			fn = folderViewHandler
		}

		fn(w, req, userID, vType == PassVault)
	}
}

// FolderViewHandler returns folder contents for the requested folder. If a
// folder ID wasn't included in the request, the user's root level folder
// (distinguished by having the same ID as their account) is returned.
func folderViewHandler(w http.ResponseWriter, req *http.Request, userID string, passVault bool) {
	var folderID string
	segments := utils.GetTrailingURLSegments(
		req.URL.Path,
		endpoints.VaultFolder,
		endpoints.PassFolder)
	if len(segments) == 0 || len(segments[0]) == 0 {
		folderID = userID
	} else {
		folderID = segments[0]
	}

	items, ownership, err := db.GetVaultItems(userID, folderID, passVault)
	if err != nil {
		log.Printf("Error fetching vault items: %v\n", err)

		if err == db.AccessError {
			http.Error(w, "Unauthorized access",
				http.StatusForbidden)
		} else {
			http.Error(w, "Error fetching vault items",
				http.StatusInternalServerError)
		}

		return
	}

	folder, err := db.GetFolderInfo(folderID, userID, ownership, false)
	if err != nil {
		log.Printf("Error fetching folder info: %v\n", err)
		http.Error(w, "Error fetching folder info", http.StatusInternalServerError)
		return
	}

	folders, err := db.GetSubfolders(folderID, userID, ownership, passVault)
	if err != nil {
		log.Printf("Error fetching subfolders: %v\n", err)
		http.Error(w, "Error fetching subfolders", http.StatusInternalServerError)
		return
	}

	keySequence, err := db.GetKeySequence(folderID, userID)
	if err != nil {
		log.Printf("Error fetching key sequence: %v\n", err)
		http.Error(w, "Error fetching key sequence", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(shared.VaultFolderResponse{
		Items:         items,
		Folders:       folders,
		CurrentFolder: folder,
		KeySequence:   keySequence,
	})
}

// newFolderHandler handles the creation of vault folders
func newFolderHandler(w http.ResponseWriter, req *http.Request, userID string, passVault bool) {
	var folder shared.NewVaultFolder
	err := utils.LimitedJSONReader(w, req.Body).Decode(&folder)
	if err != nil {
		log.Printf("Error decoding request body: %v\n", err)
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	folderID, err := db.NewFolder(folder, userID, passVault)
	if err != nil {
		log.Printf("Error creating new folder: %v\n", err)
		http.Error(w, "Error creating new folder", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(shared.NewFolderResponse{ID: folderID})
	if err != nil {
		http.Error(w, "Error sending response", http.StatusInternalServerError)
		return
	}
}

// modifyFolderHandler receives request to change or delete an existing folder.
func modifyFolderHandler(w http.ResponseWriter, req *http.Request, userID string, passVault bool) {
	segments := strings.Split(req.URL.Path, "/")
	idPart := strings.Split(segments[len(segments)-1], "?")
	id := idPart[0]

	isShared := len(req.URL.Query().Get("shared")) > 0

	var modErr error
	switch req.Method {
	case http.MethodPut:
		var folderMod shared.ModifyVaultItem
		modErr = utils.LimitedJSONReader(w, req.Body).Decode(&folderMod)
		if modErr != nil {
			break
		}

		modErr = updateVaultFolder(id, userID, folderMod)
		break
	case http.MethodDelete:
		freed, err := DeleteVaultFolder(id, userID, isShared, passVault)
		if err != nil {
			log.Printf("Error deleting folder: %v\n", err)
			http.Error(w, "Error deleting folder", http.StatusInternalServerError)
			return
		}

		resp := shared.DeleteResponse{FreedSpace: freed}
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			http.Error(w, "Error sending response", http.StatusInternalServerError)
		}
		break
	}

	if modErr != nil {
		http.Error(w, "Error modifying folder", http.StatusBadRequest)
	}
}

// GetFileHandler handlers requests for information related to a vault file
func GetFileHandler(w http.ResponseWriter, req *http.Request, userID string) {
	segments := strings.Split(req.URL.Path, "/")
	idPart := strings.Split(segments[len(segments)-1], "?")
	id := idPart[0]

	info, err := db.RetrieveFullItemInfo(id, userID)
	if err != nil {
		log.Printf("Error retrieving file info: %v\n", err)
		http.Error(w, "Error retrieving file info", http.StatusBadRequest)
		return
	}

	err = json.NewEncoder(w).Encode(info)
	if err != nil {
		http.Error(w, "Error sending response", http.StatusInternalServerError)
		return
	}
}

// ModifyFileHandler handles requests to modify an existing file in the user's vault
func ModifyFileHandler(w http.ResponseWriter, req *http.Request, userID string) {
	segments := strings.Split(req.URL.Path, "/")
	idPart := strings.Split(segments[len(segments)-1], "?")
	id := idPart[0]

	isShared := len(req.URL.Query().Get("shared")) > 0

	var modErr error
	var modResponse []byte
	switch req.Method {
	case http.MethodPut:
		var fileMod shared.ModifyVaultItem
		modErr = utils.LimitedJSONReader(w, req.Body).Decode(&fileMod)
		if modErr != nil {
			log.Printf("Error decoding request: %v\n", modErr)
			break
		}
		modErr = updateVaultFile(id, userID, fileMod)
		break
	case http.MethodDelete:
		var freed int64
		freed, modErr = deleteVaultFile(id, userID, isShared)

		if modErr == nil {
			modResponse, _ = json.Marshal(shared.DeleteResponse{FreedSpace: freed})
		}
		break
	}

	if modErr != nil {
		log.Printf("Error modifying file: %v\n", modErr)
		http.Error(w, "Error modifying file", http.StatusBadRequest)
	} else if modResponse != nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(modResponse)
	}
}

// UploadMetadataHandler initializes a file in the user's vault
func UploadMetadataHandler(w http.ResponseWriter, req *http.Request, userID string) {
	var upload shared.VaultUpload
	err := utils.LimitedJSONReader(w, req.Body).Decode(&upload)
	if err != nil {
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	if upload.PasswordData == nil || len(upload.PasswordData) == 0 {
		err = CanUserUpload(upload.Length, userID, upload.FolderID)
		if err != nil {
			log.Printf("Error checking if user can upload file: %v\n", err)
			http.Error(w, "Not enough storage available", http.StatusBadRequest)
			return
		}
	}

	itemID, err := db.AddVaultItem(userID, upload)
	if err != nil {
		log.Printf("Error initializing vault upload: %v\n", err)
		http.Error(w, "Error initializing vault upload", http.StatusBadRequest)
		return
	}

	if upload.PasswordData != nil && len(upload.PasswordData) > 0 {
		// Exit early if the user is uploading an encrypted password
		// (not stored in B2)
		err = json.NewEncoder(w).Encode(shared.MetadataUploadResponse{ID: itemID})
		if err != nil {
			http.Error(w, "Error sending response", http.StatusInternalServerError)
		}
		return
	}

	err = db.CreateNewUpload(itemID, upload.Name)
	if err != nil {
		log.Printf("Error initializing new upload: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if upload.Chunks == 1 {
		err = storage.Interface.InitUpload(itemID)
	} else {
		err = storage.Interface.InitLargeUpload(upload.Name, itemID)
	}

	if err != nil {
		http.Error(w, "Error initializing storage", http.StatusInternalServerError)
		_ = db.DeleteVaultFile(itemID, userID)
		return
	}

	err = json.NewEncoder(w).Encode(shared.MetadataUploadResponse{ID: itemID})
	if err != nil {
		http.Error(w, "Error sending response", http.StatusInternalServerError)
		return
	}
}

// UploadDataHandler processes incoming chunks of encrypted file data for a
// vault file
func UploadDataHandler(w http.ResponseWriter, req *http.Request, userID string) {
	segments := strings.Split(req.URL.Path, "/")
	id := segments[len(segments)-2]
	chunkNum, err := strconv.Atoi(segments[len(segments)-1])
	if err != nil {
		http.Error(w, "Invalid upload URL", http.StatusBadRequest)
		return
	}

	metadata, err := db.RetrieveVaultMetadata(id, userID)
	if err != nil {
		log.Printf("[YF Vault] Error fetching metadata: %v\n", err)
		http.Error(w, "No metadata found", http.StatusBadRequest)
		return
	}

	data, err := utils.LimitedChunkReader(w, req.Body)
	if err != nil {
		log.Printf("[YF Vault] Error reading uploaded data: %v\n", err)
		http.Error(w, "Error reading request", http.StatusBadRequest)
		abortUpload(metadata, userID, 0, chunkNum)
		return
	}

	if chunkNum > metadata.Chunks {
		log.Printf("[YF Vault] User uploading beyond stated # of chunks")
		http.Error(w, "Attempting to upload more chunks than specified",
			http.StatusBadRequest)
		abortUpload(metadata, userID, 0, chunkNum)
		return
	}

	totalSize := int64(len(data)) - int64(constants.TotalOverhead)

	if metadata.OwnsParentFolder {
		err = db.UpdateStorageUsed(userID, totalSize)
	} else {
		err = db.UpdateFolderOwnerStorage(metadata.FolderID, totalSize)
	}

	if err != nil {
		if metadata.OwnsParentFolder {
			abortUpload(metadata, userID, totalSize, chunkNum)
		}
		http.Error(w, "Attempting to upload beyond max storage",
			http.StatusBadRequest)
		return
	}

	fileChunk, uploadValues, err := transfer.PrepareUpload(metadata, chunkNum, data)
	if err != nil {
		http.Error(w, "Unable to initialize chunk upload",
			http.StatusBadRequest)
		abortUpload(metadata, userID, totalSize, chunkNum)
		return
	}

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
		http.Error(w, "Error uploading file", http.StatusBadRequest)
		log.Printf("[YF Vault] Error uploading file: %v\n", err)
		abortUpload(metadata, userID, totalSize, chunkNum)
		return
	}

	if finishedUploading {
		_, _ = io.WriteString(w, id)
	}
}

// DownloadHandler handles incoming requests for metadata pertaining to a file
// in the vault that a user wants to download
func DownloadHandler(w http.ResponseWriter, req *http.Request, userID string) {
	segments := strings.Split(req.URL.Path, "/")
	id := segments[len(segments)-1]

	metadata, err := db.RetrieveVaultMetadata(id, userID)
	if err != nil {
		log.Println("Error fetching metadata:", err)
		http.Error(w, "Error fetching metadata", http.StatusBadRequest)
		return
	}

	// If storage limits are in place, track bandwidth usage to prevent
	// excessive repeated downloads
	if config.YeetFileConfig.DefaultUserStorage > 0 {
		bandwidth, err := db.GetUserBandwidth(userID)
		if err != nil {
			log.Println("Server error:", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		} else if bandwidth < metadata.Length {
			log.Printf("Bandwidth limit triggered")
			http.Error(w, "Bandwidth limit reached -- contact YeetFile "+
				"support or try again tomorrow.", http.StatusForbidden)
			return
		}
	}

	var downloadID string
	if metadata.PasswordData == nil || len(metadata.PasswordData) == 0 {
		downloadID, err = db.InitDownload(metadata.RefID, userID, metadata.Chunks)
		if err != nil {
			log.Println("Error initializing download:", err)
			http.Error(w, "Error initializing download", http.StatusInternalServerError)
			return
		}
	}

	response := shared.VaultDownloadResponse{
		Name:         metadata.Name,
		ID:           downloadID,
		Chunks:       metadata.Chunks,
		Size:         metadata.Length,
		ProtectedKey: metadata.ProtectedKey,
		PasswordData: metadata.PasswordData,
	}

	jsonData, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonData)
}

// DownloadChunkHandler handles requests for encrypted file data for a file in
// the user's vault
func DownloadChunkHandler(w http.ResponseWriter, req *http.Request, userID string) {
	segments := strings.Split(req.URL.Path, "/")

	if len(segments) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	id := segments[len(segments)-2]
	chunk, _ := strconv.Atoi(segments[len(segments)-1])
	if chunk <= 0 {
		chunk = 1 // Downloads always begin with chunk 1
	}

	metadataID, err := db.GetDownload(id, userID)
	if err != nil {
		log.Printf("Error fetching download ID: %v\n", err)
		http.Error(w, "Error fetching download info", http.StatusInternalServerError)
		return
	}

	metadata, err := db.RetrieveVaultMetadata(metadataID, userID)
	if err != nil {
		log.Printf("Error fetching metadata: %v\n", err)
		http.Error(w, "No metadata found", http.StatusBadRequest)
		return
	}

	var bytes []byte
	if cache.HasFile(id, metadata.Length) {
		_, bytes = transfer.DownloadFileFromCache(id, metadata.Length, chunk)
	} else {
		cache.PrepCache(id, metadata.Length)
		_, bytes = transfer.DownloadFile(
			metadata.B2ID,
			metadata.Name,
			metadata.Length,
			chunk)
		_ = cache.Write(id, bytes)
	}

	err = db.UpdateDownload(id)
	if err != nil {
		log.Printf("Error updating download: %v\n", err)
	}

	err = db.UpdateBandwidth(userID, int64(len(bytes)-constants.TotalOverhead))
	if err != nil {
		log.Printf("Error updating bandwidth: %v\n", err)
	}

	_, _ = w.Write(bytes)
}

// ShareHandler handles requests to share files or folders within the user's
// vault, as well as modifying the shared state of those files/folders
func ShareHandler(isFolder bool) session.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request, userID string) {
		segments := strings.Split(req.URL.Path, "/")
		itemID := segments[len(segments)-1]

		if len(itemID) != db.VaultIDLength {
			http.Error(w, "Invalid item ID", http.StatusBadRequest)
			return
		}

		var shareErr error
		switch req.Method {
		case http.MethodPost:
			var share shared.ShareItemRequest
			err := utils.LimitedJSONReader(w, req.Body).Decode(&share)
			if err != nil {
				http.Error(w, "Error decoding request",
					http.StatusBadRequest)
				return
			}

			var shareInfo shared.ShareInfo
			shareInfo, shareErr = shareVaultItem(share, itemID, userID, isFolder)
			if shareErr == nil {
				jsonData, _ := json.Marshal(shareInfo)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(jsonData)
				return
			}
			break
		case http.MethodGet:
			shares, err := db.GetShareInfo(userID, itemID, isFolder)
			if err != nil {
				http.Error(w, "Error fetching share info", http.StatusInternalServerError)
			}

			jsonData, _ := json.Marshal(shares)

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(jsonData)
			return
		case http.MethodPut:
			var edit shared.ShareEdit
			err := utils.LimitedJSONReader(w, req.Body).Decode(&edit)
			if err != nil {
				http.Error(w, "Error decoding request",
					http.StatusBadRequest)
				return
			}

			shareErr = db.ModifyShare(userID, edit, isFolder)
		case http.MethodDelete:
			shareID := req.URL.Query().Get("id")
			if len(shareID) == 0 {
				http.Error(w, "Missing 'id' param", http.StatusBadRequest)
				return
			}

			shareErr = db.RemoveShare(userID, itemID, shareID, isFolder)
		}

		if shareErr != nil {
			log.Printf("Error with shared content: %v\n", shareErr)
			http.Error(w, "Error with shared content", http.StatusBadRequest)
			return
		}
	}
}
