package fixture

import "strings"

const pathSeparator = "/"

// folder is one mailbox of an account; nested local mailboxes use Parent/Child names.
type folder struct {
	Account string `json:"account"`
	Name    string `json:"name"`
}

func (folder folder) leaf() string {
	return folder.Name[strings.LastIndex(folder.Name, pathSeparator)+1:]
}
