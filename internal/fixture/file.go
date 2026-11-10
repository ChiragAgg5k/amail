package fixture

// file is an attachment of a fixture message, stored as text.
type file struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data string `json:"data"`
}
