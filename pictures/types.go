package pictures

type Picture struct {
	Base64    string
	MediaType string
}

type imageDetails struct {
	Width    int
	Height   int
	Format   string
	Animated bool
}
