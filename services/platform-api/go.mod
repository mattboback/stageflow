module github.com/mattboback/stageflow/services/platform-api

go 1.26.4

require (
	github.com/google/uuid v1.6.0
	github.com/mattboback/stageflow/libs/contracts/report/generated/go v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/bootstrap v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/config v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/diff v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/domain v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/events v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/httputil v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/logging v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/messaging v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/models v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/scannerregistry v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/libs/go/storage v0.0.0-00010101000000-000000000000
	github.com/mattn/go-sqlite3 v1.14.44
	github.com/nats-io/nats-server/v2 v2.14.0
)

require (
	github.com/antithesishq/antithesis-sdk-go v0.7.0-default-no-op // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/mattboback/stageflow/libs/contracts/scanner-manifest v0.0.0-00010101000000-000000000000 // indirect
	github.com/mattboback/stageflow/libs/go/scannercatalog v0.0.0-00010101000000-000000000000 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/highwayhash v1.0.4 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.1.0 // indirect
	github.com/nats-io/jwt/v2 v2.8.1 // indirect
	github.com/nats-io/nats.go v1.52.0 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/tinylib/msgp v1.6.1 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
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
