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
  echo
  echo cat dfilter/notice.txt:
  cat dfilter/notice.txt
  echo
  echo cat dfilter/resume.txt:
  cat dfilter/resume.txt
  umount dfilter
) 
