module github.com/mattboback/stageflow/libs/go/storage

go 1.26.2

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

require (
	github.com/mattboback/stageflow/libs/go/config v0.0.0-00010101000000-000000000000
	github.com/minio/minio-go/v7 v7.0.99
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.3 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tinylib/msgp v1.6.1 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/mattboback/stageflow/libs/go/storage => ../../../libs/go/storage

replace github.com/mattboback/stageflow/libs/contracts/scanner-manifest => ../../../libs/contracts/scanner-manifest

replace github.com/mattboback/stageflow/libs/contracts/report/generated/go => ../../../libs/contracts/report/generated/go
