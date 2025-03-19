package session

import (
	"errors"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
	"strings"
	"yeetfile/backend/config"
	"yeetfile/backend/db"
	"yeetfile/backend/utils"
	"yeetfile/shared"
	"yeetfile/shared/constants"
)

type HandlerFunc func(w http.ResponseWriter, req *http.Request, userID string)

var (
	authKey = utils.GetEnvVarBytesB64(
		"YEETFILE_SESSION_AUTH_KEY",
		securecookie.GenerateRandomKey(32))
	encKey = utils.GetEnvVarBytesB64(
		"YEETFILE_SESSION_ENC_KEY",
		securecookie.GenerateRandomKey(32))
	store = sessions.NewCookieStore(authKey, encKey)
)

const UserIDKey = "user"
const UserSessionKey = "session"
const UserSessionIDKey = "session_id"

func GetSession(req *http.Request) (*sessions.Session, error) {
	return store.Get(req, constants.AuthSessionStore)
}

func SetSession(id string, w http.ResponseWriter, req *http.Request) error {
	session, _ := GetSession(req)

	sessionKey, err := db.GetUserSessionKey(id)
	if err != nil {
		return err
	} else if len(sessionKey) == 0 {
		sessionKey = shared.GenRandomString(16)
		err = db.SetUserSessionKey(id, sessionKey)
		if err != nil {
			return err
		}
	}

	session.Values[UserIDKey] = id
	session.Values[UserSessionKey] = sessionKey
	session.Values[UserSessionIDKey] = shared.GenRandomNumbers(32)
	session.Options.SameSite = http.SameSiteStrictMode
	session.Options.HttpOnly = true
	if req.TLS != nil {
		session.Options.Secure = true
	}

	return session.Save(req, w)
}

func HasSession(req *http.Request) bool {
	session, err := GetSession(req)
	if err != nil {
		return false
	}

	id, found := session.Values[UserIDKey].(string)
	if !found || len(id) == 0 {
		return false
	}

	return true
}

func GetSessionKeyAndID(req *http.Request) (string, string, error) {
	session, err := GetSession(req)
	if err != nil {
		return "", "", err
	}

	sessionKey, found := session.Values[UserSessionKey].(string)
	if !found {
		return "", "", errors.New("session key not found")
	}

	sessionID, found := session.Values[UserSessionIDKey].(string)
	if !found {
		return "", "", errors.New("session id not found")
	}

	return sessionKey, sessionID, nil
}

func IsValidSession(w http.ResponseWriter, req *http.Request) bool {
	session, err := GetSession(req)
	if err != nil {
		return false
	}

	id, found := session.Values[UserIDKey].(string)
	if !found || len(id) == 0 {
		return false
	}

	sessionKey, found := session.Values[UserSessionKey].(string)
	if !found || len(sessionKey) == 0 {
		return false
	}

	dbKey, err := db.GetUserSessionKey(id)
	if err != nil {
		log.Printf("Error checking for user session key: %v\n", err)
		return false
	} else if sessionKey != dbKey {
		log.Println("Session key", sessionKey, "does not match db key ", dbKey)
		_ = RemoveSession(w, req)
		return false
	}

	return true
}

func InvalidateOtherSessions(w http.ResponseWriter, req *http.Request) error {
	session, _ := GetSession(req)

	id, found := session.Values[UserIDKey].(string)
	if !found || len(id) == 0 {
		return errors.New("current session isn't valid")
	}

	newSessionKey := shared.GenRandomString(16)
	err := db.SetUserSessionKey(id, newSessionKey)
	if err != nil {
		return err
	}

	session.Values[UserSessionKey] = newSessionKey
	return session.Save(req, w)
}

func RemoveSession(w http.ResponseWriter, req *http.Request) error {
	session, err := GetSession(req)

	if err == nil {
		session.Options.MaxAge = -1
		session.Values[UserSessionIDKey] = ""
		session.Values[UserSessionKey] = ""
		session.Values[UserIDKey] = ""
		return session.Save(req, w)
	}

	return nil
}

func GetSessionUserID(session *sessions.Session) string {
	sessionVal := session.Values[UserIDKey]
	if sessionVal != nil {
		return sessionVal.(string)
	}

	return ""
}

func GetSessionAndUserID(req *http.Request) (string, error) {
	s, err := GetSession(req)
	if err != nil {
		return "", errors.New("invalid session")
	}

	id := GetSessionUserID(s)
	if len(id) == 0 {
		return "", errors.New("invalid session")
	}

	return id, nil
}

func init() {
	store.Options.HttpOnly = true
	store.Options.SameSite = http.SameSiteStrictMode
	if strings.HasPrefix(config.YeetFileConfig.Domain, "https") {
		store.Options.Secure = true
	}
}
