package databases

import (
	"fmt"
	"strconv"
	"strings"
)

func readableSize(bytes int64) string {
	if bytes < KilobyteSize {
		return fmt.Sprintf(BytesFormat, bytes)
	}

	value := float64(bytes)
	units := []string{KilobyteUnit, MegabyteUnit, GigabyteUnit, TerabyteUnit}

	for _, unit := range units {
		value /= KilobyteSize

		if value < KilobyteSize {
			return fmt.Sprintf(SizeFormat, value, unit)
		}
	}

	return fmt.Sprintf(SizeFormat, value, TerabyteUnit)
}

func readableCount(value int64) string {
	if value < 0 {
		return "-" + readableCount(-value)
	}

	switch {
	case value >= Billion:
		return trimZeros(float64(value)/Billion) + BillionSuffix
	case value >= Million:
		return trimZeros(float64(value)/Million) + MillionSuffix
	case value >= Thousand:
		return trimZeros(float64(value)/Thousand) + ThousandSuffix
	}

	return strconv.FormatInt(value, 10)
}

func trimZeros(value float64) string {
	text := fmt.Sprintf(CompactFormat, value)

	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimSuffix(text, ".")
	}

	return text
}
