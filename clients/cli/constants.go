package main

import "strings"

const (
	defaultScanScanners = "axe,lighthouse,seo,link-checker"
)

// defaultScanScannerList is the slice form of defaultScanScanners, used as the
// --scanner flag default.
func defaultScanScannerList() []string {
	return strings.Split(defaultScanScanners, ",")
}
