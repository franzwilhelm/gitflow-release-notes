package githubutil

import (
	"errors"
	"fmt"
	"strings"
)

// Repository holds the owner and name of a Github repository
type Repository struct {
	Owner string
	Name  string
}

// String is part of the Value interface for cobra custom flags
func (r *Repository) String() string {
	return ""
}

// Set is part of the Value interface for cobra custom flags
func (r *Repository) Set(input string) error {
	gitRefs := strings.Split(input, "/")
	if len(gitRefs) != 2 {
		return errors.New("Invalid repository format. Example: franzwilhelm/gitflow-release-notes")
	}
	r.Owner = gitRefs[0]
	r.Name = gitRefs[1]
	return nil
}

// Type is part of the Value interface for cobra custom flags
func (r *Repository) Type() string {
	return ""
}

// Full returns the full repository name with the owner included
func (r *Repository) Full() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}
