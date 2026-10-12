package pictures

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
)

import "github.com/kettek/apng"

func processPNG(data []byte, details imageDetails) ([]byte, error) {
	if details.Animated {
		return processAnimatedPNG(data, details)
	}

	decoded, decodeError := png.Decode(bytes.NewReader(data))
	if decodeError != nil {
		return nil, decodeError
	}

	width, height := dimensions(details.Width, details.Height)
	result := &bytes.Buffer{}
	encodeError := png.Encode(result, resize(decoded, width, height))
	return result.Bytes(), encodeError
}

func processAnimatedPNG(data []byte, details imageDetails) ([]byte, error) {
	animation, decodeError := apng.DecodeAll(bytes.NewReader(data))
	if decodeError != nil {
		return nil, decodeError
	}

	width, height := dimensions(details.Width, details.Height)
	canvas := image.NewNRGBA(image.Rect(0, 0, details.Width, details.Height))
	for position, frame := range animation.Frames {
		if frame.IsDefault {
			animation.Frames[position].Image = resize(frame.Image, width, height)
			continue
		}

		var previous *image.NRGBA
		if frame.DisposeOp == apng.DISPOSE_OP_PREVIOUS {
			previous = copyCanvas(canvas)
		}

		bounds := frame.Image.Bounds().Add(image.Pt(frame.XOffset, frame.YOffset))
		operation := draw.Over
		if frame.BlendOp == apng.BLEND_OP_SOURCE {
			operation = draw.Src
		}

		draw.Draw(canvas, bounds, frame.Image, frame.Image.Bounds().Min, operation)
		animation.Frames[position].Image = resize(canvas, width, height)
		animation.Frames[position].XOffset = 0
		animation.Frames[position].YOffset = 0
		animation.Frames[position].BlendOp = apng.BLEND_OP_SOURCE
		animation.Frames[position].DisposeOp = apng.DISPOSE_OP_NONE

		switch frame.DisposeOp {
		case apng.DISPOSE_OP_BACKGROUND:
			clearCanvas(canvas, bounds)
		case apng.DISPOSE_OP_PREVIOUS:
			canvas = previous
		}
	}

	result := &bytes.Buffer{}
	encodeError := apng.Encode(result, animation)
	return result.Bytes(), encodeError
}
