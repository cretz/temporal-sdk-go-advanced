module github.com/cretz/temporal-sdk-go-advanced/temporalsqlite

go 1.17

require (
	crawshaw.io/sqlite v0.0.0-00010101000000-000000000000
	github.com/cretz/temporal-sdk-go-advanced/temporalproto v0.0.0-00010101000000-000000000000
	github.com/cretz/temporal-sdk-go-advanced/temporalutil v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.3.0
	github.com/pierrec/lz4/v4 v4.1.12
	go.temporal.io/sdk v1.12.0
	google.golang.org/protobuf v1.27.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/gogo/status v1.1.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	go.temporal.io/api v1.5.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	golang.org/x/net v0.0.0-20210913180222-943fd674d43e // indirect
	golang.org/x/sys v0.0.0-20210910150752-751e447fb3d0 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/genproto v0.0.0-20210909211513-a8c4777a87af // indirect
	google.golang.org/grpc v1.40.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace (
	// For serialize, at least until https://github.com/crawshaw/sqlite/pull/132 accepted
	crawshaw.io/sqlite => github.com/cretz/crawshaw-sqlite v0.3.3-0.20211227191323-903935e86940
	github.com/cretz/temporal-sdk-go-advanced/temporalproto => ../temporalproto
	github.com/cretz/temporal-sdk-go-advanced/temporalutil => ../temporalutil
)
