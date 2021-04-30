package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func init() {
	formatter := &log.TextFormatter{}
	formatter.TimestampFormat = "01-02-2006 15:04:05"
	formatter.FullTimestamp = true
	log.SetFormatter(formatter)
	log.SetOutput(os.Stdout)
	// log.SetReportCaller(true)

	// Only log the warning severity or above.
	// log.SetLevel(log.WarnLevel)
}

func main() {
	doMain()
}
