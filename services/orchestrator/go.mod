module github.com/mattboback/stageflow/platform/orchestrator

go 1.26.1

require (
	github.com/fergusstrange/embedded-postgres v1.33.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.8.0
	github.com/mattboback/stageflow/libs/contracts/report/generated/go v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/bootstrap v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/config v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/domain v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/events v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/httputil v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/logging v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/messaging v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/models v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/scannercatalog v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/scannerregistry v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/storage v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.18.3 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mattboback/stageflow/libs/contracts/scanner-manifest v0.0.0-00010101000000-000000000000 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.0.99 // indirect
	github.com/nats-io/nats.go v1.49.0 // indirect
	github.com/nats-io/nkeys v0.4.12 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/tinylib/msgp v1.6.1 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/mattboback/stageflow/libs/go/bootstrap => ../../libs/go/bootstrap

replace github.com/mattboback/stageflow/libs/go/config => ../../libs/go/config

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
