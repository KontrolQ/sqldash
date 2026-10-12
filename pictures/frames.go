package pictures

import "encoding/binary"

func inspectFrames(data []byte, details imageDetails) (int, bool, error) {
	switch details.Format {
	case "gif":
		frames, inspectError := inspectGIF(data, details)
		return frames, frames > 1, inspectError
	case "png":
		return inspectPNG(data, details)
	case "webp":
		frames, inspectError := inspectWebP(data, details)
		return frames, frames > 1, inspectError
	}

	return 1, false, nil
}

func skipBlocks(data []byte, position int) (int, error) {
	for position < len(data) {
		length := int(data[position])
		position++
		if length == 0 {
			return position, nil
		}
		if length > len(data)-position {
			return 0, ImageUnreadable
		}
		position += length
	}

	return 0, ImageUnreadable
}

func inspectGIF(data []byte, details imageDetails) (int, error) {
	if len(data) < 13 {
		return 0, ImageUnreadable
	}

	position := 13
	if data[10]&0x80 != 0 {
		position += 3 << (uint(data[10]&7) + 1)
	}

	frames := 0
	for position < len(data) {
		kind := data[position]
		position++
		switch kind {
		case 0x3b:
			if frames == 0 {
				return 0, ImageUnreadable
			}
			return frames, nil
		case 0x21:
			if position >= len(data) {
				return 0, ImageUnreadable
			}
			position++
		case 0x2c:
			if len(data)-position < 9 {
				return 0, ImageUnreadable
			}
			left := int(binary.LittleEndian.Uint16(data[position:]))
			top := int(binary.LittleEndian.Uint16(data[position+2:]))
			width := int(binary.LittleEndian.Uint16(data[position+4:]))
			height := int(binary.LittleEndian.Uint16(data[position+6:]))
			if width == 0 || height == 0 || left+width > details.Width || top+height > details.Height {
				return 0, ImageUnreadable
			}
			frames++
			if !permittedSize(details.Width, details.Height, frames) {
				return 0, ImageTooComplex
			}
			flags := data[position+8]
			position += 9
			if flags&0x80 != 0 {
				position += 3 << (uint(flags&7) + 1)
			}
			position++
		default:
			return 0, ImageUnreadable
		}

		var inspectError error
		position, inspectError = skipBlocks(data, position)
		if inspectError != nil {
			return 0, inspectError
		}
	}

	return 0, ImageUnreadable
}

func inspectPNG(data []byte, details imageDetails) (int, bool, error) {
	position := 8
	frames := 0
	animated := false
	defaultImage := false
	for len(data)-position >= 12 {
		length := int(binary.BigEndian.Uint32(data[position:]))
		if length < 0 || length > len(data)-position-12 {
			return 0, false, ImageUnreadable
		}
		kind := string(data[position+4 : position+8])
		chunk := data[position+8 : position+8+length]
		switch kind {
		case "acTL":
			if length != 8 || animated {
				return 0, false, ImageUnreadable
			}
			declared := int(binary.BigEndian.Uint32(chunk))
			if !permittedSize(details.Width, details.Height, declared) {
				return 0, false, ImageTooComplex
			}
			animated = true
		case "fcTL":
			if length != 26 || !animated {
				return 0, false, ImageUnreadable
			}
			width := uint64(binary.BigEndian.Uint32(chunk[4:]))
			height := uint64(binary.BigEndian.Uint32(chunk[8:]))
			left := uint64(binary.BigEndian.Uint32(chunk[12:]))
			top := uint64(binary.BigEndian.Uint32(chunk[16:]))
			if width == 0 || height == 0 || left+width > uint64(details.Width) || top+height > uint64(details.Height) {
				return 0, false, ImageUnreadable
			}
			frames++
			if !permittedSize(details.Width, details.Height, frames+1) {
				return 0, false, ImageTooComplex
			}
		case "IDAT":
			if frames == 0 {
				defaultImage = true
			}
		case "IEND":
			if !animated {
				return 1, false, nil
			}
			if frames == 0 {
				return 0, false, ImageUnreadable
			}
			if defaultImage {
				frames++
			}
			return frames, true, nil
		}
		position += length + 12
	}

	return 0, false, ImageUnreadable
}

func little24(data []byte) uint64 {
	return uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16
}

func inspectWebP(data []byte, details imageDetails) (int, error) {
	if len(data) < 12 {
		return 0, ImageUnreadable
	}

	declared := uint64(binary.LittleEndian.Uint32(data[4:])) + 8
	if declared != uint64(len(data)) {
		return 0, ImageUnreadable
	}

	position := 12
	frames := 0
	for len(data)-position >= 8 {
		length := int(binary.LittleEndian.Uint32(data[position+4:]))
		if length < 0 || length > len(data)-position-8 {
			return 0, ImageUnreadable
		}
		if string(data[position:position+4]) == "ANMF" {
			if length < 16 {
				return 0, ImageUnreadable
			}
			chunk := data[position+8:]
			left, top := little24(chunk)*2, little24(chunk[3:])*2
			width, height := little24(chunk[6:])+1, little24(chunk[9:])+1
			if left+width > uint64(details.Width) || top+height > uint64(details.Height) {
				return 0, ImageUnreadable
			}
			frames++
			if !permittedSize(details.Width, details.Height, frames) {
				return 0, ImageTooComplex
			}
		}
		position += 8 + length + length%2
	}

	if position != len(data) {
		return 0, ImageUnreadable
	}

	return max(1, frames), nil
}
