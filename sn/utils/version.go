package utils

import (
	"github.com/hashicorp/go-version"
	"github.com/ztrue/tracerr"
)

// CheckVersion checks ver with constraint string.
//	CheckVersion("1.0.1", ">= 1.0, < 1.1")
func CheckVersion(ver string, constraint string) (bool, error) {
	v, err := version.NewVersion(ver)
	if err != nil {
		return false, tracerr.Wrap(err)
	}
	cst, err := version.NewConstraint(constraint)
	if err != nil {
		return false, tracerr.Wrap(err)
	}
	return cst.Check(v), nil
}
