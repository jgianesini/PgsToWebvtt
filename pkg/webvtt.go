package pkg

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"
)

type Webvtt struct {
	Filename string
	Lines    map[int]WebvttLine
}

type WebvttLine struct {
	LineNumber int
	StartTime  string
	EndTime    string
	Text       string
}

func NewWebvtt(filename string) Webvtt {
	webvtt := Webvtt{Filename: filename}
	webvtt.Lines = make(map[int]WebvttLine)

	return webvtt
}

func (file *Webvtt) Add(key int, StartTime time.Duration, EndTime time.Duration, Text string) Webvtt {
	line := WebvttLine{LineNumber: key + 1, StartTime: formatDuration(StartTime, ".", 3), EndTime: formatDuration(EndTime, ".", 3), Text: Text}
	file.Lines[key] = line

	return *file
}
func (file *Webvtt) Write() {
	filename := file.Filename
	err := os.Remove(filename)
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	check(err)

	keys := make([]int, 0, len(file.Lines))
	for key := range file.Lines {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	lines := make([]WebvttLine, 0, len(file.Lines))
	for _, k := range keys {
		lines = append(lines, file.Lines[k])
	}
	for _, line := range lines {
		t := strconv.Itoa(line.LineNumber)
		_, err := f.WriteString(t + "\n")
		check(err)
		_, err = f.WriteString(line.StartTime + " --> " + line.EndTime + "\n")
		check(err)
		_, err = f.WriteString(line.Text + "\n")
		check(err)
		_, err = f.WriteString("\n")
		check(err)
	}
	err = f.Close()
	check(err)
}

// thx https://github.com/asticode/go-astisub
func formatDuration(i time.Duration, millisecondSep string, numberOfMillisecondDigits int) (s string) {
	// Parse hours
	var hours = int(i / time.Hour)
	var n = i % time.Hour
	if hours < 10 {
		s += "0"
	}
	s += strconv.Itoa(hours) + ":"

	// Parse minutes
	var minutes = int(n / time.Minute)
	n = i % time.Minute
	if minutes < 10 {
		s += "0"
	}
	s += strconv.Itoa(minutes) + ":"

	// Parse seconds
	var seconds = int(n / time.Second)
	n = i % time.Second
	if seconds < 10 {
		s += "0"
	}
	s += strconv.Itoa(seconds) + millisecondSep

	// Parse milliseconds
	var milliseconds = float64(n/time.Millisecond) / float64(1000)
	s += fmt.Sprintf("%."+strconv.Itoa(numberOfMillisecondDigits)+"f", milliseconds)[2:]
	return
}
