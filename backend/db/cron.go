package db

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"hash/fnv"
	"log"
	"time"
	"yeetfile/backend/config"
	"yeetfile/backend/service"
	"yeetfile/shared/constants"
)

const (
	ExpiryTask     = "expiry"
	LimiterTask    = "limiter"
	BandwidthTask  = "bandwidth"
	DownloadsTask  = "downloads"
	UpgradeTask    = "upgrade"
	UpgradeExpTask = "upgrade-expiration"
	B2AuthTask     = "b2-auth-task"
)

type CronTask struct {
	Name           string
	Interval       time.Duration
	IntervalAmount int
	Enabled        bool
	TaskFn         func()
}

// tasks: defines all background cron tasks in YeetFile. This includes:
// - an expiry task that handles expired content from YeetFile Send
// - a limiter task for preventing N requests within a specific time frame
// - a bandwidth task for resetting user bandwidth every N days
// - an upgrade monitoring task for instances with billing enabled
// - a downloads cleanup task that removes abandoned in-progress downloads
var tasks = []CronTask{
	{
		Name:           ExpiryTask,
		Interval:       time.Second,
		IntervalAmount: 15,
		Enabled:        true,
		TaskFn:         CheckExpiry,
	},
	{
		Name:           LimiterTask,
		Interval:       time.Second,
		IntervalAmount: config.YeetFileConfig.LimiterSeconds,
		Enabled:        true,
		TaskFn:         func() {}, // Set in InitCronTasks
	},
	{
		Name:           BandwidthTask,
		Interval:       time.Hour,
		IntervalAmount: constants.BandwidthMonitorDuration * 24,
		Enabled:        true,
		TaskFn:         CheckBandwidth,
	},
	{
		// Only enable if billing through BTCPay or Stripe is set up
		Name:           UpgradeTask,
		Interval:       time.Hour,
		IntervalAmount: 24,
		Enabled:        config.YeetFileConfig.BillingEnabled,
		TaskFn:         CheckActiveUpgrades,
	},
	{
		Name:           UpgradeExpTask,
		Interval:       time.Hour,
		IntervalAmount: 24,
		Enabled:        config.YeetFileConfig.BillingEnabled,
		TaskFn:         CheckUpgradeExpiration,
	},
	{
		Name:           DownloadsTask,
		Interval:       time.Hour,
		IntervalAmount: 1,
		Enabled:        true,
		TaskFn:         CleanUpDownloads,
	},
	{
		Name:           B2AuthTask,
		Interval:       time.Hour,
		IntervalAmount: 3,
		Enabled:        config.YeetFileConfig.StorageType == config.B2Storage,
		TaskFn:         service.AuthorizeB2,
	},
}

// getAdvisoryLockID returns a unique int64 value for the given cron task name
func (task CronTask) getAdvisoryLockID() int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(task.Name))
	return int64(hasher.Sum64())
}

func (task CronTask) getCronString() string {
	var intervalChar rune
	switch task.Interval {
	case time.Second:
		intervalChar = 's'
	case time.Minute:
		intervalChar = 'm'
	case time.Hour:
		intervalChar = 'h'
	default:
		log.Fatalf("Unsupported cron interval type: %s", task.Interval)
		return ""
	}

	return fmt.Sprintf("@every %d%c", task.IntervalAmount, intervalChar)

}

func (task CronTask) isLocked() bool {
	var lockedUntil time.Time

	s := `SELECT locked_until FROM cron WHERE task_name=$1`
	err := db.QueryRow(s, task.Name).Scan(&lockedUntil)
	if err != nil {
		log.Printf("Error checking locked_until for task '%s': %v\n", task.Name, err)
		return true
	}

	return lockedUntil.After(time.Now().UTC())
}

func (task CronTask) runCronTask() {
	if task.isLocked() {
		return
	}

	var lockAcquired bool
	lockID := task.getAdvisoryLockID()
	lockDuration := task.Interval * time.Duration(task.IntervalAmount)
	lockUntil := time.Now().UTC().Add(-time.Second).Add(lockDuration)

	err := db.QueryRow("SELECT pg_try_advisory_lock($1)", lockID).Scan(&lockAcquired)
	if err != nil {
		log.Printf("Error acquiring advisory lock: %v\n", err)
		return
	}

	if !lockAcquired {
		if config.IsDebugMode {
			log.Printf("'%s' task lock already acquired, skipping", task.Name)
		}
		return
	}

	if config.IsDebugMode {
		log.Printf("CRON: Running '%s' task...\n", task.Name)
	}

	// Run the task
	task.TaskFn()

	// Update cron table with the latest run and lock time
	s := `UPDATE cron SET locked_until=$1, last_run=$2 WHERE task_name=$3`
	_, err = db.Exec(s, lockUntil, time.Now().UTC(), task.Name)
	if err != nil {
		log.Printf("Error updating cron table lock time: %v\n", err)
	}

	_, err = db.Exec("SELECT pg_advisory_unlock($1)", lockID)
	if err != nil {
		log.Printf("Error releasing advisory lock: %v\n", err)
	} else if config.IsDebugMode {
		log.Printf("'%s' task completed at %v\n", task.Name, time.Now().Format(time.RFC1123))
	}
}

func InitCronTasks(limiterFn func()) {
	c := cron.New()

	for _, task := range tasks {
		// Ensure all tables already exist
		s := `INSERT INTO cron (task_name, locked_until, last_run) VALUES ($1, $2, $3)`
		_, _ = db.Exec(s, task.Name, time.Now().UTC(), time.Now().UTC())

		// Add all cron tasks
		if !task.Enabled {
			continue
		}

		// Workaround for calling the one task outside the db package
		if task.Name == LimiterTask {
			task.TaskFn = limiterFn
		}

		task.runCronTask()
		_, err := c.AddFunc(task.getCronString(), task.runCronTask)
		if err == nil {
			log.Printf("Added cron task '%s'\n", task.Name)
		} else {
			log.Printf("Error adding cron task: %v\n", err)
		}
	}

	c.Start()
}
