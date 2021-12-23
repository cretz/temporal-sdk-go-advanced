
TODO(cretz): Protoc-based code generation for workflow

Rebuild SDK protos, from this dir

    protoc --go_out=paths=source_relative:. temporalpb/sdk.proto

Rebuild test protos, but local dir on the PATH, then:

    go build ./cmd/protoc-gen-go-temporal && protoc --go_out=paths=source_relative:. --go_temporal_out=paths=source_relative:. ./test/simplepb/simple.proto