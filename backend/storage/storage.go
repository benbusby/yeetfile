package storage

import (
	"errors"
	"log"
	"yeetfile/backend/cache"
	"yeetfile/backend/config"
	"yeetfile/backend/db"
)

const MaxUploadAttempts = 5

var Interface storage
var ExceededMaximumAttemptsError = errors.New("exceeded maximum attempts")

type storage interface {
	Authorize() error
	Reauthorize()
	InitUpload(metadataID string) error
	InitLargeUpload(filename, metadataID string) error
	UploadSingleChunk(chunk FileChunk, upload db.Upload) error
	UploadMultiChunk(chunk FileChunk, upload db.Upload) (bool, error)
	CancelLargeFile(remoteID, filename string) (bool, error)
	DeleteFile(remoteID, filename string) (bool, error)
	FinishLargeUpload(remoteID, filename string, checksums []string) (string, int64, error)
	PartialDownloadById(remoteID, filename string, start, end int64) ([]byte, error)
}

type FileChunk struct {
	FileID      string
	Filename    string
	Data        []byte
	ChunkNum    int
	TotalChunks int
}

// DeleteFileByMetadata removes a file from B2 matching the provided file ID
func DeleteFileByMetadata(metadata db.FileMetadata) {
	log.Println("Deleting file by metadata (B2 errors are OK)")
	if err := cache.RemoveFile(metadata.ID); err != nil {
		log.Printf("Error removing cached file: %v\n", metadata.ID)
	} else {
		log.Printf("%s deleted from cache\n", metadata.ID)
	}

	if ok, err := Interface.CancelLargeFile(metadata.B2ID, metadata.Name); ok && err == nil {
		log.Printf("%s (large B2 upload) canceled\n", metadata.ID)
		db.ClearDatabase(metadata.ID)
	} else if ok, err = Interface.DeleteFile(metadata.B2ID, metadata.Name); ok && err == nil {
		log.Printf("%s deleted from B2\n", metadata.ID)
		db.ClearDatabase(metadata.ID)
	} else {
		if len(metadata.B2ID) == 0 {
			db.ClearDatabase(metadata.ID)
		} else {
			log.Printf("Failed to delete B2 file (id: %s, "+
				"metadata id: %s)\n",
				metadata.B2ID, metadata.ID)
			db.ClearDatabase(metadata.ID)
		}
	}
}

func init() {
	storageType := config.YeetFileConfig.StorageType

	switch storageType {
	case config.LocalStorage:
		Interface = initLocalStorage()
	case config.B2Storage:
		Interface = initB2()
	case config.S3Storage:
		Interface = initS3()
	default:
		log.Fatalf("Invalid storage type '%s', "+
			"should be either '%s', '%s', or '%s'",
			config.YeetFileConfig.StorageType,
			config.B2Storage, config.S3Storage, config.LocalStorage)

	}
}
