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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
)

func SmallCacheWrapper(span time.Duration, disableRWR bool) Middleware {
	cache := NewSmallCache(span)
	rwr := (*RWR)(nil)
	if !disableRWR {
		rwr = NewRWR()
	}
	return func(h http.HandlerFunc) http.HandlerFunc {
		return WrapSmallCache(h, cache, rwr)
	}
}

// WrapSmallCache is middleware that adds a small cache to the server, for the same request by the same tenant.
// This is very important for control plane things, like caches requesting newly changed config all at once.
// This should generally be around 1 second. Configuring this over 5 seconds or so is strongly discouraged.
//
// Note the configured cache time will be how long new changes are potentially unavailable to requestors.
// So a 5s time means new changes will delay propogating to all clients, tenants and the control-plane, for 5 seconds.
//
// This doesn't follow HTTP Cache Control rules, or respect client requests for things like max-age.
// It does however return a proper Age header.
//
func WrapSmallCache(h http.HandlerFunc, cache *SmallCache, rwr *RWR) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Infoln("SmallCache starting")
		if cache.CacheTime == 0 && rwr == nil {
			log.Infoln("SmallCache cache and rwr disabled, skipping entirely")
			h(w, r)
			return
		}
		if r.Method != http.MethodGet {
			log.Infoln("SmallCache method not get, skipping entirely")
			// only GETs are cached. TODO: determine if we should cache HEAD
			h(w, r)

			// See RFC7234§4.4
			// Note this reduces the race, it does not eliminate it.
			// Using the cache or read-while-writer is always a race.
			// Clients which need to avoid that race must send a no-cache.
			if !rfc.MethodIsSafe(r.Method) {
				cache.InvalidatePath(r.URL.Path)
			}

			return
		}

		user, err := auth.GetCurrentUser(r.Context())
		if err != nil {
			log.Infoln("SmallCache error getting user, skipping entirely")
			h(w, r) // not our job to error, pass the request along
			return
		}

		// Note no error doesn't mean a valid user, there may be no user (which is allowed for some endpoints).
		// In that case, user.TenantID will be TenantIDInvalid.
		// Which is fine, we'll use that as a cache key like any valid tenant,
		// and all unauthenticated requests will share a cache.

		// TODO: add "allowed to request no-cache" as a Tenant Role Permission
		if user.TenantID != auth.TenantIDInvalid && requestNoCache(r) {
			log.Infoln("SmallCache had valid tenant who sent no-cache, skipping entirely")
			h(w, r)
			return
		}

		cacheKey := makeSmallCacheKey(r, user)

		if cache.CacheTime == 0 {
			log.Infoln("SmallCache: '" + cacheKey + "' cache disabled, fetching")
			smallCacheInterceptAndCache(h, w, r, cache, rwr, cacheKey)
			return
		}

		iCacheObj, ok := cache.Cache.Load(cacheKey)
		if !ok {
			log.Infoln("SmallCache: '" + cacheKey + "' not in cache, fetching")
			smallCacheInterceptAndCache(h, w, r, cache, rwr, cacheKey)
			return
		}

		cacheObj, ok := iCacheObj.(CacheObj)
		if !ok {
			log.Infoln("SmallCache: '"+cacheKey+"' cache object typecast fail! Should never happen! type is %T\n", iCacheObj)
			h(w, r)
			return
		}

		age := time.Since(cacheObj.Time)
		if age > cache.CacheTime {
			log.Infoln("SmallCache: in cache expired, fetching")
			cache.Cache.Delete(cacheKey)
			smallCacheInterceptAndCache(h, w, r, cache, rwr, cacheKey)
			return
		}

		log.Infoln("SmallCache: '" + string(cacheKey) + "' in cache fresh, returning cached")
		smallCacheWriteObj(w, r, cacheObj)
	}
}

// requestNoCache returns whether the client requested not to be served from cache.
func requestNoCache(req *http.Request) bool {
	if req.Header.Get("Cache-Control") == "" {
		pragmaHdr := req.Header.Get("Pragma")
		if pragmaHdr != "" && strings.Contains(strings.ToLower(pragmaHdr), "no-cache") {
			return true
		}
	}
	cc := rfc.ParseCacheControl(req.Header)
	if _, ok := cc["no-cache"]; ok {
		return true
	}
	if _, ok := cc["no-store"]; ok {
		return true
	}
	if val, ok := cc["max-age"]; ok {
		if val == `0` || val == `"0"` {
			return true
		}
	}
	return false
}

func smallCacheInterceptAndCache(h http.HandlerFunc, w http.ResponseWriter, r *http.Request, cache *SmallCache, rwr *RWR, cacheKey CacheKey) {
	if cache.CacheTime != 0 {
		lastGC := (*time.Time)(atomic.LoadPointer(cache.LastGC))
		if cache.CacheTime != 0 && time.Since(*lastGC) > SmallCacheGCInterval {
			go func() { cache.GC() }()
		}
	}

	if rwr != nil {
		log.Infoln("RWR starting")
		if reqChan := rwr.GetOrMakeQueue(cacheKey); reqChan != nil {
			log.Infoln("RWR: GetOrMakeQueue loaded '" + cacheKey + "', there's another concurrent reader, queueing up")
			// we loaded, so a request is ongoing, we need to queue up
			obj := <-reqChan
			if obj.Code != 200 {
				log.Infof("smallCacheInterceptAndCache: '"+string(cacheKey)+"' writing code %v\n", obj.Code)
				w.WriteHeader(obj.Code)
			} else {
				log.Infof("smallCacheInterceptAndCache: '" + string(cacheKey) + "' had no code, not writing code\n")
			}
			log.Infoln("RWR: '" + cacheKey + "' Writing concurrent body and returning")
			w.Write(obj.Body)
			// return without writing to the cache: only the first RWR caller needs to write to the cache
			return
		}
	} else {
		log.Infoln("RWR disabled")
	}

	log.Infoln("SmallCache starting interceptor, calling next handler")
	iw := &util.FullInterceptor{W: w}
	h(iw, r)
	log.Infoln("SmallCache intercepted, processing")

	// If the StatusKey Context was set, prioritize it
	ctx := r.Context()
	val := ctx.Value(tc.StatusKey)
	status, ok := val.(int)
	if ok {
		iw.Code = status
	}

	log.Infoln("SmallCache: '" + cacheKey + "' got original object")

	if cache.CacheTime != 0 {
		cache.Cache.Store(cacheKey, CacheObj{
			Time:    time.Now(),
			Body:    iw.Body,
			Code:    iw.Code,
			Headers: iw.Headers,
		})
	}

	util.WriteHeaders(w, iw.Headers)
	if iw.Code != 0 {
		log.Infof("SmallCache: '"+string(cacheKey)+"' writing code %v\n", iw.Code)
		w.WriteHeader(iw.Code)
	} else {
		log.Infof("SmallCache: '" + string(cacheKey) + "' had no code, not writing code\n")
	}

	log.Infoln("SmallCache: '" + string(cacheKey) + "' writing original object")

	log.Infoln("SmallCache about to write to real writer")
	w.Write(iw.Body)
	log.Infoln("SmallCache wrote to real writer")

	if rwr != nil {
		log.Infoln("RWR: '" + cacheKey + "' starting thread to write to queued readers")
		// run in a goroutine, so we don't block this routine, and the http request can finish
		// TODO test performance, vs closing the writer (?) and calling WriteQueue in this goroutine
		// TODO test performance, of dedicated QueueWriter goroutine(s)
		go rwr.WriteQueue(cacheKey, ReqObj{Body: iw.Body, Headers: iw.Headers, Code: iw.Code})
	}
}

func smallCacheWriteObj(w http.ResponseWriter, r *http.Request, obj CacheObj) {
	w.Header().Set("Age", strconv.FormatInt(int64(time.Since(obj.Time)/time.Second)+1, 10))
	util.WriteHeaders(w, obj.Headers)
	if obj.Code != 200 {
		log.Infof("SmallCache: '"+""+"' WriteObj writing code %v\n", obj.Code)
		w.WriteHeader(obj.Code)
	} else {
		log.Infof("SmallCache: '" + "" + "' WriteObj had no code, not writing code\n")
	}
	w.Write(obj.Body)
}

func makeSmallCacheKey(r *http.Request, user *auth.CurrentUser) CacheKey {
	return CacheKey("t" + strconv.Itoa(user.TenantID) +
		"r" + strconv.Itoa(user.Role) +
		`ims"` + r.Header.Get(rfc.IfModifiedSince) + `"` +
		`etag"` + r.Header.Get(rfc.ETagHeader) + `"` +
		`ae"` + r.Header.Get(rfc.AcceptEncoding) + `"` +
		`path"` + r.URL.Path + `"` +
		`query"` + r.URL.RawQuery + `"`)
}

type CacheObj struct {
	Time    time.Time
	Body    []byte
	Headers http.Header
	Code    int
}

type SmallCache struct {
	Cache     *sync.Map
	CacheTime time.Duration
	LastGC    *unsafe.Pointer // LastGC is a *time.Time, and is atomically synchronized! Do not access outside atomics!
	InGC      *int32          // InGC is atomically synchronized! Do not access outside atomics!
}

func NewSmallCache(cacheTime time.Duration) *SmallCache {
	now := time.Now()
	inGC := int32(0)
	nowP := (unsafe.Pointer)(&now)
	return &SmallCache{
		Cache:     &sync.Map{},
		CacheTime: cacheTime,
		InGC:      &inGC,
		LastGC:    &nowP,
	}
}

const SmallCacheGCInterval = time.Second * 10

func (sc *SmallCache) GC() {
	if !atomic.CompareAndSwapInt32(sc.InGC, 0, 1) {
		return // another thread is running GC
	}

	log.Infoln("SmallCache.GC starting")

	// Note this doesn't lock the cache, so there's a race, a thread could insert after we check expiration,
	// and then we'll delete an object that was just inserted.
	// This isn't a correctness problem, just inefficient.
	// That inefficiency race will mostly only happen under high load,
	// in which case the cost of mutex contention would be far worse.

	expiredKeys := []CacheKey{}
	sc.Cache.Range(func(iCacheKey, iCacheObj interface{}) bool {
		cacheObj, ok := iCacheObj.(CacheObj)
		if !ok {
			log.Errorf("SmallCache.GC cache object typecast fail! Should never happen! type is %T\n", iCacheObj)
			return true
		}
		cacheKey, ok := iCacheKey.(CacheKey)
		if !ok {
			log.Errorf("SmallCache.GC cache key typecast fail! Should never happen! type is %T\n", iCacheKey)
			return true
		}
		if time.Since(cacheObj.Time) > sc.CacheTime {
			expiredKeys = append(expiredKeys, cacheKey)
		}
		return true
	})
	for _, key := range expiredKeys {
		log.Infoln("SmallCache.GC deleting expired '" + key + "'")
		sc.Cache.Delete(key)
	}

	now := time.Now()
	atomic.StorePointer(sc.LastGC, unsafe.Pointer(&now))
	atomic.StoreInt32(sc.InGC, 0)
}

func (sc *SmallCache) InvalidatePath(path string) {
	invalidKeys := []CacheKey{}
	sc.Cache.Range(func(iCacheKey, iCacheObj interface{}) bool {
		cacheKey, ok := iCacheKey.(CacheKey)
		if !ok {
			log.Errorf("SmallCache.InvalidteCache '%v' cache key typecast fail! Should never happen! type is %T\n", path, iCacheKey)
			return true
		}
		// This could be faster by making the cache multi-dimensional, but there are likely far more GETs than POST/PUT, so the cost of a multi-dimensional lookup for every GET likely far exceeds the string compare on POST/PUT.
		if strings.Contains(string(cacheKey), `path"`+path+`"`) {
			invalidKeys = append(invalidKeys, cacheKey)
		}
		return true
	})
	for _, key := range invalidKeys {
		log.Infoln("SmallCache.InvalidateCache deleting '" + key + "'")
		sc.Cache.Delete(key)
	}
}
