package main

import (
	"os"

	"shellsage/internal/app"
)

// Version is overridden at build time via -ldflags "-X main.Version=vX.Y.Z".
var Version = "4.1.0"

func main() {
	os.Exit(app.Run(Version, os.Args[1:]))
}
