package internal

// This file exists so `go generate ./...` is part of the repo from day one.
// Add //go:generate mockgen directives next to the interfaces they mock
// (DockerClient, GitRunner, StateInferer, Elevator, progress.Source, ...).
//
// Example (added later, next to the interface):
//
//	//go:generate go run go.uber.org/mock/mockgen -source=client.go -destination=mock_client.go -package=dockerapi
