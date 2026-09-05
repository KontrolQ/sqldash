package main

import (
	"sqldash/proxy"
	"sqldash/telemetry"
)

func runProxy() {
	telemetry.Start()
	proxy.Serve()
}
