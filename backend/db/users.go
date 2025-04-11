package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"log"
	"strings"
	"time"
	"yeetfile/backend/config"
	"yeetfile/backend/mail"
	"yeetfile/backend/server/upgrades"
	"yeetfile/shared"
	"yeetfile/shared/constants"
)

type User struct {
	ID                  string
	Email               string
	PasswordHash        []byte
	ProtectedPrivateKey []byte
	PublicKey           []byte
	PasswordHint        []byte
	Secret              []byte
	PaymentID           string
	UpgradeExp          time.Time
	StorageAvailable    int64
	StorageUsed         int64
	SendAvailable       int64
	SendUsed            int64
}

type UserStorage struct {
	StorageAvailable int64
	StorageUsed      int64
}

type UserSend struct {
	SendAvailable int64
	SendUsed      int64
}

const unsubscribedMonth = -1

var defaultExp time.Time

var UserAlreadyExists = errors.New("user already exists")
var UserLimitReached = errors.New("user limit has been reached")
var UserSendExceeded = errors.New("user exceeded monthly send limit")
var UserStorageExceeded = errors.New("user exceeded storage limit")

// NewUser creates a new user in the "users" table, ensuring that the email
// provided is not already in use.
func NewUser(user User) (string, error) {
	if config.YeetFileConfig.MaxUserCount > 0 {
		count, err := GetUserCount()
		if err != nil {
			return "", err
		} else if count == config.YeetFileConfig.MaxUserCount {
			return "", UserLimitReached
		}

		config.YeetFileConfig.CurrentUserCount = count
	}

	if len(user.Email) > 0 {
		rows, err := db.Query(`SELECT * from users WHERE email = $1`, user.Email)
		if err != nil {
			return "", err
		} else if rows.Next() {
			return "", UserAlreadyExists
		}

		rows.Close()
	}

	if len(user.ID) == 0 {
		user.ID = CreateUniqueUserID()
	} else {
		if UserIDExists(user.ID) {
			return "", UserAlreadyExists
		}
	}

	paymentID := CreateUniquePaymentID()

	s := `INSERT INTO users (
                   id,
                   email,
                   pw_hash,
                   pw_hint,
                   payment_id,
                   send_available,
                   storage_available,
                   upgrade_exp,
                   last_upgraded_month,
                   protected_key,
                   public_key,
                   bandwidth)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := db.Exec(
		s,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.PasswordHint,
		paymentID,
		config.YeetFileConfig.DefaultUserSend,
		config.YeetFileConfig.DefaultUserStorage,
		defaultExp,
		unsubscribedMonth,
		user.ProtectedPrivateKey,
		user.PublicKey,
		config.YeetFileConfig.DefaultUserStorage*
			constants.TotalBandwidthMultiplier*
			constants.BandwidthMonitorDuration)
	if err != nil {
		return "", err
	}

	if config.YeetFileConfig.MaxUserCount > 0 {
		config.YeetFileConfig.CurrentUserCount += 1
	}

	return user.ID, nil
}

// UpdateUser is used to update user values that can change as a result of a
// user changing their email or password.
func UpdateUser(user User, accountID string) error {
	s := `UPDATE users 
	      SET email=$1, pw_hash=$2, protected_key=$3
	      WHERE id=$4`
	_, err := db.Exec(
		s,
		user.Email,
		user.PasswordHash,
		user.ProtectedPrivateKey,
		accountID)
	return err
}

// GetUserCount returns the total number of users in the table
func GetUserCount() (int, error) {
	rows, err := db.Query(`SELECT COUNT(*) from users`)
	if err != nil || !rows.Next() {
		return 0, err
	}

	defer rows.Close()

	var count int
	err = rows.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CreateUniqueUserID creates a 16 digit user ID that is not already being used
// in the user database.
func CreateUniqueUserID() string {
	id := shared.GenRandomNumbers(16)
	for UserIDExists(id) {
		id = shared.GenRandomNumbers(16)
	}

	return id
}

// CreateUniquePaymentID creates a 16 character payment ID that is not already
// being used in the user database.
func CreateUniquePaymentID() string {
	paymentID := shared.GenRandomString(16)
	for PaymentIDExists(paymentID) {
		paymentID = shared.GenRandomString(16)
	}

	return paymentID
}

// GetUserPassCount returns the number of items in the vault are associated
// with the user and are flagged with `is_pw`.
func GetUserPassCount(id string) (int, int, error) {
	var count int
	maxPassCount := -1
	if config.YeetFileConfig.BillingEnabled && config.YeetFileConfig.DefaultMaxPasswords > 0 {
		maxPassCount = config.YeetFileConfig.DefaultMaxPasswords
	}

	s := `SELECT 
	    COUNT(
	        CASE 
	            WHEN v.pw_data IS NOT NULL AND LENGTH(v.pw_data) > 0 THEN 1 
	            END
	        ) AS pw_count,
	    CASE 
		WHEN u.upgrade_exp < CURRENT_DATE THEN $2
		ELSE 0 
	    END AS exp_status
	FROM vault v
	JOIN users u ON v.owner_id = u.id
	WHERE v.owner_id = $1 GROUP BY u.upgrade_exp`
	err := db.QueryRow(s, id, maxPassCount).Scan(&count, &maxPassCount)
	if err == sql.ErrNoRows {
		return 0, maxPassCount, nil
	}

	return count, maxPassCount, err
}

// GetUserStorage returns UserStorage and UserSend struct containing the user's
// available and used limits for storing and sending files
func GetUserStorage(id string) (UserStorage, UserSend, error) {
	rows, err := db.Query(`
	    SELECT storage_available, storage_used, send_available, send_used 
	    FROM users 
	    WHERE id = $1`, id)
	if err != nil {
		return UserStorage{}, UserSend{}, err
	} else if !rows.Next() {
		errorStr := fmt.Sprintf("unable to find user with id '%s'", id)
		return UserStorage{}, UserSend{}, errors.New(errorStr)
	}

	defer rows.Close()

	var (
		storageAvailable int64
		storageUsed      int64

		sendAvailable int64
		sendUsed      int64
	)

	err = rows.Scan(&storageAvailable, &storageUsed, &sendAvailable, &sendUsed)
	if err != nil {
		return UserStorage{}, UserSend{}, err
	}

	return UserStorage{
			StorageAvailable: storageAvailable, StorageUsed: storageUsed,
		},
		UserSend{
			SendAvailable: sendAvailable, SendUsed: sendUsed,
		},
		nil
}

// UpdateStorageUsed updates the amount of storage used by the user. Can be a
// negative number to remove storage space.
func UpdateStorageUsed(userID string, amount int64) error {
	var storageUsed int
	var storageAvailable int
	s := `UPDATE users 
	      SET storage_used = CASE 
	                           WHEN storage_used + $1 < 0 THEN 0
	                           ELSE storage_used + $1
	                         END
	      WHERE id=$2 AND storage_available > 0
	      RETURNING storage_used, storage_available`
	err := db.QueryRow(s, amount, userID).Scan(&storageUsed, &storageAvailable)
	if err != nil && err != sql.ErrNoRows {
		return err
	} else if storageUsed > storageAvailable && amount > 0 && config.YeetFileConfig.DefaultUserStorage > 0 {
		return UserStorageExceeded
	}

	return nil
}

// UpdateBandwidth subtracts bandwidth from the user's bandwidth column, returning
// an error if the value goes below 0.
func UpdateBandwidth(userID string, amount int64) error {
	// Skip db update if send limits aren't configured
	if config.YeetFileConfig.DefaultUserStorage < 0 {
		return nil
	}

	s := `UPDATE users SET bandwidth=bandwidth-$2 WHERE id=$1`
	_, err := db.Exec(s, userID, amount)
	return err
}

func UpdatePasswordHint(userID string, encHint []byte) error {
	s := `UPDATE users SET pw_hint=$2 WHERE id=$1 AND email IS NOT NULL AND email != ''`
	_, err := db.Exec(s, userID, encHint)
	return err
}

// RecycleUserPaymentID overwrites the user's previous payment ID. This can be
// performed whenever a user wants.
func RecycleUserPaymentID(paymentID string) error {
	rows, err := db.Query(`SELECT id from users WHERE payment_id = $1`, paymentID)
	if err != nil {
		return err
	} else if !rows.Next() {
		errorStr := fmt.Sprintf("unable to find user with payment id '%s'", paymentID)
		return errors.New(errorStr)
	}

	defer rows.Close()

	newID := shared.GenRandomString(16)
	for PaymentIDExists(newID) {
		newID = shared.GenRandomString(16)
	}

	// Read in account ID for the user
	var accountID string
	err = rows.Scan(&accountID)

	// Replace payment ID
	s := `UPDATE users
	      SET payment_id=$1
	      WHERE id=$2`

	_, err = db.Exec(s, newID, accountID)
	if err != nil {
		return err
	}

	return nil
}

// UserIDExists checks the users table to see if the provided id is already
// being used for another user.
func UserIDExists(id string) bool {
	rows, err := db.Query(`SELECT id FROM users WHERE id = $1`, id)
	if err != nil {
		log.Fatalf("Error querying user id: %v", err)
		return true
	}

	defer rows.Close()

	// If any rows are returned, the id exists
	if rows.Next() {
		return true
	}

	return false
}

// PaymentIDExists checks the user table to see if the provided payment ID
// (for Stripe + BTCPay) already exists for another user.
func PaymentIDExists(paymentID string) bool {
	rows, err := db.Query(`SELECT * FROM users WHERE payment_id = $1`, paymentID)
	if err != nil {
		log.Fatalf("Error querying user payment id: %v", err)
		return true
	}

	defer rows.Close()

	// If any rows are returned, the id exists
	if rows.Next() {
		return true
	}

	return false
}

// GetUserPasswordHashByEmail retrieves the password hash and the encrypted
// secret for a user with the provided email address.
func GetUserPasswordHashByEmail(email string) ([]byte, []byte, error) {
	var pwHash []byte
	var secret []byte
	err := db.QueryRow(`
		SELECT pw_hash, secret
		FROM users 
		WHERE email = $1`, email).Scan(&pwHash, &secret)
	if err != nil {
		log.Printf("Error querying for user by email: %v", err)
		return nil, nil, err
	}

	return pwHash, secret, nil
}

// GetUserPasswordHashByID retrieves the password hash and the encrypted secret
// for a user with the provided ID.
func GetUserPasswordHashByID(id string) ([]byte, []byte, error) {
	var pwHash []byte
	var secret []byte
	err := db.QueryRow(`
		SELECT pw_hash, secret
		FROM users 
		WHERE id = $1`, id).Scan(&pwHash, &secret)
	if err != nil {
		log.Printf("Error querying for user by id: %v", err)
		return nil, nil, err
	}

	return pwHash, secret, nil
}

// GetUserKeys retrieves the user's public key and their private key, the latter
// is encrypted with their user key (which is generated client side and never stored)
func GetUserKeys(id string) ([]byte, []byte, error) {
	rows, err := db.Query(`
		SELECT protected_key, public_key
		FROM users 
		WHERE id = $1`, id)
	if err != nil {
		log.Printf("Error querying for user by id: %v", err)
		return nil, nil, err
	}

	defer rows.Close()
	if rows.Next() {
		var protectedKey []byte
		var publicKey []byte
		err = rows.Scan(&protectedKey, &publicKey)
		if err != nil {
			return nil, nil, err
		}

		return protectedKey, publicKey, nil
	}

	return nil, nil, errors.New("unable to find user")
}

// GetUserByID retrieves a User struct for given user ID.
func GetUserByID(id string) (User, error) {
	var (
		email            string
		paymentID        string
		expiration       time.Time
		sendAvailable    int64
		sendUsed         int64
		storageAvailable int64
		storageUsed      int64
		pwHint           []byte
		secret           []byte
	)
	s := `SELECT email, payment_id, upgrade_exp,
	             send_available, send_used, 
		     storage_available, storage_used,
		     pw_hint, secret
	      FROM users
	      WHERE id = $1`
	err := db.QueryRow(s, id).Scan(
		&email, &paymentID, &expiration,
		&sendAvailable, &sendUsed,
		&storageAvailable, &storageUsed,
		&pwHint, &secret,
	)

	if err != nil {
		log.Printf("Error querying for user by id: %s\n", id)
		return User{}, err
	}

	return User{
		ID:               id,
		Email:            email,
		PaymentID:        paymentID,
		UpgradeExp:       expiration,
		SendAvailable:    sendAvailable,
		SendUsed:         sendUsed,
		StorageAvailable: storageAvailable,
		StorageUsed:      storageUsed,
		PasswordHint:     pwHint,
		Secret:           secret,
	}, nil
}

func GetUserSecret(userID string) ([]byte, error) {
	var secret []byte
	s := `SELECT secret FROM users WHERE id=$1`
	err := db.QueryRow(s, userID).Scan(&secret)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return secret, err
}

func SetUserSecret(userID string, secret []byte) error {
	s := `UPDATE users SET secret=$2 WHERE id=$1`
	_, err := db.Exec(s, userID, secret)
	return err
}

func RemoveUser2FA(userID string) error {
	s := `UPDATE users 
	      SET secret='\x'::bytea, recovery_hashes='{}'::bytea[]
	      WHERE id=$1`
	_, err := db.Exec(s, userID)
	return err
}

func GetUserRecoveryCodeHashes(userID string) ([]string, error) {
	var hashes []string
	s := `SELECT recovery_hashes FROM users WHERE id=$1`
	err := db.QueryRow(s, userID).Scan(pq.Array(&hashes))

	return hashes, err
}

func SetUserRecoveryCodeHashes(userID string, hashes []string) error {
	s := `UPDATE users SET recovery_hashes=$2 WHERE id=$1`
	_, err := db.Exec(s, userID, pq.Array(hashes))
	return err
}

func GetUserPubKey(userID string) ([]byte, error) {
	rows, err := db.Query(`SELECT public_key FROM users WHERE id=$1`, userID)
	if err != nil {
		log.Printf("Error querying for public key by user id: %v\n", err)
		return nil, err
	}

	defer rows.Close()
	if rows.Next() {
		var publicKey []byte
		err = rows.Scan(&publicKey)
		if err != nil {
			return nil, err
		}

		return publicKey, nil
	}

	return nil, errors.New("user public key not found")
}

func GetUserPublicName(userID string) (string, error) {
	var email string
	err := db.QueryRow(`SELECT email FROM users WHERE id=$1`, userID).Scan(&email)
	if err != nil {
		log.Printf("Error querying for user's public name")
		return "", err
	}

	if len(email) == 0 {
		idTail := userID[len(userID)-4:]
		return fmt.Sprintf("*%s", idTail), nil
	}

	return email, nil
}

// GetUserIDByEmail returns a user's ID given their email address.
func GetUserIDByEmail(email string) (string, error) {
	var id string
	err := db.QueryRow(`
		SELECT id
		FROM users 
		WHERE email = $1`, email).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error querying for user by email: %v", err)
		return "", err
	}

	return id, nil
}

func GetUserSubByPaymentID(paymentID string) (string, time.Time, error) {
	var (
		subType string
		subExp  time.Time
	)

	err := db.QueryRow(`
	        SELECT upgrade_tag, upgrade_exp 
	        FROM users 
	        WHERE payment_id=$1`, paymentID).
		Scan(&subType, &subExp)

	if err != nil {
		return "", time.Time{}, err
	}

	return subType, subExp, nil
}

func GetUserPasswordHintByEmail(email string) ([]byte, error) {
	var hint []byte
	err := db.QueryRow(`SELECT pw_hint FROM users WHERE email = $1`, email).
		Scan(&hint)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return hint, nil
}

// GetUserUsage fetches the storage used/available and send used/available for
// the user matching the provided ID.
func GetUserUsage(id string) (shared.UsageResponse, error) {
	var (
		storageUsed      int64
		storageAvailable int64
		sendUsed         int64
		sendAvailable    int64
	)

	s := `SELECT storage_used, storage_available, send_used, send_available
	      FROM users
	      WHERE id = $1`
	err := db.QueryRow(s, id).Scan(
		&storageUsed,
		&storageAvailable,
		&sendUsed,
		&sendAvailable)
	if err != nil {
		return shared.UsageResponse{}, err
	}

	return shared.UsageResponse{
		StorageAvailable: storageAvailable,
		StorageUsed:      storageUsed,
		SendAvailable:    sendAvailable,
		SendUsed:         sendUsed,
	}, nil
}

// GetUserSendLimits returns the amount of used and available bytes for
// sending files
func GetUserSendLimits(id string) (int64, int64, error) {
	var sendUsed int64
	var sendAvailable int64
	err := db.QueryRow(`
		SELECT send_used, send_available
		FROM users
		WHERE id = $1`, id).Scan(&sendUsed, &sendAvailable)
	if err == sql.ErrNoRows {
		return 0, 0, errors.New("unable to find user by id")
	} else if err != nil {
		log.Printf("Error querying for user by id: %s\n", id)
		return 0, 0, err
	}

	return sendUsed, sendAvailable, nil
}

// GetUserStorageLimits returns the amount of used and available bytes for
// storing files
func GetUserStorageLimits(id string) (int64, int64, error) {
	var storageUsed int64
	var storageAvailable int64
	err := db.QueryRow(`
		SELECT storage_used, storage_available
		FROM users
		WHERE id = $1`, id).Scan(&storageUsed, &storageAvailable)
	if err == sql.ErrNoRows {
		return 0, 0, errors.New("unable to find user by id")
	} else if err != nil {
		log.Printf("Error querying for user by id: %s\n", id)
		return 0, 0, err
	}

	return storageUsed, storageAvailable, nil
}

// GetUserBandwidth returns the user's bandwidth, which can be used to determine
// if a file download can be performed.
func GetUserBandwidth(id string) (int64, error) {
	var bandwidth int64
	s := `SELECT bandwidth FROM users WHERE id=$1`
	err := db.QueryRow(s, id).Scan(&bandwidth)
	if err != nil {
		return 0, err
	}

	return bandwidth, nil
}

func GetPaymentIDByUserID(userID string) (string, error) {
	rows, err := db.Query(`
		SELECT payment_id
		FROM users
		WHERE id = $1`, userID)
	if err != nil {
		log.Println("Error querying for payment_id")
		return "", err
	}

	defer rows.Close()
	if rows.Next() {
		var paymentID string
		err = rows.Scan(&paymentID)
		if err != nil {
			log.Println("Error fetching payment ID")
			return "", err
		}

		return paymentID, nil
	}

	return "", errors.New("unable to find payment id by user id")
}

func GetUserEmailByID(userID string) (string, error) {
	var email string
	s := `SELECT email FROM users WHERE id=$1`
	err := db.QueryRow(s, userID).Scan(&email)
	return email, err
}

func GetUserSessionKey(userID string) (string, error) {
	var sessionKey string
	s := `SELECT session_key FROM users WHERE id=$1`
	err := db.QueryRow(s, userID).Scan(&sessionKey)
	return sessionKey, err
}

func SetUserSessionKey(userID, sessionKey string) error {
	s := `UPDATE users SET session_key=$2 WHERE id=$1`
	_, err := db.Exec(s, userID, sessionKey)
	return err
}

func GetUserEmailByPaymentID(paymentID string) (string, error) {
	rows, err := db.Query(`
		SELECT email
		FROM users
		WHERE payment_id = $1`, paymentID)
	if err != nil {
		log.Printf("Error querying for user by payment_id: %s\n", paymentID)
		return "", err
	}

	defer rows.Close()
	if rows.Next() {
		var email string
		err = rows.Scan(&email)
		if err != nil {
			log.Printf("Error fetching email for user with payment id %s\n", paymentID)
			return "", err
		}

		return email, nil
	}

	return "", errors.New("unable to find user by payment id")
}

// SetUserVaultUpgrade updates a user's vault to have an updated amount
// of storage available
func SetUserVaultUpgrade(
	paymentID string,
	upgradeTag string,
	exp time.Time,
	storage int64,
) error {
	totalWeeklyBandwidth := storage *
		constants.TotalBandwidthMultiplier *
		constants.BandwidthMonitorDuration

	s := `UPDATE users
              SET upgrade_exp=$1,
                  storage_available=$2,
                  upgrade_tag=$3,
                  last_upgraded_month=$4, bandwidth=$5
              WHERE payment_id=$6`

	_, err := db.Exec(s,
		exp,
		storage,
		upgradeTag,
		int(time.Now().Month()),
		totalWeeklyBandwidth,
		paymentID)
	if err != nil {
		return err
	}

	return nil
}

// SetUserSendUpgrade updates a user's available send
func SetUserSendUpgrade(paymentID string, sendUpgradeBytes int64) error {
	s := `UPDATE users
              SET send_available = send_available - send_used + $1,
                  send_used = 0
              WHERE payment_id=$2`

	_, err := db.Exec(s,
		sendUpgradeBytes,
		paymentID)
	if err != nil {
		return err
	}

	return nil
}

// UpdateUserSendUsed adds an amount of bytes (size) to a user's send_used
// given their user ID.
func UpdateUserSendUsed(id string, size int) error {
	s := `UPDATE users 
	      SET send_used = CASE 
	                           WHEN send_used + $1 < 0 THEN 0
	                           ELSE send_used + $1
	                         END
	      WHERE id=$2 AND send_available > 0
	      RETURNING send_used, send_available`

	var sendUsed int
	var sendAvailable int
	err := db.QueryRow(s, size, id).Scan(&sendUsed, &sendAvailable)
	if err != nil && err != sql.ErrNoRows {
		return err
	} else if sendUsed > sendAvailable {
		return UserSendExceeded
	}

	return nil
}

func CheckBandwidth() {
	bandwidthUpdate := `UPDATE users SET bandwidth = storage_available * $1 * $2;`
	_, err := db.Exec(
		bandwidthUpdate,
		constants.TotalBandwidthMultiplier,
		constants.BandwidthMonitorDuration)
	if err != nil {
		log.Printf("Failed to update user bandwidths")
	}
}

// CheckActiveUpgrades inspects each user's upgraded account and updates their
// available transfer if their upgrade is still valid
func CheckActiveUpgrades() {
	s := `SELECT id, upgrade_tag, upgrade_exp FROM users
              WHERE last_upgraded_month != $1 AND last_upgraded_month > 0`
	rows, err := db.Query(s, int(time.Now().Month()))
	if err != nil {
		log.Printf("Error retrieving user upgrades: %v", err)
		return
	}

	var revertIDs []string
	now := time.Now()

	// Upgrade map matches upgrade tags to user IDs
	upgradeMap := make(map[string][]string)

	defer rows.Close()
	for rows.Next() {
		var id string
		var upgradeTag string
		var upgradeExp time.Time

		err = rows.Scan(&id, &upgradeTag, &upgradeExp)

		if err != nil {
			log.Printf("Error scanning user rows: %v", err)
			return
		}

		if upgradeExp.Before(time.Now()) {
			// User doesn't have an active upgrade, set send to
			// default amount
			revertIDs = append(revertIDs, id)
			continue
		} else if now.Day() == upgradeExp.Day() || ExpDateRollover(now, upgradeExp) {
			// User has an active upgrade
			upgradeMap[upgradeTag] = append(upgradeMap[upgradeTag], id)
		}
	}

	updateFunc := func(ids []string, storageAvailable int64, month int) error {
		if ids == nil || len(ids) == 0 {
			return nil
		}

		idStr := fmt.Sprintf("{%s}", strings.Join(ids, ","))

		u := `UPDATE users
		      SET storage_available=$1,
		          last_upgraded_month=$2
		      WHERE id=ANY($3)`
		_, err := db.Exec(u, storageAvailable, month, idStr)
		return err
	}

	err = updateFunc(
		revertIDs,
		config.YeetFileConfig.DefaultUserStorage,
		unsubscribedMonth)
	if err != nil {
		log.Printf("Error resetting unpaid user storage/send")
	}

	for upgradeTag, ids := range upgradeMap {
		vaultUpgrade, err := upgrades.GetUpgradeByTag(
			upgradeTag,
			upgrades.GetAllUpgrades())
		if err != nil {
			log.Printf("Error locating upgrade in cron: %v\n", err)
			continue
		}

		err = updateFunc(ids, vaultUpgrade.Bytes, int(now.Month()))
		if err != nil {
			log.Printf("Error updating user storage/send: %v\n", err)
		}
	}
}

// CheckUpgradeExpiration checks for users who are a week away from having their
// purchased upgrade expire.
func CheckUpgradeExpiration() {
	s := `SELECT email, upgrade_exp
	      FROM users
	      WHERE email != ''
	        AND upgrade_exp < current_date + interval '8' day
	        AND upgrade_exp > current_date;`

	rows, err := db.Query(s)
	if err != nil {
		log.Printf("Error retrieving upcoming user upgrade expirations: %v", err)
		return
	}

	var notifyEmails []string

	defer rows.Close()
	for rows.Next() {
		var (
			email      string
			upgradeExp time.Time
		)

		err = rows.Scan(&email, &upgradeExp)
		if err != nil {
			log.Printf("Error reading rows in user upgrade expirations: %v\n", err)
			return
		}

		oneWeekExp := time.Now().UTC().AddDate(0, 0, 7)
		if len(email) == 0 ||
			oneWeekExp.Day() != upgradeExp.Day() ||
			oneWeekExp.Month() != upgradeExp.Month() ||
			oneWeekExp.Year() != upgradeExp.Year() {
			continue
		}

		notifyEmails = append(notifyEmails, email)
	}

	if len(notifyEmails) == 0 {
		return
	}

	err = mail.SendUpgradeExpirationEmail(notifyEmails)
	if err != nil {
		log.Printf("Error sending upgrade expiration emails: %v\n", err)
	}
}

// ExpDateRollover checks to see if the user's upgrade expiration date takes
// place on a day that doesn't exist in other months. If so, the user's transfer
// limit should be upgraded "early". For example:
//
// - Expiration: Dec 31
// - Today: June 30
//
// In this scenario, the upgrade should be upgraded today. The 31st will
// never occur in June, but the following day would be a new month.
func ExpDateRollover(now time.Time, exp time.Time) bool {
	if exp.Day() <= 28 {
		// Skip check, the expiration date is within the bounds of all
		// monthly days
		return false
	}

	return exp.Day() > now.Day() && now.AddDate(0, 0, 1).Month() > now.Month()
}

func DeleteUser(id string) error {
	s := `DELETE FROM users WHERE id=$1`
	_, err := db.Exec(s, id)
	if err != nil {
		return err
	}

	return nil
}

func UpdateUserLogin(id string, loginKeyHash, protectedKey []byte) error {
	s := `UPDATE users
          SET pw_hash=$2, protected_key=$3
          WHERE id=$1`

	_, err := db.Exec(s, id, loginKeyHash, protectedKey)
	if err != nil {
		return err
	}

	return nil
}

// OverrideUserStorage sets the user's storage_available to the specified amount,
// and sets their last_upgraded_month to -1 to prevent this value from being
// overwritten by the subscription cron task.
func OverrideUserStorage(userID string, storage int64) error {
	s := `UPDATE users SET storage_available=$2, last_upgraded_month=$3 WHERE id=$1`
	_, err := db.Exec(s, userID, storage, unsubscribedMonth)
	if err != nil {
		return err
	}

	return nil
}

// OverrideUserSend sets the user's send_available to the specified amount.
func OverrideUserSend(userID string, send int64) error {
	s := `UPDATE users SET send_available=$2 WHERE id=$1`
	_, err := db.Exec(s, userID, send)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	var err error
	defaultExp, err = time.Parse(time.RFC1123, time.RFC1123)
	if err != nil {
		panic(err)
	}
}
