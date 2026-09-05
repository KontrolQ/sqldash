package tags

import (
	"strconv"
	"time"

	"github.com/flosch/pongo2/v6"
)

func timeago(value *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	moment, valid := value.Interface().(time.Time)
	if !valid || moment.IsZero() {
		return pongo2.AsValue(""), nil
	}

	return pongo2.AsValue(relativeLabel(time.Since(moment))), nil
}

func relativeLabel(elapsed time.Duration) string {
	switch {
	case elapsed < time.Minute:
		return JustNowLabel
	case elapsed < time.Hour:
		return strconv.Itoa(int(elapsed.Minutes())) + MinutesSuffix
	case elapsed < 24*time.Hour:
		return strconv.Itoa(int(elapsed.Hours())) + HoursSuffix
	case elapsed < 7*24*time.Hour:
		return strconv.Itoa(int(elapsed.Hours())/24) + DaysSuffix
	case elapsed < 30*24*time.Hour:
		return strconv.Itoa(int(elapsed.Hours())/(24*7)) + WeeksSuffix
	case elapsed < 365*24*time.Hour:
		return strconv.Itoa(int(elapsed.Hours())/(24*30)) + MonthsSuffix
	default:
		return strconv.Itoa(int(elapsed.Hours())/(24*365)) + YearsSuffix
	}
}
