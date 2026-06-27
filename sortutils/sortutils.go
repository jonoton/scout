package sortutils

import (
	"os"

	"github.com/jonoton/go-dir"
)

// DescendingTime sorts []os.FileInfo by modification time descending,
// falling back to filename descending if modification times are equal.
type DescendingTime []os.FileInfo

func (s DescendingTime) Len() int      { return len(s) }
func (s DescendingTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s DescendingTime) Less(i, j int) bool {
	t1 := s[i].ModTime()
	t2 := s[j].ModTime()
	if !t1.Equal(t2) {
		return t1.After(t2)
	}
	return s[i].Name() > s[j].Name()
}

// DescendingTimeName sorts []string by timestamp in name descending,
// falling back to the full filename descending if timestamps are equal or missing.
type DescendingTimeName []string

func (s DescendingTimeName) Len() int      { return len(s) }
func (s DescendingTimeName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s DescendingTimeName) Less(i, j int) bool {
	first := dir.FilenameTimestampRegex.FindString(s[i])
	second := dir.FilenameTimestampRegex.FindString(s[j])
	if first != second {
		return first > second
	}
	return s[i] > s[j]
}

// AscendingTime sorts []os.FileInfo by modification time ascending,
// falling back to filename ascending if modification times are equal.
type AscendingTime []os.FileInfo

func (s AscendingTime) Len() int      { return len(s) }
func (s AscendingTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s AscendingTime) Less(i, j int) bool {
	t1 := s[i].ModTime()
	t2 := s[j].ModTime()
	if !t1.Equal(t2) {
		return t1.Before(t2)
	}
	return s[i].Name() < s[j].Name()
}
