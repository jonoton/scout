package main

import (
	"os"

	"github.com/jonoton/scout/http"
	"github.com/jonoton/scout/manage"
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
	if len(os.Args) > 1 {
		log.Printf("How to run:\n\t%s NO ARGS\n", os.Args[0])
		return
	}

	m := manage.NewManage()
	m.Start()
	h := http.NewHttp(m)
	h.Listen()
	m.Wait()
}
