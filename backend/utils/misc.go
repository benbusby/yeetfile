package utils

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"yeetfile/shared/constants"
	"yeetfile/shared/endpoints"
)

// GetEnvVar is the primary method for reading variables from the environment.
// Note that variables are unset after they are retrieved, so the value needs
// to be stored in some way if it needs to be accessed more than once.
func GetEnvVar(key string, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}

	err := os.Unsetenv(key)
	if err != nil {
		log.Fatalf("Failed to unset %s key: %v\n", key, err)
	}

	return strings.TrimSpace(value)
}

// GetEnvVarBytesB64 retrieves a base64 string from the environment and returns
// the value as a []byte.
func GetEnvVarBytesB64(key string, fallback []byte) []byte {
	value := GetEnvVar(key, "")
	if value == "" {
		return fallback
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		log.Fatalf("Error decoding %s (this should be a base64 value)", key)
	}

	return decoded
}

// GetEnvVarInt retrieves a string value from the environment and converts it
// into an integer.
func GetEnvVarInt(key string, fallback int) int {
	value := GetEnvVar(key, strconv.Itoa(fallback))
	if value == "" {
		return fallback
	}

	num, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("WARNING: Value for %s is not a valid number, using fallback...\n", key)
		return fallback
	}

	return num
}

// GetEnvVarInt64 retrieves a string valkue from the environment and converts it
// into a 64-bit integer.
func GetEnvVarInt64(key string, fallback int64) int64 {
	value := GetEnvVar(key, strconv.FormatInt(fallback, 10))
	if value == "" {
		return fallback
	}

	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}

	return num
}

// GetEnvVarBool retrieves a value from the environment and interprets it as a
// bool value -- 0/n/false == false, 1/y/true == true
func GetEnvVarBool(key string, fallback bool) bool {
	value := GetEnvVar(key, "")
	value = strings.ToLower(value)

	if value == "" {
		return fallback
	} else if value == "0" || value == "n" || value == "false" {
		return false
	} else if value == "1" || value == "y" || value == "true" {
		return true
	}

	return fallback
}

func StrToDuration(str string, isDebug bool) time.Duration {
	unit := string(str[len(str)-1])
	length, _ := strconv.Atoi(str[:len(str)-1])

	if unit == "d" {
		return time.Duration(length) * time.Hour * 24
	} else if unit == "h" {
		return time.Duration(length) * time.Hour
	} else if unit == "m" {
		return time.Duration(length) * time.Minute
	} else if unit == "s" {
		if !isDebug {
			// N sec expiry is only available in debug mode
			return time.Minute
		}

		return time.Duration(length) * time.Second
	}

	return 0
}

func GenChecksum(data []byte) ([]byte, string) {
	h := sha1.New()
	h.Write(data)

	checksum := h.Sum(nil)
	return checksum, fmt.Sprintf("%x", checksum)
}

func LogStruct(v any) {
	s, _ := json.MarshalIndent(v, "", "\t")
	log.Println(string(s))
}

// DayDiff returns the number of days between two dates
func DayDiff(begin, end time.Time) int {
	duration := end.Sub(begin)
	days := int(duration.Hours() / 24)
	return days
}

func IsTLSReq(req *http.Request) bool {
	return req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https"
}

// IsStructMissingAnyField checks to see if any generic struct is missing a
// values in its string or array fields. Numeric fields are not checked since
// 0 is a valid field value.
func IsStructMissingAnyField(s interface{}) bool {
	val := reflect.ValueOf(s)
	for i := 0; i < val.Type().NumField(); i++ {
		switch val.Field(i).Type().Kind() {
		case reflect.String:
			fallthrough
		case reflect.Slice:
			if val.Field(i).Len() == 0 {
				return true
			}
			break
		}
	}

	return false
}

func IsAnyStringMissing(s ...string) bool {
	for _, str := range s {
		if len(str) == 0 {
			return true
		}
	}

	return false
}

func IsAnyByteSliceMissing(b ...[]byte) bool {
	for _, bs := range b {
		if len(bs) == 0 {
			return true
		}
	}

	return false
}

func CheckDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func ParseSizeString(str string) int64 {
	pattern := regexp.MustCompile(`^(\d+)([a-zA-Z]+)$`)
	matches := pattern.FindStringSubmatch(str)

	if len(matches) == 3 {
		numStr := matches[1]
		num, err := strconv.Atoi(numStr)
		if err != nil {
			log.Printf("Error converting number: %v\n", err)
			return 0
		}

		i64num := int64(num)
		letters := strings.ToUpper(matches[2])

		switch letters[0] {
		case 'T': // Terabyte
			return int64(1024) * 1024 * 1024 * 1024 * i64num
		case 'G': // Gigabyte
			return int64(1024) * 1024 * 1024 * i64num
		case 'M': // Megabyte
			return int64(1024) * 1024 * i64num
		case 'K': // Kilobyte
			return int64(1024) * i64num
		default:
			return i64num
		}
	} else {
		log.Printf("No match found for size string: %s\n", str)
	}

	return 0
}

// LimitedChunkReader reads the request body, limited to max chunk size + encryption
// overhead + 1024 bytes. This is big enough for all data-containing requests
// made to the YeetFile API.
func LimitedChunkReader(w http.ResponseWriter, body io.ReadCloser) ([]byte, error) {
	return limitedReader(w, body, constants.ChunkSize+constants.TotalOverhead+1024)
}

// LimitedReader reads the request body, limited to 4096 bytes. This is an arbitrary
// limit, but should always be more than big enough for all API requests.
func LimitedReader(w http.ResponseWriter, body io.ReadCloser) ([]byte, error) {
	return limitedReader(w, body, 4096)
}

func limitedReader(w http.ResponseWriter, body io.ReadCloser, limit int) ([]byte, error) {
	limitedBody := http.MaxBytesReader(w, body, int64(limit))
	return io.ReadAll(limitedBody)
}

func LimitedJSONReader(w http.ResponseWriter, body io.ReadCloser) *json.Decoder {
	return limitedJSONReader(w, body, 12288)
}

func limitedJSONReader(w http.ResponseWriter, body io.ReadCloser, limit int) *json.Decoder {
	limitedBody := http.MaxBytesReader(w, body, int64(limit))
	return json.NewDecoder(limitedBody)
}

func GetTrailingURLSegments(path string, strip ...endpoints.Endpoint) []string {
	if strings.HasSuffix(path, "/") {
		path = path[0 : len(path)-1]
	}

	for _, endpoint := range strip {
		endpointBase := strings.ReplaceAll(string(endpoint), "/*", "")
		path = strings.Replace(path, endpointBase, "", 1)
		if strings.HasSuffix(path, string(endpoint)) {
			// There is no trailing segment, it ends with the base endpoint
			return []string{}
		}
	}

	path = strings.TrimPrefix(path, "/")
	return strings.Split(path, "/")
}

func GetReqSource(req *http.Request) (string, error) {
	ip := req.Header.Get("X-Forwarded-For")

	if len(ip) == 0 {
		fallbackIP, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			return "", err
		}

		ip = fallbackIP
	}

	return ip, nil
}

// IsLocalUpload validates that the URL being used for an upload is a valid URL
func IsLocalUpload(uploadURL string) bool {
	_, err := url.ParseRequestURI(uploadURL)
	if err != nil {
		return true
	}

	return false
}
