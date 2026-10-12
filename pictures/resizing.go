package pictures

import "image"

import "golang.org/x/image/draw"

func dimensions(width int, height int) (int, int) {
	if width <= MaximumSide && height <= MaximumSide {
		return width, height
	}

	if width >= height {
		return MaximumSide, max(1, height*MaximumSide/width)
	}

	return max(1, width*MaximumSide/height), MaximumSide
}

func resize(source image.Image, width int, height int) *image.NRGBA {
	result := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(result, result.Bounds(), source, source.Bounds(), draw.Src, nil)

	return result
}

func permittedSize(width int, height int, frames int) bool {
	return width > 0 && height > 0 && frames > 0 && frames <= MaximumFrames &&
		int64(width) <= MaximumDecodedPixels && int64(height) <= MaximumDecodedPixels &&
		int64(width)*int64(height) <= MaximumDecodedPixels/int64(frames)
}
