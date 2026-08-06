module github.com/Rajeevlang/ecommercesagapattern/order

go 1.25.0

require (
	github.com/Rajeevlang/ecommercesagapattern/shared v0.0.0-20260727180943-2790dc835c06
	github.com/jackc/pgx/v5 v5.10.0
	github.com/segmentio/kafka-go v0.4.47
	google.golang.org/grpc v1.82.1
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
)

replace github.com/Rajeevlang/ecommercesagapattern/shared => ../shared
