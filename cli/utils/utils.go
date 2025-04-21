package utils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	"yeetfile/cli/styles"
	"yeetfile/lang"
)

var (
	httpErrorCodeFormat = "[code: %d]"
)

// ParseDownloadString processes a URL such as
// "[http(s)://...]this.example.path#<hex key>"
// into separate usable components: the path to the file (this.example.path),
// and a [32]byte key to use for decrypting the encrypted salt from the server.
func ParseDownloadString(tag string) (string, []byte, error) {
	splitURL := strings.Split(tag, "/")
	splitTag := strings.Split(splitURL[len(splitURL)-1], "#")

	if len(splitTag) != 2 {
		return "", nil, errors.New(lang.I18n.T("cli.utils.error.invaliddl"))
	}

	path := splitTag[0]
	secret := splitTag[1]

	return path, []byte(secret), nil
}

/*
	func CopyToFile(contents string, to string) error {
		return CopyBytesToFile([]byte(contents), to)
	}

	func CopyBytesToFile(contents []byte, to string) error {
		err := os.WriteFile(to, contents, 0o644)
		if err != nil {
			return err
		}

		return err
	}
*/
func CreateHeader(title string, desc string) *huh.Note {
	return huh.NewNote().
		Title(GenerateTitle(title)).
		Description(GenerateWrappedText(desc))
}

func GenerateTitle(s string) string {
	prefix := "YeetFile CLI: "
	verticalEdge := strings.Repeat("═", len(s)+len(prefix)+2)
	title := styles.TitleStyle.Render(fmt.Sprintf(
		"╔"+verticalEdge+"╗\n"+
			"║ %s%s ║\n"+
			"╚"+verticalEdge+"╝", prefix, s))
	return title
}

func GenerateWrappedString(text string, width int) string {
	if len(text) <= width {
		return text
	}

	var result []string
	for len(text) > width {
		result = append(result, text[:width])
		text = text[width:]
	}
	result = append(result, text)
	return strings.Join(result, "\n")
}

func GenerateWrappedText(s string) string {
	maxLen := 50
	words := strings.Split(s, " ")
	var wrappedWords []string

	lineLen := 0
	i := 0
	for j, word := range words {
		if strings.HasSuffix(word, "\n") {
			lineLen = 0
			continue
		}

		if lineLen+len(word) > maxLen {
			lineLen = 0
			wrappedWords = append(wrappedWords, words[i:j]...)
			wrappedWords = append(wrappedWords, "\n")
			i = j
		}

		lineLen += len(word)
	}

	wrappedWords = append(wrappedWords, words[i:]...)
	joined := strings.Join(wrappedWords, " ")
	formatted := strings.ReplaceAll(joined, "\n ", "\n")
	return formatted
}

// GenerateDescription generates a text box with the provided description
func GenerateDescription(desc string, minLen int) string {
	return GenerateDescriptionSection("", desc, minLen)
}

func B64Encode(val []byte) string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(val)
}

func B64Decode(str string) []byte {
	val, _ := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(str)
	return val
}

// GenerateDescriptionSection generates a text box with a title positioned
// above the provided description.
func GenerateDescriptionSection(title, desc string, minLen int) string {
	var out string
	split := strings.Split(desc, "\n")
	maxLen := minLen
	for _, s := range split {
		maxLen = max(maxLen, len(s))
	}

	verticalEdge := strings.Repeat("─", maxLen+2)
	out += styles.BoldStyle.Render("┌"+verticalEdge+"┐") + "\n"
	if len(title) > 0 {
		out += styles.BoldStyle.Render("│ ") +
			title +
			strings.Repeat(" ", maxLen-len(title)) +
			styles.BoldStyle.Render(" │") + "\n"
		out += styles.BoldStyle.Render("│ ") +
			strings.Repeat("-", maxLen) +
			styles.BoldStyle.Render(" │") + "\n"
	}

	for _, s := range split {
		out += styles.BoldStyle.Render("│ ") +
			s +
			strings.Repeat(" ", maxLen-len(s)+strings.Count(s, "\\")) +
			styles.BoldStyle.Render(" │") + "\n"
	}
	out += styles.BoldStyle.Render("└" + verticalEdge + "┘")

	return out
}

func GetFilenameFromPath(path string) string {
	fullPath := strings.Split(path, string(os.PathSeparator))
	name := fullPath[len(fullPath)-1]
	return name
}

func GenerateListIdxSpacing(length int) string {
	lenStr := strconv.Itoa(length)
	return strings.Repeat(" ", len(lenStr))
}

func GetListIdxSpacing(spacing string, idx, length int) string {
	idxStr := strconv.Itoa(idx)
	lenStr := strconv.Itoa(length)
	return spacing[0 : len(lenStr)-len(idxStr)+1]
}

func LocalTimeFromUTC(utcTime time.Time) time.Time {
	return utcTime.In(time.Now().Location())
}

func ParseHTTPError(response *http.Response) error {
	body, _ := io.ReadAll(response.Body)
	errCode := fmt.Sprintf(httpErrorCodeFormat, response.StatusCode)
	msg := fmt.Sprintf(lang.I18n.T("cli.utils.error.servererr")+" %s: %s", errCode, body)
	return errors.New(msg)
}

func ShowErrorForm(msg string) {
	_ = huh.NewForm(huh.NewGroup(
		huh.NewNote().
			Title(styles.ErrStyle.Render(GenerateTitle(lang.I18n.T("cli.utils.error.errtitle")))).
			Description(msg),
		huh.NewConfirm().
			Affirmative("OK").
			Negative("")),
	).WithTheme(styles.Theme).Run()
}

func RunCmd(stdOut bool, cmd string, args ...string) error {
	if cmd == "clear" {
		switch runtime.GOOS {
		case "windows":
			cmd = "cmd"
			args = []string{"/c", "cls"}
		}
	}
	newCmd := exec.Command(cmd, args...)
	if stdOut {
		newCmd.Stdout = os.Stdout
	}
	err := newCmd.Run()
	return err
}
