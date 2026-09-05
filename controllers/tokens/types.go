package tokens

type MintRequest struct {
	Label string `form:"label"`
	Scope string `form:"scope"`
}
