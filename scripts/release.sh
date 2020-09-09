#!/bin/sh -x

( cd ../vbus-cmd ;  go build -o linux-x64-vbus-cmd -ldflags "-s -w")
mv ../vbus-cmd/linux-x64-vbus-cmd .

( cd ../vbus-cmd ; env GOOS=linux GOARCH=arm64 go build -o elf-linux-arm64-vbus-cmd -ldflags "-s -w")
mv ../vbus-cmd/elf-linux-arm64-vbus-cmd .


( cd ../vbus-cmd ; env GOOS=linux GOARCH=arm go build -o elf-linux-arm-vbus-cmd -ldflags "-s -w")
mv ../vbus-cmd/elf-linux-arm-vbus-cmd .

(cd ../vbus-cmd; env GOOS=darwin GOARCH=amd64 go build -o darwin-x64-vbus-cmd -ldflags "-s -w")
mv ../vbus-cmd/darwin-x64-vbus-cmd .
