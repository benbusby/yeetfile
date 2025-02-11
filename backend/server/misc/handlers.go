package misc

import (
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/blake2b"
	"net/http"
	"strings"
	"time"
	"yeetfile/backend/config"
	"yeetfile/backend/server/session"
	"yeetfile/backend/static"
	"yeetfile/backend/utils"
	"yeetfile/shared"
	"yeetfile/shared/constants"
)

// UpHandler is used as the health check endpoint for load balancing, docker, etc.
func UpHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// InfoHandler returns information about the current instance
func InfoHandler(w http.ResponseWriter, _ *http.Request) {
	info := config.GetServerInfoStruct()
	_ = json.NewEncoder(w).Encode(info)
}

// FileHandler uses the embedded files from staticFiles to return a file
// resource based on its name
func FileHandler(strip string, prepend string, files embed.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		req.URL.Path = prepend + req.URL.Path
		req.URL.Path = strings.Replace(
			req.URL.Path,
			fmt.Sprintf("/%s", config.YeetFileConfig.Version), "", 1)
		path := strings.Split(req.URL.Path, "/")
		name := path[len(path)-1]

		queryIdx := strings.Index(name, "?")
		if queryIdx > 0 {
			name = name[:queryIdx]
		}

		w.Header().Set("Cache-Control", "private, max-age=604800")
		w.Header().Set("Expires", time.Now().Add(time.Hour*24*7).Format(http.TimeFormat))

		minFile, ok := static.MinifiedFiles[name]
		if ok {
			if strings.Contains(name, shared.DBFilename) {
				dbKey, canCache := generateJSDBKey(req)
				if !canCache {
					// Prevent caching non-authenticated db.js
					w.Header().Set("Cache-Control",
						"no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
					addr, _ := utils.GetReqSource(req)
					dbKey = blake2b.Sum256([]byte(addr))
				}

				minFile = []byte(strings.ReplaceAll(
					string(minFile),
					constants.JSSessionKey,
					hex.EncodeToString(dbKey[:])))
			}

			// Found minified file, return this instead
			if strings.HasSuffix(name, "js") {
				w.Header().Set("Content-Type", "application/javascript")
			} else if strings.HasSuffix(name, "css") {
				w.Header().Set("Content-Type", "text/css")
			}

			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")

			_, _ = w.Write(minFile)
			return
		}

		http.StripPrefix(strip, http.FileServer(http.FS(files))).ServeHTTP(w, req)
	}
}

// generateJSDBKey creates a deterministic hash using the current session key
// and session ID. This is used as a salt if the user's vault is password
// protected, otherwise its used as the key for encrypting the user's key pair.
// If the user doesn't have a session, the hash is generated from the source of
// the request and the value of config.YeetFileConfig.WebDBSecret.
func generateJSDBKey(req *http.Request) ([32]byte, bool) {
	canCache := false
	var dbKey [32]byte

	key, id, err := session.GetSessionKeyAndID(req)
	if err == nil && len(key) > 0 && len(id) > 0 {
		canCache = true
		dbKey = blake2b.Sum256([]byte(key + id))
	} else {
		// Prevent caching non-authenticated db.js
		addr, _ := utils.GetReqSource(req)
		dbKey = blake2b.Sum256(append(config.YeetFileConfig.FallbackWebSecret, []byte(addr)...))
	}

	return dbKey, canCache
}
