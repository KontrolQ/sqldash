package bytesize

import "errors"

var Invalid = errors.New("Enter a whole-byte memory size, such as 256 MiB, 1.5 GB or 268435456 bytes.")
