package main

import (
	"log/slog"
	"os"

	"github.com/pilarjs/prscd/chirp"
)

func init() {
	chirp.AuthUserAndGetYoMoCredential = func(publicKey string) (appID, credential string, ok bool) {
		slog.Info("Node| auth_user", "publicKey", publicKey)
		return "YOMO_APP", os.Getenv("YOMO_CREDENTIAL"), true
	}
}
