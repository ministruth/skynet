// Package sn provide common interface for skynet and plugins. All plugins should ONLY
// use this package to interact with skynet to prevent golang complain version mismatch.
//
// This package will not be updated for each patch version change, and for minor version,
// sometimes we will change and sometimes not, so you shouldn't relay on this promise.
// When this package changes, all plugins need to update to be compatible with new version,
// otherwise golang will refuse to load.
package sn
