package bytesize

import (
	"math/big"
	"strconv"
	"strings"
)

func Parse(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if len(value) > 80 {
		return 0, Invalid
	}
	parts := pattern.FindStringSubmatch(value)
	if len(parts) != 3 {
		return 0, Invalid
	}
	multiplier, known := multipliers[strings.ToLower(parts[2])]
	if !known {
		return 0, Invalid
	}
	number, valid := new(big.Rat).SetString(parts[1])
	if !valid {
		return 0, Invalid
	}
	number.Mul(number, big.NewRat(multiplier, 1))
	if !number.IsInt() || !number.Num().IsInt64() {
		return 0, Invalid
	}
	return number.Num().Int64(), nil
}

func Format(value int64) string {
	for _, unit := range binaryUnits {
		if value >= unit.Bytes && value%unit.Bytes == 0 {
			return strconv.FormatInt(value/unit.Bytes, 10) + " " + unit.Label
		}
	}
	for _, unit := range decimalUnits {
		if value >= unit.Bytes {
			number := new(big.Rat).SetFrac(big.NewInt(value), big.NewInt(unit.Bytes))
			text := strings.TrimRight(strings.TrimRight(number.FloatString(12), "0"), ".")
			return text + " " + unit.Label
		}
	}
	return strconv.FormatInt(value, 10) + " B"
}
