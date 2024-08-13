package transfer

import (
	"yeetfile/backend/cache"
	"yeetfile/backend/service"
	"yeetfile/shared/constants"
)

type DownloadRequest struct {
	Password string `json:"password"`
}

func DownloadFile(b2ID string, length int, chunk int) (bool, []byte) {
	start, end, eof := getReadBoundaries(chunk, length)
	data, _ := service.B2.PartialDownloadById(b2ID, start, end)
	return eof, data
}

func DownloadFileFromCache(fileID string, length int, chunk int) (bool, []byte) {
	start, end, eof := getReadBoundaries(chunk, length)
	data, _ := cache.Read(fileID, start, end)
	return eof, data
}

// getReadBoundaries calculates the correct start and end bytes to read from for
// a specific file chunk, and determines if this read operation reaches the end
// of the file
func getReadBoundaries(chunk, length int) (int, int, bool) {
	eof := false

	start := (chunk-1)*constants.ChunkSize +
		((constants.TotalOverhead) * (chunk - 1))

	end := constants.ChunkSize +
		constants.TotalOverhead +
		start - 1

	if end >= length-1 {
		end = length - 1
		eof = true
	}

	return start, end, eof
}