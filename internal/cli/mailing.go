package cli

// mailing holds the templated fields of a merge, one draft per CSV row.
type mailing struct {
	to      []string
	cc      []string
	subject string
	dryRun  bool
}
