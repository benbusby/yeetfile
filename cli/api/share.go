package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"yeetfile/cli/lang"
	"yeetfile/cli/requests"
	"yeetfile/shared"
	"yeetfile/shared/endpoints"
)

// FetchUserPubKey fetches a YeetFile user's public key, which is used to
// encrypt a file or folder's key before sharing the content with them.
func (ctx *Context) FetchUserPubKey(
	userIdentifier string,
) (shared.PubKeyResponse, error) {
	url := endpoints.PubKey.Format(ctx.Server)
	url = url + fmt.Sprintf("?user=%s", userIdentifier)

	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return shared.PubKeyResponse{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.PubKeyResponse{}, shared.ParseHTTPError(resp)
	}

	var pubKeyResponse shared.PubKeyResponse
	err = json.NewDecoder(resp.Body).Decode(&pubKeyResponse)
	if err != nil {
		return shared.PubKeyResponse{}, err
	}

	return pubKeyResponse, nil
}

// ShareFileWithUser shares a new file with a user. This file will appear in the
// recipient's home folder.
func (ctx *Context) ShareFileWithUser(
	request shared.ShareItemRequest,
	fileID string,
) (shared.ShareInfo, error) {
	url := endpoints.ShareFile.Format(ctx.Server, fileID)
	return shareContentWithUser(ctx.Session, url, request)
}

// ShareFolderWithUser shares a new folder with a user. This folder will appear
// in the recipient's home folder.
func (ctx *Context) ShareFolderWithUser(
	request shared.ShareItemRequest,
	folderID string,
) (shared.ShareInfo, error) {
	url := endpoints.ShareFolder.Format(ctx.Server, folderID)
	return shareContentWithUser(ctx.Session, url, request)
}

// GetSharedFileInfo retrieves a list of shares that are active with the
// specified file
func (ctx *Context) GetSharedFileInfo(id string) ([]shared.ShareInfo, error) {
	url := endpoints.ShareFile.Format(ctx.Server, id)
	return getSharedInfo(ctx.Session, url)
}

// GetSharedFolderInfo retrieves a list of shares that are active with the
// specified folder
func (ctx *Context) GetSharedFolderInfo(id string) ([]shared.ShareInfo, error) {
	url := endpoints.ShareFolder.Format(ctx.Server, id)
	return getSharedInfo(ctx.Session, url)
}

// RemoveSharedFileUsers takes a list of shared users to remove access to a file,
// returning a list of the successfully removed users and any errors encountered.
func (ctx *Context) RemoveSharedFileUsers(
	fileID string,
	remove []shared.ShareInfo,
) ([]shared.ShareInfo, error) {
	url := endpoints.ShareFile.Format(ctx.Server, fileID)
	return removeSharedUsers(ctx.Session, url, remove)
}

// RemoveSharedFolderUsers takes a list of shared users to remove access to a folder,
// returning a list of the successfully removed users and any errors encountered.
func (ctx *Context) RemoveSharedFolderUsers(
	folderID string,
	remove []shared.ShareInfo,
) ([]shared.ShareInfo, error) {
	url := endpoints.ShareFolder.Format(ctx.Server, folderID)
	return removeSharedUsers(ctx.Session, url, remove)
}

// UpdateSharedFileUsers updates read/write permissions for the provided users,
// returning a list of the successfully updated shared users and any errors.
func (ctx *Context) UpdateSharedFileUsers(
	fileID string,
	update []shared.ShareInfo,
) ([]shared.ShareInfo, error) {
	url := endpoints.ShareFile.Format(ctx.Server, fileID)
	return updateSharedUsers(ctx.Session, fileID, url, update)
}

// UpdateSharedFolderUsers updates read/write permissions for the provided users,
// returning a list of the successfully updated shared users and any errors.
func (ctx *Context) UpdateSharedFolderUsers(
	folderID string,
	update []shared.ShareInfo,
) ([]shared.ShareInfo, error) {
	url := endpoints.ShareFolder.Format(ctx.Server, folderID)
	return updateSharedUsers(ctx.Session, folderID, url, update)
}

func shareContentWithUser(
	session,
	url string,
	request shared.ShareItemRequest,
) (shared.ShareInfo, error) {
	reqData, _ := json.Marshal(request)
	resp, err := requests.PostRequest(session, url, reqData)
	if err != nil {
		return shared.ShareInfo{}, err
	}

	var shareInfo shared.ShareInfo
	err = json.NewDecoder(resp.Body).Decode(&shareInfo)
	if err != nil {
		return shared.ShareInfo{}, err
	}

	return shareInfo, nil
}

func getSharedInfo(
	session,
	url string,
) ([]shared.ShareInfo, error) {
	resp, err := requests.GetRequest(session, url)
	if err != nil {
		return nil, err
	}

	var shares []shared.ShareInfo
	err = json.NewDecoder(resp.Body).Decode(&shares)
	if err != nil {
		return nil, err
	}

	return shares, nil
}

func removeSharedUsers(
	session,
	url string,
	remove []shared.ShareInfo,
) ([]shared.ShareInfo, error) {
	var removed []shared.ShareInfo
	for _, share := range remove {
		deleteURL := fmt.Sprintf("%s?id=%s", url, share.ID)
		resp, err := requests.DeleteRequest(session, deleteURL, nil)
		if err != nil {
			msg := fmt.Sprintf(lang.I18n.T("cli.api.error.remove_failed")+" %s -- %s",
				share.Recipient, err.Error())
			return removed, errors.New(msg)
		} else if resp.StatusCode != http.StatusOK {
			return removed, shared.ParseHTTPError(resp)
		}

		removed = append(removed, share)
	}

	return removed, nil
}

func updateSharedUsers(
	session,
	itemID,
	url string,
	update []shared.ShareInfo,
) ([]shared.ShareInfo, error) {
	var updated []shared.ShareInfo
	for _, share := range update {
		reqData, _ := json.Marshal(shared.ShareEdit{
			ID:        share.ID,
			ItemID:    itemID,
			CanModify: share.CanModify,
		})

		resp, err := requests.PutRequest(session, url, reqData)
		if err != nil {
			return updated, err
		} else if resp.StatusCode != http.StatusOK {
			return updated, shared.ParseHTTPError(resp)
		}

		updated = append(updated, share)
	}

	return updated, nil
}
