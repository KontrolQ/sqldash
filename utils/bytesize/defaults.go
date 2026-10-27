package bytesize

import "regexp"

type unit struct {
	Label string
	Bytes int64
}

const MaximumFractionDigits = 40

var pattern = regexp.MustCompile(`^([0-9]+(?:\.[0-9]{1,40})?)\s*([a-zA-Z]*)$`)
var multipliers = map[string]int64{"": 1, "b": 1, "byte": 1, "bytes": 1, "kb": 1000, "mb": 1000000, "gb": 1000000000, "tb": 1000000000000, "kib": 1024, "mib": 1048576, "gib": 1073741824, "tib": 1099511627776}
var binaryUnits = []unit{{"TiB", 1099511627776}, {"GiB", 1073741824}, {"MiB", 1048576}, {"KiB", 1024}}
var decimalUnits = []unit{{"TB", 1000000000000}, {"GB", 1000000000}, {"MB", 1000000}, {"KB", 1000}}
