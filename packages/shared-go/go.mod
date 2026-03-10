module github.com/mattboback/stageflow/packages/shared-go

go 1.25.4

replace github.com/mattboback/stageflow/packages/contracts/scanner-manifest => ../contracts/scanner-manifest

replace github.com/mattboback/stageflow/packages/contracts/report/generated/go => ../contracts/report/generated/go

require (
	github.com/mattboback/stageflow/packages/contracts/report/generated/go v0.0.0-00010101000000-000000000000
	github.com/mattboback/stageflow/packages/contracts/scanner-manifest v0.0.0-00010101000000-000000000000
	github.com/minio/minio-go/v7 v7.0.98
	github.com/nats-io/nats.go v1.48.0
	gopkg.in/yaml.v3 v3.0.1
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
	github.com/nats-io/nkeys v0.4.12 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tinylib/msgp v1.6.1 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
