package cards

// Media is an asset attached to a card. Paths are filesystem paths as seen at
// import time; callers normalise before display.
type Media struct {
	Kind   string
	Path   string
	SHA256 string
	Mime   string
}
