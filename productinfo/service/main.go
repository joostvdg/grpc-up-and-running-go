package main

import (
	"os"
	"productinfo/service/internal"
)

const (
	defaulPort = ":50051"
)

func main() {
	port := defaulPort
	overridePort := os.Getenv("PORT")
	if overridePort != "" {
		port = ":" + overridePort
	}

	internal.StartGRPCServer(port)

}
