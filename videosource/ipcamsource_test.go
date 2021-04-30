// +build config

package videosource

import (
	"go/build"
	"io/ioutil"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"gocv.io/x/gocv"
	"gopkg.in/yaml.v2"
)

func TestIPCamSource(t *testing.T) {
	url := getURLFromYaml()
	if url == "" {
		t.Fatal("url empty")
		return
	}
	f := NewVideoReader(NewIPCamSource("test1", url), 30, 2)
	images := f.Start()
	defer f.Stop()

	go func() {
		tick := time.NewTicker(35 * time.Second)
	Loop:
		for {
			select {
			case <-tick.C:
				f.Stop()
				break Loop
			}
		}
		tick.Stop()
	}()

	window := gocv.NewWindow("Test Window")
	defer window.Close()
	for img := range images {
		mat := img.SharedMat.Mat
		window.IMShow(mat)
		window.WaitKey(5)
	}
	f.Wait() // should return immediately
	window.WaitKey(5000)
}

func getURLFromYaml() string {
	type urlYaml struct {
		URL string `yaml:"url"`
	}
	url := ""
	y := &urlYaml{}
	yamlFile, err := ioutil.ReadFile(build.Default.GOPATH + "/src/github.com/jonoton/scout/.config/ipcam-test.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, y)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	url = y.URL
	return url
}
