package fixture

// headers are the threading headers a fixture message carries.
type headers struct {
	InReplyTo  string `json:"In-Reply-To,omitempty"`
	References string `json:"References,omitempty"`
}
