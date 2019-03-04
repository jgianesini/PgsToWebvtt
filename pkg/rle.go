package pkg

import "io"

type RleStream struct {
	Reader      *BitReader
	Position    int
	RunLength   uint64
	ColorIndex  uint64
	ToEndofLine bool
}

func (stream RleStream) GetPosition() int {
	return stream.Position
}
func (stream RleStream) GetRunLength() uint64 {
	return stream.RunLength
}
func (stream RleStream) GetColorIndex() uint64 {
	return stream.ColorIndex
}
func (stream RleStream) GetToEndofLine() bool {
	return stream.ToEndofLine
}

func (stream *RleStream) NextRun() {
	stream.ToEndofLine = false
	stream.RunLength = 0
	stream.ColorIndex = 0

	// Process
	firstByte := stream.Reader.bits(8)
	if firstByte != 0 {
		stream.RunLength = 1
		stream.ColorIndex = firstByte
		return
	}

	firstBitOn := stream.Reader.bool()
	secondBitOn := stream.Reader.bool()

	stream.RunLength = stream.Reader.bits(6)
	if secondBitOn {
		stream.RunLength = (stream.RunLength << 8) + stream.Reader.bits(8)
	}
	if firstBitOn {
		stream.ColorIndex = stream.Reader.bits(8)
	}

	if !firstBitOn && !secondBitOn && stream.RunLength == 0 {
		stream.ToEndofLine = true
	}
}

type BitReader struct {
	reader       io.ByteReader
	byte         byte
	offset       int
	bytePosition uint
	current      uint8
}

func NewRLEStream(r io.ByteReader) *BitReader {
	return &BitReader{r, 0, 0, 8, 0}
}

func (r *BitReader) readNextByte() {
	r.offset++
	value, err := r.reader.ReadByte()
	if err != nil {
		panic(err)
	}
	r.current = value
	r.bytePosition = 0
}
func (r *BitReader) bool() bool {
	return r.bit() == 1
}
func (r *BitReader) bit() int {
	if r.bytePosition == 8 {
		r.readNextByte()
	}
	bit := 0
	if (r.current & (0x80 >> r.bytePosition)) != 0 {
		bit = 1
	}
	r.bytePosition++
	return bit
}
func (r *BitReader) bits(nbits int) uint64 {
	result := uint64(0)
	for i := 0; i < nbits; i++ {
		v := r.bit()
		result = (result << 1) + uint64(v)
	}
	return result
}
