package notify

import (
	"crypto/tls"

	log "github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"
)

// Notify Constants
const (
	ATT     = "att"
	TMOBILE = "tmobile"
	VERIZON = "verizon"
	SPRINT  = "sprint"
)

// Phone contains phone information
type Phone struct {
	Number   string
	Provider string
}

// NewPhone creates a new Phone
func NewPhone(number string, provider string) *Phone {
	p := &Phone{
		Number:   number,
		Provider: provider,
	}
	return p
}

// Notify can send emails to email or phone number recipients
type Notify struct {
	dialer *gomail.Dialer
}

// NewNotify creates a new Notify
func NewNotify(host string, port int, user string, password string) *Notify {
	n := &Notify{
		dialer: gomail.NewDialer(host, port, user, password),
	}
	n.dialer.TLSConfig = &tls.Config{ServerName: host}
	return n
}

// SendEmail sends an email with attachments
func (n *Notify) SendEmail(to []string, subject string, body string, filenamesAttach []string, filenamesEmbed []string) {
	m := gomail.NewMessage()
	m.SetHeader("From", n.dialer.Username)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	for i := range filenamesAttach {
		m.Attach(filenamesAttach[i])
	}
	for i := range filenamesEmbed {
		m.Embed(filenamesEmbed[i])
	}
	if err := n.dialer.DialAndSend(m); err != nil {
		log.Errorln(err)
	}
}

// SendText sends a text message with attachments
// Note: text messages tend to be slower as the target service provider decides
//    when to relay messages from email to mobile devices.
func (n *Notify) SendText(phones []Phone, subject string, body string, filenames []string) {
	to := []string{}
	for i := range phones {
		phone := phones[i]
		email := getTextEmail(phone, len(filenames) > 0)
		to = append(to, email)
	}
	n.SendEmail(to, subject, body, filenames, make([]string, 0))
}

func getTextEmail(phone Phone, attachments bool) string {
	email := phone.Number
	var (
		smsExtAtt     = "@txt.att.net"
		mmsExtAtt     = "@mms.att.net"
		smsExtTmobile = "@tmomail.net"
		mmsExtTmobile = "@tmomail.net"
		smsExtVerizon = "@vtext.com"
		mmsExtVerizon = "@vzwpix.com"
		smsExtSprint  = "@pm.sprint.com"
		mmsExtSprint  = "@pm.sprint.com"
	)
	if phone.Provider == ATT {
		if attachments {
			email += mmsExtAtt
		} else {
			email += smsExtAtt
		}
	} else if phone.Provider == TMOBILE {
		if attachments {
			email += mmsExtTmobile
		} else {
			email += smsExtTmobile
		}
	} else if phone.Provider == VERIZON {
		if attachments {
			email += mmsExtVerizon
		} else {
			email += smsExtVerizon
		}
	} else if phone.Provider == SPRINT {
		if attachments {
			email += mmsExtSprint
		} else {
			email += smsExtSprint
		}
	}
	return email
}
