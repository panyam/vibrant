module github.com/panyam/viberunner

go 1.24.0

require (
	github.com/panyam/goutils v0.1.3
	github.com/panyam/templar v0.0.17
)

replace github.com/panyam/templar v0.0.17 => ./locallinks/templar/

// replace github.com/panyam/s3gen v0.0.28 => ./locallinks/s3gen/
