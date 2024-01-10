// package main start the service
package main

import (
	"log/slog"

	"github.com/joho/godotenv"
	prscd "github.com/pilarjs/prscd"
)

func main() {
	slog.Info("loading .env file...")
	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file")
	}

	prscd.StartServer()
}
