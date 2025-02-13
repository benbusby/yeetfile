package db

import (
	"database/sql"
	"time"
)

func InitCronTask(task string, lockedUntil, lastRun time.Time) {
	s := `INSERT INTO cron (task_name, locked_until, last_run) VALUES ($1, $2, $3)`
	_, _ = db.Exec(s, task, lockedUntil, lastRun)
}

func GetCronLockedUntil(task string) (time.Time, error) {
	var lockedUntil time.Time

	s := `SELECT locked_until FROM cron WHERE task_name::text=$1`
	err := db.QueryRow(s, task).Scan(&lockedUntil)
	return lockedUntil, err
}

func AcquireCronTaskLock(lockID int64) (bool, error) {
	var lockAcquired bool
	err := db.QueryRow("SELECT pg_try_advisory_lock($1)", lockID).Scan(&lockAcquired)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	} else if err == sql.ErrNoRows {
		return true, nil
	}

	return lockAcquired, nil
}

func UpdateCronTaskLockDetails(lockUntil, lastRun time.Time, task string) error {
	s := `UPDATE cron SET locked_until=$1, last_run=$2 WHERE task_name=$3`
	_, err := db.Exec(s, lockUntil, time.Now().UTC(), task)
	if err != nil {
		return err
	}

	return nil
}

func ReleaseCronTaskLock(lockID int64) error {
	_, err := db.Exec("SELECT pg_advisory_unlock($1)", lockID)
	return err
}
