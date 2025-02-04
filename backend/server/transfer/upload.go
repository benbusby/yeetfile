package transfer

import (
	"errors"
	"github.com/benbusby/b2"
	"log"
	"net/url"
	db "yeetfile/backend/db"
	"yeetfile/backend/service"
	"yeetfile/backend/utils"
)

const MaxUploadAttempts = 5

var ExceededMaximumAttemptsError = errors.New("exceeded maximum attempts")

type FileUpload struct {
	filename string
	data     []byte
	checksum string
	chunk    int
	chunks   int
}

func InitB2Upload(upload db.B2Upload) error {
	info, err := service.B2.GetUploadURL(service.B2BucketID)
	if err != nil {
		return err
	}

	return db.UpdateUploadValues(
		upload.MetadataID,
		info.UploadURL,
		info.AuthorizationToken,
		info.BucketID, // Single chunk files use the bucket ID for uploading
		info.Dummy)
}

func PrepareUpload(
	metadata db.FileMetadata,
	chunk int,
	data []byte,
) (FileUpload, db.B2Upload, error) {
	_, checksum := utils.GenChecksum(data)
	db.UpdateChecksums(metadata.ID, chunk, checksum)

	b2Values := db.GetB2UploadValues(metadata.ID)

	upload := FileUpload{
		data:     data,
		filename: metadata.Name,
		checksum: checksum,
		chunk:    chunk,
		chunks:   metadata.Chunks,
	}

	return upload, b2Values, nil
}

func (upload FileUpload) Upload(b2Values db.B2Upload) (bool, error) {
	if upload.chunks > 1 {
		return UploadMultiChunk(upload, b2Values)
	} else {
		return UploadSingleChunk(upload, b2Values)
	}
}

func UploadSingleChunk(upload FileUpload, b2Values db.B2Upload) (bool, error) {
	file := b2.FileInfo{
		BucketID:           b2Values.UploadID,
		AuthorizationToken: b2Values.Token,
		UploadURL:          b2Values.UploadURL,
		Dummy:              b2Values.Local,
	}

	resp, err := b2.UploadFile(
		file,
		upload.filename,
		upload.checksum,
		upload.data)

	if err != nil {
		log.Printf("Error uploading to B2: %v\n", err)
		return false, err
	}

	err = db.UpdateB2Metadata(
		b2Values.MetadataID,
		resp.FileID,
		resp.ContentLength)

	return true, err
}

func UploadMultiChunk(upload FileUpload, b2Values db.B2Upload) (bool, error) {
	uploadChunk := func() error {
		info, err := service.B2.GetUploadPartURL(b2Values.UploadID)
		if err != nil {
			return err
		}

		largeFile := b2.FilePartInfo{
			FileID:             b2Values.UploadID,
			AuthorizationToken: info.AuthorizationToken,
			UploadURL:          info.UploadURL,
			Dummy:              b2Values.Local,
		}

		err = b2.UploadFilePart(
			largeFile,
			upload.chunk,
			upload.checksum,
			upload.data)

		if err != nil {
			log.Printf("Error: %v\n", err)
			return err
		}

		return nil
	}

	attempt := 0
	err := uploadChunk()
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

	if upload.chunk == upload.chunks {
		b2ID, length := FinishLargeB2Upload(
			b2Values.UploadID,
			b2Values.Checksums)
		return true, db.UpdateB2Metadata(b2Values.MetadataID, b2ID, length)
	} else {
		return false, nil
	}
}

func InitLargeB2Upload(filename string, upload db.B2Upload) error {
	init, err := service.B2.StartLargeFile(filename, service.B2BucketID)
	if err != nil {
		return err
	}

	info, err := service.B2.GetUploadPartURL(init.FileID)
	if err != nil {
		return err
	}

	isDummy := info.Dummy
	if !isDummy {
		// Ensure that the dummy option is enabled if the request URI
		// is not actually valid
		_, err = url.ParseRequestURI(upload.UploadURL)
		if err != nil {
			isDummy = true
		}
	}

	return db.UpdateUploadValues(
		upload.MetadataID,
		info.UploadURL,
		info.AuthorizationToken,
		info.FileID, // Multi-chunk files use the file ID for uploading
		isDummy)
}

func ResetLargeUpload(b2FileID string, metadataID string) (b2.FilePartInfo, error) {
	info, err := service.B2.GetUploadPartURL(b2FileID)
	if err != nil {
		return b2.FilePartInfo{}, err
	}

	err = db.UpdateUploadValues(
		metadataID,
		info.UploadURL,
		info.AuthorizationToken,
		info.FileID,
		info.Dummy)

	if err != nil {
		return b2.FilePartInfo{}, err
	}

	return info, nil
}

func FinishLargeB2Upload(b2ID string, checksums []string) (string, int64) {
	largeFile, err := service.B2.FinishLargeFile(b2ID, checksums)
	if err != nil {
		panic(err)
	}

	return largeFile.FileID, largeFile.ContentLength
}
