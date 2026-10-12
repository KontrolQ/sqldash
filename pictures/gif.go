package pictures

import (
	"bytes"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
)

func copyCanvas(source *image.NRGBA) *image.NRGBA {
	result := image.NewNRGBA(source.Bounds())
	copy(result.Pix, source.Pix)
	return result
}

func clearCanvas(canvas *image.NRGBA, bounds image.Rectangle) {
	draw.Draw(canvas, bounds, image.Transparent, image.Point{}, draw.Src)
}

func gifFrame(source image.Image) *image.Paletted {
	colours := append(color.Palette{color.Transparent}, palette.Plan9[:255]...)
	result := image.NewPaletted(source.Bounds(), colours)
	draw.FloydSteinberg.Draw(result, result.Bounds(), source, source.Bounds().Min)
	for row := source.Bounds().Min.Y; row < source.Bounds().Max.Y; row++ {
		for column := source.Bounds().Min.X; column < source.Bounds().Max.X; column++ {
			_, _, _, alpha := source.At(column, row).RGBA()
			if alpha < 0x8000 {
				result.SetColorIndex(column, row, 0)
			}
		}
	}
	return result
}

func gifBackground(animation *gif.GIF) color.Color {
	for _, frame := range animation.Image {
		for _, colour := range frame.Palette {
			_, _, _, alpha := colour.RGBA()
			if alpha == 0 {
				return color.Transparent
			}
		}
	}
	if colours, matched := animation.Config.ColorModel.(color.Palette); matched && int(animation.BackgroundIndex) < len(colours) {
		return colours[animation.BackgroundIndex]
	}
	return color.Transparent
}

func processGIF(data []byte, details imageDetails) ([]byte, error) {
	animation, decodeError := gif.DecodeAll(bytes.NewReader(data))
	if decodeError != nil {
		return nil, decodeError
	}

	width, height := dimensions(details.Width, details.Height)
	if width == details.Width && height == details.Height {
		result := &bytes.Buffer{}
		encodeError := gif.EncodeAll(result, animation)
		return result.Bytes(), encodeError
	}

	canvas := image.NewNRGBA(image.Rect(0, 0, details.Width, details.Height))
	background := image.NewUniform(gifBackground(animation))
	draw.Draw(canvas, canvas.Bounds(), background, image.Point{}, draw.Src)
	processed := &gif.GIF{LoopCount: animation.LoopCount, Delay: animation.Delay, BackgroundIndex: 0}
	for position, frame := range animation.Image {
		var previous *image.NRGBA
		disposal := byte(gif.DisposalNone)
		if position < len(animation.Disposal) {
			disposal = animation.Disposal[position]
		}
		if disposal == gif.DisposalPrevious {
			previous = copyCanvas(canvas)
		}
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
		processed.Image = append(processed.Image, gifFrame(resize(canvas, width, height)))
		processed.Disposal = append(processed.Disposal, gif.DisposalBackground)
		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, frame.Bounds(), background, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			canvas = previous
		}
	}

	processed.Config = image.Config{Width: width, Height: height, ColorModel: processed.Image[0].Palette}
	result := &bytes.Buffer{}
	encodeError := gif.EncodeAll(result, processed)
	return result.Bytes(), encodeError
}
