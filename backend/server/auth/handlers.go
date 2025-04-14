package auth

import (
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"strings"
	"yeetfile/backend/config"
	"yeetfile/backend/crypto"
	"yeetfile/backend/db"
	"yeetfile/backend/mail"
	"yeetfile/backend/server/session"
	"yeetfile/backend/utils"
	"yeetfile/shared"
	"yeetfile/shared/constants"
)

// LoginHandler handles a POST request to /login to log the user in.
func LoginHandler(w http.ResponseWriter, req *http.Request) {
	var login shared.Login
	if utils.LimitedJSONReader(w, req.Body).Decode(&login) != nil {
		log.Printf("Error decoding login request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID, err := ValidateCredentials(login.Identifier, login.LoginKeyHash, login.Code, true)
	if err != nil {
		if err == Missing2FAErr {
			log.Printf("Error: Missing TOTP")
			http.Error(w, "TOTP required", http.StatusForbidden)
			return
		} else if err == Failed2FAErr {
			log.Printf("Error: Incorrect TOTP")
			http.Error(w, "TOTP incorrect", http.StatusForbidden)
			return
		}

		http.Error(w, "User not found, or incorrect password", http.StatusNotFound)
		return
	}

	protectedKey, publicKey, err := db.GetUserKeys(userID)
	if err != nil {
		http.Error(w, "Error retrieving user keys", http.StatusInternalServerError)
		return
	}

	_ = session.SetSession(userID, w, req)
	_ = json.NewEncoder(w).Encode(shared.LoginResponse{
		PublicKey:    publicKey,
		ProtectedKey: protectedKey,
	})
}

// SignupHandler uses data from the incoming POST request to create a new user.
// The data received must match the shared.Signup struct.
func SignupHandler(w http.ResponseWriter, req *http.Request) {
	var signupData shared.Signup
	if utils.LimitedJSONReader(w, req.Body).Decode(&signupData) != nil {
		log.Printf("Unable to parse shared.Signup request\n")
		http.Error(w, "Unable to parse request", http.StatusBadRequest)
		return
	}

	// Check if server has a password
	if config.YeetFileConfig.PasswordHash != nil {
		err := bcrypt.CompareHashAndPassword(
			config.YeetFileConfig.PasswordHash,
			[]byte(signupData.ServerPassword))
		errMsg := "Missing or invalid server password"

		if err != nil && config.InvitesAllowed && len(signupData.Identifier) > 0 {
			// Check if the provided password is an invite code
			codeHash, dbErr := db.GetInviteCodeHash(signupData.Identifier)
			if dbErr == nil && len(codeHash) > 0 {
				err = bcrypt.CompareHashAndPassword(
					codeHash,
					[]byte(signupData.ServerPassword))
				errMsg = "Invalid invite code"
			}
		}

		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(errMsg))
			return
		} else if err != nil && config.InvitesAllowed {

		}
	} else {
		signupData.ServerPassword = "-"
	}

	var response shared.SignupResponse
	status := http.StatusOK

	if len(signupData.Identifier) == 0 {
		// No email, so this is an account ID only signup
		isCLI := req.UserAgent() == constants.CLIUserAgent
		id, captcha, err := SignupAccountIDOnly(isCLI)
		if err != nil {
			status = http.StatusBadRequest
			response = shared.SignupResponse{
				Error: "Error creating account ID",
			}
			log.Printf("Error: %v\n", err)
		} else {
			response = shared.SignupResponse{
				Identifier: id,
				Captcha:    captcha,
			}
		}
	} else {
		// Email signup
		if !config.YeetFileConfig.Email.Configured {
			http.Error(w, "Email signup not configured for this instance", http.StatusNotFound)
			return
		}

		err := SignupWithEmail(signupData)
		if err != nil && err != db.VerificationCodeExistsError {
			log.Printf("Error creating (email) account: %v\n", err)
			errMsg := "Error creating account"
			if err == db.UserAlreadyExists {
				errMsg = "User already exists"
			}
			status = http.StatusBadRequest
			response = shared.SignupResponse{
				Error: errMsg,
			}
		} else {
			response = shared.SignupResponse{
				Identifier: signupData.Identifier,
			}
		}
	}

	w.WriteHeader(status)
	if len(response.Error) > 0 {
		_, _ = w.Write([]byte(response.Error))
	} else {
		_ = json.NewEncoder(w).Encode(response)
	}
}

// AccountHandler handles fetching and returning the user's account information.
func AccountHandler(w http.ResponseWriter, req *http.Request, id string) {
	if !session.IsValidSession(w, req) {
		http.Redirect(w, req, "/login", http.StatusTemporaryRedirect)
		return
	}

	switch req.Method {
	case http.MethodDelete:
		var deleteAccount shared.DeleteAccount
		if utils.LimitedJSONReader(w, req.Body).Decode(&deleteAccount) != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("error decoding request"))
			return
		}

		err := DeleteUser(id, deleteAccount)
		if err != nil {
			http.Error(w, "Error deleting account", http.StatusBadRequest)
			return
		}

		_ = session.RemoveSession(w, req)
		w.WriteHeader(http.StatusOK)

		return
	case http.MethodGet:
		user, err := db.GetUserByID(id)
		if err != nil {
			log.Printf("Error fetching user by id: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		obscuredEmail, _ := shared.ObscureEmail(user.Email)
		_ = json.NewEncoder(w).Encode(shared.AccountResponse{
			Email:            obscuredEmail,
			PaymentID:        user.PaymentID,
			StorageAvailable: user.StorageAvailable,
			StorageUsed:      user.StorageUsed,
			SendAvailable:    user.SendAvailable,
			SendUsed:         user.SendUsed,
			UpgradeExp:       user.UpgradeExp,
			HasPasswordHint:  len(user.PasswordHint) > 0,
			Has2FA:           len(user.Secret) > 0,
		})
	}
}

// AccountUsageHandler handles authenticated GET requests to fetch the user's
// current vault and send usage
func AccountUsageHandler(w http.ResponseWriter, _ *http.Request, id string) {
	usage, err := db.GetUserUsage(id)
	if err != nil {
		log.Printf("Error fetching usage: %v\n", err)
		http.Error(w, "Error fetching usage", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(usage)
}

// VerifyEmailHandler handles account verification using the link sent to a user's
// email immediately after signup.
func VerifyEmailHandler(w http.ResponseWriter, req *http.Request) {
	var verifyEmail shared.VerifyEmail
	if utils.LimitedJSONReader(w, req.Body).Decode(&verifyEmail) != nil {
		http.Error(w, "Error decoding request", http.StatusBadRequest)
		return
	}

	// Ensure the request has the correct params for verification, otherwise
	// it should return the HTML verification page
	if len(verifyEmail.Email) == 0 || len(verifyEmail.Code) == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Verify user verification code and fetch password hash
	accountValues, err := db.VerifyUser(verifyEmail.Email, verifyEmail.Code)
	if err != nil {
		http.Error(w, "Incorrect verification code", http.StatusUnauthorized)
		return
	}

	var id string
	if len(accountValues.AccountID) == 0 {
		id, err = createNewUser(accountValues)
		if err != nil {
			http.Error(w, "Error creating account", http.StatusInternalServerError)
			return
		}
	} else {
		// User is verifying a new email, need to validate auth too
		if !session.IsValidSession(w, req) {
			http.Error(w, "You must be logged in", http.StatusUnauthorized)
			return
		}

		userID, err := session.GetSessionAndUserID(req)
		if err != nil || userID != accountValues.AccountID {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		err = updateUser(accountValues)
		if err != nil {
			http.Error(w, "Unable to verify email", http.StatusInternalServerError)
			return
		}

		err = session.InvalidateOtherSessions(w, req)
		if err != nil {
			log.Printf("Error invalidating user's other sessions")
		}
	}

	// Remove invite entry
	if config.InvitesAllowed {
		err = db.RemoveInvites([]string{verifyEmail.Email})
		if err != nil {
			log.Printf("Error removing invite: %v\n", err)
		}
	}

	// Remove verification entry
	_ = db.DeleteVerification(verifyEmail.Email)
	_ = session.SetSession(id, w, req)
}

// VerifyAccountHandler handles account verification using the CAPTCHA displayed
// to the user containing a multi-digit code.
func VerifyAccountHandler(w http.ResponseWriter, req *http.Request) {
	var verify shared.VerifyAccount
	err := utils.LimitedJSONReader(w, req.Body).Decode(&verify)
	if err != nil {
		log.Printf("Unable to parse VerifyAccount request: %v\n", err)
		http.Error(w, "Unable to parse request", http.StatusBadRequest)
		return
	} else if utils.IsStructMissingAnyField(verify) {
		log.Println("Missing required fields for verification")
		http.Error(w, "Unable to parse request", http.StatusBadRequest)
		return
	}

	// Verify user verification code
	_, err = db.VerifyUser(verify.ID, verify.Code)
	if err != nil {
		log.Printf("Error verifying user: %v\n", err)
		http.Error(w, "Incorrect verification code", http.StatusUnauthorized)
		return
	}

	hash, err := bcrypt.GenerateFromPassword(verify.LoginKeyHash, 8)
	if err != nil {
		log.Printf("Error generating bcrypt login hash: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	_, err = createNewUser(db.VerifiedAccountValues{
		AccountID:               verify.ID,
		Email:                   "",
		PasswordHash:            hash,
		ProtectedPrivateKey:     verify.ProtectedPrivateKey,
		PublicKey:               verify.PublicKey,
		ProtectedVaultFolderKey: verify.ProtectedVaultFolderKey,
	})

	if err != nil {
		log.Printf("Error creating user: %v\n", err)
		http.Error(w, "Error creating account", http.StatusInternalServerError)
		return
	}

	// Remove verification entry
	_ = db.DeleteVerification(verify.ID)
	_ = session.SetSession(verify.ID, w, req)
}

// LogoutHandler handles a PUT request to /logout to log the user out of their
// current session.
func LogoutHandler(w http.ResponseWriter, req *http.Request) {
	err := session.RemoveSession(w, req)
	if err != nil {
		log.Printf("Error logging out: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	http.Redirect(w, req, "/", http.StatusTemporaryRedirect)
}

// ForgotPasswordHandler handles a request for a user's password hint, if one
// has been set. If it has, an email will be sent to the user. If not, nothing
// is sent.
func ForgotPasswordHandler(w http.ResponseWriter, req *http.Request) {
	var forgot shared.ForgotPassword
	if utils.LimitedJSONReader(w, req.Body).Decode(&forgot) != nil {
		http.Error(w, "Unable to parse request", http.StatusBadRequest)
		return
	}

	canRequest, err := db.CanRequestPasswordHint(forgot.Email)
	if err != nil {
		log.Printf("Error checking forgot table: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	} else if !canRequest {
		w.WriteHeader(http.StatusOK)
		return
	}

	hint, err := db.GetUserPasswordHintByEmail(forgot.Email)
	if err != nil {
		log.Printf("Error fetching user pw hint: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if hint == nil || len(hint) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	decryptedHint, err := crypto.Decrypt(hint)
	if err != nil {
		log.Printf("Error decrypting user pw hint: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	err = mail.SendPasswordHintEmail(decryptedHint, forgot.Email)
	if err != nil {
		log.Printf("Error sending password hint email: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	err = db.AddForgotEntry(forgot.Email)
	if err != nil {
		log.Printf("Error adding forgot table entry: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
}

// PubKeyHandler handles requests for a YeetFile user's public key, which can
// be used for sharing files/folders with the user.
func PubKeyHandler(w http.ResponseWriter, req *http.Request, _ string) {
	userIdentifier := req.URL.Query().Get("user")

	var userID string
	var err error
	if len(userIdentifier) == 0 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	} else if strings.Contains(userIdentifier, "@") {
		userID, err = db.GetUserIDByEmail(userIdentifier)
	} else {
		userID = userIdentifier
		_, err = db.GetUserByID(userID)
	}

	if err != nil || len(userID) == 0 {
		log.Printf("Error in user lookup for pub key: %v\n", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	pubKey, err := db.GetUserPubKey(userID)
	if err != nil {
		log.Printf("Error fetching pub key: %v\n", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	jsonData, _ := json.Marshal(shared.PubKeyResponse{PublicKey: pubKey})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonData)
}

// ProtectedKeyHandler returns the user's protected key (private key encrypted
// with their user key). This is used when updating the protected key when
// a user changes their email or password.
func ProtectedKeyHandler(w http.ResponseWriter, _ *http.Request, id string) {
	protectedKey, _, err := db.GetUserKeys(id)
	if err != nil {
		log.Printf("Error fetching user keys: %v\n", err)
		http.Error(w, "Error fetching protected key", http.StatusInternalServerError)
		return
	}

	jsonData, _ := json.Marshal(shared.ProtectedKeyResponse{
		ProtectedKey: protectedKey,
	})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonData)
}

// ChangeEmailHandler validates the user's old login information, and uses the
// ChangeEmail request struct to send a verification email to their new email
// in preparation for updating their login key hash, encrypted protected key, etc
func ChangeEmailHandler(w http.ResponseWriter, req *http.Request, id string) {
	var fn session.HandlerFunc
	switch req.Method {
	case http.MethodPost:
		fn = startEmailChangeHandler
	case http.MethodPut:
		fn = finishEmailChangeHandler
	}

	fn(w, req, id)
}

func startEmailChangeHandler(w http.ResponseWriter, _ *http.Request, id string) {
	email, err := db.GetUserEmailByID(id)
	if err != nil {
		log.Printf("Error fetching user email: %v\n", err)
		http.Error(w, "Error fetching user email", http.StatusBadRequest)
		return
	} else if len(email) == 0 {
		// Account ID-only user is setting up an email
		changeID, err := db.NewChangeEmailEntry(id, "")
		if err != nil && err != db.ChangeEmailEntryTooNew {
			log.Printf("Error creating email change entry for account ID user: %v\n", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		jsonData, _ := json.Marshal(shared.StartEmailChangeResponse{
			ChangeID: changeID,
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonData)
		return
	}

	changeID, err := db.NewChangeEmailEntry(id, email)
	if err != nil && err != db.ChangeEmailEntryTooNew {
		log.Printf("Error creating new change email entry: %v\n", err)
		http.Error(w, "Error creating new change email entry", http.StatusInternalServerError)
		return
	} else if err == db.ChangeEmailEntryTooNew {
		log.Printf("Change email request is too new")
		w.WriteHeader(http.StatusOK)
		return
	}

	err = mail.SendEmailChangeNotification(email, changeID)
	if err != nil {
		log.Printf("Error sending email change notification: %v\n", err)
		http.Error(w, "Error sending email", http.StatusInternalServerError)
		return
	}

	jsonData, _ := json.Marshal(shared.StartEmailChangeResponse{
		ChangeID: "", // Change ID is sent to the user's current email
	})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonData)
}

func finishEmailChangeHandler(w http.ResponseWriter, req *http.Request, id string) {
	var changeEmail shared.ChangeEmail
	if utils.LimitedJSONReader(w, req.Body).Decode(&changeEmail) != nil {
		http.Error(w, "Unable to decode request", http.StatusBadRequest)
		return
	}

	pathSegments := strings.Split(req.URL.Path, "/")
	changeID := pathSegments[len(pathSegments)-1]
	if !db.IsChangeIDValid(changeID, id) {
		log.Printf("Change email ID is invalid")
		http.Error(w, "Invalid email change ID", http.StatusUnauthorized)
		return
	}

	fetchedID, err := db.GetUserIDByEmail(changeEmail.NewEmail)
	if err == nil && len(fetchedID) > 0 {
		http.Error(w, "An account with this email already exists", http.StatusBadRequest)
		return
	}

	userID, err := ValidateCredentials(id, changeEmail.OldLoginKeyHash, "", false)
	if err != nil || id != userID {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		return
	}

	bcryptHash, err := bcrypt.GenerateFromPassword(changeEmail.NewLoginKeyHash, 8)
	if err != nil {
		log.Printf("Error generating bcrypt hash: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	code, err := db.NewVerification(shared.Signup{
		Identifier:          changeEmail.NewEmail,
		ProtectedPrivateKey: changeEmail.ProtectedKey,
	}, bcryptHash, userID)
	if err != nil {
		log.Printf("Error creating email verification entry: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	err = mail.SendVerificationEmail(code, changeEmail.NewEmail)
	if err != nil {
		log.Printf("Error sending verification email: %v\n", err)
		http.Error(w, "SMTP error", http.StatusInternalServerError)
		return
	}

	_ = db.RemoveEmailChangeByChangeID(changeID)
}

// ChangePasswordHandler validates the user's old login information, and uses
// the ChangePassword request struct to update their login info and protected
// key with new values.
func ChangePasswordHandler(w http.ResponseWriter, req *http.Request, id string) {
	var changePassword shared.ChangePassword
	if utils.LimitedJSONReader(w, req.Body).Decode(&changePassword) != nil {
		http.Error(w, "Unable to decode request", http.StatusBadRequest)
		return
	}

	userID, err := ValidateCredentials(id, changePassword.OldLoginKeyHash, "", false)
	if err != nil || id != userID {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		return
	}

	bcryptHash, err := bcrypt.GenerateFromPassword(
		changePassword.NewLoginKeyHash, 8)
	if err != nil {
		log.Printf("Error generating bcrypt hash: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	err = db.UpdateUserLogin(id, bcryptHash, changePassword.ProtectedKey)
	if err != nil {
		log.Printf("Error updating user login credentials: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
}

// ChangeHintHandler handles a plaintext hint sent to the server, which is
// encrypted and stored in the user's db entry.
func ChangeHintHandler(w http.ResponseWriter, req *http.Request, id string) {
	var changeHint shared.ChangePasswordHint
	if utils.LimitedJSONReader(w, req.Body).Decode(&changeHint) != nil {
		http.Error(w, "Unable to decode request", http.StatusBadRequest)
		return
	}

	if len(changeHint.Hint) > constants.MaxHintLen {
		http.Error(w, "Hint is too long", http.StatusBadRequest)
		return
	}

	var encHint []byte
	var err error
	if len(changeHint.Hint) == 0 {
		encHint = nil
	} else {
		encHint, err = crypto.Encrypt(changeHint.Hint)
		if err != nil {
			log.Printf("Error encrypting hint: %v\n", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
	}

	err = db.UpdatePasswordHint(id, encHint)
	if err != nil {
		log.Printf("Error updating pw hint: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func TwoFactorHandler(w http.ResponseWriter, req *http.Request, userID string) {
	switch req.Method {
	case http.MethodGet:
		newTOTP, err := generateUserTotp(userID)
		if err != nil {
			log.Printf("Error generating 2FA: %v\n", err)
			http.Error(w, "Error generating 2FA", http.StatusBadRequest)
			return
		}

		err = json.NewEncoder(w).Encode(newTOTP)
		if err != nil {
			http.Error(w, "Error sending response", http.StatusInternalServerError)
		}
	case http.MethodPost:
		var totp shared.SetTOTP
		if err := utils.LimitedJSONReader(w, req.Body).Decode(&totp); err != nil {
			http.Error(w, "Error decoding request body", http.StatusBadRequest)
			return
		}

		response, err := setTOTP(userID, totp)
		if err != nil {
			log.Printf("Failed to set totp: %v\n", err)
			http.Error(w, "Failed to set totp", http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			http.Error(w, "Error sending response", http.StatusInternalServerError)
		}
	case http.MethodDelete:
		code := req.URL.Query().Get("code")
		if len(code) == 0 {
			http.Error(w, "Missing TOTP code", http.StatusBadRequest)
			return
		}

		err := removeTOTP(userID, code)
		if err != nil {
			http.Error(w, "Invalid TOTP code", http.StatusUnauthorized)
			return
		}
	}
}

// RecyclePaymentIDHandler handles replacing the user's current payment ID with
// a new value
func RecyclePaymentIDHandler(w http.ResponseWriter, _ *http.Request, userID string) {
	paymentID, err := db.GetPaymentIDByUserID(userID)
	if err != nil {
		log.Println("Error fetching user payment ID", err)
		http.Error(w, "Error fetching user", http.StatusBadRequest)
		return
	}

	err = db.RecycleUserPaymentID(paymentID)
	if err != nil {
		log.Println("Error recycling payment ID", err)
		http.Error(w, "Error recycling payment ID", http.StatusBadRequest)
		return
	}
}
