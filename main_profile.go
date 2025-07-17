//go:build profile
// +build profile

package main

// View all Profiles
//   Browser goto `http://localhost:6060/debug/pprof/`
//
// Profile Examples
//   Terminal
//     Tab1 Run `go run -tags profile github.com/jonoton/scout`
//     Tab2 Run `go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/profile?seconds=2`
//
//     Tab2 Run `go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/heap`
//     Tab2 Run `go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/goroutine`
//     Tab2 Run `go tool pprof -http localhost:8081 http://localhost:6060/debug/pprof/github.com/jonoton/go-sharedmat/sharedmat.counts`
//

import (
	baseHttp "net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jonoton/go-sharedmat"
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

	// HTTP for Profiling
	go func() {
		log.Println(baseHttp.ListenAndServe(":6060", nil))
	}()

	m := manage.NewManage()
	h := http.NewHttp(m)
	go func() {
		<-ctlc
		log.Println("Captured ctrl-c")
		m.Stop()
		h.Stop()
	}()
	m.Start()
	h.Listen()
	m.Wait()
	h.Wait()
	for i := 0; i < 20; i++ {
		time.Sleep(time.Second)
		log.Infoln("SharedMat Profile Count:", sharedmat.SharedMatProfile.Count())
	}
}
