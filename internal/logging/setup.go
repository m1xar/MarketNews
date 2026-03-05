package logging

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

const defaultLogFile = "logs/marketnews.log"

func Setup() (*os.File, error) {
	path := os.Getenv("APP_LOG_FILE")
	if path == "" {
		path = defaultLogFile
	}
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	log.SetOutput(io.MultiWriter(os.Stdout, f))
	log.SetFlags(log.LstdFlags | log.LUTC)
	return f, nil
}
