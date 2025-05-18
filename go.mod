module github.com/panyam/vibrant

go 1.24.0

require (
	github.com/gorilla/websocket v1.5.0
	github.com/panyam/goutils v0.1.3
	github.com/panyam/templar v0.0.17
	github.com/spf13/cobra v1.9.1
	golang.design/x/clipboard v0.7.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	golang.org/x/exp v0.0.0-20190731235908-ec7cb31e5a56 // indirect
	golang.org/x/image v0.6.0 // indirect
	golang.org/x/mobile v0.0.0-20230301163155-e0f57694e12c // indirect
	golang.org/x/sys v0.19.0 // indirect
)

replace github.com/panyam/templar v0.0.17 => ./locallinks/templar/

// replace github.com/panyam/s3gen v0.0.28 => ./locallinks/s3gen/
