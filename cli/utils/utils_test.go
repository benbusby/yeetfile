package utils

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"yeetfile/cli/lang"
)

func TestParseDownloadString(t *testing.T) {
	path := "abc123"
	secret := "secret.goes.here"
	downloadString := fmt.Sprintf("%s#%s", path, secret)

	parsedPath, parsedSecret, err := ParseDownloadString(downloadString)
	if err != nil {
		t.Fatalf("Error parsing download string: %v", err)
	}

	if len(parsedPath) == 0 || len(parsedSecret) == 0 {
		t.Fatalf("Error retrieving path and secret from download str")
	} else if parsedPath != path || !bytes.Equal([]byte(secret), parsedSecret) {
		t.Fatalf("Parsed path or secret values are incorrect")
	}

	invalidDownloadString := "invalid"
	_, _, err = ParseDownloadString(invalidDownloadString)
	if err == nil {
		t.Fatalf("Invalid download string was parsed without an error")
	}
}

// ParseDownloadString processes a URL such as
// "[http(s)://...]this.example.path#<hex key>"
// into separate usable components: the path to the file (this.example.path),
// and a [32]byte key to use for decrypting the encrypted salt from the server.

func ParseDownloadString(tag string) (string, []byte, error) {
	splitURL := strings.Split(tag, "/")
	splitTag := strings.Split(splitURL[len(splitURL)-1], "#")

	if len(splitTag) != 2 {
		return "", nil, errors.New(lang.I18n.T("cli.utils.error.invalid_dl_link"))
	}

	path := splitTag[0]
	secret := splitTag[1]

	return path, []byte(secret), nil
}
