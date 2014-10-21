#!/bin/sh
mkdir -p ~/Buildspace/cloudio
export GOROOT=/usr/local/go
export GOPATH=~/Buildspace/cloudio
export GOBIN=~/bin
git clone https://github.com/stormstack/cloudio.git
cd $GOPATH/src/stormstack.org/cloudio
go get .
go install
