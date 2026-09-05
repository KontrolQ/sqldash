package collections

type Option struct {
	Value string
	Label string
}

func OptionsOf(pairs ...string) []Option {
	options := make([]Option, 0, len(pairs)/2)

	for index := 0; index+1 < len(pairs); index += 2 {
		options = append(options, Option{Value: pairs[index], Label: pairs[index+1]})
	}

	return options
}

func LabelledOptions(values []string) []Option {
	options := make([]Option, 0, len(values))

	for _, value := range values {
		options = append(options, Option{Value: value, Label: value})
	}

	return options
}
