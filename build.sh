#!/bin/bash

GOOS=linux GOARCH=amd64 go build -o dist/adslproxy_agent cmd/agent.go
GOOS=linux GOARCH=amd64 go build -o dist/adslproxy_server cmd/server.go
