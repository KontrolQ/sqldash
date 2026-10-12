package pictures

import (
	"bytes"
	"image/jpeg"
)

func processJPEG(data []byte, details imageDetails) ([]byte, error) {
	decoded, decodeError := jpeg.Decode(bytes.NewReader(data))
	if decodeError != nil {
		return nil, decodeError
	}

	width, height := dimensions(details.Width, details.Height)
	result := &bytes.Buffer{}
	encodeError := jpeg.Encode(result, resize(decoded, width, height), &jpeg.Options{Quality: JPEGQuality})

	return result.Bytes(), encodeError
}
