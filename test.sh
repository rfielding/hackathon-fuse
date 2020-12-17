#!/bin/bash

(
  cd cmd/filter
  rmdir filter
  mkdir filter
  ( go run main.go filter /tmp ) &
  sleep 2
  ls -al filter
  umount filter
) 
