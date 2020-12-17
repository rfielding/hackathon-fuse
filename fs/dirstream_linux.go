// Copyright 2019 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fs

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/hanwen/go-fuse/v2/fuse"
)

type loopbackDirStream struct {
	buf        []byte
	todo       []byte
	name       string
	nextResult *fuse.DirEntry
	loadResult syscall.Errno
	// Protects fd so we can guard against double close
	mu sync.Mutex
	fd int
}

// NewLoopbackDirStream open a directory for reading as a DirStream
func NewLoopbackDirStream(name string) (DirStream, syscall.Errno) {
	fd, err := syscall.Open(name, syscall.O_DIRECTORY, 0755)
	if err != nil {
		return nil, ToErrno(err)
	}

	ds := &loopbackDirStream{
		name: name,
		buf:  make([]byte, 4096),
		fd:   fd,
	}

	if err := ds.load(); err != 0 {
		ds.Close()
		return nil, err
	}
	return ds, OK
}

func (ds *loopbackDirStream) Close() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if ds.fd != -1 {
		syscall.Close(ds.fd)
		ds.fd = -1
	}
}

func (ds *loopbackDirStream) HasNext() bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if len(ds.todo) <= 0 {
		return false
	}
	for ds.advance() {
		// we may ignore files
	}
	return ds.nextResult != nil
}

func (ds *loopbackDirStream) Next() (fuse.DirEntry, syscall.Errno) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	for ds.advance() {
		// we may ignore files
	}
	// we must CONSUME the result by zeroing it out after next
	n, e := *ds.nextResult, ds.loadResult
	ds.nextResult = nil
	ds.loadResult = 0
	return n, e
}

func (ds *loopbackDirStream) advance() bool {
	if ds.nextResult != nil {
		return false
	}
	if len(ds.todo) <= 0 {
		return false
	}
	de := (*syscall.Dirent)(unsafe.Pointer(&ds.todo[0]))

	nameBytes := ds.todo[unsafe.Offsetof(syscall.Dirent{}.Name):de.Reclen]
	ds.todo = ds.todo[de.Reclen:]

	// After the loop, l contains the index of the first '\0'.
	l := 0
	for l = range nameBytes {
		if nameBytes[l] == 0 {
			break
		}
	}
	nameBytes = nameBytes[:l]
	name := string(nameBytes)
	ds.nextResult = &fuse.DirEntry{
		Ino:  de.Ino,
		Mode: (uint32(de.Type) << 12),
		Name: name,
	}
	ds.loadResult = ds.load()
	if name == "." || name == ".." {
		// leave it as normal
	} else if strings.HasPrefix(name, ".rego-") {
		regoFileName := fmt.Sprintf("%s%s%s", ds.name, "/", name)
		ds.canList(name, regoFileName)
	} else {
		regoFileName := fmt.Sprintf("%s%s%s", ds.name, "/.rego-", name)
		ds.canList(name, regoFileName)
	}
	return true
}

func (ds *loopbackDirStream) canList(name, regoFileName string) {
	fd, err := os.Open(regoFileName)
	if err == nil {
		defer fd.Close()
		fdBytes, err := ioutil.ReadAll(fd)
		if err != nil {
			log.Printf("error: %v", err)
			ds.nextResult = nil
			ds.loadResult = 0
		}
		eval, err := evalRego(JwtInput, string(fdBytes))
		if err != nil {
			log.Printf("error: %v", err)
			ds.nextResult = nil
			ds.loadResult = 0
		}
		readok, ok := eval["R"].(bool)
		if readok && ok {
			// keep this file
		} else {
			ds.nextResult = nil
			ds.loadResult = 0
		}
	}
}

func (ds *loopbackDirStream) load() syscall.Errno {
	if len(ds.todo) > 0 {
		return OK
	}

	n, err := syscall.Getdents(ds.fd, ds.buf)
	if err != nil {
		return ToErrno(err)
	}
	ds.todo = ds.buf[:n]
	return OK
}
