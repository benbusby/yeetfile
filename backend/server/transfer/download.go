package transfer

import (
	"yeetfile/backend/cache"
	"yeetfile/backend/storage"
	"yeetfile/shared/constants"
)

type DownloadRequest struct {
	Password string `json:"password"`
}

func DownloadFile(b2ID, filename string, length int64, chunk int) (bool, []byte) {
	start, end, eof := getReadBoundaries(chunk, length)
	data, _ := storage.Interface.PartialDownloadById(b2ID, filename, start, end)
	return eof, data
}

func DownloadFileFromCache(fileID string, length int64, chunk int) (bool, []byte) {
	start, end, eof := getReadBoundaries(chunk, length)
	data, _ := cache.Read(fileID, start, end)
	return eof, data
}

// getReadBoundaries calculates the correct start and end bytes to read from for
// a specific file chunk, and determines if this read operation reaches the end
// of the file
func getReadBoundaries(chunk int, length int64) (int64, int64, bool) {
	var start int64
	var end int64
	eof := false

	start = int64((chunk-1)*constants.ChunkSize +
		((constants.TotalOverhead) * (chunk - 1)))

	end = int64(constants.ChunkSize) +
		int64(constants.TotalOverhead) +
		start - 1

	if end >= length-1 {
		end = length - 1
		eof = true
	}

	return start, end, eof
}
