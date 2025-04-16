package db

import (
	"fmt"
	"github.com/lib/pq"
	"strings"
)

type Invite struct {
	Email    string
	Code     string
	CodeHash []byte
}

func AddInviteCodeHashes(inviteCodes []Invite) error {
	valueStrings := make([]string, 0, len(inviteCodes))
	valueArgs := make([]interface{}, 0, len(inviteCodes)*2)

	for i, inviteCode := range inviteCodes {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		valueArgs = append(valueArgs, inviteCode.Email)
		valueArgs = append(valueArgs, inviteCode.CodeHash)
	}

	stmt := fmt.Sprintf(
		"INSERT INTO invites (email, code_hash) VALUES %s",
		strings.Join(valueStrings, ","),
	)

	_, err := db.Exec(stmt, valueArgs...)
	return err
}

func GetInviteCodeHash(email string) ([]byte, error) {
	var codeHash []byte
	s := `SELECT code_hash FROM invites WHERE email=$1`
	err := db.QueryRow(s, email).Scan(&codeHash)
	if err != nil {
		return nil, err
	}

	return codeHash, nil
}

func GetInvitesList() ([]string, error) {
	var emails []string
	s := `SELECT email FROM invites`
	rows, err := db.Query(s)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var email string
		err = rows.Scan(&email)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	return emails, nil
}

func RemoveInvites(email []string) error {
	s := `DELETE FROM invites WHERE email=any($1)`
	_, err := db.Exec(s, pq.Array(email))
	if err != nil {
		return err
	}

	return nil
}
