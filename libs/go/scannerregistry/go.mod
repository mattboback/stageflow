module github.com/mattboback/stageflow/libs/go/scannerregistry

go 1.26.4

replace github.com/mattboback/stageflow/libs/go/bootstrap => ../../../libs/go/bootstrap

replace github.com/mattboback/stageflow/libs/go/config => ../../../libs/go/config

replace github.com/mattboback/stageflow/libs/go/domain => ../../../libs/go/domain

replace github.com/mattboback/stageflow/libs/go/events => ../../../libs/go/events

replace github.com/mattboback/stageflow/libs/go/httputil => ../../../libs/go/httputil

replace github.com/mattboback/stageflow/libs/go/logging => ../../../libs/go/logging

replace github.com/mattboback/stageflow/libs/go/messaging => ../../../libs/go/messaging

replace github.com/mattboback/stageflow/libs/go/models => ../../../libs/go/models

replace github.com/mattboback/stageflow/libs/go/scannercatalog => ../../../libs/go/scannercatalog

replace github.com/mattboback/stageflow/libs/go/storage => ../../../libs/go/storage

require (
	github.com/mattboback/stageflow/libs/go/scannercatalog v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mattboback/stageflow/libs/contracts/scanner-manifest v0.0.0-00010101000000-000000000000 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/mattboback/stageflow/libs/go/scannerregistry => ../../../libs/go/scannerregistry

replace github.com/mattboback/stageflow/libs/contracts/scanner-manifest => ../../../libs/contracts/scanner-manifest

replace github.com/mattboback/stageflow/libs/contracts/report/generated/go => ../../../libs/contracts/report/generated/go
