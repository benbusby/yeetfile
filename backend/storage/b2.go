package storage

import (
	"errors"
	"github.com/benbusby/b2"
	"log"
	"os"
	"strconv"
	"yeetfile/backend/db"
	"yeetfile/backend/utils"
)

const defaultStoragePath = "uploads"

type B2 struct {
	client      *b2.Service
	bucketID    string
	bucketKeyID string
	bucketKey   string

	local bool
}

func (b2Backend *B2) Authorize() error {
	tmp, _, err := b2.AuthorizeAccount(b2Backend.bucketKeyID, b2Backend.bucketKey)
	if err != nil {
		log.Println("Error authorizing B2 account", err)
		return err
	}

	b2Backend.client = tmp
	return nil
}

func (b2Backend *B2) Reauthorize() {
	if b2Backend.local {
		return
	}

	err := b2Backend.Authorize()
	if err != nil {
		log.Printf("ERROR: Unable to reauthorize B2 client: %v\n", err)
	}
}

func (b2Backend *B2) InitUpload(metadataID string) error {
	info, err := b2Backend.client.GetUploadURL(b2Backend.bucketID)
	if err != nil {
		return err
	}

	err = db.UpdateUploadValues(
		metadataID,
		info.UploadURL,
		info.AuthorizationToken,
		info.BucketID, // Single chunk files use the bucket ID for uploading
		info.Dummy)

	return err
}

func (b2Backend *B2) InitLargeUpload(filename, metadataID string) error {
	init, err := b2Backend.client.StartLargeFile(filename, b2Backend.bucketID)
	if err != nil {
		return err
	}

	info, err := b2Backend.client.GetUploadPartURL(init.FileID)
	if err != nil {
		return err
	}

	localUpload := utils.IsLocalUpload(info.UploadURL)
	return db.UpdateUploadValues(
		metadataID,
		info.UploadURL,
		info.AuthorizationToken,
		info.FileID, // Multi-chunk files use the file ID for uploading
		localUpload)
}

func (b2Backend *B2) UploadSingleChunk(chunk FileChunk, upload db.Upload) error {
	file := b2.FileInfo{
		BucketID:           upload.UploadID,
		AuthorizationToken: upload.Token,
		UploadURL:          upload.UploadURL,
		Dummy:              upload.Local,
	}

	_, checksum := utils.GenChecksum(chunk.Data)
	_, err := db.UpdateChecksums(chunk.FileID, chunk.ChunkNum, checksum)
	if err != nil {
		log.Printf("Error updating checksums: %v\n", err)
		return err
	}

	resp, err := b2.UploadFile(
		file,
		chunk.Filename,
		checksum,
		chunk.Data)

	if err != nil {
		log.Printf("Error uploading to B2: %v\n", err)
		return err
	}

	err = db.UpdateMetadata(
		upload.MetadataID,
		resp.FileID,
		resp.ContentLength)

	return err
}

func (b2Backend *B2) UploadMultiChunk(chunk FileChunk, upload db.Upload) (bool, error) {
	_, checksum := utils.GenChecksum(chunk.Data)
	checksums, err := db.UpdateChecksums(chunk.FileID, chunk.ChunkNum, checksum)
	if err != nil {
		log.Printf("Failed to update checksums: %v\n", err)
		return false, err
	}

	uploadChunk := func() error {
		info, err := b2Backend.client.GetUploadPartURL(upload.UploadID)
		if err != nil {
			return err
		}

		localUpload := utils.IsLocalUpload(info.UploadURL)
		log.Printf("Uploading chunk to: %s\n", info.UploadURL)
		log.Printf("(local: %t\n)", localUpload)
		largeFile := b2.FilePartInfo{
			FileID:             upload.UploadID,
			AuthorizationToken: info.AuthorizationToken,
			UploadURL:          info.UploadURL,
			Dummy:              localUpload,
		}

		err = b2.UploadFilePart(
			largeFile,
			chunk.ChunkNum,
			checksum,
			chunk.Data)

		if err != nil {
			log.Printf("Error: %v\n", err)
			return err
		}

		return nil
	}

	attempt := 0
	err = uploadChunk()
	for err != nil && attempt < MaxUploadAttempts {
		// Try again
		attempt += 1
		log.Printf("Retrying (attempt %d)\n", attempt+1)
		err = uploadChunk()
	}

	if attempt >= MaxUploadAttempts {
		return false, ExceededMaximumAttemptsError
	} else if err != nil {
		return false, err
	}

	if len(checksums) == chunk.TotalChunks {
		// All chunks accounted for, finalize the upload
		b2ID, length, err := b2Backend.FinishLargeUpload(
			upload.UploadID,
			"",
			checksums)
		if err != nil {
			return false, err
		}

		return true, db.UpdateMetadata(upload.MetadataID, b2ID, length)
	} else {
		return false, nil
	}
}

func (b2Backend *B2) FinishLargeUpload(b2ID, _ string, checksums []string) (string, int64, error) {
	largeFile, err := b2Backend.client.FinishLargeFile(b2ID, checksums)
	if err != nil {
		return "", 0, err
	}

	return largeFile.FileID, largeFile.ContentLength, nil
}

func (b2Backend *B2) CancelLargeFile(remoteID, _ string) (bool, error) {
	return b2Backend.client.CancelLargeFile(remoteID)
}

func (b2Backend *B2) DeleteFile(remoteID, filename string) (bool, error) {
	if len(remoteID) == 0 {
		return false, errors.New("b2 ID cannot be empty")
	}
	return b2Backend.client.DeleteFile(remoteID, filename)
}

func (b2Backend *B2) PartialDownloadById(remoteID, _ string, start, end int64) ([]byte, error) {
	return b2Backend.client.PartialDownloadById(remoteID, start, end)
}

// =============================================================================

// initLocalStorage configures the backblaze B2 Go library to store files locally
// rather than actually storing in B2. This allows the same functionality as
// remote storage without having to change or add any upload logic.
func initLocalStorage() storage {
	var (
		client *b2.Service
		err    error
	)

	log.Println("Setting up local storage...")
	// Storage will bypass B2 and just store encrypted files on the
	// machine in the specified path or "uploads/"
	var limit int64
	limitStr := os.Getenv("YEETFILE_LOCAL_STORAGE_LIMIT")
	path := utils.GetEnvVar("YEETFILE_LOCAL_STORAGE_PATH", defaultStoragePath)

	if len(limitStr) > 0 {
		limit, err = strconv.ParseInt(limitStr, 10, 64)
		if err != nil {
			log.Fatalf("Invalid storage limit \"%s\"", limitStr)
		}
	}

	if limit > 0 {
		client, err = b2.AuthorizeLimitedDummyAccount(path, limit)
	} else {
		client, err = b2.AuthorizeDummyAccount(path)
	}

	return &B2{
		client: client,
		local:  true,
	}
}

// initB2 initializes the Backblaze B2 storage backend and fetches an authorization
// token using the provided credentials.
func initB2() storage {
	bucketID := utils.GetEnvVar("YEETFILE_B2_BUCKET_ID", "")
	bucketKeyID := utils.GetEnvVar("YEETFILE_B2_BUCKET_KEY_ID", "")
	bucketKey := utils.GetEnvVar("YEETFILE_B2_BUCKET_KEY", "")

	if len(bucketID) == 0 || len(bucketKeyID) == 0 || len(bucketKey) == 0 {
		log.Fatalf("Missing required B2 environment variables:\n"+
			"- YEETFILE_B2_BUCKET_ID: %v\n"+
			"- YEETFILE_B2_BUCKET_KEY_ID: %v\n"+
			"- YEETFILE_B2_BUCKET_KEY: %v\n",
			len(bucketID) > 0,
			len(bucketKeyID) > 0,
			len(bucketKey) > 0)
	}

	log.Println("Authorizing B2 account...")
	b2Backend := &B2{
		bucketID:    bucketID,
		bucketKeyID: bucketKeyID,
		bucketKey:   bucketKey,
		local:       false,
	}

	err := b2Backend.Authorize()
	if err != nil {
		log.Fatal(err)
	}

	return b2Backend
}
