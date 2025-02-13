package main

import (
	_ "github.com/joho/godotenv/autoload"
	"yeetfile/backend/cron"
	"yeetfile/backend/db"
	"yeetfile/backend/server"
	"yeetfile/backend/utils"
)

func main() {
	defer db.Close()
	cron.InitCronTasks(server.ManageLimiters)

	host := utils.GetEnvVar("YEETFILE_HOST", "0.0.0.0")
	port := utils.GetEnvVar("YEETFILE_PORT", "8090")

	server.Run(host, port)
}
