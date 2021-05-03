package utils

import (
	"github.com/hashicorp/go-version"
)

const VERSION = "1.0.0"

func CheckSkynetVersion(c string) (bool, error) {
	return CheckVersion(VERSION, c)
}

func CheckVersion(v string, c string) (bool, error) {
	ver, err := version.NewVersion(v)
	if err != nil {
		return false, err
	}
	cst, err := version.NewConstraint(c)
	if err != nil {
		return false, err
	}
	return cst.Check(ver), nil
}
