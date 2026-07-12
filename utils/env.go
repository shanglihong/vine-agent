package utils

import (
	"os"
	"strings"
)

func IsDev() bool {
	env := strings.ToLower(os.Getenv("APP_ENV"))
	return env == "dev"
}
