package db

import (
	"database/sql"
	"embed"
	_ "embed"
	"fmt"
	_ "github.com/lib/pq"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"yeetfile/backend/utils"
)

var db *sql.DB

//go:embed scripts/migrations/*.sql
var migrationScripts embed.FS

const migrationDir = "scripts/migrations"

func init() {
	var (
		host     = utils.GetEnvVar("YEETFILE_DB_HOST", "localhost")
		port     = utils.GetEnvVar("YEETFILE_DB_PORT", "5432")
		user     = utils.GetEnvVar("YEETFILE_DB_USER", "postgres")
		password = utils.GetEnvVar("YEETFILE_DB_PASS", "")
		dbname   = utils.GetEnvVar("YEETFILE_DB_NAME", "yeetfile")
		cert     = utils.GetEnvVar("YEETFILE_DB_CERT", "")
	)

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		user,
		password,
		host,
		port,
		dbname)

	if len(cert) > 0 {
		cert = strings.ReplaceAll(cert, "\\n", "\n")

		certFile, err := os.CreateTemp("", ".*")
		if err != nil {
			log.Fatalln("Error creating tmp dir for db cert:", err)
		}

		if _, err = certFile.WriteString(cert); err != nil {
			log.Fatalf("Unable to write tmp CA cert file: %v\n", err)
		}

		if err = certFile.Close(); err != nil {
			log.Fatalf("Unable to close tmp CA cert file: %v\n", err)
		}

		connStr += fmt.Sprintf(
			"?sslmode=verify-full&sslrootcert=%s",
			certFile.Name())
	} else {
		connStr += "?sslmode=disable"
	}

	// Open db connection
	var err error
	db, err = sql.Open("postgres", connStr)

	ping := db.Ping()
	if err != nil || ping != nil {
		log.Fatalf("Unable to connect to database!\n"+
			"Error: %v\nPing: %v\n", err, ping)
	}

	version, err := getMigrationVersion()
	if err != nil {
		version = -1
	}

	// Init db contents from migration scripts
	dir, err := migrationScripts.ReadDir(migrationDir)
	if err != nil {
		log.Fatal(err)
	}

	sort.Slice(dir, func(i, j int) bool {
		iName := dir[i].Name()
		jName := dir[j].Name()
		return getScriptVersion(iName) < getScriptVersion(jName)
	})

	for _, file := range dir {
		scriptVersion := getScriptVersion(file.Name())
		if scriptVersion <= version {
			continue
		}

		log.Println("Running script:", file.Name())

		fullPath := fmt.Sprintf("%s/%s", migrationDir, file.Name())
		script, err := migrationScripts.Open(fullPath)
		if err != nil {
			log.Fatal(err)
		}

		scriptBytes, err := io.ReadAll(script)
		_, err = db.Exec(string(scriptBytes))
		if err != nil {
			log.Fatal(err)
		}

		_ = script.Close()

		err = setMigrationVersion(scriptVersion)
		if err != nil {
			log.Fatal(err)
		}

		version = scriptVersion
	}
}

func getScriptVersion(name string) int {
	scriptVersionStr := strings.Split(name, "_")[0]
	scriptVersion, _ := strconv.Atoi(scriptVersionStr)
	return scriptVersion
}

// ClearDatabase removes all instances of a file ID from all tables in the database
func ClearDatabase(id string) {
	if DeleteMetadata(id) {
		log.Printf("%s metadata deleted\n", id)
	} else {
		log.Printf("Failed to delete metadata for %s\n", id)
	}

	if DeleteUploads(id) {
		log.Printf("%s B2 info deleted\n", id)
	} else {
		log.Printf("Failed to delete B2 info for %s\n", id)
	}

	if DeleteExpiry(id) {
		log.Printf("%s expiry fields deleted\n", id)
	} else {
		log.Printf("Failed to delete expiry fields for %s\n", id)
	}
}

// TableIDExists checks any table to see if the `id` column already has a value
// matching the provided id param
func TableIDExists(tableName, id string) bool {
	rows, err := db.Query(`SELECT * FROM `+tableName+` WHERE id=$1`, id)
	if err != nil {
		log.Printf("Error checking for id in table '%s': %v", tableName, err)
		return true
	}

	// If any rows are returned, the id exists
	defer rows.Close()
	if rows.Next() {
		return true
	}

	return false
}

func Close() {
	log.Println("Closing DB connection")
	err := db.Close()
	if err != nil {
		panic(err)
	}
}
