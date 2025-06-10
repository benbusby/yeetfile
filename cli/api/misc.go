package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"yeetfile/cli/requests"
	"yeetfile/shared"
	"yeetfile/shared/endpoints"
)

// GetServerInfo returns information about the current YeetFile instance/server
func (ctx *Context) GetServerInfo() (shared.ServerInfo, error) {
	url := endpoints.ServerInfo.Format(ctx.Server)
	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return shared.ServerInfo{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.ServerInfo{}, shared.ParseHTTPError(resp)
	}

	var serverInfo shared.ServerInfo
	err = json.NewDecoder(resp.Body).Decode(&serverInfo)
	if err != nil {
		log.Println("Error decoding server response: ", err)
		return shared.ServerInfo{}, err
	}

	return serverInfo, nil
}

func (ctx *Context) GetStaticFile(dir, file string) ([]byte, error) {
	url := endpoints.StaticFile.Format(ctx.Server, dir, file)
	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, shared.ParseHTTPError(resp)
	}

	bytes, err := io.ReadAll(resp.Body)
	return bytes, err
}
