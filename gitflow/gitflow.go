package gitflow

import "strings"

const (
	// Feature is the branch name prefix standard for GitFlow feature branches
	Feature = "feature"
	// Bugfix is the branch name prefix standard for GitFlow bugfix branches
	Bugfix = "bugfix"
	// Hotfix is the branch name prefix standard for GitFlow hotfix branches
	Hotfix = "hotfix"
	// Release is the branch name prefix standard for GitFlow release branches
	Release = "release"
)

var prefixes = []string{Feature, Bugfix, Hotfix, Release}
var dashRemover = strings.NewReplacer("-", " ", "_", " ")

// RemovePrefixes removes GitFlow prefixes from strings
// Example input: Feature/new_logIN-pages
// Example output: New Login Pages
func RemovePrefixes(s string) string {
	s = strings.ToLower(s)
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix+"/") {
			s = strings.TrimPrefix(s, prefix+"/")
			s = dashRemover.Replace(s)
			break
		}
	}
	return strings.Title(s)
}
