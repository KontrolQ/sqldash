package pictures

import "errors"

var (
	UploadTooLarge    = errors.New("The picture exceeds the upload size limit.")
	FormatUnsupported = errors.New("Choose a JPG, WebP, GIF, or PNG picture.")
	ImageUnreadable   = errors.New("That picture could not be read. Choose a valid image file.")
	ImageTooComplex   = errors.New("That picture is too large or has too many animation frames to process safely.")
)
