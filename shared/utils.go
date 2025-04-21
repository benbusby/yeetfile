package shared

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"yeetfile/lang"
	"yeetfile/shared/constants"
)

var Characters = []rune("abcdefghijklmnopqrstuvwxyz1234567890")
var Numbers = []rune("1234567890")

var (
	httpErrorCodeFormat = "[code: %d]"
)

func ReadableFileSize(b int64) string {
	if b < 0 {
		return "Unlimited"
	}

	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	size := float64(b) / float64(div)
	if math.Mod(size, 1) == 0 {
		return fmt.Sprintf("%d %cB", int64(b)/div, "KMGT"[exp])
	}

	return fmt.Sprintf("%.1f %cB", size, "KMGT"[exp])
}

// IsPlaintext takes a string determines if it contains non-ascii characters
func IsPlaintext(text string) bool {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		for _, r := range scanner.Text() {
			if r > 127 {
				return false // Non-ASCII character found
			}
		}
	}

	return true
}

func GenRandomStringWithPrefix(n int, prefix string) string {
	randStr := GenRandomArray(n, Characters)

	if len(prefix) == 0 {
		return string(randStr)
	}

	return fmt.Sprintf("%s_%s", prefix, string(randStr))
}

func GenRandomString(n int) string {
	randStr := GenRandomArray(n, Characters)
	return string(randStr)
}

func GenRandomNumbers(n int) string {
	randNums := GenRandomArray(n, Numbers)
	return string(randNums)
}

func GenRandomArray(n int, runes []rune) []rune {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	b := make([]rune, n)
	for i := range b {
		b[i] = runes[r.Intn(len(runes))]
	}

	return b
}

func EscapeString(s string) string {
	s = strings.ReplaceAll(s, "*", "\\*")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

func UnescapeString(s string) string {
	s = strings.ReplaceAll(s, "\\*", "*")
	s = strings.ReplaceAll(s, "\\_", "_")
	return s
}

func CalculateNumChunks(fileSize int64) int {
	return int(math.Ceil(float64(fileSize) / float64(constants.ChunkSize)))
}

func RemoveOverlap[T comparable](source []T, remove []T) []T {
	// Create a map to store the elements to be removed for quick lookup
	toRemove := make(map[T]struct{}, len(remove))
	for _, item := range remove {
		toRemove[item] = struct{}{}
	}

	// Create a new slice to store the filtered elements
	var filtered []T
	for _, item := range source {
		if _, found := toRemove[item]; !found {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func FormatIDTail(fullID string) string {
	idTail := fullID[len(fullID)-4:]
	return fmt.Sprintf("*%s", idTail)
}

func GetFileInfo(filepath string) (*os.File, os.FileInfo, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}

	return file, stat, nil
}

func CreateNewSaveName(filename string) string {
	var nameOnly string
	var ext string

	if strings.Contains(filename, ".") {
		filenameSegments := strings.Split(filename, ".")
		nameOnly = strings.Join(
			filenameSegments[0:len(filenameSegments)-1], ".")
		ext = filenameSegments[len(filenameSegments)-1]
	} else {
		nameOnly = filename
	}

	match, _ := regexp.MatchString(".*-\\d", nameOnly)
	if match {
		nameSegments := strings.Split(nameOnly, "-")
		saveNum := nameSegments[len(nameSegments)-1]
		digit, err := strconv.Atoi(saveNum)
		if err == nil {
			baseName := strings.Join(
				nameSegments[0:len(nameSegments)-1], "-")
			if len(ext) > 0 {
				return fmt.Sprintf("%s-%d.%s", baseName, digit+1, ext)
			} else {
				return fmt.Sprintf("%s-%d", baseName, digit+1)
			}
		}
	}

	newName := nameOnly + "-1"
	if len(ext) == 0 {
		return newName
	}

	return strings.Join([]string{newName, ext}, ".")
}

func ArrayContains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// ObscureEmail takes an email and strips out the majority of the address and
// domain, adding "***" as an indicator of the obfuscation for both.
func ObscureEmail(email string) (string, error) {
	segments := strings.Split(email, "@")
	if len(segments) != 2 {
		return "", errors.New(lang.I18n.T("cli.utils.error.invalid_email"))
	}

	address := segments[0]
	domain := segments[1]

	segments = strings.Split(email, ".")
	ext := segments[len(segments)-1]

	var hiddenEmail string
	if len(address) > 1 {
		hiddenEmail = fmt.Sprintf(
			"%c%c***%c@%c***.%s",
			address[0],
			address[1],
			address[len(address)-1],
			domain[0],
			ext)
	} else {
		hiddenEmail = fmt.Sprintf(
			"%c***%c@%c***.%s",
			address[0],
			address[len(address)-1],
			domain[0],
			ext)
	}

	return hiddenEmail, nil
}

func TrimEmptyLines(s string) string {
	lines := strings.Split(s, "\n")

	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}

	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}

	return strings.Join(lines[start:end], "\n")
}

func ParseHTTPError(response *http.Response) error {
	body, _ := io.ReadAll(response.Body)
	errCode := fmt.Sprintf(httpErrorCodeFormat, response.StatusCode)
	msg := fmt.Sprintf(lang.I18n.T("cli.utils.error.server_error")+" %s: %s", errCode, body)
	return errors.New(msg)
}
