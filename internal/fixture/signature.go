package fixture

// signature is a Mail signature that drafts may name.
type signature struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}
