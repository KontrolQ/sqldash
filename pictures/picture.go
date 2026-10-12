package pictures

import (
	"bytes"
	"encoding/base64"
	"image"
	"io"
)

func Transform(source io.Reader, maximumBytes int64) (*Picture, error) {
	data, readError := io.ReadAll(io.LimitReader(source, maximumBytes+1))
	if readError != nil {
		return nil, ImageUnreadable
	}

	if int64(len(data)) > maximumBytes {
		return nil, UploadTooLarge
	}

	configuration, format, inspectError := image.DecodeConfig(bytes.NewReader(data))
	if inspectError != nil {
		return nil, ImageUnreadable
	}

	if format == "apng" {
		format = "png"
	}

	mediaType := mediaTypeOf(format)
	if mediaType == "" {
		return nil, FormatUnsupported
	}

	if !permittedSize(configuration.Width, configuration.Height, 1) {
		return nil, ImageTooComplex
	}

	details := imageDetails{Width: configuration.Width, Height: configuration.Height, Format: format}
	frames, animated, inspectError := inspectFrames(data, details)
	if inspectError != nil {
		return nil, inspectError
	}

	if !permittedSize(details.Width, details.Height, frames) {
		return nil, ImageTooComplex
	}

	details.Animated = animated
	var processed []byte
	var processError error

	switch format {
	case "jpeg":
		processed, processError = processJPEG(data, details)
	case "png":
		processed, processError = processPNG(data, details)
	case "gif":
		processed, processError = processGIF(data, details)
	case "webp":
		processed, processError = processWebP(data, details)
	}

	if processError != nil {
		if processError == ImageTooComplex {
			return nil, processError
		}
		return nil, ImageUnreadable
	}

	return &Picture{Base64: base64.StdEncoding.EncodeToString(processed), MediaType: mediaType}, nil
}

func Source(encoded string, mediaType string) string {
	if encoded == "" || (mediaType != "image/jpeg" && mediaType != "image/png" && mediaType != "image/gif" && mediaType != "image/webp") {
		return ""
	}

	return "data:" + mediaType + ";base64," + encoded
}

func mediaTypeOf(format string) string {
	switch format {
	case "jpeg", "png", "gif", "webp":
		return "image/" + format
	}

	return ""
}
