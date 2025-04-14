package admin

import (
	"golang.org/x/crypto/bcrypt"
	"log"
	"yeetfile/backend/db"
	"yeetfile/backend/mail"
	"yeetfile/shared"
)

func createInvites(emails []string) error {
	var invites []db.Invite
	for _, email := range emails {
		code := shared.GenRandomString(12)
		codeHash, err := bcrypt.GenerateFromPassword([]byte(code), 8)
		if err != nil {
			log.Printf("Error generating invite code: %v\n", err)
			return err
		}

		invite := db.Invite{
			Email:    email,
			Code:     code,
			CodeHash: codeHash,
		}

		invites = append(invites, invite)
	}

	err := db.AddInviteCodeHashes(invites)
	if err != nil {
		return err
	}

	for _, invite := range invites {
		err = mail.SendInviteEmail(invite.Code, invite.Email)
		if err != nil {
			return err
		}
	}

	return nil
}

func deleteInvites(emails []string) error {
	err := db.RemoveInvites(emails)
	if err != nil {
		log.Printf("Error deleting invites: %v\n", err)
		return err
	}

	return nil
}
