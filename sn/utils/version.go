package utils

import (
	"skynet/sn"

	"github.com/hashicorp/go-version"
)

// CheckSkynetVersion checks skynet version with constraint string.
//	CheckSkynetVersion(">= 1.0, < 1.1")
func CheckSkynetVersion(constraint string) (bool, error) {
	return CheckVersion(sn.VERSION, constraint)
}

// CheckVersion checks ver with constraint string.
//	CheckVersion("1.0.1", ">= 1.0, < 1.1")
func CheckVersion(ver string, constraint string) (bool, error) {
	v, err := version.NewVersion(ver)
	if err != nil {
		return false, err
	}
	cst, err := version.NewConstraint(constraint)
	if err != nil {
		return false, err
	}
	return cst.Check(v), nil
}
