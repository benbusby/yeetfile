package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
	"yeetfile/shared"
)

var FolderNotFoundError = errors.New("folder not found")

func NewRootFolder(id string, protectedKey []byte) error {
	if len(id) == 0 {
		return errors.New("invalid folder ID length")
	}

	// User should exist for this folder ID
	if _, err := GetUserByID(id); err != nil {
		return errors.New("error fetching user by ID when creating root folder")
	}

	return insertFolder(id, id, shared.NewVaultFolder{
		Name:         "",
		ParentID:     "",
		ProtectedKey: protectedKey,
	}, false)
}

func NewFolder(folder shared.NewVaultFolder, ownerID string, pwFolder bool) (string, error) {
	folderID := shared.GenRandomString(VaultIDLength)
	for FolderIDExists(folderID) {
		folderID = shared.GenRandomString(VaultIDLength)
	}

	if len(folder.ParentID) == 0 {
		// Assume parent is the user's root folder
		folder.ParentID = ownerID
	} else {
		parentOwnerID, err := GetFolderOwner(folder.ParentID)
		if err != nil {
			return "", err
		}

		if parentOwnerID != ownerID {
			_, err = CheckFolderOwnership(ownerID, folder.ParentID)
			if err != nil {
				return "", err
			}
		}

		ownerID = parentOwnerID
	}

	return folderID, insertFolder(folderID, ownerID, folder, pwFolder)
}

func insertFolder(id, ownerID string, folder shared.NewVaultFolder, pwFolder bool) error {
	s := `INSERT INTO folders 
	      (id, name, owner_id, protected_key, modified, parent_id, pw_folder, ref_id)
	      VALUES 
	      ($1, $2, $3, $4, $5, $6, $7, $1)`

	_, err := db.Exec(
		s,
		id,
		folder.Name,
		ownerID,
		folder.ProtectedKey,
		time.Now().UTC(),
		folder.ParentID,
		pwFolder)

	return err
}

func FolderIDExists(id string) bool {
	rows, err := db.Query(`SELECT * FROM folders WHERE id=$1`, id)
	if err != nil {
		log.Printf("Error checking folder id: %v", err)
		return true
	}

	// If any rows are returned, the id exists
	defer rows.Close()
	if rows.Next() {
		return true
	}

	return false
}

// GetFolderTree returns a tree representation of the user's vault
func GetFolderTree(userID string, isPassVault bool) error {
	s := `WITH RECURSIVE folder_tree AS (
	          SELECT 
	              f.id AS folder_id, f.name, 0::bigint as length, f.modified, f.protected_key,
	              f.shared_by, f.link_tag, f.ref_id, f.can_modify,
	              (SELECT COUNT(*) FROM sharing s WHERE s.item_id = f.id) AS share_count,
	              true AS is_folder
	          FROM folders f
	          WHERE f.parent_id = $1

	          UNION ALL

	          SELECT
	              f.id AS folder_id, f.name, 0::bigint as length, f.modified, f.protected_key,
	              f.shared_by, f.link_tag, f.ref_id, f.can_modify,
	              (SELECT COUNT(*) FROM sharing s WHERE s.item_id = f.id) AS share_count,
	              true AS is_folder
	          FROM folders f
	          INNER JOIN folder_tree ft ON f.parent_id = ft.folder_id
	      )
	      SELECT *
	      FROM folder_tree
	      UNION ALL

	      -- Retrieve all vault items associated with the folders in the tree
	      SELECT 
	          v.id, v.name, v.length, v.modified, v.protected_key,
	          v.shared_by, v.link_tag, v.ref_id, v.can_modify,
	          (SELECT COUNT(*) FROM sharing s WHERE s.item_id = v.id) AS share_count,
                  false as is_folder
	      FROM vault v  
	      WHERE v.folder_id IN (SELECT folder_id FROM folder_tree)

	      ORDER BY modified DESC;`

	rows, err := db.Query(s, userID)
	for rows.Next() {
		// TODO: Create tree of index-able vault items
	}

	return err
}

func GetSubfolders(
	folderID,
	ownerID string,
	ownership shared.FolderOwnershipInfo,
	pwFolder bool,
) ([]shared.VaultFolder, error) {
	var err error
	if ownership == (shared.FolderOwnershipInfo{}) {
		ownership, err = CheckFolderOwnership(ownerID, folderID)
		if err != nil {
			return nil, err
		}
	}

	query := `SELECT 
	          f.id, 
	          f.name, 
     	          f.modified, 
	          f.protected_key, 
	          f.shared_by, 
	          f.link_tag, 
	          f.ref_id, 
	          f.can_modify,
	          f.pw_folder,
	          (SELECT COUNT(*) FROM sharing s WHERE s.item_id = f.id) AS share_count
	          FROM folders f
	          WHERE f.parent_id = $1
	          AND f.pw_folder = $2
	          ORDER BY f.modified DESC`

	rows, err := db.Query(query, folderID, pwFolder)
	if err != nil {
		return []shared.VaultFolder{}, err
	}

	var subfolders []shared.VaultFolder
	defer rows.Close()
	for rows.Next() {
		var id string
		var name string
		var modified time.Time
		var protectedKey []byte
		var sharedBy string
		var linkTag string
		var refID string
		var canModify bool
		var passwordFolder bool
		var shareCount int

		err = rows.Scan(
			&id,
			&name,
			&modified,
			&protectedKey,
			&sharedBy,
			&linkTag,
			&refID,
			&canModify,
			&passwordFolder,
			&shareCount)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return []shared.VaultFolder{}, err
		}

		if !ownership.CanModify {
			canModify = false
		}

		isOwner := refID == id
		if len(folderID) != 0 && folderID != ownerID {
			canModify = ownership.CanModify
			isOwner = ownership.IsOwner
		}

		if !ownership.IsOwner {
			// Hide share count for non-owners
			shareCount = 0
		}

		subfolders = append(subfolders, shared.VaultFolder{
			ID:             id,
			Name:           name,
			Modified:       modified,
			ProtectedKey:   protectedKey,
			SharedWith:     shareCount,
			SharedBy:       sharedBy,
			LinkTag:        linkTag,
			CanModify:      canModify,
			RefID:          refID,
			IsOwner:        isOwner,
			PasswordFolder: passwordFolder,
		})
	}

	if len(subfolders) == 0 {
		return []shared.VaultFolder{}, nil
	}

	return subfolders, nil
}

func GetParentFolderID(folderID string) (string, error) {
	query := `SELECT parent_id from folders WHERE id=$1`
	rows, err := db.Query(query, folderID)
	if err == nil && rows.Next() {
		defer rows.Close()
		var parentID string

		err = rows.Scan(&parentID)
		if err != nil {
			return "", err
		}

		return parentID, nil
	}

	return "", FolderNotFoundError
}

func GetFolderOwner(folderID string) (string, error) {
	query := `SELECT owner_id from folders WHERE id=$1`
	rows, err := db.Query(query, folderID)
	if err == nil && rows.Next() {
		defer rows.Close()
		var ownerID string

		err = rows.Scan(&ownerID)
		if err != nil {
			return "", err
		}

		return ownerID, nil
	}

	return "", FolderNotFoundError
}

func ChangeFolderPermission(folderID, ownerID string, canModify bool) error {
	s := `UPDATE folders
	      SET can_modify=$1
	      WHERE ref_id=$2 and owner_id=$3`
	_, err := db.Exec(s, canModify, folderID, ownerID)
	if err != nil {
		log.Printf("Error updating folder permissions: %v\n", err)
		return err
	}

	return nil
}

func GetFolderOwnerStorage(folderID string) (int64, int64, error) {
	var (
		storageUsed      int64
		storageAvailable int64
	)

	s := `
	    WITH folder_owner AS (
	        SELECT owner_id FROM folders WHERE id = $1
	    )
	    SELECT storage_used, storage_available
		FROM users
		WHERE id = (SELECT owner_id FROM folder_owner);`

	err := db.QueryRow(s, folderID).Scan(&storageUsed, &storageAvailable)
	if err != nil {
		return 0, 0, err
	}

	return storageUsed, storageAvailable, nil
}

func UpdateFolderOwnerStorage(folderID string, amount int64) error {
	var (
		storageUsed      int64
		storageAvailable int64
	)

	s := `
	    WITH folder_owner AS (
	        SELECT owner_id FROM folders WHERE id = $1
	    )
	    UPDATE users 
	      SET storage_used = CASE 
	                           WHEN storage_used + $2 < 0 THEN 0
	                           ELSE storage_used + $2
	                         END
	      WHERE id = (SELECT owner_id FROM folder_owner) 
	      RETURNING storage_used, storage_available`

	err := db.QueryRow(s, folderID, amount).Scan(&storageUsed, &storageAvailable)
	if err != nil {
		return err
	} else if storageUsed > storageAvailable && amount > 0 {
		return UserStorageExceeded
	}

	return nil
}

func GetFolderOwnership(
	folderID,
	ownerID string,
) (shared.FolderOwnershipInfo, error) {
	query := `SELECT id, ref_id, can_modify FROM folders WHERE ref_id=$1 and owner_id=$2`
	rows, err := db.Query(query, folderID, ownerID)
	if err == nil && rows.Next() {
		defer rows.Close()
		var id string
		var refID string
		var canModify bool

		err = rows.Scan(&id, &refID, &canModify)
		if err != nil {
			return shared.FolderOwnershipInfo{}, err
		}

		return shared.FolderOwnershipInfo{
			ID:        id,
			RefID:     refID,
			CanModify: canModify,
			IsOwner:   id == refID,
		}, nil
	} else if err != nil {
		return shared.FolderOwnershipInfo{}, err
	}

	return shared.FolderOwnershipInfo{}, FolderNotFoundError
}

// GetFolderInfo returns metadata for a particular folder
func GetFolderInfo(
	folderID,
	ownerID string,
	ownership shared.FolderOwnershipInfo,
	ownerOnly bool,
) (shared.VaultFolder, error) {
	var err error
	if ownership == (shared.FolderOwnershipInfo{}) {
		ownership, err = CheckFolderOwnership(ownerID, folderID)
		if err != nil || len(ownership.ID) == 0 {
			return shared.VaultFolder{}, AccessError
		}
	}

	var rows *sql.Rows
	if ownerOnly {
		query := `SELECT id, owner_id, name, modified, protected_key, parent_id, ref_id, pw_folder
	                  FROM folders 
	                  WHERE id=$1 AND owner_id=$2`
		rows, err = db.Query(query, folderID, ownerID)
	} else {
		query := `SELECT id, owner_id, name, modified, protected_key, parent_id, ref_id, pw_folder
	                  FROM folders 
	                  WHERE id=$1 OR ref_id=$1`
		rows, err = db.Query(query, folderID)
		if err != nil {
			return shared.VaultFolder{}, err
		}
	}

	if err != nil {
		return shared.VaultFolder{}, err
	}

	defer rows.Close()
	hasNext := rows.Next()
	for hasNext {
		var id string
		var folderOwnerID string
		var name string
		var modified time.Time
		var protectedKey []byte
		var parentID string
		var refID string
		var pwFolder bool

		err = rows.Scan(
			&id,
			&folderOwnerID,
			&name,
			&modified,
			&protectedKey,
			&parentID,
			&refID,
			&pwFolder)
		if err != nil {
			return shared.VaultFolder{}, err
		}

		if ownerOnly && refID != id {
			return shared.VaultFolder{}, errors.New("user is not the owner of the folder")
		}

		if parentID == ownerID {
			// Ignore parent ID if the parent is the root folder
			parentID = ""
		}

		hasNext = rows.Next()
		if ownerID != folderOwnerID && hasNext {
			// Keep looking for the folder's correct owner info
			continue
		}

		// Don't send actual user ID in root folder response
		if id == ownerID {
			id = "root"
			refID = "root"
		}

		return shared.VaultFolder{
			ID:             id,
			Name:           name,
			Modified:       modified,
			ProtectedKey:   protectedKey,
			ParentID:       parentID,
			RefID:          refID,
			IsOwner:        ownership.IsOwner,
			CanModify:      ownership.CanModify,
			PasswordFolder: pwFolder,
		}, nil
	}

	return shared.VaultFolder{}, errors.New("folder not found")
}

// GetKeySequence starts with a specific folder and recursively climbs up to each
// parent folder, retrieving the parent's protected key. This is required to decrypt
// a folder's contents, since a folder's key is always encrypted with its parents key.
func GetKeySequence(folderID, ownerID string) ([][]byte, error) {
	s := `WITH RECURSIVE parent_hierarchy AS (
	         SELECT ref_id, owner_id, parent_id, protected_key, 1 as depth
	         FROM folders
	         WHERE ref_id=$1 AND (parent_id=$2 OR id = ref_id)
	
	         UNION ALL
	
	         SELECT f.ref_id, f.owner_id, f.parent_id, f.protected_key, ph.depth + 1
	         FROM folders f
	         INNER JOIN parent_hierarchy ph ON f.ref_id = ph.parent_id
	     ),
	     hierarchy_with_depth_count AS (
	         SELECT ph.*, COUNT(*) OVER (PARTITION BY depth) AS depth_count,
	             ROW_NUMBER() OVER (PARTITION BY depth ORDER BY (CASE WHEN owner_id = $2 THEN 1 ELSE 2 END)) AS rn
	         FROM parent_hierarchy ph
	     )
	     SELECT protected_key
	     FROM hierarchy_with_depth_count
	     WHERE (depth_count = 1)
	        OR (depth_count > 1 AND rn = 1)
	     ORDER BY depth DESC;`

	rows, err := db.Query(s, folderID, ownerID)
	if err != nil {
		return nil, err
	}

	var keySequence [][]byte
	defer rows.Close()
	for rows.Next() {
		var protectedKey []byte
		err = rows.Scan(&protectedKey)
		if err != nil {
			return nil, err
		}

		keySequence = append(keySequence, protectedKey)
	}

	// First key is excluded since it is always returned in the folder
	// response body
	return keySequence[1:], nil
}

// ShareFolder shares a user's folder with another user via the recipient's user
// ID (determined before calling ShareFolder).
func ShareFolder(share shared.NewSharedItem, userID string) (string, error) {
	if share.ItemID == share.UserID {
		return "", errors.New("cannot share user's root folder")
	} else if len(share.RecipientID) == 0 {
		return "", errors.New("invalid recipient id")
	} else if share.RecipientID == userID {
		return "", errors.New("cannot share a user's folder with themself")
	}

	ownership, err := GetFolderOwnership(share.ItemID, userID)
	if err != nil {
		return "", err
	} else if !ownership.IsOwner {
		return "", errors.New("cannot share within a shared folder")
	}

	isAlreadyShared, err := IsSharedWithRecipient(share.UserID, share.ItemID, share.RecipientID)
	if err != nil {
		return "", err
	} else if isAlreadyShared {
		return "", AlreadySharedError
	}

	folder, err := GetFolderInfo(share.ItemID, share.UserID, ownership, true)
	if err != nil {
		return "", err
	}

	folderID := shared.GenRandomString(VaultIDLength)
	for FolderIDExists(folderID) {
		folderID = shared.GenRandomString(VaultIDLength)
	}

	sharedByName, _ := GetUserPublicName(userID)

	// Add new folder entry for recipient
	s1 := `INSERT INTO folders (id, name, parent_id, owner_id, 
                     protected_key, shared_by, modified, ref_id, can_modify,
                     pw_folder)
	       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = db.Exec(s1, folderID, folder.Name,
		share.RecipientID, share.RecipientID, share.ProtectedKey,
		sharedByName, time.Now().UTC(), folder.ID, share.CanModify,
		folder.PasswordFolder)
	if err != nil {
		return "", err
	}

	shareID, shareErr := AddSharingEntry(share.UserID, share.RecipientID, share.ItemID, true, share.CanModify)
	if shareErr != nil {
		return "", shareErr
	}

	return shareID, nil
}

// UpdateVaultFolderName updates the name of a folder in the vault. Note that the
// name is always an encrypted string
func UpdateVaultFolderName(id, ownerID, newName string) error {
	ownership, err := CheckFolderOwnership(ownerID, id)
	if err != nil {
		return err
	} else if !ownership.CanModify {
		return errors.New("unable to modify read-only shared folder")
	}

	s := `UPDATE folders
	      SET name=$1, modified=$2
	      WHERE ref_id=$3`
	_, err = db.Exec(s, newName, time.Now().UTC(), id)
	if err != nil {
		log.Printf("Error updating folder name: %v\n", err)
		return err
	}

	return nil
}

// DeleteSharedFolder removes a folder that has been shared with the current user
func DeleteSharedFolder(id, ownerID string) error {
	s := `DELETE FROM folders WHERE id=$1 AND owner_id=$2 RETURNING ref_id`
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

func DeleteSharedFolderByRefID(id, ownerID string) error {
	s := `DELETE FROM folders WHERE ref_id=$1 AND owner_id=$2`
	_, err := db.Exec(s, id, ownerID)
	return err
}

// DeleteVaultFolder removes the specified folder from the table
func DeleteVaultFolder(id, ownerID string) error {
	ownership, err := CheckFolderOwnership(ownerID, id)
	if err != nil {
		return err
	} else if !ownership.IsOwner {
		return errors.New("unable to modify read-only shared folder")
	}

	s := `DELETE FROM folders WHERE ref_id=$1`
	_, err = db.Exec(s, id)
	if err != nil {
		return err
	}

	return nil
}
