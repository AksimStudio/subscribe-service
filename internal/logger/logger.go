package logger

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func InitLogger(level string) {
	log = logrus.New()

	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.SetOutput(os.Stdout)
		log.Warnf("failed to open log file: %v; logs will be output only to stdout", err)
	} else {
		multiWriter := io.MultiWriter(os.Stdout, file)
		log.SetOutput(multiWriter)
	}

	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	log.SetLevel(logLevel)
}

func GetLogger() *logrus.Logger {
	if log == nil {
		InitLogger("info")
	}
	return log
}

func SetOutput(w io.Writer) {
	log.SetOutput(w)
}
