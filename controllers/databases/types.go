package databases

type CreateRequest struct {
	Name string `form:"name"`
}

type SettingsRequest struct {
	BlockReads  string `form:"block_reads"`
	BlockWrites string `form:"block_writes"`
	BlockReason string `form:"block_reason"`
	AllowAttach string `form:"allow_attach"`
	Protected   string `form:"protected"`
}

type ImportRequest struct {
	Name string `form:"name"`
}

type ForkRequest struct {
	Name   string `form:"name"`
	Moment string `form:"moment"`
}

type RuleRequest struct {
	Network string `form:"network"`
	Note    string `form:"note"`
}
