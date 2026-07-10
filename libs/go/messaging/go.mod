module github.com/mattboback/stageflow/libs/go/messaging

go 1.26.5

replace github.com/mattboback/stageflow/libs/go/bootstrap => ../../../libs/go/bootstrap

replace github.com/mattboback/stageflow/libs/go/config => ../../../libs/go/config

replace github.com/mattboback/stageflow/libs/go/domain => ../../../libs/go/domain

replace github.com/mattboback/stageflow/libs/go/events => ../../../libs/go/events

replace github.com/mattboback/stageflow/libs/go/httputil => ../../../libs/go/httputil

replace github.com/mattboback/stageflow/libs/go/logging => ../../../libs/go/logging

replace github.com/mattboback/stageflow/libs/go/models => ../../../libs/go/models

replace github.com/mattboback/stageflow/libs/go/scannercatalog => ../../../libs/go/scannercatalog

replace github.com/mattboback/stageflow/libs/go/scannerregistry => ../../../libs/go/scannerregistry

replace github.com/mattboback/stageflow/libs/go/storage => ../../../libs/go/storage

require (
	github.com/mattboback/stageflow/libs/go/config v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/events v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/logging v0.0.0-00010101000000-000000000000
	github.com/nats-io/nats.go v1.52.0
)

require (
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/mattboback/stageflow/libs/go/models v0.0.0-00010101000000-000000000000 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
)

replace github.com/mattboback/stageflow/libs/contracts/scanner-manifest => ../../../libs/contracts/scanner-manifest

replace github.com/mattboback/stageflow/libs/contracts/report/generated/go => ../../../libs/contracts/report/generated/go
