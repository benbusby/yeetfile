package api

import (
	"encoding/json"
	"log"
	"net/http"
	"yeetfile/cli/requests"
	"yeetfile/shared"
	"yeetfile/shared/endpoints"
)

// InitSendFile initializes a new file to send via YeetFile Send
func (ctx *Context) InitSendFile(
	meta shared.UploadMetadata,
) (shared.MetadataUploadResponse, error) {
	reqData, err := json.Marshal(meta)
	if err != nil {
		return shared.MetadataUploadResponse{}, err
	}

	url := endpoints.UploadSendFileMetadata.Format(ctx.Server)
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

// FetchSendFileMetadata fetches metadata for a file sent using YeetFile Send
// using the file's id
func (ctx *Context) FetchSendFileMetadata(server, id string) (shared.DownloadResponse, error) {
	url := endpoints.DownloadSendFileMetadata.Format(server, id)
	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return shared.DownloadResponse{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.DownloadResponse{}, shared.ParseHTTPError(resp)
	}

	var downloadResponse shared.DownloadResponse
	err = json.NewDecoder(resp.Body).Decode(&downloadResponse)
	if err != nil {
		return shared.DownloadResponse{}, err
	}

	return downloadResponse, nil
}
