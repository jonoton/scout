package gzip

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"
)

// Header is the gzip header
type Header struct {
	Name    string
	Comment string
	Date    time.Time
}

// Encode payload to gzip result bytes
func Encode(payload []byte, header *Header) (result []byte) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	if header != nil {
		zw.Name = header.Name
		zw.Comment = header.Comment
		zw.ModTime = header.Date
	}

	_, err := zw.Write(payload)
	if err != nil {
		log.Error(err)
	}
	err = zw.Flush()
	if err != nil {
		log.Error(err)
	}
	err = zw.Close()
	if err != nil {
		log.Error(err)
	}
	result = buf.Bytes()
	return
}

// Decode gzip to result bytes
func Decode(data []byte) (result []byte, header *Header) {
	buf := bytes.NewBuffer(data)
	zr, err := gzip.NewReader(buf)
	if err != nil {
		log.Error(err)
	}
	if zr.Name != "" || zr.Comment != "" || !zr.ModTime.Equal(time.Time{}) {
		header = &Header{
			Name:    zr.Name,
			Comment: zr.Comment,
			Date:    zr.ModTime,
		}
	}
	result, err = ioutil.ReadAll(zr)
	if err != nil {
		log.Error(err)
	}
	err = zr.Close()
	if err != nil {
		log.Error(err)
	}
	return
}
