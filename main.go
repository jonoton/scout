package main

import (
	"os"

	"github.com/jonoton/go-runtime"
	"github.com/jonoton/scout/http"
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

func handleArgs() bool {
	if len(os.Args) > 1 {
		if os.Args[1] == "--secure-http-passwords" {
			cfgPath := runtime.GetRuntimeDirectory(".config") + http.ConfigFilename
			conf := http.NewConfig(cfgPath)
			if conf == nil {
				log.Fatalf("Required config file %s not found.", cfgPath)
			}
			err := conf.SecureConfig(cfgPath)
			if err != nil {
				log.Fatalf("Failed to secure config: %v", err)
			}
			return true
		}
		log.Printf("How to run:\n\t%s\n\t%s --secure-http-passwords\n", os.Args[0], os.Args[0])
		return true
	}
	return false
}

func main() {
	if handleArgs() {
		return
	}
	doMain()
}
