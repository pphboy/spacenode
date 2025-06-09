#!/bin/bash
GOOS=windows GOARCH=amd64 go build
cp win.exe /mnt/d/Projects/go/spacenode/