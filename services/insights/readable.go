package insights

import (
	"fmt"
	"strconv"
	"strings"
)

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
	places := CompactFormat
	if value >= TightAbove {
		places = TightFormat
	}

	text := fmt.Sprintf(places, value)

	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimSuffix(text, ".")
	}

	return text
}

func readableChange(now int64, before int64) (string, string) {
	if before == 0 {
		if now == 0 {
			return "", ChangeSteady
		}

		return NewChange, ChangeUp
	}

	share := (float64(now) - float64(before)) / float64(before) * 100

	switch {
	case share > 0.05:
		return fmt.Sprintf(RiseFormat, share), ChangeUp
	case share < -0.05:
		return fmt.Sprintf(FallFormat, -share), ChangeDown
	}

	return SteadyChange, ChangeSteady
}
