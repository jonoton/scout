package dir

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// RegexEndsWith returns the string regex
func RegexEndsWith(val string) string {
	return fmt.Sprintf("^.*(%s)$", val)
}

// RegexBeginsWith returns the string regex
func RegexBeginsWith(val string) string {
	return fmt.Sprintf("^(%s).*$", val)
}

// Size returns the directory size in Bytes
func Size(path string, regex string) (uint64, error) {
	var size uint64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if matched, _ := regexp.MatchString(regex, info.Name()); matched || len(regex) == 0 {
				size += uint64(info.Size())
			}
		}
		return err
	})
	return size, err
}

// List returns the files
func List(path string, regex string) ([]os.FileInfo, error) {
	result := make([]os.FileInfo, 0)
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if matched, _ := regexp.MatchString(regex, info.Name()); matched || len(regex) == 0 {
				result = append(result, info)
			}
		}
		return err
	})
	return result, err
}

// Expired returns the files that have expired
func Expired(path string, regex string, nowTime time.Time, maxTime time.Duration) ([]os.FileInfo, error) {
	result := make([]os.FileInfo, 0)
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if matched, _ := regexp.MatchString(regex, info.Name()); matched || len(regex) == 0 {
				delta := nowTime.Sub(info.ModTime())
				if delta > maxTime {
					result = append(result, info)
				}
			}
		}
		return err
	})
	return result, err
}

// BytesToMegaBytes converts Bytes to MegaBytes
func BytesToMegaBytes(in uint64) float64 {
	return float64(in) / 1000 / 1000
}

// BytesToGigaBytes converts Bytes to GigaBytes
func BytesToGigaBytes(in uint64) float64 {
	return float64(in) / 1000 / 1000 / 1000
}

// AscendingTime sorting FileInfo by time
type AscendingTime []os.FileInfo

func (a AscendingTime) Len() int { return len(a) }
func (a AscendingTime) Less(i, j int) bool {
	return a[i].ModTime().Before(a[j].ModTime())
}
func (a AscendingTime) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// DescendingTime sorting FileInfo by time
type DescendingTime []os.FileInfo

func (a DescendingTime) Len() int { return len(a) }
func (a DescendingTime) Less(i, j int) bool {
	return a[i].ModTime().After(a[j].ModTime())
}
func (a DescendingTime) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
