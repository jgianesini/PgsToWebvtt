package pkg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/otiai10/gosseract"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"runtime"
	"sync"
	"time"
)

// Obj
type Palette struct {
	Red   int
	Green int
	Blue  int
	Alpha int
}
type SubFile struct {
	File        *os.File
	Size        int64
	SectionList []Section
	Current     *Section
	OutFile     string
	Data        []byte
	Position    int64
}
type Section struct {
	Key       int
	StartTime uint32
	EndTime   uint32
	WDS       WDS
	PDS       PDS
	ODS       ODS
}
type WDS struct {
	X           uint16
	Y           uint16
	Height      int
	Width       int
	SectionSize int
}
type PDS struct {
	Palette     map[uint64]Palette
	SectionSize int
}
type ODS struct {
	SectionSize uint16
	StartByte   int64
	Height      uint16
	Width       uint16
}

// Methods
func (subFile *SubFile) Open(filename string) error {
	var err error
	subFile.File, err = os.Open(filename)

	stats, err := subFile.File.Stat()
	subFile.Size = stats.Size()

	return err
}
func (subFile *SubFile) Run() error {
	var err error

	// Initial star point
	subFile.Parse(0, 13)
	if len(subFile.SectionList) == 0 {
		fmt.Println("Data is empty")
		return nil
	}

	return err
}

var startTime = time.Time{}

func (subFile *SubFile) Save(filename string) error {
	startTime = time.Now()
	webvtt := NewWebvtt(filename)
	subFile.process(subFile.SectionList, 0, webvtt)
	return nil
}
func (subFile *SubFile) process(data []Section, pos int, webvtt Webvtt) {
	var cpus = runtime.GOMAXPROCS(0)
	var start = 0
	var end = cpus
	var c = float64(len(data)) / float64(end)
	var max = int(math.Floor(c))
	start = pos * end
	end = (pos + 1) * end

	if pos == max {
		end = len(data)
	}
	current := data[start:end]
	var wg sync.WaitGroup
	wg.Add(len(current))
	for _, element := range current {
		subFile.Read(element.ODS.StartByte+24, int64(element.ODS.SectionSize))
		go subFile.async(element, &webvtt, subFile.Data, &wg)
	}
	wg.Wait()
	secondsRemaining := int64(0)
	if pos >= 1 {
		now := time.Now()
		elapsedMS := now.UnixNano() - startTime.UnixNano()
		if elapsedMS != 0 {
			ticksPerMS := elapsedMS / int64(pos)
			ticksRemaining := max - pos
			msRemaining := int64(ticksRemaining) * ticksPerMS
			secondsRemaining = int64(math.Round(float64(msRemaining) / 1000000000))
		}
	}
	fmt.Printf("\r\033[K%d/%d tasks (%d/%d bitmaps) (GOMAXPROCS -> %d) ETA : %s", pos, max, pos+1*cpus, len(data), cpus, format(secondsRemaining))
	if pos == max {
		webvtt.Write()
		fmt.Println()
		return
	}
	subFile.process(data, pos+1, webvtt)
}
func (subFile *SubFile) async(section Section, webvtt *Webvtt, bitmap []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	webvtt.Add(section.Key, time.Duration(int64(section.StartTime)*1000000), time.Duration(int64(section.EndTime)*1000000), Decode(section, bitmap))
}

func (subFile *SubFile) Parse(pos int64, length int64) {

	status := subFile.Next(pos, length)
	// if end file
	if !status {
		return
	}

	switch fmt.Sprintf("%x", subFile.Data[10:11]) {
	case "16":
		now := binary.BigEndian.Uint32(subFile.Data[2:6])

		if subFile.Current != nil {
			if subFile.Current.ODS.Height != 0 {
				subFile.Current.EndTime = now / 90
				subFile.SectionList = append(subFile.SectionList, *subFile.Current)
			}
			subFile.Current = nil
		}
		subFile.Current = &Section{Key: len(subFile.SectionList), StartTime: now / 90}
		subFile.Parse(13+int64(subFile.SectionSize()), 19)
		break
	case "17":
		x := binary.BigEndian.Uint16(subFile.Data[15:17])
		y := binary.BigEndian.Uint16(subFile.Data[17:19])

		wds := WDS{X: x, Y: y}
		subFile.Current.WDS = wds
		subFile.Parse(13+int64(subFile.SectionSize()), 13)
		break
	case "14":
		size := subFile.SectionSize()
		subFile.Read(subFile.GetCurrentPosition()+int64(15), 13+int64(size))

		pds := PDS{}
		pds.Palette = make(map[uint64]Palette)
		for i := 0; i < len(subFile.Data); i += 5 {
			index := int(subFile.Data[i])
			y := int(subFile.Data[i+1]) - 16
			cb := int(subFile.Data[i+2]) - 128
			cr := int(subFile.Data[i+3]) - 128

			r := math.Max(float64(0), math.Min(float64(255), math.Round(1.1644*float64(y)+1.596*float64(cr))))
			g := math.Max(float64(0), math.Min(float64(255), math.Round(1.1644*float64(y)-0.813*float64(cr)-0.391*float64(cb))))
			b := math.Max(float64(0), math.Min(float64(255), math.Round(1.1644*float64(y)+2.018*float64(cb))))

			var palette Palette
			if r == 137 {
				r = 0
			}
			palette.Red = int(r)
			palette.Green = int(g)
			palette.Blue = int(b)
			palette.Alpha = 255

			pds.Palette[uint64(index)] = palette
		}
		subFile.Current.PDS = pds
		subFile.Parse(13+int64(size), 24)
		break
	case "15":
		if subFile.Current.ODS.SectionSize != 0 {
			panic("many")
		}
		size := subFile.SectionSize()
		ods := ODS{SectionSize: size, StartByte: subFile.Position, Width: binary.BigEndian.Uint16(subFile.Data[20:22]), Height: binary.BigEndian.Uint16(subFile.Data[22:24])}
		subFile.Current.ODS = ods

		subFile.Parse(13+int64(size), 13)
		break
	case "80":
		size := subFile.SectionSize()
		subFile.Parse(13+int64(size), 13)
		break
	}
}
func (subFile *SubFile) SectionSize() uint16 {
	return binary.BigEndian.Uint16(subFile.Data[11:13])
}
func (subFile *SubFile) GetCurrentPosition() int64 {
	return subFile.Position
}
func (subFile *SubFile) Next(pos int64, length int64) bool {
	subFile.Position = subFile.Position + pos
	if subFile.Position >= subFile.Size {
		return false
	}
	_, err := subFile.File.Seek(subFile.Position, 0)
	check(err)

	subFile.Data = make([]byte, length)
	_, err = subFile.File.Read(subFile.Data)
	check(err)

	return true
}
func (subFile *SubFile) Read(pos int64, length int64) {
	_, err := subFile.File.Seek(pos, 0)
	check(err)

	subFile.Data = make([]byte, length)
	_, err = subFile.File.Read(subFile.Data)
	check(err)
}

func Decode(section Section, bitmap []byte) string {
	currentX := uint16(0)
	currentY := uint16(0)

	frameTotalX := currentX + section.ODS.Width
	frameTotalY := currentY + section.ODS.Height

	r := NewRLEStream(bytes.NewBuffer(bitmap))

	rleStream := RleStream{Reader: r, Position: 0, RunLength: 0, ColorIndex: 0, ToEndofLine: false}

	img := image.NewRGBA(image.Rect(0, 0, int(section.ODS.Width), int(section.ODS.Height)))
	for currentY < frameTotalY {
		rleStream.NextRun()

		palette := section.PDS.Palette[rleStream.ColorIndex]

		fillUntilX := uint64(currentX) + rleStream.RunLength
		if rleStream.ToEndofLine {
			fillUntilX = uint64(frameTotalX)
		}
		for ; uint64(currentX) < fillUntilX; currentX++ {
			img.Set(int(currentX), int(currentY), color.RGBA{R: uint8(palette.Red), G: uint8(palette.Green), B: uint8(palette.Blue), A: uint8(palette.Alpha)})
		}
		if rleStream.ToEndofLine {
			currentX = 0
			currentY++
		}
	}
	f, err := os.Create("image.png")
	if err != nil {
		panic(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
	buf := new(bytes.Buffer)
	err = png.Encode(buf, img)
	if err != nil {
		panic(err)
	}
	client := gosseract.NewClient()
	err = client.SetLanguage("fra+eng")
	if err != nil {
		panic(err)
	}
	err = client.SetImageFromBytes(buf.Bytes())
	text, _ := client.Text()
	err = client.Close()

	return text
}
func humanizeDuration(duration time.Duration) string {
	if duration.Seconds() < 60.0 {
		return fmt.Sprintf("%d seconds", int64(duration.Seconds()))
	}
	if duration.Minutes() < 60.0 {
		remainingSeconds := math.Mod(duration.Seconds(), 60)
		return fmt.Sprintf("%d minutes %d seconds", int64(duration.Minutes()), int64(remainingSeconds))
	}
	if duration.Hours() < 24.0 {
		remainingMinutes := math.Mod(duration.Minutes(), 60)
		remainingSeconds := math.Mod(duration.Seconds(), 60)
		return fmt.Sprintf("%d hours %d minutes %d seconds",
			int64(duration.Hours()), int64(remainingMinutes), int64(remainingSeconds))
	}
	remainingHours := math.Mod(duration.Hours(), 24)
	remainingMinutes := math.Mod(duration.Minutes(), 60)
	remainingSeconds := math.Mod(duration.Seconds(), 60)

	return fmt.Sprintf("%d days %d hours %d minutes %d seconds",
		int64(duration.Hours()/24), int64(remainingHours),
		int64(remainingMinutes), int64(remainingSeconds))
}
func format(duration int64) string {
	modTime := time.Now().Round(0).Add(-time.Duration(duration) * time.Second)
	since := time.Since(modTime)
	return humanizeDuration(since)
}
