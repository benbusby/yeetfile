package admin

import (
	"database/sql"
	"log"
	"yeetfile/backend/db"
	"yeetfile/shared"
)

func deleteFile(fileID string) error {
	metadata, err := db.AdminRetrieveMetadata(fileID)
	if err == nil {
		// Delete vault file
		err = db.AdminDeleteFile(fileID)
		if err != nil {
			log.Printf("Error deleting file: %v\n", err)
			return err
		}

		_ = db.UpdateStorageUsed(metadata.OwnerID, -metadata.RawSize)
	} else {
		// Attempt to delete Send file instead
		metadata, err := db.RetrieveMetadata(fileID)
		if err != nil {
			return err
		}

		db.DeleteFileByMetadata(metadata)
	}

	return nil
}

func fetchFileMetadata(fileID string) (shared.AdminFileInfoResponse, error) {
	fileInfo, err := db.AdminRetrieveMetadata(fileID)
	if err != nil && err != sql.ErrNoRows {
		return shared.AdminFileInfoResponse{}, err
	} else if err == nil {
		return fileInfo, nil
	}

	sendFileInfo, err := db.AdminRetrieveSendMetadata(fileID)
	if err != nil && err != sql.ErrNoRows {
		return shared.AdminFileInfoResponse{}, err
	} else if err == nil {
		return sendFileInfo, nil
	}

	return shared.AdminFileInfoResponse{}, err
}
