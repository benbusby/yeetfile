package api

import (
	"encoding/json"
	"log"
	"net/http"
	"yeetfile/cli/requests"
	"yeetfile/shared"
	"yeetfile/shared/endpoints"
)

// InitVaultFile initializes a new vault upload using the contents of a
// shared.VaultUpload struct.
func (ctx *Context) InitVaultFile(
	upload shared.VaultUpload,
) (shared.MetadataUploadResponse, error) {
	reqData, err := json.Marshal(upload)
	if err != nil {
		return shared.MetadataUploadResponse{}, err
	}

	url := endpoints.UploadVaultFileMetadata.Format(ctx.Server)
	resp, err := requests.PostRequest(ctx.Session, url, reqData)
	if err != nil {
		return shared.MetadataUploadResponse{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.MetadataUploadResponse{}, shared.ParseHTTPError(resp)
	}

	var metaResponse shared.MetadataUploadResponse
	err = json.NewDecoder(resp.Body).Decode(&metaResponse)
	if err != nil {
		log.Println("Error decoding server response: ", err)
		return shared.MetadataUploadResponse{}, err
	}

	return metaResponse, nil
}

// GetVaultItemMetadata retrieves metadata for a file using the file's ID.
// Returns the metadata response and any errors.
func (ctx *Context) GetVaultItemMetadata(
	id string,
) (shared.VaultDownloadResponse, error) {
	url := endpoints.DownloadVaultFileMetadata.Format(ctx.Server, id)
	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return shared.VaultDownloadResponse{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.VaultDownloadResponse{}, shared.ParseHTTPError(resp)
	}

	var metadata shared.VaultDownloadResponse
	err = json.NewDecoder(resp.Body).Decode(&metadata)
	if err != nil {
		return shared.VaultDownloadResponse{}, err
	}

	return metadata, nil
}

// FetchFolderContents fetches the contents of a folder in the user's vault
// using the folder's ID. The ID can be left empty to fetch the user's home
// vault folder.
func (ctx *Context) FetchFolderContents(
	id string,
	isPassVault bool,
) (shared.VaultFolderResponse, error) {
	var endpoint endpoints.Endpoint
	if isPassVault {
		endpoint = endpoints.PassFolder
	} else {
		endpoint = endpoints.VaultFolder
	}

	url := endpoint.Format(ctx.Server, id)
	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return shared.VaultFolderResponse{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.VaultFolderResponse{}, shared.ParseHTTPError(resp)
	}

	var folderResp shared.VaultFolderResponse
	err = json.NewDecoder(resp.Body).Decode(&folderResp)
	if err != nil {
		return shared.VaultFolderResponse{}, err
	}

	return folderResp, nil
}

// CreateVaultFolder creates a new folder in the user's vault
func (ctx *Context) CreateVaultFolder(
	newFolder shared.NewVaultFolder,
	isPassVault bool,
) (shared.NewFolderResponse, error) {
	var endpoint endpoints.Endpoint
	if isPassVault {
		endpoint = endpoints.PassFolder
	} else {
		endpoint = endpoints.VaultFolder
	}

	reqData, err := json.Marshal(newFolder)
	if err != nil {
		return shared.NewFolderResponse{}, err
	}

	url := endpoint.Format(ctx.Server)
	resp, err := requests.PostRequest(ctx.Session, url, reqData)
	if err != nil {
		return shared.NewFolderResponse{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.NewFolderResponse{}, shared.ParseHTTPError(resp)
	}

	var folderResponse shared.NewFolderResponse
	err = json.NewDecoder(resp.Body).Decode(&folderResponse)
	if err != nil {
		log.Println("error decoding new folder response")
		return shared.NewFolderResponse{}, err
	}

	return folderResponse, nil
}

func (ctx *Context) DeleteVaultFile(id string, isShared bool) error {
	url := endpoints.VaultFile.Format(ctx.Server, id)
	if isShared {
		url += "?shared=true"
	}
	return deleteItem(ctx.Session, url)
}

func (ctx *Context) DeleteVaultFolder(id string, isShared bool) error {
	url := endpoints.VaultFolder.Format(ctx.Server, id)
	if isShared {
		url += "?shared=true"
	}
	return deleteItem(ctx.Session, url)
}

func (ctx *Context) ModifyVaultFile(
	id string,
	mod shared.ModifyVaultItem,
) error {
	url := endpoints.VaultFile.Format(ctx.Server, id)
	return modifyItem(ctx.Session, url, mod)
}

func (ctx *Context) ModifyVaultFolder(
	id string,
	mod shared.ModifyVaultItem,
) error {
	url := endpoints.VaultFolder.Format(ctx.Server, id)
	return modifyItem(ctx.Session, url, mod)
}

func deleteItem(session, url string) error {
	resp, err := requests.DeleteRequest(session, url, nil)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return shared.ParseHTTPError(resp)
	}

	return nil
}

func modifyItem(
	session,
	url string,
	mod shared.ModifyVaultItem,
) error {
	reqData, err := json.Marshal(mod)
	if err != nil {
		return err
	}

	resp, err := requests.PutRequest(session, url, reqData)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return shared.ParseHTTPError(resp)
	}

	return nil
}
