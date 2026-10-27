package pictures

import "errors"

var (
	ReferenceInvalid     = errors.New("That picture reference is not valid.")
	StoredPictureDamaged = errors.New("The stored picture could not be verified.")
	UploadTooLarge       = errors.New("The picture exceeds the upload size limit.")
	FormatUnsupported    = errors.New("Choose a JPG, WebP, GIF, or PNG picture.")
	ImageUnreadable      = errors.New("That picture could not be read. Choose a valid image file.")
	ImageTooComplex      = errors.New("That picture is too large or has too many animation frames to process safely.")
)
