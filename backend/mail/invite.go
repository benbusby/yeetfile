package mail

import (
	"bytes"
	"text/template"
	"yeetfile/shared/endpoints"
)

type InviteEmail struct {
	Code     string
	Email    string
	Domain   string
	Endpoint string
}

var inviteSubject = "YeetFile Invite"
var inviteBodyTemplate = template.Must(template.New("").Parse(
	"Hello,\n\nYou have been invited to join a YeetFile instance at this " +
		"domain: {{.Domain}}\n\n" +
		"YeetFile is an open source platform that allows encrypted file " +
		"sharing and storage.\n\n" +
		"To create an account, you can use the following link:\n\n" +
		"{{.Domain}}{{.Endpoint}}?email={{.Email}}&code={{.Code}}"))

func SendInviteEmail(code string, to string) error {
	var buf bytes.Buffer

	inviteEmail := InviteEmail{
		Code:     code,
		Email:    to,
		Domain:   smtpConfig.CallbackDomain,
		Endpoint: string(endpoints.HTMLSignup),
	}

	err := inviteBodyTemplate.Execute(&buf, inviteEmail)
	if err != nil {
		return err
	}

	body := buf.String()
	go sendEmail(to, inviteSubject, body)
	return nil
}
