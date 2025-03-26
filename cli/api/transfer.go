package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"yeetfile/cli/requests"
	"yeetfile/cli/utils"
	"yeetfile/shared"
	"yeetfile/shared/endpoints"
)

// UploadFileChunk uploads a chunk of file data to the server. This API call
// requires a pre-formatted endpoint (either endpoints.UploadSendFileData or
// endpoints.UploadVaultFileData) that contains the chunk number.
func (ctx *Context) UploadFileChunk(
	endpoint string,
	encData []byte,
) (string, error) {
	resp, err := requests.PostRequest(ctx.Session, endpoint, encData)
	if err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusOK {
		return "", utils.ParseHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// UploadText uploads text to YeetFile (only used by YeetFile Send). Since text
// only uploads are limited to 2K chars, metadata and encrypted text content
// can be uploaded together in one call.
func (ctx *Context) UploadText(
	upload shared.TextUpload,
) (string, error) {
	reqData, err := json.Marshal(upload)
	if err != nil {
		return "", err
	}

	url := endpoints.UploadSendText.Format(ctx.Server)
	resp, err := requests.PostRequest(ctx.Session, url, reqData)
	if err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusOK {
		return "", utils.ParseHTTPError(resp)
	}

	var metaResponse shared.MetadataUploadResponse
	err = json.NewDecoder(resp.Body).Decode(&metaResponse)
	if err != nil {
		log.Println("Error decoding server response: ", err)
		return "", err
	}

	return metaResponse.ID, nil
}

// DownloadFileChunk downloads a chunk of encrypted file data. Note that a
// pre-formatted endpoint url must be provided, since the server and/or chunk
// number can change per request.
func (ctx *Context) DownloadFileChunk(url string) ([]byte, error) {
	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, utils.ParseHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	return body, err
}
