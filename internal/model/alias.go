package model

// AliasTarget points an alias such as "stable" at a concrete model version.
type AliasTarget struct {
	Model   string
	Version string
}

// Key returns a stable identifier for the alias target.
func (a AliasTarget) Key() string {
	return a.Model + "/" + a.Version
}
