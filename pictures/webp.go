package pictures

import (
	"bytes"
	"encoding/binary"
	"image"
)

import "github.com/gen2brain/webp"

func webPLoopCount(data []byte) int {
	for position := 12; len(data)-position >= 8; {
		length := int(binary.LittleEndian.Uint32(data[position+4:]))
		if string(data[position:position+4]) == "ANIM" && length >= 6 {
			return int(binary.LittleEndian.Uint16(data[position+12:]))
		}
		position += 8 + length + length%2
	}
	return 0
}

func processWebP(data []byte, details imageDetails) ([]byte, error) {
	animation, decodeError := webp.DecodeAll(bytes.NewReader(data))
	if decodeError != nil {
		return nil, decodeError
	}

	if !permittedSize(details.Width, details.Height, len(animation.Image)) {
		return nil, ImageTooComplex
	}

	width, height := dimensions(details.Width, details.Height)
	for position, frame := range animation.Image {
		if rgba, matched := frame.(*image.RGBA); matched {
			frame = &image.NRGBA{Pix: rgba.Pix, Stride: rgba.Stride, Rect: rgba.Rect}
		}
		animation.Image[position] = resize(frame, width, height)
	}
	animation.LoopCount = webPLoopCount(data)

	result := &bytes.Buffer{}
	options := webp.Options{Lossless: true}
	var encodeError error
	if len(animation.Image) == 1 {
		encodeError = webp.Encode(result, animation.Image[0], options)
	} else {
		encodeError = webp.EncodeAll(result, animation, options)
	}

	return result.Bytes(), encodeError
}
