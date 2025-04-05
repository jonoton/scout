package monitor

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jonoton/go-dir"
	"github.com/jonoton/go-notify"
	"github.com/jonoton/go-runtime"
	"github.com/jonoton/go-videosource"
)

// Alert Constants
const (
	MaxTextFileSize = 300000 // 300 KB

	NewLine = "<br>"
)

// AlertTimes for alerts
type AlertTimes struct {
	Object time.Time
	Person time.Time
	Face   time.Time
}

// Alert buffers ProcessedImages and sends notifications
type Alert struct {
	name          string
	notifier      *notify.Notify
	notifyRxConf  *notify.RxConfig
	saveDirectory string
	alertConf     *AlertConfig
	ringBuffer    videosource.RingBufferProcessedImage
	intervalTick  *time.Ticker
	hourTick      *time.Ticker
	hourSent      int
	done          chan bool
	cancel        chan bool
	LastAlert     AlertTimes
}

// NewAlert creates a new Alert
func NewAlert(name string, notifier *notify.Notify, notifyRxConf *notify.RxConfig, saveDirectory string, alertConf *AlertConfig) *Alert {
	if saveDirectory == "" || alertConf == nil {
		return nil
	}
	alertDir := filepath.Clean(saveDirectory+"/alerts") + string(filepath.Separator)
	os.MkdirAll(alertDir, os.ModePerm)

	a := &Alert{
		name:          name,
		notifier:      notifier,
		notifyRxConf:  notifyRxConf,
		saveDirectory: alertDir,
		alertConf:     alertConf,
		ringBuffer:    *videosource.NewRingBufferProcessedImage(alertConf.MaxImagesPerInterval),
		intervalTick:  time.NewTicker(time.Duration(alertConf.IntervalMinutes) * time.Minute),
		hourTick:      time.NewTicker(time.Hour),
		hourSent:      0,
		done:          make(chan bool),
		cancel:        make(chan bool),
		LastAlert:     AlertTimes{},
	}
	a.ringBuffer.IsSortByContent = true
	return a
}

// Wait until done
func (a *Alert) Wait() {
	<-a.done
}

// Start the processes
func (a *Alert) Start() {
	go func() {
	Loop:
		for {
			select {
			case <-a.cancel:
				a.prune()
				a.doAlerts()
				break Loop
			case <-a.hourTick.C:
				a.hourSent = 0
				a.prune()
			case <-a.intervalTick.C:
				a.doAlerts()
			}
		}
		a.intervalTick.Stop()
		a.hourTick.Stop()
		close(a.done)
	}()
}

// Push a processed image to buffer
func (a *Alert) Push(img videosource.ProcessedImage) {
	if img.HasObject() {
		popped := a.ringBuffer.Push(*img.Ref())
		popped.Cleanup()
	}
	img.Cleanup()
}

// Stop the processes
func (a *Alert) Stop() {
	close(a.cancel)
	<-a.done
	for a.ringBuffer.Len() > 0 {
		cur := a.ringBuffer.Pop()
		cur.Cleanup()
	}
}

func (a *Alert) prune() {
	a.deleteOldAlerts()
	a.deleteWhenFull()
}

func (a *Alert) doAlerts() {
	poppedList := a.ringBuffer.PopAll()
	sort.Sort(videosource.ProcessedImageByCreatedTime(poppedList))
	nowTime := time.Now()
	nowTimeStr := getFormattedKitchenTimestamp(nowTime)

	a.setLastAlerts(poppedList)
	imageInfos := a.saveAlerts(poppedList)
	a.sendAlerts(imageInfos, nowTimeStr)
}

func (a *Alert) deleteOldAlerts() {
	expiredFiles, _ := dir.Expired(a.saveDirectory, dir.RegexBeginsWith(a.name),
		time.Now(), time.Duration(a.alertConf.DeleteAfterHours)*time.Hour)
	for _, fileInfo := range expiredFiles {
		fullPath := filepath.Clean(a.saveDirectory + string(filepath.Separator) + fileInfo.Name())
		err := os.Remove(fullPath)
		if err != nil {
			log.Errorln(err)
		}
	}
}

func (a *Alert) deleteWhenFull() {
	dirSize, _ := dir.Size(a.saveDirectory, dir.RegexBeginsWith(a.name))
	if int(math.Ceil(dir.BytesToGigaBytes(dirSize))) > a.alertConf.DeleteAfterGB {
		files, _ := dir.List(a.saveDirectory, dir.RegexBeginsWith(a.name))
		sort.Sort(dir.AscendingTime(files))
		for _, fileInfo := range files {
			if int(math.Ceil(dir.BytesToGigaBytes(dirSize))) <= a.alertConf.DeleteAfterGB {
				break
			}
			dirSize -= uint64(fileInfo.Size())
			fullPath := filepath.Clean(a.saveDirectory + string(filepath.Separator) + fileInfo.Name())
			err := os.Remove(fullPath)
			if err != nil {
				log.Errorln(err)
			}
		}
	}
}

func hasPersonObject(objects []videosource.ObjectInfo) (found bool) {
	for _, obj := range objects {
		if strings.ToLower(obj.Description) == "person" {
			found = true
			break
		}
	}
	return
}

func getFormattedKitchenTimestamp(t time.Time) string {
	return t.Format("03:04:05 PM 01-02-2006")
}

func (a *Alert) setLastAlerts(poppedList []videosource.ProcessedImage) {
	for _, curPop := range poppedList {
		createdTime := curPop.Original.CreatedTime
		if curPop.HasObject() {
			if hasPersonObject(curPop.Objects) {
				if createdTime.After(a.LastAlert.Person) {
					a.LastAlert.Person = createdTime
				}
			} else {
				if createdTime.After(a.LastAlert.Object) {
					a.LastAlert.Object = createdTime
				}
			}
		}
		if curPop.HasFace() {
			if createdTime.After(a.LastAlert.Face) {
				a.LastAlert.Face = createdTime
			}
		}
	}
}

func (a *Alert) saveAlerts(poppedList []videosource.ProcessedImage) (result []imageInfo) {
	if len(poppedList) == 0 {
		return
	}

	result = make([]imageInfo, 0)

	for index, curPop := range poppedList {
		imageInfo := imageInfo{
			Name: fmt.Sprintf("Image %d", index+1),
			Time: getFormattedKitchenTimestamp(curPop.Original.CreatedTime),
		}
		infos := make([]attachedInfo, 0)
		if a.alertConf.SaveOriginal && curPop.Original.IsFilled() {
			title := "Original"
			percentage := ""
			videosource.SavePreview(curPop.Original, curPop.Original.CreatedTime, a.saveDirectory, a.name, title, percentage)
			s := videosource.SaveImage(curPop.Original, curPop.Original.CreatedTime, a.saveDirectory, a.alertConf.SaveQuality, a.name, title, percentage)
			info := attachedInfo{
				Title:      title,
				Percentage: percentage,
				Filename:   filepath.Base(s),
				FullPath:   s,
			}
			infos = append(infos, info)
		}
		if a.alertConf.SaveHighlighted && curPop.HasObject() {
			title := "Highlighted"
			percentage := ""
			highlighted := curPop.HighlightedAll()
			videosource.SavePreview(*highlighted, curPop.Original.CreatedTime, a.saveDirectory, a.name, title, percentage)
			s := videosource.SaveImage(*highlighted, curPop.Original.CreatedTime, a.saveDirectory, a.alertConf.SaveQuality, a.name, title, percentage)
			highlighted.Cleanup()
			info := attachedInfo{
				Title:      title,
				Percentage: percentage,
				Filename:   filepath.Base(s),
				FullPath:   s,
			}
			infos = append(infos, info)
		}
		for i, cur := range curPop.Objects {
			if i < a.alertConf.SaveObjectsCount {
				title := cur.Description
				percentage := fmt.Sprintf("%d", cur.Percentage)
				object := curPop.Object(i)
				videosource.SavePreview(*object, curPop.Original.CreatedTime, a.saveDirectory, a.name, title, percentage)
				s := videosource.SaveImage(*object, curPop.Original.CreatedTime, a.saveDirectory, 100, a.name, title, percentage)
				object.Cleanup()
				info := attachedInfo{
					Title:      title,
					Percentage: fmt.Sprintf("%d%%", cur.Percentage),
					Filename:   filepath.Base(s),
					FullPath:   s,
				}
				infos = append(infos, info)
			}
		}
		for i, cur := range curPop.Faces {
			if i < a.alertConf.SaveFacesCount {
				title := "Face"
				percentage := fmt.Sprintf("%d", cur.Percentage)
				face := curPop.Face(i)
				videosource.SavePreview(*face, curPop.Original.CreatedTime, a.saveDirectory, a.name, title, percentage)
				s := videosource.SaveImage(*face, curPop.Original.CreatedTime, a.saveDirectory, 100, a.name, title, percentage)
				face.Cleanup()
				info := attachedInfo{
					Title:      title,
					Percentage: fmt.Sprintf("%d%%", cur.Percentage),
					Filename:   filepath.Base(s),
					FullPath:   s,
				}
				infos = append(infos, info)
			}
		}
		imageInfo.AttachedInfo = infos
		result = append(result, imageInfo)
		curPop.Cleanup()
	}

	return
}

func getText(monitorName string, alertTime string, imageInfos []imageInfo) string {
	title := "Scout Alert " + monitorName
	txtBody := fmt.Sprintf("%s%s - %s", NewLine, title, alertTime)
	txtBody += fmt.Sprintf("%sTotal Images: %d", NewLine, len(imageInfos))

	for index, imageInfo := range imageInfos {
		imgNum := index + 1
		if imgNum > 1 {
			txtBody += NewLine
		}
		txtBody += fmt.Sprintf("%s%s", NewLine, imageInfo.Name)
		txtBody += fmt.Sprintf("%sCreated Time: %s", NewLine, imageInfo.Time)

		for _, attached := range imageInfo.AttachedInfo {
			if len(attached.Percentage) > 0 {
				txtBody += fmt.Sprintf("%s%s - %s", NewLine, attached.Title, attached.Percentage)
			} else {
				txtBody += fmt.Sprintf("%s%s", NewLine, attached.Title)
			}
		}
	}
	return txtBody
}

func (a *Alert) sendAlerts(imageInfos []imageInfo, alertTime string) {
	if a.notifier == nil || a.notifyRxConf == nil {
		return
	}
	sendAttachments := a.hourSent < a.alertConf.MaxSendAttachmentsPerHour
	emails := a.notifyRxConf.Email
	phones := a.notifyRxConf.GetPhones()
	hasEmails := len(emails) > 0
	hasPhones := len(phones) > 0
	if len(imageInfos) > 0 && (hasEmails || hasPhones) {
		attachments := make([]string, 0)
		title := "Scout Alert " + a.name
		html := getHTML(a.name, alertTime, imageInfos, sendAttachments)
		txtBody := getText(a.name, alertTime, imageInfos)

		// get all attachments paths
		if sendAttachments {
			for _, imageInfo := range imageInfos {
				for _, attachedInfo := range imageInfo.AttachedInfo {
					attachments = append(attachments, attachedInfo.FullPath)
				}
			}
		}

		if hasEmails {
			attachments = append([]string{runtime.GetRuntimeDirectory("http") + "public/images/hawk-wing.png"}, attachments...)
			a.notifier.SendEmail(emails, title, html, make([]string, 0), attachments)
		}
		if hasPhones {
			a.notifier.SendText(phones, title, txtBody, []string{})
			if a.alertConf.TextAttachments {
				for _, cur := range attachments {
					if fileInfo, err := os.Stat(cur); err == nil && fileInfo.Size() <= MaxTextFileSize {
						a.notifier.SendText(phones, title, "", []string{cur})
					}
				}
			}
		}

		log.Infoln("Sent alert", a.name)
		a.hourSent++
	}
}

type imageInfo struct {
	Name         string
	Time         string
	AttachedInfo []attachedInfo
}
type attachedInfo struct {
	Title      string
	Percentage string
	Filename   string
	FullPath   string
}

func getHTML(monitorName string, alertTime string, imageInfos []imageInfo, sendAttachments bool) string {
	html := ""
	top := `
	<html>
	<head>
		<meta name=”x-apple-disable-message-reformatting”>
		<style>
			.bgColorLight {
				background-color: rgb(32, 160, 255);
			}

			.bgColorDark {
				background-color: rgb(32, 128, 255);
			}

			body {
				width: 100% !important;
				padding: 0 !important;
				margin: 0 !important;
				font-size: 0.9rem;
				font-family: sans-serif;
			}

			.contentBody {
				width: 100%;
				margin: auto;
				min-height: 20rem;
			}

			.image {
				max-width: 100%;
				max-height: 100%;
				object-fit: contain;
				padding: 0 !important;
				margin: 0 !important;
			}

			.title {
				font-size: 1.5rem;
			}

			.logo {
				min-height: 5rem;
			}

			.footer {
				font-size: 0.6rem;
				font-weight: bold;
			}

			.rowSmallXX {
				height: 0.5rem;
			}
			
			.rowSmallX {
				height: 1rem;
			}

			.rowSmall {
				height: 3rem;
				line-height: 3rem;
			}

			.rowMedium {
				height: 5rem;
				line-height: 5rem;
			}

			.column {
				float: left;
				width: 50%;
				height: 100%;
			}

			.row:before,
			.row:after {
				content: "";
				display: table;
			}
			.row:after {
				clear: both;
			}
			.row {
				zoom: 1;
				display: flow-root;
			}
			
			.w20 {
				width: 20%;
			}

			.w80 {
				width: 80%;
			}

			.w100 {
				width: 100%;
			}

			.textAlignCenter {
				text-align: center;
			}

			.textAlignRight {
				text-align: right;
			}

			.textDate {
				font-size: 0.75rem;
			}

			.rowImage {
				height: 15rem;
				background-color: rgb(228, 228, 228);
			}

			.alertImage {
				min-height: 15rem;
			}
        </style>
    </head>

	<body>
    	<div class='contentBody'>
			<div class='bgColorLight row rowMedium'>
				<div class='column w20'>
					<img class='image logo' src='cid:hawk-wing.png' alt='' />
				</div>
				<div class='column w80'>
					<div class='title'> Scout Alert ` + monitorName + ` </div>
				</div>
			</div>
			<div class='row rowSmall bgColorLight'>
				<div class='column'> Total Images ` + fmt.Sprintf("%d", len(imageInfos)) + ` </div>
				<div class='column textAlignRight textDate'> ` + alertTime + ` </div>
			</div>
			<div class='row rowSmallXX'></div>
	`
	bottom := `
			<div class='row rowSmallXX'></div>
			<div class='bgColorLight row rowMedium'>
				<div class='footer column w100 textAlignCenter'>
					Provided by Scout
				</div>
			</div>
		</div>
	</body>
	</html>
	`
	html += top

	for index, imageInfo := range imageInfos {
		if index > 0 {
			html += `
			<div class='row rowSmallXX'></div>
			`
		}
		middle := getImageHTML(imageInfo, sendAttachments)
		html += middle
	}

	html += bottom

	return html
}

func getImageHTML(imageInfo imageInfo, sendAttachments bool) string {
	html := ""
	top := `
			<div class='row rowSmall bgColorLight'>
				<div class='column'> ` + imageInfo.Name + ` </div>
				<div class='column textAlignRight textDate'> ` + imageInfo.Time + ` </div>
			</div>
	`
	html += top
	for _, attached := range imageInfo.AttachedInfo {
		middleTop := `
				<div class='row rowSmall bgColorDark'>
					<div class='column w100'> ` + attached.Title + ` </div>
				</div>
		`
		if len(attached.Percentage) > 0 {
			middleTop = `
				<div class='row rowSmall bgColorDark'>
					<div class='column'> ` + attached.Title + ` </div>
					<div class='column textAlignRight'> ` + attached.Percentage + ` </div>
				</div>
			`
		}
		html += middleTop
		if len(attached.Filename) > 0 && sendAttachments {
			middleMiddle := `
				<div class='row rowImage'>
					<div class='column w100 textAlignCenter'>
						<img class='image alertImage' src='cid:` + attached.Filename + `' alt='` + attached.Filename + `' />
					</div>
				</div>
			`
			html += middleMiddle
		}

	}

	return html
}
