package pgtransfer

import (
	"bufio"
	"io"
	"os"
	"strings"
)

func nativePublicSchemaRange(input *os.File, start, finish int64) (int64, int64) {
	reader := bufio.NewReaderSize(io.NewSectionReader(input, start, min(finish-start, int64(WrapperInspectionBytes))), WrapperInspectionBytes)
	position := start
	pendingPublic := false
	for {
		line, readError := reader.ReadSlice('\n')
		if readError != nil {
			return 0, 0
		}
		lineStart := position
		position += int64(len(line))
		text := strings.TrimSuffix(strings.TrimSuffix(string(line), "\n"), "\r")
		if text == "" || text == "--" {
			continue
		}
		if strings.HasPrefix(text, "-- Name: ") {
			if !strings.Contains(text, "; Type: SCHEMA; Schema: -; Owner: ") {
				return 0, 0
			}
			pendingPublic = strings.HasPrefix(text, "-- Name: public; Type: SCHEMA; Schema: -; Owner: ")
			continue
		}
		if strings.HasPrefix(text, "-- ") {
			continue
		}
		if pendingPublic && text == "CREATE SCHEMA public;" {
			return lineStart, position
		}
		if safeDumpPrologue(text) || safeSchemaCreation(text) {
			pendingPublic = false
			continue
		}
		return 0, 0
	}
}

func safeDumpPrologue(line string) bool {
	switch line {
	case "SET statement_timeout = 0;", "SET lock_timeout = 0;", "SET idle_in_transaction_session_timeout = 0;", "SET transaction_timeout = 0;",
		"SET standard_conforming_strings = on;", "SELECT pg_catalog.set_config('search_path', '', false);", "SET check_function_bodies = false;",
		"SET xmloption = content;", "SET client_min_messages = warning;", "SET row_security = off;":
		return true
	}
	if strings.HasPrefix(line, "SET client_encoding = '") && strings.HasSuffix(line, "';") {
		value := strings.TrimSuffix(strings.TrimPrefix(line, "SET client_encoding = '"), "';")
		if value == "" {
			return false
		}
		for _, character := range value {
			if character != '_' && character != '-' && (character < '0' || character > '9') && (character < 'A' || character > 'Z') && (character < 'a' || character > 'z') {
				return false
			}
		}
		return true
	}
	return false
}

func safeSchemaCreation(line string) bool {
	if !strings.HasPrefix(line, "CREATE SCHEMA ") || !strings.HasSuffix(line, ";") {
		return false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(line, "CREATE SCHEMA "), ";")
	if name == "" {
		return false
	}
	if name[0] == '"' {
		for position := 1; position < len(name); position++ {
			if name[position] != '"' {
				continue
			}
			if position+1 < len(name) && name[position+1] == '"' {
				position++
				continue
			}
			return position == len(name)-1
		}
		return false
	}
	for _, character := range name {
		if character != '_' && (character < '0' || character > '9') && (character < 'A' || character > 'Z') && (character < 'a' || character > 'z') {
			return false
		}
	}
	return true
}
