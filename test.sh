#!/bin/bash

(
  echo ------------------$2--------------------
  go mod vendor
  go mod tidy
  cd cmd/filter
  sudo umount $2
  rmdir $2
  mkdir $2
  ( go run main.go $2 originalData $1 ) &
  sleep 2
  for f in $2/.rego-*
  do
    echo
    echo //$f
    cat $f
  done
  echo
  ls -al $2
  echo
  echo cat $2/notice.txt:
  cat $2/notice.txt
  echo
  echo cat $2/resume.txt:
  cat $2/resume.txt
  #umount $2
) 
