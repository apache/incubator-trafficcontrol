package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

func main() {
	port := flag.Int("port", 80, "Port to serve on")
	timeoutMS := flag.Int("timeout-ms", 10000, "HTTP read and write timeout")
	debug := flag.Bool("debug", false, "Whether to enable debug HTTP directives. Unsecured, should never be enabled in a production environment.")
	flag.Parse()

	timeout := time.Duration(*timeoutMS) * time.Millisecond

	server := &http.Server{
		Addr:         ":" + strconv.Itoa(*port),
		Handler:      MakeHandler(&HandlerCfg{Debug: *debug}),
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	fmt.Println("Listening on " + server.Addr)
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("ERROR: " + err.Error())
		os.Exit(1)
	}
	os.Exit(0) // should never happen; unless we add a "shutdown" directive
}

type HandlerCfg struct {
	Debug bool
	// Atomic, MUST NOT be accessed outside atomic.
	// DO NOT change to a bool or make non-atomic. Booleans still must be atomic.
	Unavailable uint32
}

// MakeHandler currently takes just the debug bool, but it can be changed to take lots of things if necessary.
func MakeHandler(cfg *HandlerCfg) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, cfg)
	}
}

func handler(w http.ResponseWriter, r *http.Request, cfg *HandlerCfg) {
	if cfg.Debug && strings.HasPrefix(r.URL.Path, PathPrefixDebug) {
		handlerDebug(w, r, cfg)
		return
	}

	handlerHealth(w, r, cfg)
}

const PathPrefixDebug = `/debug`

func handlerDebug(w http.ResponseWriter, r *http.Request, cfg *HandlerCfg) {
	switch r.Method {
	case http.MethodPost:
		query := r.URL.Query()
		if availStrs, ok := query["available"]; ok {
			if len(availStrs) > 0 && strings.HasPrefix(availStrs[0], "f") {
				atomic.StoreUint32(&cfg.Unavailable, 1)
			} else {
				atomic.StoreUint32(&cfg.Unavailable, 0)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "unknown debug directive POST\n")
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func handlerHealth(w http.ResponseWriter, r *http.Request, cfg *HandlerCfg) {
	switch r.Method {
	case http.MethodHead:
		fallthrough
	case http.MethodGet:
		if atomic.LoadUint32(&cfg.Unavailable) != 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}
