module github.com/mattboback/stageflow/libs/contracts/scanner-manifest

go 1.26.4

require github.com/santhosh-tekuri/jsonschema/v5 v5.3.1

replace github.com/mattboback/stageflow/libs/go/bootstrap => ../../../libs/go/bootstrap

replace github.com/mattboback/stageflow/libs/go/config => ../../../libs/go/config

replace github.com/mattboback/stageflow/libs/go/domain => ../../../libs/go/domain

replace github.com/mattboback/stageflow/libs/go/events => ../../../libs/go/events

replace github.com/mattboback/stageflow/libs/go/httputil => ../../../libs/go/httputil

replace github.com/mattboback/stageflow/libs/go/logging => ../../../libs/go/logging

replace github.com/mattboback/stageflow/libs/go/messaging => ../../../libs/go/messaging

replace github.com/mattboback/stageflow/libs/go/models => ../../../libs/go/models

replace github.com/mattboback/stageflow/libs/go/scannercatalog => ../../../libs/go/scannercatalog

replace github.com/mattboback/stageflow/libs/go/scannerregistry => ../../../libs/go/scannerregistry

replace github.com/mattboback/stageflow/libs/go/storage => ../../../libs/go/storage

replace github.com/mattboback/stageflow/libs/contracts/report/generated/go => ../../../libs/contracts/report/generated/go
