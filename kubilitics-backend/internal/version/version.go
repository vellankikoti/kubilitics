package version

// Version is set at build time via -ldflags:
//
//	go build -ldflags="-X github.com/kubilitics/kubilitics-backend/internal/version.Version=0.1.0"
var Version = "dev"
