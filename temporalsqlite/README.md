
TODO(cretz): Embedded sqlite DB as part of a Temporal workflow

Regen protos:

    go install ../temporalproto/cmd/protoc-gen-go_temporal

    protoc --go_out=paths=source_relative:. --go_temporal_out=paths=source_relative:. -I . -I ../temporalproto ./sqlitepb/sqlite.proto