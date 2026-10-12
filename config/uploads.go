package config

func RequestBodyLimit() int {
	return max(MinimumRequestBodyLimit, Server.MaximumImageUploadBytes+MultipartUploadOverhead)
}
