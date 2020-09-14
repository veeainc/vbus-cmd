#!/bin/sh

if [ -z "$1" ]
then
      echo "No version number provided, just building binaries"
else
    version="$1"
    echo "version $version"
    echo "modifying version number in version.go"
    sed -i 's/\([[:blank:]]version = "\)[^"]*"/\1'"$version"'"/' ../version.go

    echo "comitting changes"
    git add ../version.go
    git commit -m "version $version"

    echo "tagging git repository "
    git tag $version

    echo "pushing changes"
    git push
    git push origin $version
fi

echo "building binaries"
( cd .. ;  go build -o linux-x64-vbus-cmd -ldflags "-s -w")
mv ../linux-x64-vbus-cmd .

( cd .. ; env GOOS=linux GOARCH=arm64 go build -o elf-linux-arm64-vbus-cmd -ldflags "-s -w")
mv ../elf-linux-arm64-vbus-cmd .

( cd .. ; env GOOS=linux GOARCH=arm go build -o elf-linux-arm-vbus-cmd -ldflags "-s -w")
mv ../elf-linux-arm-vbus-cmd .

(cd ..; env GOOS=darwin GOARCH=amd64 go build -o darwin-x64-vbus-cmd -ldflags "-s -w")
mv ../darwin-x64-vbus-cmd .

echo "done"
