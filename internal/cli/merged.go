package cli

// merged is one row of a merge's report.
type merged struct {
	Row     int      `json:"row"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	// Body is reported on a dry run only.
	Body string `json:"body,omitempty"`
}
