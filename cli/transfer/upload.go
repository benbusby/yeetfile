package transfer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"yeetfile/cli/config"
	"yeetfile/cli/crypto"
	"yeetfile/cli/requests"
	"yeetfile/shared"
	"yeetfile/shared/constants"
	"yeetfile/shared/endpoints"
)

type PendingUpload struct {
	ID                  string
	Key                 []byte
	File                *os.File
	NumChunks           int
	UnformattedEndpoint endpoints.Endpoint
}

type FileChunk struct {
	Chunk         int
	EncryptedData []byte
	Endpoint      string
}

type WorkerCtx struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// worker sends chunked and encrypted file data to the endpoint specified in the
// provided FileChunk.
func worker(wCtx WorkerCtx, chunks <-chan FileChunk, progress func(), wg *sync.WaitGroup) {
	defer wg.Done()
	for chunk := range chunks {
		select {
		case <-wCtx.ctx.Done():
			log.Println("workers stopped due to cancellation")
			return
		default:
			_, err := sendChunk(chunk)
			if err != nil {
				log.Printf("Worker error: %v\n", err)
				wCtx.cancel()
				return
			}
			progress()
		}
	}
}

// CreateVaultFolder creates a new folder in the user's vault. Returns the new
// folder's ID if successful.
func CreateVaultFolder(
	folderName,
	folderID string,
	protectedKey,
	key []byte,
) (shared.NewFolderResponse, error) {
	encName, err := crypto.EncryptChunk(key, []byte(folderName))
	if err != nil {
		return shared.NewFolderResponse{}, err
	}

	name := hex.EncodeToString(encName)
	newFolder := shared.NewVaultFolder{
		Name:         name,
		ProtectedKey: protectedKey,
		ParentID:     folderID,
	}

	reqData, err := json.Marshal(newFolder)
	if err != nil {
		return shared.NewFolderResponse{}, err
	}

	url := endpoints.VaultFolder.Format(config.UserConfig.Server)
	resp, err := requests.PostRequest(url, reqData)
	if err != nil {
		return shared.NewFolderResponse{}, err
	} else if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("server error %d - %s", resp.StatusCode)
		return shared.NewFolderResponse{}, errors.New(msg)
	}

	var folderResponse shared.NewFolderResponse
	err = json.NewDecoder(resp.Body).Decode(&folderResponse)
	if err != nil {
		log.Println("error decoding new folder response")
		return shared.NewFolderResponse{}, err
	}

	return folderResponse, nil
}

// InitVaultFile initializes a vault file's metadata, which is required prior to
// uploading the file contents.
func InitVaultFile(
	file *os.File,
	stat os.FileInfo,
	folderID string,
	protectedKey,
	key []byte,
) (PendingUpload, error) {
	encName, err := crypto.EncryptChunk(key, []byte(stat.Name()))
	if err != nil {
		return PendingUpload{}, err
	}

	name := hex.EncodeToString(encName)
	size := int(stat.Size())
	numChunks := GetNumChunks(stat.Size())
	upload := shared.VaultUpload{
		Name:         name,
		Length:       size,
		Chunks:       numChunks,
		FolderID:     folderID,
		ProtectedKey: protectedKey,
	}

	reqData, err := json.Marshal(upload)
	if err != nil {
		return PendingUpload{}, err
	}

	url := endpoints.UploadVaultFileMetadata.Format(config.UserConfig.Server)
	resp, err := requests.PostRequest(url, reqData)
	if err != nil {
		return PendingUpload{}, err
	} else if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := fmt.Sprintf("server error [%d]: %s", resp.StatusCode, body)
		return PendingUpload{}, errors.New(msg)
	}

	var metaResponse shared.MetadataUploadResponse
	err = json.NewDecoder(resp.Body).Decode(&metaResponse)
	if err != nil {
		log.Println("Error decoding server response: ", err)
		return PendingUpload{}, err
	}

	return PendingUpload{
		ID:                  metaResponse.ID,
		Key:                 key,
		File:                file,
		NumChunks:           numChunks,
		UnformattedEndpoint: endpoints.UploadVaultFileData,
	}, nil
}

// InitSendFile initializes a file's metadata for sending.
func InitSendFile(
	file *os.File,
	meta shared.UploadMetadata,
	key []byte,
) (PendingUpload, error) {
	reqData, err := json.Marshal(meta)
	if err != nil {
		return PendingUpload{}, err
	}

	url := endpoints.UploadSendFileMetadata.Format(config.UserConfig.Server)
	resp, err := requests.PostRequest(url, reqData)
	if err != nil {
		return PendingUpload{}, err
	} else if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := fmt.Sprintf("server error [%d]: %s", resp.StatusCode, body)
		return PendingUpload{}, errors.New(msg)
	}

	var metaResponse shared.MetadataUploadResponse
	err = json.NewDecoder(resp.Body).Decode(&metaResponse)
	if err != nil {
		log.Println("Error decoding server response: ", err)
		return PendingUpload{}, err
	}

	return PendingUpload{
		ID:                  metaResponse.ID,
		Key:                 key,
		File:                file,
		NumChunks:           meta.Chunks,
		UnformattedEndpoint: endpoints.UploadSendFileData,
	}, nil
}

// UploadData encrypts and uploads a file's contents chunk-by-chunk. The upload
// threads for multi-chunk uploads are limited by constants.MaxTransferThreads.
func (p PendingUpload) UploadData(progress func()) (string, error) {
	var wg sync.WaitGroup
	var fileChunk FileChunk
	var prepErr error

	stat, _ := p.File.Stat()
	ctx, cancel := context.WithCancel(context.Background())
	wCtx := WorkerCtx{ctx: ctx, cancel: cancel}
	defer cancel()

	jobs := make(chan FileChunk, constants.MaxTransferThreads)
	for i := 1; i <= constants.MaxTransferThreads; i++ {
		wg.Add(1)
		go worker(wCtx, jobs, progress, &wg)
	}

	// Send all but the final file chunk to the workers. The final chunk
	// will indicate if Backblaze has accepted all file contents.
	for chunk := 0; chunk < p.NumChunks-1; chunk++ {
		fileChunk, prepErr = p.prepareChunk(chunk, stat.Size())
		if prepErr != nil {
			cancel()
			break
		}

		jobs <- fileChunk
	}

	close(jobs)
	wg.Wait()

	if prepErr != nil || ctx.Err() != nil {
		return "", errors.Join(prepErr, ctx.Err())
	}

	// Prepare final chunk
	fileChunk, prepErr = p.prepareChunk(p.NumChunks-1, stat.Size())
	if prepErr != nil {
		return "", prepErr
	}

	// Send final chunk
	response, err := sendChunk(fileChunk)
	if err != nil {
		return "", err
	}

	return response, nil
}

// prepareChunk reads a chunk of a file and encrypts it, returning a FileChunk
// struct containing the encrypted data, the chunk number, and the endpoint
// to send the chunk to.
func (p PendingUpload) prepareChunk(chunk int, size int64) (FileChunk, error) {
	endpoint := p.UnformattedEndpoint.Format(
		config.UserConfig.Server,
		p.ID,
		strconv.Itoa(chunk+1))

	start, end := GetReadBounds(chunk, size)
	contents := make([]byte, end-start)
	_, err := p.File.ReadAt(contents, start)
	if err != nil {
		return FileChunk{}, err
	}

	encData, _ := crypto.EncryptChunk(p.Key, contents)
	return FileChunk{
		Chunk:         chunk,
		Endpoint:      endpoint,
		EncryptedData: encData,
	}, nil
}

// sendChunk sends the encrypted file data to the server
func sendChunk(fileChunk FileChunk) (string, error) {
	resp, err := requests.PostRequest(fileChunk.Endpoint, fileChunk.EncryptedData)
	if err != nil {
		log.Printf("Error sending data: %v\n", err)
		return "", err
	} else if resp.StatusCode != http.StatusOK {
		log.Printf("Error sending data (status %d)\n", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error parsing chunk response: %v\n", err)
		return "", err
	}

	if len(body) > 0 {
		return string(body), nil
	}

	return "", nil
}

func UploadEncryptedText(upload shared.PlaintextUpload) (string, error) {
	reqData, err := json.Marshal(upload)
	if err != nil {
		return "", err
	}

	url := endpoints.UploadSendText.Format(config.UserConfig.Server)
	resp, err := requests.PostRequest(url, reqData)
	if err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := fmt.Sprintf("server error [%d]: %s", resp.StatusCode, body)
		return "", errors.New(msg)
	}

	var metaResponse shared.MetadataUploadResponse
	err = json.NewDecoder(resp.Body).Decode(&metaResponse)
	if err != nil {
		log.Println("Error decoding server response: ", err)
		return "", err
	}

	return metaResponse.ID, nil
}