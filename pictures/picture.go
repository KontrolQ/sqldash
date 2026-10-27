package pictures

import (
	"bytes"
	"encoding/base64"
	"image"
	"io"
)

func Transform(source io.Reader, maximumBytes int64) (*Picture, error) {
	if maximumBytes < 0 {
		return nil, UploadTooLarge
	}
	if maximumBytes > 0 {
		source = io.LimitReader(source, maximumBytes+1)
	}
	data, readError := io.ReadAll(source)
	if readError != nil {
		return nil, ImageUnreadable
	}

	if maximumBytes > 0 && int64(len(data)) > maximumBytes {
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

func Source(reference string, mediaType string) string {
	filename, storedType, validationError := StoredName(reference)
	if validationError != nil || storedType != mediaType {
		return ""
	}
	return URLPrefix + filename
}

func mediaTypeOf(format string) string {
	switch format {
	case "jpeg", "png", "gif", "webp":
		return "image/" + format
	}

	return ""
}
