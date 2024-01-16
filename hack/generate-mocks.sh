#!/bin/bash
go get -u go.uber.org/mock/mockgen
mockgen -package logr github.com/go-logr/logr Logger  > pkg/mock/logr/logr_mock.go
