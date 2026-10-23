package pgtransfer

func dumpArguments(arguments []string, format, snapshot string, schemas []string) []string {
	if format == FormatSQL {
		format = "plain"
	}
	arguments = append(arguments, "--format="+format, "--no-owner", "--no-privileges", "--no-tablespaces", "--no-security-labels", "--large-objects", "--strict-names", "--snapshot="+snapshot)
	for _, schema := range schemas {
		arguments = append(arguments, "--schema="+schema)
	}
	return arguments
}
