module github.com/panyam/vibrant

go 1.24.0

require (
	github.com/gorilla/websocket v1.5.0
	github.com/panyam/goutils v0.1.3
	github.com/panyam/templar v0.0.17
	github.com/spf13/cobra v1.9.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
)

replace github.com/panyam/templar v0.0.17 => ./locallinks/templar/

// replace github.com/panyam/s3gen v0.0.28 => ./locallinks/s3gen/
