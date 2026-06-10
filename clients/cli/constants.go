package main

import "strings"

const (
	jobStateDone   = "DONE"
	jobStateFailed = "FAILED"

	defaultScanScanners   = "axe,lighthouse,seo,link-checker"
	defaultMaxIssues      = 200
	defaultMaxOccurrences = 3
)

// defaultScanScannerList is the slice form of defaultScanScanners, used as the
// --scanner flag default.
func defaultScanScannerList() []string {
	return strings.Split(defaultScanScanners, ",")
}
