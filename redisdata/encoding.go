package redisdata

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/hex"
	"io"
	"time"
)

func Encode(codec string, value []byte) ([]byte, error) {
	if len(value) > maximumBytes {
		return nil, LimitExceeded
	}
	switch codec {
	case "base64":
		return []byte(base64.StdEncoding.EncodeToString(value)), nil
	case "hex":
		return []byte(hex.EncodeToString(value)), nil
	case "gzip":
		var output bytes.Buffer
		compressor := gzip.NewWriter(&output)
		if _, err := compressor.Write(value); err != nil {
			return nil, err
		}
		if err := compressor.Close(); err != nil {
			return nil, err
		}
		return output.Bytes(), nil
	default:
		return nil, Unsupported
	}
}

func Decode(ctx context.Context, codec string, value []byte, options DecodeOptions) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if options.MaxBytes == 0 {
		options.MaxBytes = defaultBytes
	}
	if options.MaxRatio == 0 {
		options.MaxRatio = maximumDecodeRatio
	}
	if options.MaxBytes < 1 || options.MaxBytes > maximumBytes || options.MaxRatio < 1 || options.MaxRatio > maximumDecodeRatio || len(value) > maximumBytes*2 {
		return nil, LimitExceeded
	}
	var reader io.Reader
	switch codec {
	case "base64":
		reader = base64.NewDecoder(base64.StdEncoding.Strict(), bytes.NewReader(value))
	case "hex":
		reader = hex.NewDecoder(bytes.NewReader(value))
	case "gzip":
		decompressor, err := gzip.NewReader(bytes.NewReader(value))
		if err != nil {
			return nil, InvalidInput
		}
		defer decompressor.Close()
		reader = decompressor
	default:
		return nil, Unsupported
	}
	limit := min(options.MaxBytes, max(1, len(value))*options.MaxRatio)
	deadline := time.Now().Add(maximumDecodeTime)
	output := make([]byte, 0, min(limit, 4096))
	buffer := make([]byte, 4096)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, context.DeadlineExceeded
		}
		count, err := reader.Read(buffer[:min(len(buffer), limit-len(output)+1)])
		if len(output)+count > limit {
			return nil, LimitExceeded
		}
		output = append(output, buffer[:count]...)
		if err == io.EOF {
			return output, nil
		}
		if err != nil {
			return nil, InvalidInput
		}
	}
}
