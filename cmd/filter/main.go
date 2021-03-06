// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is main program driver for the loopback filesystem from
// github.com/hanwen/go-fuse/fs/, a filesystem that shunts operations
// to an underlying file system.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/rfielding/hackathon-fuse/fs"
)

func writeMemProfile(fn string, sigs <-chan os.Signal) {
	i := 0
	for range sigs {
		fn := fmt.Sprintf("%s-%d.memprof", fn, i)
		i++

		log.Printf("Writing mem profile to %s\n", fn)
		f, err := os.Create(fn)
		if err != nil {
			log.Printf("Create: %v", err)
			continue
		}
		pprof.WriteHeapProfile(f)
		if err := f.Close(); err != nil {
			log.Printf("close %v", err)
		}
	}
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	// Scans the arg list and sets up flags
	debug := flag.Bool("debug", false, "print debugging messages.")
	other := flag.Bool("allow-other", false, "mount with -o allowother.")
	quiet := flag.Bool("q", false, "quiet")
	keypairCreate := flag.String("makekeypair", "", "make keypair for jwt signing")
	jwtSign := flag.String("jwtsign", "", "sign a jwt")
	jwtClaims := flag.String("jwtclaims", "", "actual jwt claims to sign")
	jwtOut := flag.String("jwtout", "", "where to write the jwt out")
	ro := flag.Bool("ro", false, "mount read-only")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to this file")
	memprofile := flag.String("memprofile", "", "write memory profile to this file")
	flag.Parse()

	if *keypairCreate != "" {
		privName := fmt.Sprintf("%s.priv", *keypairCreate)
		pubName := fmt.Sprintf("%s.pub", *keypairCreate)
		err := fs.JwtKeygen(privName, pubName)
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}

	if *jwtSign != "" {
		claimBytes, err := ioutil.ReadFile(*jwtClaims)
		if err != nil {
			panic(err)
		}
		sig := fs.Sign(*jwtSign, string(claimBytes))
		err = ioutil.WriteFile(*jwtOut, []byte(sig), 0700)
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}

	if flag.NArg() < 2 {
		fmt.Printf("usage: %s MOUNTPOINT ORIGINAL\n", path.Base(os.Args[0]))
		fmt.Printf("\noptions:\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	if *cpuprofile != "" {
		if !*quiet {
			fmt.Printf("Writing cpu profile to %s\n", *cpuprofile)
		}
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Println(err)
			os.Exit(3)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if *memprofile != "" {
		if !*quiet {
			log.Printf("send SIGUSR1 to %d to dump memory profile", os.Getpid())
		}
		profSig := make(chan os.Signal, 1)
		signal.Notify(profSig, syscall.SIGUSR1)
		go writeMemProfile(*memprofile, profSig)
	}
	if *cpuprofile != "" || *memprofile != "" {
		if !*quiet {
			fmt.Printf("Note: You must unmount gracefully, otherwise the profile file(s) will stay empty!\n")
		}
	}

	orig := flag.Arg(1)
	loopbackRoot, err := fs.NewLoopbackRoot(orig)
	if err != nil {
		log.Fatalf("NewLoopbackRoot(%s): %v\n", orig, err)
	}

	sec := time.Second
	opts := &fs.Options{
		// These options are to be compatible with libfuse defaults,
		// making benchmarking easier.
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	}
	opts.Debug = *debug
	opts.AllowOther = *other
	if opts.AllowOther {
		// Make the kernel check file permissions for us
		opts.MountOptions.Options = append(opts.MountOptions.Options, "default_permissions")
	}
	if *ro {
		opts.MountOptions.Options = append(opts.MountOptions.Options, "ro")
	}
	// First column in "df -T": original dir
	opts.MountOptions.Options = append(opts.MountOptions.Options, "fsname="+orig)
	// Second column in "df -T" will be shown as "fuse." + Name
	opts.MountOptions.Name = "loopback"
	// Leave file permissions on "000" files as-is
	opts.NullPermissions = true
	// Enable diagnostics logging
	if !*quiet {
		opts.Logger = log.New(os.Stderr, "", 0)
	}

	// Set up an http server for signalling out of band
	http.HandleFunc("/jwt-for-pid", func(res http.ResponseWriter, req *http.Request) {
		log.Printf("got a request")
		if req.Method == "POST" {
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				res.Write([]byte(err.Error()))
				return
			}
			jwtBytes := strings.Trim(string(body), " \t\r\n")
			authenticated, err := fs.Authenticate(fs.FindIssuer(string(jwtBytes)), jwtBytes)
			if err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				res.Write([]byte(err.Error()))
				return
			}
			pid := authenticated.Pid
			log.Printf("map pid %d to %s", pid, fs.AsJsonPretty(authenticated))
			var j fs.JwtData
			j.Claims.Values = authenticated.Values
			fs.JwtDataByPid[uint32(pid)] = &j
		}
	})
	go http.ListenAndServe("127.0.0.1:9494", nil)
	log.Printf("Started up jwt control at 127.0.0.1:9494")
	server, err := fs.Mount(flag.Arg(0), loopbackRoot, opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}
	if !*quiet {
		fmt.Println("\nMounted!\n")
	}
	server.Wait()
}
