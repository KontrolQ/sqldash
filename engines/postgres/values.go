package postgres

import (
	"math"
	"strconv"
	"strings"

	"sqldash/config"
	"sqldash/engines"
	"sqldash/sqld"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func modifiedRows(command pgconn.CommandTag) int64 {
	if command.Insert() || command.Update() || command.Delete() || strings.HasPrefix(command.String(), "MERGE ") || strings.HasPrefix(command.String(), "COPY ") {
		return command.RowsAffected()
	}
	return 0
}

func resultColumns(types *pgtype.Map, fields []pgconn.FieldDescription) []sqld.Column {
	columns := make([]sqld.Column, len(fields))
	for index, field := range fields {
		kind := ""
		if dataType, known := types.TypeForOID(field.DataTypeOID); known {
			kind = dataType.Name
		}
		columns[index] = sqld.Column{Name: field.Name, Kind: kind}
	}
	return columns
}

func normalizeResult(types *pgtype.Map, fields []pgconn.FieldDescription, rows [][][]byte) (*engines.Result, error) {
	if config.Query.MaximumRows > 0 && len(rows) > config.Query.MaximumRows {
		return nil, ResultLimit
	}
	result := &engines.Result{Columns: resultColumns(types, fields)}
	totalBytes := 0
	for _, values := range rows {
		row, rowBytes, normalizeError := normalizeRow(types, fields, values)
		if normalizeError != nil {
			return nil, normalizeError
		}
		totalBytes += rowBytes
		if config.Query.MaximumResultBytes > 0 && int64(totalBytes) > config.Query.MaximumResultBytes {
			return nil, ResultLimit
		}
		result.Rows = append(result.Rows, row)
	}
	return result, nil
}

func normalizeRow(types *pgtype.Map, fields []pgconn.FieldDescription, values [][]byte) ([]any, int, error) {
	if len(fields) != len(values) {
		return nil, 0, InvalidValue
	}
	row := make([]any, len(values))
	totalBytes := 0
	for index, value := range values {
		totalBytes += len(value)
		if value == nil {
			continue
		}
		normalized, normalizeError := normalizeValue(types, fields[index], value)
		if normalizeError != nil {
			return nil, 0, normalizeError
		}
		row[index] = normalized
	}
	return row, totalBytes, nil
}

func normalizeValue(types *pgtype.Map, field pgconn.FieldDescription, value []byte) (any, error) {
	if field.Format != pgtype.TextFormatCode {
		return nil, InvalidValue
	}
	switch field.DataTypeOID {
	case pgtype.Int2OID, pgtype.Int4OID, pgtype.Int8OID, pgtype.OIDOID, pgtype.XIDOID, pgtype.CIDOID, pgtype.XID8OID:
		integer, parseError := strconv.ParseInt(string(value), 10, 64)
		if parseError != nil {
			return string(value), nil
		}
		return integer, nil
	case pgtype.BoolOID:
		return string(value) == "t", nil
	case pgtype.Float4OID, pgtype.Float8OID:
		number, parseError := strconv.ParseFloat(string(value), 64)
		if parseError != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return string(value), nil
		}
		return number, nil
	case pgtype.ByteaOID:
		dataType, known := types.TypeForOID(field.DataTypeOID)
		if !known {
			return nil, InvalidValue
		}
		decoded, decodeError := dataType.Codec.DecodeValue(types, field.DataTypeOID, field.Format, value)
		if decodeError != nil {
			return nil, InvalidValue
		}
		return decoded, nil
	default:
		return string(value), nil
	}
}
