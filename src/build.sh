#!/bin/bash

export     GOPATH=$HOME/gocode
go build dockerJVM.go && mv dockerJVM ..
