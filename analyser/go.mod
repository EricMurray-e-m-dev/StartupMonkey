module github.com/EricMurray-e-m-dev/StartupMonkey/analyser

go 1.25.1

require (
	github.com/EricMurray-e-m-dev/StartupMonkey/proto v0.0.0-20251013104841-b2acacaf2bd5
	github.com/stretchr/testify v1.11.1
	google.golang.org/grpc v1.76.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Add this line at the bottom:
replace github.com/EricMurray-e-m-dev/StartupMonkey/proto => ../proto
