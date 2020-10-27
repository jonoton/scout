// +build config

package notify

import (
	"go/build"
	"testing"
)

func TestEmail(t *testing.T) {
	sender := NewSenderConfig(build.Default.GOPATH + "/src/github.com/jonoton/scout/.config/" + SenderConfigFilename)
	rx := NewRxConfig(build.Default.GOPATH + "/src/github.com/jonoton/scout/.config/" + RxConfigFilename)
	n := NewNotify(sender.Host, sender.Port, sender.User, sender.Password)
	images := []string{}
	n.SendEmail(rx.Email, "test subject", "test body", images, images)
}

func TestText(t *testing.T) {
	sender := NewSenderConfig(build.Default.GOPATH + "/src/github.com/jonoton/scout/.config/" + SenderConfigFilename)
	rx := NewRxConfig(build.Default.GOPATH + "/src/github.com/jonoton/scout/.config/" + RxConfigFilename)
	n := NewNotify(sender.Host, sender.Port, sender.User, sender.Password)
	images := []string{}
	phones := rx.GetPhones()
	n.SendText(phones, "test subject", "test body", images)
}
