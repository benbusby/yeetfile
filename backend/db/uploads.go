package db

import (
	"github.com/lib/pq"
	"log"
)

type Upload struct {
	MetadataID string
	UploadURL  string
	Token      string
	UploadID   string
	Checksums  []string
	Local      bool
	Name       string
}

func CreateNewUpload(id string, name string) error {
	s := `INSERT INTO uploads (metadata_id, upload_id, name)
	      VALUES ($1, $2, $3)`
	_, err := db.Exec(s, id, name, name)
	if err != nil {
		return err
	}

	return nil
}

func UpdateUploadValues(
	metadataID string,
	uploadURL string,
	token string,
	uploadID string,
	local bool,
) error {
	s := `UPDATE uploads
	      SET upload_url=$1, 
	          token=$2, 
	          upload_id=CASE WHEN $3 = '' THEN upload_id ELSE $3 END, 
	          local=$4
	      WHERE metadata_id=$5`
	_, err := db.Exec(s, uploadURL, token, uploadID, local, metadataID)
	if err != nil {
		log.Printf("Error updating b2 upload values: %v\n", err)
		return err
	}

	return nil
}

func UpdateUploadID(id string, metadataID string) bool {
	s := `UPDATE uploads SET upload_id=$1 WHERE metadata_id=$2`
	_, err := db.Exec(s, id, metadataID)
	if err != nil {
		log.Printf("Error updating b2 upload id: %v\n", err)
		return false
	}

	return true
}

func UpdateChecksums(id string, chunk int, checksum string) ([]string, error) {
	var checksums []string
	s := `UPDATE uploads
	      SET checksums[$1] = $2
	      WHERE metadata_id=$3
	      RETURNING checksums`

	err := db.QueryRow(s, chunk, checksum, id).Scan(pq.Array(&checksums))
	if err != nil {
		return nil, err
	}

	return checksums, nil
}

func GetUploadValues(id string) Upload {
	s := `SELECT *
	      FROM uploads
	      WHERE metadata_id = $1`

	rows, err := db.Query(s, id)
	if err != nil {
		log.Printf("Error retrieving upload values: %v", err)
		return Upload{}
	}

	defer rows.Close()
	if rows.Next() {
		var metadataID string
		var uploadURL string
		var token string
		var uploadID string
		var checksums []string
		var local bool
		var name string

		err = rows.Scan(
			&metadataID,
			&uploadURL,
			&token,
			&uploadID,
			pq.Array(&checksums),
			&local,
			&name)

		return Upload{
			MetadataID: metadataID,
			UploadURL:  uploadURL,
			Token:      token,
			UploadID:   uploadID,
			Checksums:  checksums,
			Local:      local,
			Name:       name,
		}
	}

	return Upload{}
}

func DeleteUploads(id string) bool {
	s := `DELETE FROM uploads
	      WHERE metadata_id = $1`
	_, err := db.Exec(s, id)
	if err != nil {
		return false
	}

	return true
}
