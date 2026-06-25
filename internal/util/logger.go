package util

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// SetupLogger configures the global zerolog logger based on config:
// a colorized human-readable console in development, raw JSON in production.
func SetupLogger(config Config) {
	level, err := zerolog.ParseLevel(config.Log.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	var output io.Writer = os.Stderr
	if config.Environment != "production" {
		output = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	}

	log.Logger = zerolog.New(output).With().Timestamp().Logger()
}
