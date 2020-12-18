#!/bin/bash

checkIt() {
  ./filter --jwtsign usa --jwtclaims ${who}claims.json --jwtout ${who}.jwt
  echo ---------------- ${who} ${spid} ---------------
  curl -X POST --data-binary @${who}.jwt http://127.0.0.1:9494/jwt-for-pid
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
  go build
  ./filter --makekeypair usa
  echo USA signs off on our JWTs
  ls -al usa.*
  umount dmount
  rmdir dmount
  mkdir dmount
  ( ./filter dmount originalData $1 ) &
  sleep 5
  who=rob
  spid=$$
  (
    who=danica
    checkIt
  )
  (
    who=rob
    checkIt
  )
) 
