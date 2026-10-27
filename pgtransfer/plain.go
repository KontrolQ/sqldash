package pgtransfer

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

func (self *job) inputFormat(input *os.File) (string, error) {
	information, statError := input.Stat()
	if statError != nil || !information.Mode().IsRegular() || information.Size() == 0 {
		return "", InvalidSource
	}
	if self.maximumBytes > 0 && information.Size() > self.maximumBytes {
		return "", SizeExceeded
	}
	magic := make([]byte, len(ArchiveMagic))
	readCount, readError := input.ReadAt(magic, 0)
	if readError != nil && readError != io.EOF {
		return "", TransferFailed
	}
	if readCount == len(magic) && string(magic) == ArchiveMagic {
		return FormatCustom, nil
	}
	if textError := self.validateText(io.NewSectionReader(input, 0, information.Size())); textError != nil {
		return "", textError
	}
	return FormatSQL, nil
}

func (self *job) validateText(source io.Reader) error {
	buffer := make([]byte, CopyBufferBytes+utf8.UTFMax)
	carried := 0
	for {
		if contextError := self.context.Err(); contextError != nil {
			return contextError
		}
		readCount, readError := source.Read(buffer[carried:CopyBufferBytes])
		data := buffer[:carried+readCount]
		if bytes.IndexByte(data, 0) != -1 {
			return InvalidSQL
		}
		position := 0
		for position < len(data) {
			character, width := utf8.DecodeRune(data[position:])
			if character == utf8.RuneError && width == 1 {
				if !utf8.FullRune(data[position:]) && readError == nil {
					break
				}
				return InvalidSQL
			}
			position += width
		}
		carried = copy(buffer, data[position:])
		if readError == io.EOF {
			if carried != 0 {
				return InvalidSQL
			}
			return nil
		}
		if readError != nil {
			return TransferFailed
		}
		if readCount == 0 && carried == 0 {
			return TransferFailed
		}
	}
}

func restrictedSQL(input *os.File) (io.Reader, error) {
	information, statError := input.Stat()
	if statError != nil {
		return nil, InvalidSource
	}
	prefix := make([]byte, min(int64(WrapperInspectionBytes), information.Size()))
	if _, readError := input.ReadAt(prefix, 0); readError != nil && readError != io.EOF {
		return nil, TransferFailed
	}
	start, finish := int64(0), information.Size()
	newline := "\n"
	header := PlainHeader
	if bytes.HasPrefix(prefix, []byte(strings.ReplaceAll(PlainHeader, "\n", "\r\n"))) {
		newline = "\r\n"
		header = strings.ReplaceAll(PlainHeader, "\n", newline)
	}
	wrapped := bytes.HasPrefix(prefix, []byte(header))
	if wrapped {
		lineStart := len(header)
		lineFinish := bytes.Index(prefix[lineStart:], []byte(newline))
		if lineFinish == -1 {
			return nil, InvalidWrapper
		}
		line := string(prefix[lineStart : lineStart+lineFinish])
		key, keyError := restrictionKey(line, "\\restrict ")
		if keyError != nil {
			return nil, keyError
		}
		start = int64(lineStart + lineFinish + len(newline))
		suffix := make([]byte, min(int64(WrapperInspectionBytes), information.Size()))
		if _, readError := input.ReadAt(suffix, information.Size()-int64(len(suffix))); readError != nil && readError != io.EOF {
			return nil, TransferFailed
		}
		footer := strings.ReplaceAll(PlainFooter, "\n", newline)
		unlock := "\\unrestrict " + key + newline
		matched := false
		for _, ending := range []string{unlock + newline, unlock} {
			if bytes.HasSuffix(suffix, []byte(footer+ending)) {
				finish = information.Size() - int64(len(ending))
				matched = true
				break
			}
		}
		if !matched || finish <= start {
			return nil, InvalidWrapper
		}
	}
	keyBytes := make([]byte, RestrictionKeyBytes)
	if _, randomError := rand.Read(keyBytes); randomError != nil {
		return nil, TransferFailed
	}
	key := hex.EncodeToString(keyBytes)
	clear(keyBytes)
	trustedPrefix := strings.NewReader("\\restrict " + key + "\n")
	if wrapped {
		publicStart, publicFinish := nativePublicSchemaRange(input, start, finish)
		if publicStart != 0 {
			return io.MultiReader(trustedPrefix, io.NewSectionReader(input, 0, int64(len(header))), io.NewSectionReader(input, start, publicStart-start), io.NewSectionReader(input, publicFinish, finish-publicFinish), strings.NewReader("\n")), nil
		}
		return io.MultiReader(trustedPrefix, io.NewSectionReader(input, 0, int64(len(header))), io.NewSectionReader(input, start, finish-start), strings.NewReader("\n")), nil
	}
	return io.MultiReader(trustedPrefix, io.NewSectionReader(input, 0, information.Size()), strings.NewReader("\n")), nil
}

func restrictionKey(line, command string) (string, error) {
	if !strings.HasPrefix(line, command) {
		return "", InvalidWrapper
	}
	key := strings.TrimPrefix(line, command)
	if len(key) == 0 || len(key) > 128 {
		return "", InvalidWrapper
	}
	for _, character := range key {
		if (character < '0' || character > '9') && (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') {
			return "", InvalidWrapper
		}
	}
	return key, nil
}
