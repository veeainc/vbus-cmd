#!/bin/sh -x

( cd .. ;  go build -o linux-x64-vbus-cmd -ldflags "-s -w")
mv ../linux-x64-vbus-cmd .

( cd .. ; env GOOS=linux GOARCH=arm64 go build -o elf-linux-arm64-vbus-cmd -ldflags "-s -w")
mv ../elf-linux-arm64-vbus-cmd .


( cd .. ; env GOOS=linux GOARCH=arm go build -o elf-linux-arm-vbus-cmd -ldflags "-s -w")
mv ../elf-linux-arm-vbus-cmd .

(cd ..; env GOOS=darwin GOARCH=amd64 go build -o darwin-x64-vbus-cmd -ldflags "-s -w")
mv ../darwin-x64-vbus-cmd .
