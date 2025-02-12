package transfer

import (
	db "yeetfile/backend/db"
	"yeetfile/backend/storage"
)

func PrepareUpload(
	metadata db.FileMetadata,
	chunk int,
	data []byte,
) (storage.FileChunk, db.Upload, error) {
	uploadValues := db.GetUploadValues(metadata.ID)
	fileChunk := storage.FileChunk{
		FileID:      metadata.ID,
		Data:        data,
		Filename:    metadata.Name,
		ChunkNum:    chunk,
		TotalChunks: metadata.Chunks,
	}

	return fileChunk, uploadValues, nil
}
