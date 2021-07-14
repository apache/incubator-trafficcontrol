package middleware

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"net/http"
	"sync"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
)

// ReadWhileWriterWrapper blocks multiple requests for the same object,
// and makes a single request to the real handler (which is probably expensive, e.g. database calls),
// and returns the result to all callers.
func ReadWhileWriterWrapper() Middleware {
	rwr := NewRWR()
	return func(h http.HandlerFunc) http.HandlerFunc {
		return WrapReadWhileWriter(h, rwr)
	}
}

func WrapReadWhileWriter(h http.HandlerFunc, rwr *RWR) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Infoln("ReadWhileWriter starting")
		if r.Method != http.MethodGet {
			// only GETs use RWR. TODO: determine if we should rwr HEAD
			h(w, r)
			return
		}
		user, err := auth.GetCurrentUser(r.Context())
		if err != nil {
			h(w, r) // not our job to error, pass the request along
			return
		}
		// Note no error doesn't mean a valid user, there may be no user (which is allowed for some endpoints).
		// In that case, user.TenantID will be TenantIDInvalid.
		// Which is fine, we'll use that as a cache key like any valid tenant,
		// and all unauthenticated requests will share requests.

		// TODO: add "allowed to request no-cache" as a Tenant Role Permission
		if user.TenantID != auth.TenantIDInvalid && requestNoCache(r) {
			h(w, r)
			return
		}

		cacheKey := makeSmallCacheKey(r, user)

		multiRequestWrite(h, w, r, rwr, cacheKey)
	}
}

type ReqObj struct {
	Body    []byte
	Code    int
	Headers http.Header
}

func multiRequestWrite(h http.HandlerFunc, w http.ResponseWriter, r *http.Request, rwr *RWR, cacheKey CacheKey) {
	log.Infoln("RWR starting")
	if reqChan := rwr.GetOrMakeQueue(cacheKey); reqChan != nil {
		log.Infoln("RWR: GetOrMakeQueue loaded, there's another concurrent reader, queueing up")
		// we loaded, so a request is ongoing, we need to queue up
		obj := <-reqChan
		util.WriteHeaders(w, obj.Headers)
		if obj.Code != 200 && obj.Code != 0 {
			log.Infof("RWR: '"+string(cacheKey)+"' concurrent writing code %v\n", obj.Code)
			w.WriteHeader(obj.Code)
		} else {
			log.Infof("RWR: '" + string(cacheKey) + "' concurrent had no code, not writing code\n")
		}
		w.Write(obj.Body)
		return
	}

	log.Infoln("RWR GetOrMakeQueue made queue, no concurrent reader")

	// we didn't load, so we're the first - make the req, respond to queued requestors, and close the queue.

	// To test the read-while-writer (which is normally very fast and difficult to test),
	// you can uncomment the following lines:
	// log.Infoln("DEBUG multiRequestWrite debug sleep")
	// time.Sleep(time.Second * 10) // debug

	iw := &util.FullInterceptor{W: w}
	h(iw, r)

	// If the StatusKey Context was set, prioritize it
	ctx := r.Context()
	val := ctx.Value(tc.StatusKey)
	status, ok := val.(int)
	if ok {
		iw.Code = status
	}

	util.WriteHeaders(w, iw.Headers)
	if iw.Code != 200 && iw.Code != 0 {
		log.Infof("RWR: '"+string(cacheKey)+"' writing code %v\n", iw.Code)
		w.WriteHeader(iw.Code)
	} else {
		log.Infof("RWR: '" + string(cacheKey) + "' had no code, not writing code\n")
	}
	w.Write(iw.Body)

	// run in a goroutine, so we don't block this routine, and the http request can finish
	// TODO test performance, vs closing the writer and calling WriteQueue in this goroutine
	// TODO test performance, of dedicated QueueWriter goroutine(s)
	go rwr.WriteQueue(cacheKey, ReqObj{Body: iw.Body, Code: iw.Code, Headers: iw.Headers})
}

type CacheKey string

type RWR struct {
	reqs map[CacheKey][]chan<- ReqObj
	m    *sync.Mutex
}

func NewRWR() *RWR {
	// now := time.Now()
	// inGC := int32(0)
	// nowP := (unsafe.Pointer)(&now)
	return &RWR{
		reqs: map[CacheKey][]chan<- ReqObj{},
		m:    &sync.Mutex{},
		// CacheTime: cacheTime,
		// InGC:      &inGC,
		// LastGC:    &nowP,
	}
}

// GetOrMakeQueue gets the queue if it exists, or creates it if it doesn't.
// If another request is happening, chan ReqObj will not be nil. In which case, the caller must read from it to get the object when it's ready.
// If the chan is nil, then no queue existed, and the caller must do the request itself, and then write what it gets via rwr.WriteQueue.
func (rwr *RWR) GetOrMakeQueue(cacheKey CacheKey) <-chan ReqObj {
	newQueue := []chan<- ReqObj{}
	reqChan := make(chan ReqObj, 1) // buffer 1, so the WriteQueue writes don't block
	rwr.m.Lock()
	if reqQueue, ok := rwr.reqs[cacheKey]; ok {
		rwr.reqs[cacheKey] = append(reqQueue, reqChan)
		rwr.m.Unlock()
		return reqChan
	}
	rwr.reqs[cacheKey] = newQueue
	rwr.m.Unlock()
	return nil
}

// WriteQueue should be called iff GetOrMakeQueue returned a nil chan, which means a queue was started.
// It writes to all subscribers, and deletes the queue.
func (rwr *RWR) WriteQueue(cacheKey CacheKey, obj ReqObj) {
	log.Infoln("RWR rwr.WriteQueue starting")
	rwr.m.Lock()
	reqQueue, ok := rwr.reqs[cacheKey]
	if !ok {
		rwr.m.Unlock()
		log.Errorln("RWR.WriteQueue was called, but there's no queue. This is a critical code error, and should never happen!")
		return
	}
	delete(rwr.reqs, cacheKey)
	rwr.m.Unlock()

	log.Infof("RWR rwr.WriteQueue writing to %v readers\n", len(reqQueue))
	for _, ch := range reqQueue {
		ch <- obj
	}
}
