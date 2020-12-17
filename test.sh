#!/bin/bash

(
  go mod vendor
  go mod tidy
  cd cmd/filter
  rmdir dfilter
  mkdir dfilter
  ( go run main.go dfilter /tmp ) &
  sleep 2
  ls -al dfilter
  umount dfilter
) 
