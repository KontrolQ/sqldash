package redis

import "sqldash/utils/collections"

func dataOptions(data *PageData) {
	data.ViewOptions = collections.OptionsOf("pretty", "Pretty", "raw", "Raw")
	data.ByteEncodings = collections.OptionsOf("text", "Text", "hex", "Hex", "base64", "Base64")
	data.KeyTypes = collections.OptionsOf("string", "String", "hash", "Hash", "list", "List", "set", "Set", "zset", "Sorted set", "stream", "Stream", "ReJSON-RL", "JSON")
	data.FilterKeyTypes = append([]collections.Option{{Value: "", Label: "All types"}}, data.KeyTypes...)
	data.ExpiryModes = collections.OptionsOf("persistent", "No expiry", "relative", "Expire after", "absolute", "Expire at")
	data.ExpiryUnits = collections.OptionsOf("1000", "Seconds", "60000", "Minutes", "3600000", "Hours", "86400000", "Days")
}
