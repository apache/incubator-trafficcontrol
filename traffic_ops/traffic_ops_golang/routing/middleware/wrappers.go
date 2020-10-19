// Package middleware provides symbols for HTTP "middleware" which wraps handlers to perform common behaviors, such as authentication, headers, and compression.
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
	"bytes"
	"compress/gzip"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/about"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tocookie"
)

// DefaultRequestTimeout is the default request timeout, if no timeout is configured.
// This should be used by all Traffic Ops routing, and is recommended for use by plugins.
const DefaultRequestTimeout = time.Second * time.Duration(60)

// These are the different status values associated with an If-Modified-Since(IMS) request
// NONIMS when the request doesn't contain the If-Modified-Since header
const NONIMS = "NON_IMS"

// IMSHIT when a 304(Not Modified) was returned
const IMSHIT = "IMS_HIT"

// IMSMISS when anything other than a 304 was returned, meaning that something changed after the If-Modified-Since time of the request
const IMSMISS = "IMS_MISS"

// RouteID
const RouteID = "RouteID"

// ServerName is the name and version of Traffic Ops.
// Things that print the server application name and version, for example in headers or logs, should use this.
var ServerName = "traffic_ops_golang" + "/" + about.About.Version

// Middleware is an HTTP dispatch "middleware" function.
// A Middleware is a function which takes an http.HandlerFunc and returns a new http.HandlerFunc, typically wrapping the execution of the given handlerFunc with some additional behavior. For example, adding headers or gzipping the body.
type Middleware func(handlerFunc http.HandlerFunc) http.HandlerFunc

// GetDefault returns the default middleware for Traffic Ops.
// This includes writing to the access log, a request timeout, default headers such as CORS, and compression.
func GetDefault(secret string, requestTimeout time.Duration) []Middleware {
	return []Middleware{GetWrapAccessLog(secret), TimeOutWrapper(requestTimeout), WrapHeaders, WrapPanicRecover}
}

// Use takes a slice of middlewares, and applies them in reverse order (which is the intuitive behavior) to the given HandlerFunc h.
// It returns a HandlerFunc which will call all middlewares, and then h.
func Use(h http.HandlerFunc, middlewares []Middleware) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- { //apply them in reverse order so they are used in a natural order.
		h = middlewares[i](h)
	}
	return h
}

// AuthBase is the basic authentication object for middleware.
// It contains the data required for authentication, as well as a function to get a Middleware to perform authentication.
type AuthBase struct {
	Secret   string
	Override Middleware
}

// GetWrapper returns a Middleware which performs authentication of the current user at the given privilege level.
// The returned Middleware also adds the auth.CurrentUser object to the request context, which may be retrieved by a handler via api.NewInfo or auth.GetCurrentUser.
func (a AuthBase) GetWrapper(privLevelRequired int) Middleware {
	if a.Override != nil {
		return a.Override
	}
	return func(handlerFunc http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, errs := api.GetUserFromReq(w, r, a.Secret)
			if errs.Occurred() {
				api.HandleErr(w, r, nil, errs.Code, errs.UserError, errs.SystemError)
				return
			}
			if user.PrivLevel < privLevelRequired {
				api.HandleErr(w, r, nil, http.StatusForbidden, errors.New("Forbidden."), nil)
				return
			}
			api.AddUserToReq(r, user)
			handlerFunc(w, r)
		}
	}
}

// TimeOutWrapper is a Middleware which adds the given timeout to the request.
// This causes the request to abort and return an error to the user if the handler takes longer than the timeout to execute.
func TimeOutWrapper(timeout time.Duration) Middleware {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.TimeoutHandler(h, timeout, "server timed out").ServeHTTP(w, r)
		}
	}
}

// WrapHeaders is a Middleware which adds common headers and behavior to the handler. It specifically:
//  - Adds default CORS headers to the response.
//  - Adds the Whole-Content-SHA512 checksum header to the response.
//  - Gzips the response and sets the Content-Encoding header, if the client sent an Accept-Encoding: gzip header.
//  - Adds the Vary: Accept-Encoding header to the response
func WrapHeaders(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie")
		w.Header().Set("Access-Control-Allow-Methods", "POST,GET,OPTIONS,PUT,DELETE")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set(rfc.Vary, rfc.AcceptEncoding)
		w.Header().Set("X-Server-Name", ServerName)
		iw := &util.BodyInterceptor{W: w}
		h(iw, r)

		sha := sha512.Sum512(iw.Body())
		w.Header().Set("Whole-Content-SHA512", base64.StdEncoding.EncodeToString(sha[:]))

		GzipResponse(w, r, iw.Body())

	}
}

// WrapPanicRecover is a Middleware which adds a panic recover call to the given HandlerFunc h.
// If h throws an unhandled panic, an error is logged and an Internal Server Error is returned to the client.
func WrapPanicRecover(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				api.HandleErr(w, r, nil, http.StatusInternalServerError, nil, fmt.Errorf("panic: (err: %v) stacktrace:\n%s\n", err, util.Stacktrace()))
				return
			}
		}()
		h(w, r)
	}
}

// AccessLogTimeFormat is the time format of the access log, as used by time.Time.Format.
const AccessLogTimeFormat = "02/Jan/2006:15:04:05 -0700"

// GetWrapAccessLog returns a Middleware which writes to the Access Log (which is the lib/go-log EventLog) after the HandlerFunc finishes.
func GetWrapAccessLog(secret string) Middleware {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return WrapAccessLog(secret, h)
	}
}

// WrapAccessLog takes the cookie secret and a http.Handler, and returns a HandlerFunc which writes to the Access Log (which is the lib/go-log EventLog) after the HandlerFunc finishes.
// This is not a Middleware, because it needs the secret as a parameter. For a Middleware, see GetWrapAccessLog.
func WrapAccessLog(secret string, h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var imsType = NONIMS
		iw := &util.Interceptor{W: w}
		user := "-"
		cookie, err := r.Cookie(tocookie.Name)
		if err == nil && cookie != nil {
			cookie, err := tocookie.Parse(secret, cookie.Value)
			if err == nil {
				user = cookie.AuthData
			}
		}
		start := time.Now()
		defer func() {
			_, ok := r.Header[rfc.IfModifiedSince]
			if ok {
				if iw.Code == http.StatusNotModified {
					imsType = IMSHIT
				} else {
					imsType = IMSMISS
				}
			}
			log.EventfRaw(`%s - %s [%s] "%v %v?%v %s" %v %v %v "%v" %v %s`, r.RemoteAddr, user, time.Now().Format(AccessLogTimeFormat), r.Method, r.URL.Path, r.URL.RawQuery, r.Proto, iw.Code, iw.ByteCount, int(time.Now().Sub(start)/time.Millisecond), r.UserAgent(), r.Header.Get(RouteID), imsType)
		}()
		h.ServeHTTP(iw, r)
	}
}

// GzipResponse takes a function which cannot error and returns only bytes, and wraps it as a http.HandlerFunc. The errContext is logged if the write fails, and should be enough information to trace the problem (function name, endpoint, request parameters, etc).
// It gzips the given bytes and writes them to w, as well as writing the appropriate 'Content-Encoding: gzip' header, if the request included an 'Accept-Encoding: gzip' header.
// If the request doesn't accept gzip, the bytes are written to w unmodified.
func GzipResponse(w http.ResponseWriter, r *http.Request, bytes []byte) {
	bytes, err := GzipIfAccepts(r, w, bytes)
	if err != nil {
		log.Errorf("gzipping request '%v': %v\n", r.URL.EscapedPath(), err)
		code := http.StatusInternalServerError
		w.WriteHeader(code)
		if _, err := w.Write([]byte(http.StatusText(code))); err != nil {
			log.Warnf("received error writing data request %v: %v\n", r.URL.EscapedPath(), err)
		}
		return
	}

	ctx := r.Context()
	val := ctx.Value(tc.StatusKey)
	status, ok := val.(int)
	if ok { //if not we assume it is a 200
		w.WriteHeader(status)
	}

	w.Write(bytes)
}

// GzipIfAccepts gzips the given bytes, writes a `Content-Encoding: gzip` header to the given writer, and returns the gzipped bytes, if the Request supports GZip (has an Accept-Encoding header). Else, returns the bytes unmodified. Note the given bytes are NOT written to the given writer. It is assumed the bytes may need to pass thru other middleware before being written.
//TODO: drichardson - refactor these to a generic area
func GzipIfAccepts(r *http.Request, w http.ResponseWriter, b []byte) ([]byte, error) {
	// TODO this could be made more efficient by wrapping ResponseWriter with the GzipWriter, and letting callers writer directly to it - but then we'd have to deal with Closing the gzip.Writer.
	if len(b) == 0 || !rfc.AcceptsGzip(r) {
		return b, nil
	}
	w.Header().Set(rfc.ContentEncoding, rfc.Gzip)

	buf := bytes.Buffer{}
	zw := gzip.NewWriter(&buf)

	if _, err := zw.Write(b); err != nil {
		return nil, fmt.Errorf("gzipping bytes: %v", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip writer: %v", err)
	}

	return buf.Bytes(), nil
}

// NotImplementedHandler returns a http.Handler which returns to the client a HTTP 501 Not Implemented status code, and a body which is a standard Traffic Ops error JSON.
// Note this is common usage in Traffic Ops for unimplemented endpoints (for example, which were impossible or impractical to rewrite from Perl). This is a something of a misuse of the HTTP spec. RFC 7231 states 501 Not Implemented is intended for HTTP Methods which are not implemented. However, it's not a clear violation of the Spec, and Traffic Ops has determined it is the safest and least ambiguous way to indicate endpoints which no longer or don't yet exist.
func NotImplementedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"alerts":[{"level":"error","text":"The requested api version is not implemented by this server. If you are using a newer client with an older server, you will need to use an older client version or upgrade your server."}]}`))
	})
}

// DisabledRouteHandler returns a http.Handler which returns a HTTP 5xx code to the client, and an error message indicating the route is currently disabled.
// This is used for routes which have been disabled via configuration. See config.ConfigTrafficOpsGolang.RoutingBlacklist.DisabledRoutes.
func DisabledRouteHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"alerts":[{"level":"error","text":"The requested route is currently disabled."}]}` + "\n"))
	})
}
