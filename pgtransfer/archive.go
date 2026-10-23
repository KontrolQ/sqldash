package pgtransfer

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (self *job) archiveList(input *os.File, size int64) (string, error) {
	listing, createError := self.createFile("archive-toc")
	if createError != nil {
		return "", createError
	}
	defer listing.Close()
	if listError := self.runClient(RestoreClient, []string{"--list"}, io.NewSectionReader(input, 0, size), listing, MaximumArchiveListBytes); listError != nil {
		return "", listError
	}
	if _, seekError := listing.Seek(0, io.SeekStart); seekError != nil {
		return "", TransferFailed
	}
	filtered, createError := self.createFile("archive-toc-filtered")
	if createError != nil {
		return "", createError
	}
	defer filtered.Close()
	reader := bufio.NewReaderSize(listing, CopyBufferBytes)
	omitted := 0
	for {
		line, readError := reader.ReadString('\n')
		if len(line) > WrapperInspectionBytes {
			return "", InvalidFormat
		}
		if publicSchemaEntry(line) {
			omitted++
		} else if _, writeError := io.WriteString(filtered, line); writeError != nil {
			return "", TransferFailed
		}
		if omitted > 1 {
			return "", InvalidFormat
		}
		if readError == io.EOF {
			break
		}
		if readError != nil {
			return "", TransferFailed
		}
		if contextError := self.context.Err(); contextError != nil {
			return "", contextError
		}
	}
	return filepath.Join(self.directory, "archive-toc-filtered"), nil
}

func publicSchemaEntry(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 7 || fields[3] != "SCHEMA" || fields[4] != "-" || fields[5] != "public" || !strings.HasSuffix(fields[0], ";") {
		return false
	}
	identifier, identifierError := strconv.ParseUint(strings.TrimSuffix(fields[0], ";"), 10, 32)
	catalog, catalogError := strconv.ParseUint(fields[1], 10, 32)
	object, objectError := strconv.ParseUint(fields[2], 10, 32)
	return identifierError == nil && identifier != 0 && catalogError == nil && catalog == 2615 && objectError == nil && object != 0
}
