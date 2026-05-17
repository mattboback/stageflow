module github.com/mattboback/stageflow/clients/cli

go 1.26.3

require (
	github.com/mattboback/stageflow/libs/contracts/report/generated/go v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/diff v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/models v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/mattboback/stageflow/libs/go/bootstrap => ../../libs/go/bootstrap

replace github.com/mattboback/stageflow/libs/go/config => ../../libs/go/config

replace github.com/mattboback/stageflow/libs/go/diff => ../../libs/go/diff

replace github.com/mattboback/stageflow/libs/go/domain => ../../libs/go/domain

replace github.com/mattboback/stageflow/libs/go/events => ../../libs/go/events

replace github.com/mattboback/stageflow/libs/go/httputil => ../../libs/go/httputil

replace github.com/mattboback/stageflow/libs/go/logging => ../../libs/go/logging

replace github.com/mattboback/stageflow/libs/go/messaging => ../../libs/go/messaging

replace github.com/mattboback/stageflow/libs/go/models => ../../libs/go/models

replace github.com/mattboback/stageflow/libs/go/scannercatalog => ../../libs/go/scannercatalog

replace github.com/mattboback/stageflow/libs/go/scannerregistry => ../../libs/go/scannerregistry

replace github.com/mattboback/stageflow/libs/go/storage => ../../libs/go/storage

replace github.com/mattboback/stageflow/libs/contracts/scanner-manifest => ../../libs/contracts/scanner-manifest

replace github.com/mattboback/stageflow/libs/contracts/report/generated/go => ../../libs/contracts/report/generated/go
