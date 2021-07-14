package util

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
)

// Interceptor implements http.ResponseWriter.
// It intercepts writes to w, and tracks the HTTP code and the count of bytes written, while still writing to w.
type Interceptor struct {
	W         http.ResponseWriter
	Code      int
	ByteCount int
}

// WriteHeader implements http.ResponseWriter.
// It does the real write to Interceptor's internal ResponseWriter, while keeping track of the code.
func (i *Interceptor) WriteHeader(rc int) {
	i.W.WriteHeader(rc)
	i.Code = rc
}

// Write implements http.ResponseWriter.
// It does the real write to Interceptor's internal ResponseWriter, while keeping track of the count of bytes written.
// It also sets Interceptor's tracked code to 200 if WriteHeader wasn't called (which is what the real http.ResponseWriter will write to the client).
func (i *Interceptor) Write(b []byte) (int, error) {
	wi, werr := i.W.Write(b)
	i.ByteCount += wi
	if i.Code == 0 {
		i.Code = 200
	}
	return wi, werr
}

// Header implements http.ResponseWriter.
// It returns Interceptor's internal ResponseWriter.Header, without modification or tracking.
func (i *Interceptor) Header() http.Header {
	return i.W.Header()
}

// BodyInterceptor fulfills the Writer interface, but records the body and doesn't actually write. This allows performing operations on the entire body written by a handler, for example, compressing or hashing. To actually write, call `RealWrite()`. Note this means `len(b)` and `nil` are always returned by `Write()`, any real write errors will be returned by `RealWrite()`.
type BodyInterceptor struct {
	W         http.ResponseWriter
	BodyBytes []byte
}

// WriteHeader implements http.ResponseWriter.
// It does the real write to Interceptor's internal ResponseWriter, without modification or tracking.
func (i *BodyInterceptor) WriteHeader(rc int) {
	i.W.WriteHeader(rc)
}

// Write implements http.ResponseWriter.
// It writes the given bytes to BodyInterceptor's internal tracking bytes, and does not write them to the internal ResponseWriter.
// To write the internal bytes, call BodyInterceptor.RealWrite.
func (i *BodyInterceptor) Write(b []byte) (int, error) {
	i.BodyBytes = append(i.BodyBytes, b...)
	return len(b), nil
}

// Header implements http.ResponseWriter.
// It returns BodyInterceptor's internal ResponseWriter.Header, without modification or tracking.
func (i *BodyInterceptor) Header() http.Header {
	return i.W.Header()
}

// RealWrite writes BodyInterceptor's internal bytes, which were stored by calls to Write.
func (i *BodyInterceptor) RealWrite(b []byte) (int, error) {
	wi, werr := i.W.Write(i.BodyBytes)
	return wi, werr
}

// Body returns the internal bytes stored by calls to Write.
func (i *BodyInterceptor) Body() []byte {
	return i.BodyBytes
}

// FullInterceptor fulfills the Writer interface, but records the body, headers, and code, and doesn't actually write. This allows performing operations on the entire content written by a handler, for example, compressing or hashing. To actually write, call `RealWrite()`. Note this means `len(b)` and `nil` are always returned by `Write()`, any real write errors will be returned by `RealWrite()`.
type FullInterceptor struct {
	W       http.ResponseWriter
	Body    []byte
	Code    int
	Headers http.Header
}

// WriteHeader implements http.ResponseWriter.
// It saves the given code without writing it to the underlying ResponseWriter.
// To write the code and header to the underlying ResponseWriter, call RealWriteHeader.
func (fi *FullInterceptor) WriteHeader(code int) {
	fi.Code = code
}

// Write implements http.ResponseWriter.
// It writes the given bytes to FullInterceptor's internal tracking bytes, and does not write them to the internal ResponseWriter.
// To write the internal bytes, call FullInterceptor.RealWrite.
func (fi *FullInterceptor) Write(b []byte) (int, error) {
	fi.Body = append(fi.Body, b...)
	return len(b), nil
}

// Header implements http.ResponseWriter.
// It returns FullInterceptor's internal ResponseWriter.Header, without modification or tracking.
func (fi *FullInterceptor) Header() http.Header {
	if fi.Headers == nil {
		fi.Headers = http.Header{}
	}
	return fi.Headers
}

func WriteHeaders(w http.ResponseWriter, hdr http.Header) {
	for name, vals := range hdr {
		for _, val := range vals {
			w.Header().Add(name, val)
		}
	}
}
