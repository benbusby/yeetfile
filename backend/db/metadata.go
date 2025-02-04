package db

import (
	"database/sql"
	"errors"
	"log"
	"time"
	"yeetfile/shared"
	"yeetfile/shared/constants"
)

const uploadIDLength = 12

type FileMetadata struct {
	ID                string
	RefID             string
	Chunks            int
	Name              string
	B2ID              string
	Length            int64
	FolderID          string
	ProtectedKey      []byte
	PasswordData      []byte
	OwnsParentFolder  bool
	ParentFolderOwner string
	Expiration        time.Time
	Downloads         int
}

// InsertMetadata creates a new metadata entry in the db and returns a unique ID for
// that entry.
func InsertMetadata(chunks int, ownerID, name string, textOnly bool) (string, error) {
	prefix := constants.FileIDPrefix
	if textOnly {
		prefix = constants.PlaintextIDPrefix
	}

	id := shared.GenRandomStringWithPrefix(uploadIDLength, prefix)

	// Ensure the id isn't already being used in the table
	for MetadataIDExists(id) {
		id = shared.GenRandomStringWithPrefix(uploadIDLength, prefix)
	}

	s := `INSERT INTO metadata
	      (id, chunks, filename, b2_id, length, owner_id, modified)
	      VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := db.Exec(s, id, chunks, name, "", -1, ownerID, time.Now().UTC())
	if err != nil {
		panic(err)
	}

	return id, nil
}

func MetadataIDExists(id string) bool {
	rows, err := db.Query(`SELECT * FROM metadata WHERE id = $1`, id)
	if err != nil {
		log.Fatalf("Error validating metadata id: %v", err)
		return true
	}

	// If any rows are returned, the id exists
	defer rows.Close()
	if rows.Next() {
		return true
	}

	return false
}

func RetrieveMetadata(id string) (FileMetadata, error) {
	s := `SELECT m.id, m.chunks, m.filename, m.b2_id, m.length, e.downloads, e.date
	      FROM metadata m
	      JOIN expiry e on m.id = e.id
	      WHERE m.id = $1`
	rows, err := db.Query(s, id)
	if err != nil {
		log.Fatalf("Error retrieving metadata: %v", err)
		return FileMetadata{}, err
	}

	defer rows.Close()
	if rows.Next() {
		return ParseMetadata(rows), nil
	}

	log.Printf("No metadata found for id: %s", id)
	return FileMetadata{}, errors.New("no metadata found")
}

func UpdateB2Metadata(id string, b2ID string, length int64) error {
	s := `UPDATE metadata
	      SET b2_id=$1, length=$2
	      WHERE id=$3`
	_, err := db.Exec(s, b2ID, length, id)
	if err != nil {
		return err
	}

	s = `UPDATE vault
	      SET b2_id=$1, length=$2
	      WHERE id=$3`

	_, err = db.Exec(s, b2ID, length, id)
	if err != nil {
		return err
	}

	return nil
}

func ParseMetadata(rows *sql.Rows) FileMetadata {
	var id string
	var chunks int
	var name string
	var b2ID string
	var length int64
	var downloads int
	var date time.Time

	err := rows.Scan(&id, &chunks, &name, &b2ID, &length, &downloads, &date)

	if err != nil {
		return FileMetadata{}
	}

	return FileMetadata{
		ID:         id,
		Chunks:     chunks,
		Name:       name,
		B2ID:       b2ID,
		Length:     length,
		Downloads:  downloads,
		Expiration: date,
	}
}

func DeleteMetadata(id string) bool {
	s := `DELETE FROM metadata
	      WHERE id = $1`
	_, err := db.Exec(s, id)
	if err != nil {
		return false
	}

	return true
}

func AdminRetrieveSendMetadata(fileID string) (shared.AdminFileInfoResponse, error) {
	var (
		id       string
		name     string
		length   int64
		ownerID  string
		modified time.Time
	)

	s := `SELECT id, filename, length, owner_id, modified FROM metadata WHERE id=$1`
	err := db.QueryRow(s, fileID).Scan(&id, &name, &length, &ownerID, &modified)
	return shared.AdminFileInfoResponse{
		ID:         id,
		BucketName: name,
		Size:       shared.ReadableFileSize(length),
		OwnerID:    ownerID,
		Modified:   modified,

		RawSize: length,
	}, err
}

func AdminFetchSentFiles(userID string) ([]shared.AdminFileInfoResponse, error) {
	result := []shared.AdminFileInfoResponse{}

	s := `SELECT id, filename, length, owner_id, modified
	      FROM metadata
	      WHERE owner_id=$1`

	rows, err := db.Query(s, userID)
	if err != nil {
		return result, err
	}

	defer rows.Close()
	for rows.Next() {
		var (
			id       string
			filename string
			length   int64
			ownerID  string
			modified time.Time
		)

		err = rows.Scan(&id, &filename, &length, &ownerID, &modified)
		if err != nil {
			return result, err
		}

		result = append(result, shared.AdminFileInfoResponse{
			ID:         id,
			BucketName: filename,
			Size:       shared.ReadableFileSize(length),
			OwnerID:    ownerID,
			Modified:   modified,
			RawSize:    length,
		})
	}

	return result, nil
}
