#!/bin/bash

(
  go mod vendor
  go mod tidy
  cd cmd/filter
  rmdir dfilter
  mkdir dfilter
  ( go run main.go dfilter originalData $1 ) &
  sleep 2
  for f in dfilter/.rego-*
  do
    echo
    echo //$f
    cat $f
  done
  echo
  ls -al dfilter
  umount dfilter
) 
