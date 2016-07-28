#!/bin/bash
CGO_ENABLED=0 go build -a -installsuffix cgo .

#docker build -t nhurel/dim:latest .