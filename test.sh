#!/bin/bash

checkIt() {
  who=$1
  spid=$$
  echo ---------------- ${who} ${spid} ---------------
  curl -X POST --data-binary @${who}.jwt http://127.0.0.1:9494/jwt-for-pid/${spid}
  for f in dmount/.rego-*
  do
    echo
    echo //$f
    cat $f
  done
  echo
  ls -al dmount
  echo
  echo cat dmount/notice.txt:
  cat dmount/notice.txt
  echo
  echo cat dmount/resume.txt:
  cat dmount/resume.txt
}

(
  go mod vendor
  go mod tidy
  cd cmd/filter
  go run main.go --makekeypair usa
  echo USA signs off on our JWTs
  ls -al usa.*
  go run main.go --jwtsign usa --jwtclaims robclaims.json > rob.jwt
  go run main.go --jwtsign usa --jwtclaims danicaclaims.json > danica.jwt
  ls -al *.jwt
  umount dmount
  rmdir dmount
  mkdir dmount
  ( go run main.go dmount originalData $1 ) &
  sleep 5
  who=rob
  spid=$$
  curl -X POST --data-binary @${who}.jwt http://127.0.0.1:9494/jwt-for-pid/${spid}
  (
    checkIt danica
  )
  (
    checkIt rob
  )
) 
