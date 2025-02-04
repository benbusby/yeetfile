package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
	"yeetfile/shared"
)

const publicOwnerID = "public"
const VaultIDLength = 20

var ReadOnlyError = errors.New("attempting to modify in read-only context")
var AccessError = errors.New("unauthorized access")

// GetFileOwnership retrieves ownership details for a particular file
func GetFileOwnership(fileID, userID string) (shared.FileOwnershipInfo, error) {
	s := `SELECT can_modify FROM vault WHERE owner_id=$1 AND ref_id=$2`
	rows, err := db.Query(s, userID, fileID)
	if err != nil {
		return shared.FileOwnershipInfo{}, err
	}

	defer rows.Close()
	if rows.Next() {
		var canModify bool
		err = rows.Scan(&canModify)
		if err != nil {
			return shared.FileOwnershipInfo{}, err
		}

		return shared.FileOwnershipInfo{CanModify: canModify}, nil
	}

	return shared.FileOwnershipInfo{}, err
}

func CheckFolderOwnership(userID, folderID string) (shared.FolderOwnershipInfo, error) {
	parentID, err := GetParentFolderID(folderID)
	if err != nil {
		return shared.FolderOwnershipInfo{}, err
	}

	// Query for the current folder to see if it's owned by the current user
	ownership, err := GetFolderOwnership(folderID, userID)
	if err == FolderNotFoundError {
		return CheckFolderOwnership(userID, parentID)
	} else if err != nil {
		log.Printf("Error checking for folder ownership: %v\n", err)
		return shared.FolderOwnershipInfo{}, err
	}

	// Check to see if the folder is valid
	if len(ownership.ID) > 0 {
		return ownership, nil
	}

	if len(parentID) == 0 {
		return ownership, nil
	}

	return CheckFolderOwnership(userID, parentID)
}

func GetVaultItems(
	userID,
	folderID string,
	pwFiles bool,
) ([]shared.VaultItem, shared.FolderOwnershipInfo, error) {
	var rows *sql.Rows
	var err error
	var ownership shared.FolderOwnershipInfo

	var qFilter string
	if pwFiles {
		qFilter = ` AND (v.pw_data IS NOT NULL AND LENGTH(v.pw_data) > 0)
		            ORDER BY modified DESC`
	} else {
		qFilter = ` AND (v.pw_data IS NULL OR LENGTH(v.pw_data) = 0)
		            ORDER BY modified DESC`
	}

	if len(folderID) == 0 || folderID == userID {
		query := `SELECT v.id, v.name, v.length, v.modified, v.protected_key,
       		                 v.shared_by, v.link_tag, v.can_modify, v.ref_id, v.pw_data,
       		                 (SELECT COUNT(*) FROM sharing s WHERE s.item_id = v.id) AS share_count
       		                 FROM vault v WHERE owner_id=$1 AND folder_id=$1`

		query += qFilter
		rows, err = db.Query(query, userID)
	} else {
		ownership, err = CheckFolderOwnership(userID, folderID)
		if err != nil || len(ownership.ID) == 0 {
			log.Printf("error checking folder ownership: %v\n", err)
			return nil, shared.FolderOwnershipInfo{}, AccessError
		}

		query := `SELECT v.id, v.name, v.length, v.modified, v.protected_key,
       		                 v.shared_by, v.link_tag, v.can_modify, v.ref_id, v.pw_data,
       		                 (SELECT COUNT(*) FROM sharing s WHERE s.item_id = v.id) AS share_count
		          FROM vault v WHERE folder_id=$1`
		query += qFilter
		rows, err = db.Query(query, folderID)
	}

	if err != nil {
		log.Printf("Error retrieving vault contents: %v", err)
		return nil, shared.FolderOwnershipInfo{}, err
	}

	var result []shared.VaultItem
	defer rows.Close()
	for rows.Next() {
		var id string
		var name string
		var length int64
		var modified time.Time
		var protectedKey []byte
		var sharedBy string
		var linkTag string
		var canModify bool
		var refID string
		var pwData []byte
		var shareCount int

		err = rows.Scan(&id, &name, &length, &modified, &protectedKey,
			&sharedBy, &linkTag, &canModify, &refID, &pwData,
			&shareCount)
		if err != nil {
			return nil, shared.FolderOwnershipInfo{}, err
		}

		isOwner := refID == id
		if len(folderID) != 0 && folderID != userID {
			canModify = ownership.CanModify
			isOwner = ownership.IsOwner
		}

		if refID != id {
			// Hide share count for non-owners
			shareCount = 0
		}

		result = append(result, shared.VaultItem{
			ID:           id,
			Name:         name,
			Size:         length,
			Modified:     modified,
			ProtectedKey: protectedKey,
			SharedWith:   shareCount,
			SharedBy:     sharedBy,
			LinkTag:      linkTag,
			CanModify:    canModify,
			RefID:        refID,
			IsOwner:      isOwner,
			PasswordData: pwData,
		})
	}

	if len(result) == 0 {
		return []shared.VaultItem{}, ownership, nil
	}

	return result, ownership, nil
}

// AddVaultItem inserts file metadata into the vault table
func AddVaultItem(userID string, item shared.VaultUpload) (string, error) {
	if len(userID) == 0 || len(item.Name) == 0 || len(item.ProtectedKey) == 0 {
		errorMsg := fmt.Sprintf("missing required fields for a new item\n"+
			"user id len: %d\nitem name len: %d\nkey len: %d\n",
			len(userID), len(item.Name), len(item.ProtectedKey))
		return "", errors.New(errorMsg)
	} else if item.Length == 0 || item.Chunks == 0 {
		return "", errors.New("file length cannot be 0")
	}

	if len(item.FolderID) == 0 {
		// Assume user's root folder
		item.FolderID = userID
	}

	// Ensure user has write permissions for this folder
	ownership, err := CheckFolderOwnership(userID, item.FolderID)
	if err != nil {
		return "", err
	} else if !ownership.CanModify {
		return "", ReadOnlyError
	}

	itemID := shared.GenRandomString(VaultIDLength)
	for VaultItemIDExists(itemID) {
		itemID = shared.GenRandomString(VaultIDLength)
	}

	pwData := item.PasswordData
	if len(pwData) == 0 {
		pwData = nil
	}

	s := `INSERT INTO vault
	      (
	       id, owner_id, name, length, folder_id, 
	       chunks, protected_key, modified, pw_data, 
	       ref_id
	      )
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $1)`
	_, err = db.Exec(
		s,
		itemID,
		userID,
		item.Name,
		item.Length,
		item.FolderID,
		item.Chunks,
		item.ProtectedKey,
		time.Now().UTC(),
		pwData)
	if err != nil {
		return "", err
	}

	return itemID, nil
}

// ChangeFilePermission enables/disables a user's ability to modify a file
func ChangeFilePermission(fileID, ownerID string, canModify bool) error {
	s := `UPDATE vault
	      SET can_modify=$1
	      WHERE ref_id=$2 and owner_id=$3`
	_, err := db.Exec(s, canModify, fileID, ownerID)
	if err != nil {
		log.Printf("Error updating file permissions: %v\n", err)
		return err
	}

	return nil
}

// ShareFile shares a user's file with another user via the recipient's user
// ID (determined before calling ShareFile).
// Returns the ID created in the `sharing` table.
func ShareFile(share shared.NewSharedItem, userID string) (string, error) {
	if len(share.RecipientID) == 0 {
		return "", errors.New("invalid recipient id")
	}

	folderID, err := GetFileFolderID(share.ItemID, userID)
	if err != nil {
		return "", err
	}

	ownership, err := GetFolderOwnership(folderID, userID)
	if err != nil {
		return "", err
	} else if !ownership.IsOwner {
		return "", errors.New("cannot share within shared folder")
	}

	isAlreadyShared, err := IsSharedWithRecipient(share.UserID, share.ItemID, share.RecipientID)
	if err != nil {
		return "", err
	} else if isAlreadyShared {
		return "", AlreadySharedError
	}

	file, err := RetrieveVaultMetadata(share.ItemID, share.UserID)
	if err != nil {
		return "", err
	}

	itemID := shared.GenRandomString(VaultIDLength)
	for VaultItemIDExists(itemID) {
		itemID = shared.GenRandomString(VaultIDLength)
	}

	sharedByName, _ := GetUserPublicName(userID)

	s1 := `INSERT INTO vault
    	           (id, name, folder_id, owner_id, b2_id, length, chunks,
                    protected_key, shared_by, modified, can_modify, ref_id, pw_data)
	       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err = db.Exec(s1,
		itemID, file.Name,
		share.RecipientID, share.RecipientID,
		file.B2ID, file.Length, file.Chunks,
		share.ProtectedKey, sharedByName,
		time.Now().UTC(), share.CanModify, file.ID, file.PasswordData)
	if err != nil {
		return "", err
	}

	shareID, shareErr := AddSharingEntry(
		share.UserID,
		share.RecipientID,
		share.ItemID,
		false,
		share.CanModify)

	return shareID, shareErr
}

func VaultItemIDExists(id string) bool {
	rows, err := db.Query(`SELECT * FROM vault WHERE id=$1`, id)
	if err != nil {
		log.Printf("Error checking vault item id: %v", err)
		return true
	}

	// If any rows are returned, the id exists
	defer rows.Close()
	if rows.Next() {
		return true
	}

	return false
}

// UpdateVaultFile updates the contents of a file in the vault. Note that the
// name is always an encrypted string, and that password data is always an
// encrypted byte array
func UpdateVaultFile(id, ownerID string, mod shared.ModifyVaultItem) error {
	err := UserCanEditItem(id, ownerID, false)
	if err != nil {
		return err
	}

	s := `UPDATE vault
	      SET name=$1, pw_data=$2, modified=$3
	      WHERE ref_id=$4`
	_, err = db.Exec(s, mod.Name, mod.PasswordData, time.Now().UTC(), id)
	if err != nil {
		return err
	}

	return nil
}

// DeleteVaultFile deletes an entry in the file vault
func DeleteVaultFile(id, ownerID string) error {
	err := UserCanEditItem(id, ownerID, false)
	if err != nil {
		return err
	}

	s := `DELETE FROM vault WHERE ref_id=$1`
	_, err = db.Exec(s, id)
	return err
}

// AdminDeleteFile deletes an entry in the file vault regardless of owner
func AdminDeleteFile(id string) error {
	s := `DELETE FROM vault WHERE id=$1 OR ref_id=$1`
	_, err := db.Exec(s, id)
	return err
}

// AdminFetchVaultFiles fetches all files for a specific user
func AdminFetchVaultFiles(userID string) ([]shared.AdminFileInfoResponse, error) {
	response := []shared.AdminFileInfoResponse{}

	s := `SELECT id, name, length, owner_id, modified FROM vault WHERE owner_id=$1`
	rows, err := db.Query(s, userID)
	if err != nil {
		log.Printf("Error retrieving files: %v\n", err)
		return response, err
	}

	defer rows.Close()
	for rows.Next() {
		var (
			id       string
			name     string
			length   int64
			ownerID  string
			modified time.Time
		)

		err = rows.Scan(&id, &name, &length, &ownerID, &modified)
		if err != nil {
			return response, err
		}

		response = append(response, shared.AdminFileInfoResponse{
			ID:         id,
			BucketName: name,
			Size:       shared.ReadableFileSize(length),
			OwnerID:    ownerID,
			Modified:   modified,

			RawSize: length,
		})
	}

	return response, err
}

// DeleteSharedFile deletes a shared file from the recipient's vault
func DeleteSharedFile(id, ownerID string) error {
	s := `DELETE FROM vault WHERE id=$1 AND owner_id=$2 RETURNING ref_id`
	rows, err := db.Query(s, id, ownerID)
	if err != nil {
		return err
	}

	defer rows.Close()
	if rows.Next() {
		var refID string
		err = rows.Scan(&refID)
		if err != nil {
			return err
		}

		err = RemoveShareEntryByRecipient(ownerID, refID)
	}

	return err
}

// DeleteSharedFileByRefID deletes a file shared with a recipient using the file's
// reference ID. This should only get called when a user removes a recipient from
// their shared file.
func DeleteSharedFileByRefID(id, ownerID string) error {
	s := `DELETE FROM vault WHERE ref_id=$1 AND owner_id=$2`
	_, err := db.Exec(s, id, ownerID)
	return err
}

// GetFileFolderID returns the parent folder ID for a particular file
func GetFileFolderID(fileID, ownerID string) (string, error) {
	s := `WITH result_count AS (
	          SELECT COUNT(*) AS c FROM vault WHERE ref_id = $1
	      )
	      SELECT folder_id
	      FROM vault
	      WHERE ref_id = $1
	      AND (CASE
	          WHEN (SELECT c FROM result_count) = 1 THEN TRUE
	          ELSE owner_id = $2
	      END);`
	rows, err := db.Query(s, fileID, ownerID)
	if err != nil {
		log.Printf("Error retrieving folder ID: %v\n", err)
		return "", err
	}

	defer rows.Close()
	if rows.Next() {
		var folderID string
		err = rows.Scan(&folderID)
		if err != nil {
			return "", err
		}

		return folderID, nil
	}

	return "", errors.New("folder ID not found for file")
}

// RetrieveFullItemInfo returns all metadata necessary to interact with a file
// in an isolated (non-folder) environment.
func RetrieveFullItemInfo(id, ownerID string) (shared.VaultItemInfo, error) {
	metadata, err := RetrieveVaultMetadata(id, ownerID)
	if err != nil {
		return shared.VaultItemInfo{}, err
	}

	keySequence, err := GetKeySequence(metadata.FolderID, ownerID)
	if err != nil {
		return shared.VaultItemInfo{}, err
	}

	return shared.VaultItemInfo{
		ID:           id,
		Name:         metadata.Name,
		Size:         metadata.Length,
		Modified:     time.Time{},
		ProtectedKey: metadata.ProtectedKey,
		CanModify:    false,
		IsOwner:      false,
		RefID:        "",
		KeySequence:  keySequence,
	}, nil
}

func AdminRetrieveMetadata(fileID string) (shared.AdminFileInfoResponse, error) {
	var (
		id       string
		name     string
		length   int64
		ownerID  string
		modified time.Time
	)

	s := `SELECT id, name, length, owner_id, modified FROM vault WHERE id=$1`
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

// RetrieveVaultMetadata returns a FileMetadata struct containing a specific
// file's metadata
func RetrieveVaultMetadata(id, ownerID string) (FileMetadata, error) {
	folderID, err := GetFileFolderID(id, ownerID)

	if err != nil || len(folderID) == 0 {
		msg := fmt.Sprintf("unable to fetch parent folder for file: %v", err)
		return FileMetadata{}, errors.New(msg)
	}

	ownership, err := CheckFolderOwnership(ownerID, folderID)
	if err != nil {
		// Check if this is a file shared from another user's home dir
		isShared, err := IsSharedWithRecipient(folderID, id, ownerID)
		if !isShared || err != nil {
			return FileMetadata{}, AccessError
		}
	}

	s := `SELECT id, b2_id, ref_id, name, length, chunks, protected_key, pw_data
	      FROM vault
	      WHERE ref_id = $1`

	var rows *sql.Rows
	if folderID == ownerID {
		// This file is in the user's root folder, which requires filtering
		// by owner_id as well.
		s += " and owner_id = $2"
		rows, err = db.Query(s, id, ownerID)
	} else {
		rows, err = db.Query(s, id)
	}

	if err != nil {
		log.Printf("Error retrieving metadata: %v\n", err)
		return FileMetadata{}, err
	}

	defer rows.Close()
	if rows.Next() {
		var itemID string
		var b2ID string
		var refID string
		var name string
		var length int64
		var chunks int
		var protectedKey []byte
		var passwordData []byte
		err = rows.Scan(
			&itemID, &b2ID, &refID, &name,
			&length, &chunks, &protectedKey, &passwordData)
		if err != nil {
			log.Printf("Error scanning rows: %v\n", err)
			return FileMetadata{}, err
		}

		return FileMetadata{
			ID:                itemID,
			B2ID:              b2ID,
			RefID:             refID,
			Name:              name,
			Length:            length,
			Chunks:            chunks,
			FolderID:          folderID,
			ProtectedKey:      protectedKey,
			PasswordData:      passwordData,
			OwnsParentFolder:  ownership.IsOwner,
			ParentFolderOwner: ownership.ID,
		}, nil
	}

	log.Printf("No metadata found for id: %s", id)
	return FileMetadata{}, errors.New("no metadata found")
}
