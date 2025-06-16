//go:build !profile
// +build !profile

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/jonoton/scout/http"
	"github.com/jonoton/scout/manage"
	log "github.com/sirupsen/logrus"
)

func doMain() {
	if len(os.Args) > 1 {
		log.Printf("How to run:\n\t%s NO ARGS\n", os.Args[0])
		return
	}
	ctlc := make(chan os.Signal, 1)
	signal.Notify(ctlc, os.Interrupt, syscall.SIGTERM)

	m := manage.NewManage()
	h := http.NewHttp(m)
	go func() {
		<-ctlc
		log.Println("Captured ctrl-c")
		h.Stop()
		m.Stop()
	}()
	m.Start()
	h.Listen()
	m.Wait()
}
