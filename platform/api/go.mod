module github.com/mattboback/stageflow/platform/api

go 1.25

replace github.com/mattboback/stageflow/packages/shared-go => ../../packages/shared-go

replace github.com/mattboback/stageflow/packages/contracts/report/generated/go => ../../packages/contracts/report/generated/go

replace github.com/mattboback/stageflow/packages/contracts/scanner-manifest => ../../packages/contracts/scanner-manifest

require (
	github.com/google/uuid v1.6.0
	github.com/mattboback/stageflow/packages/contracts/report/generated/go v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/packages/shared-go v0.0.0-00010101000000-000000000000
	github.com/mattn/go-sqlite3 v1.14.34
	github.com/nats-io/nats-server/v2 v2.12.3
)

require (
	github.com/antithesishq/antithesis-sdk-go v0.5.0-default-no-op // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/klauspost/compress v1.18.3 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattboback/stageflow/packages/contracts/scanner-manifest v0.0.0-00010101000000-000000000000 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/highwayhash v1.0.4-0.20251030100505-070ab1a87a76 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.0.98 // indirect
	github.com/nats-io/jwt/v2 v2.8.0 // indirect
	github.com/nats-io/nats.go v1.48.0 // indirect
	github.com/nats-io/nkeys v0.4.12 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/tinylib/msgp v1.6.1 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
