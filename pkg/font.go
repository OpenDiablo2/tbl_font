package pkg

import (
	"fmt"
	"io"

	"github.com/OpenDiablo2/bitstream"
)

const (
	knownSignature = "Woo!\x01"
)

const (
	numHeaderBytes          = 12
	bytesPerGlyph           = 14
	signatureBytesCount     = 5
	unknownHeaderBytesCount = 7
	unknown1BytesCount      = 1
	unknown2BytesCount      = 3
	unknown3BytesCount      = 4
)

// FontTable represents a displayable font
type FontTable struct {
	Glyphs map[rune]*FontGlyph
}

// Load loads a new font from byte slice
func Load(data io.ReadSeeker) (*FontTable, error) {
	stream := bitstream.NewReader(data)
	font := &FontTable{}
	err := font.Decode(stream)

	return font, err
}

// GetTextMetrics returns the dimensions of the FontTable element in pixels
func (f *FontTable) GetTextMetrics(text string) (width, height int) {
	var (
		lineWidth  int
		lineHeight int
	)

	for _, c := range text {
		if c == '\n' {
			width = max(width, lineWidth)
			height += lineHeight
			lineWidth = 0
			lineHeight = 0
		} else if glyph, ok := f.Glyphs[c]; ok {
			lineWidth += glyph.Width()
			lineHeight = max(lineHeight, glyph.Height())
		}
	}

	width = max(width, lineWidth)
	height += lineHeight

	return width, height
}

func (f *FontTable) Decode(stream *bitstream.Reader) error {
	signature, err := stream.Next(signatureBytesCount).Bytes().AsBytes()
	if err != nil {
		return err
	}

	if string(signature) != knownSignature {
		return fmt.Errorf("invalid font table format")
	}

	stream.Next(unknownHeaderBytesCount).Bytes()

	glyphs := make(map[rune]*FontGlyph)

	// for i := numHeaderBytes; i < len(f.table); i += bytesPerGlyph {
	for i := numHeaderBytes; true; i += bytesPerGlyph {
		code, err := stream.Next(2).Bytes().AsUInt16()
		if err != nil {
			break
		}

		// byte of 0
		stream.Next(unknown1BytesCount).Bytes()

		width, err := stream.Next(1).Bytes().AsByte()
		if err != nil {
			return err
		}

		height, err := stream.Next(1).Bytes().AsByte()
		if err != nil {
			return err
		}

		// 1, 0, 0
		stream.Next(unknown2BytesCount).Bytes()

		frame, err := stream.Next(2).Bytes().AsUInt16()
		if err != nil {
			return err
		}

		// 1, 0, 0, character code repeated, and further 0.
		stream.Next(unknown3BytesCount).Bytes()

		glyph := newGlyph(int(frame), int(width), int(height))

		glyphs[rune(code)] = glyph
	}

	f.Glyphs = glyphs

	return nil
}

// Encode font back into byte slice
func (f *FontTable) Encode(sw bitstream.Writer) error {
	if _, err := sw.WriteBytes([]byte("Woo!\x01")); err != nil {
		return err
	}

	// unknown header bytes - constant
	if _, err := sw.WriteBytes([]byte{1, 0, 0, 0, 0, 1}); err != nil {
		return err
	}

	// Expected Height of character cell and Expected Width of character cell
	// not used in decoder
	if _, err := sw.WriteBytes([]byte{0, 0}); err != nil {
		return err
	}

	for c, i := range f.Glyphs {
		// only check the last error for brevity
		_, _ = sw.WriteByte(byte(c))
		_, _ = sw.WriteBytes(i.Unknown1())
		_, _ = sw.WriteByte(byte(i.Width()))
		_, _ = sw.WriteByte(byte(i.Height()))
		_, _ = sw.WriteBytes(i.Unknown2())
		_, _ = sw.WriteByte(byte(i.FrameIndex()))
		_, err := sw.WriteBytes(i.Unknown3())
		
		if err != nil {
			return err
		}
	}

	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}
