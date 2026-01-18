#!/usr/bin/env bash

VERSION=$(git describe --tags)

if [ $VERSION = ""]; then
    VERSION="dev"
fi

function build() {
    printf "Compiling version ${VERSION} for ${1}/${2}\t"
    GOOS=${1} GOARCH=${2} go build -ldflags "-X 'main.Version=${VERSION}'" -o ./dist/${VERSION}/journal_${1}_${3}
    echo "✅"
}

echo Output dir: ./dist/${VERSION}/

build linux "386" i386
build linux amd64 amd64
build linux arm arm
build linux arm64 arm64
