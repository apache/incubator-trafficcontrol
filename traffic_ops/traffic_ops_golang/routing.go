package main

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
	"context"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"

	"github.com/jmoiron/sqlx"
)

// RoutePrefix ...
const RoutePrefix = "api" // TODO config?

// Middleware ...
type Middleware func(handlerFunc http.HandlerFunc) http.HandlerFunc

// Route ...
type Route struct {
	// Order matters! Do not reorder this! Routes() uses positional construction for readability.
	Version       float64
	Method        string
	Path          string
	Handler       http.HandlerFunc
	Authenticated bool
	Middlewares   []Middleware
}

// RawRoute is an HTTP route to be served at the root, rather than under /api/version. Raw Routes should be rare, and almost exclusively converted old Perl routes which have yet to be moved to an API path.
type RawRoute struct {
	// Order matters! Do not reorder this! Routes() uses positional construction for readability.
	Method        string
	Path          string
	Handler       http.HandlerFunc
	Authenticated bool
	Middlewares   []Middleware
}

func getDefaultMiddleware(secret string, requestTimeout time.Duration) []Middleware {
	return []Middleware{getWrapAccessLog(secret), timeOutWrapper(requestTimeout), wrapHeaders, wrapPanicRecover}
}

// ServerData ...
type ServerData struct {
	config.Config
	DB        *sqlx.DB
	Profiling *bool // Yes this is a field in the config but we want to live reload this value and NOT the entire config
}

// CompiledRoute ...
type CompiledRoute struct {
	Handler       http.HandlerFunc
	Regex         *regexp.Regexp
	Params        []string
	APICapability string
}

func getSortedRouteVersions(rs []Route) []float64 {
	m := map[float64]struct{}{}
	for _, r := range rs {
		m[r.Version] = struct{}{}
	}
	versions := []float64{}
	for v := range m {
		versions = append(versions, v)
	}
	sort.Float64s(versions)
	return versions
}

// PathHandler ...
type PathHandler struct {
	// Path is the full path, including the API prefix and version, e.g. `/api/1.2/cdns/{id}`.
	Path string
	// RawPath is the path not including the API prefix and version, e.g. `cdns/{id}`.
	RawPath string
	// Handler is the HTTP handler function for this path.
	Handler http.HandlerFunc
}

// CreateRouteMap returns a map of methods to a slice of paths and handlers; wrapping the handlers in the appropriate middleware. Uses Semantic Versioning: routes are added to every subsequent minor version, but not subsequent major versions. For example, a 1.2 route is added to 1.3 but not 2.1. Also truncates '2.0' to '2', creating succinct major versions.
func CreateRouteMap(rs []Route, rawRoutes []RawRoute, authBase AuthBase, reqTimeOutSeconds int) map[string][]PathHandler {
	// TODO strong types for method, path
	versions := getSortedRouteVersions(rs)
	requestTimeout := time.Second * time.Duration(60)
	if reqTimeOutSeconds > 0 {
		requestTimeout = time.Second * time.Duration(reqTimeOutSeconds)
	}
	m := map[string][]PathHandler{}
	for _, r := range rs {
		versionI := sort.SearchFloat64s(versions, r.Version)
		nextMajorVer := float64(int(r.Version) + 1)
		for _, version := range versions[versionI:] {
			if version >= nextMajorVer {
				break
			}
			vstr := strconv.FormatFloat(version, 'f', -1, 64)
			path := RoutePrefix + "/" + vstr + "/" + r.Path
			middlewares := getRouteMiddleware(r.Middlewares, authBase, r.Authenticated, requestTimeout)
			m[r.Method] = append(m[r.Method], PathHandler{Path: path, Handler: use(r.Handler, middlewares), RawPath: r.Path})
			log.Infof("adding route %v %v\n", r.Method, path)
		}
	}
	for _, r := range rawRoutes {
		middlewares := getRouteMiddleware(r.Middlewares, authBase, r.Authenticated, requestTimeout)
		m[r.Method] = append(m[r.Method], PathHandler{Path: r.Path, Handler: use(r.Handler, middlewares)})
		log.Infof("adding raw route %v %v\n", r.Method, r.Path)
	}
	return m
}

func getRouteMiddleware(middlewares []Middleware, authBase AuthBase, authenticated bool, requestTimeout time.Duration) []Middleware {
	if middlewares == nil {
		middlewares = getDefaultMiddleware(authBase.secret, requestTimeout)
	}
	if authenticated {
		authWrapper := authBase.GetWrapper()
		middlewares = append([]Middleware{authWrapper}, middlewares...)
	}
	return middlewares
}

// CompileRoutes - takes a map of methods to paths and handlers, and returns a map of methods to CompiledRoutes
func CompileRoutes(routes map[string][]PathHandler) map[string][]CompiledRoute {
	compiledRoutes := map[string][]CompiledRoute{}
	for method, mRoutes := range routes {
		for _, pathHandler := range mRoutes {
			route := pathHandler.Path
			var params []string
			for open := strings.Index(route, "{"); open > 0; open = strings.Index(route, "{") {
				close := strings.Index(route, "}")
				if close < 0 {
					panic("malformed route")
				}
				param := route[open+1 : close]

				params = append(params, param)
				route = route[:open] + `([^/]+)` + route[close+1:]
			}
			compiledRoutes[method] = append(compiledRoutes[method], CompiledRoute{
				Handler:       pathHandler.Handler,
				Regex:         regexp.MustCompile(route),
				Params:        params,
				APICapability: MakeAPICapability(pathHandler.RawPath),
			})
		}
	}
	return compiledRoutes
}

// MakeAPICapability returns an route "api capability" string, from a route. This panics if route has mismatched braces.
func MakeAPICapability(route string) string {
	route = strings.Replace(route, `$`, ``, -1)
	route = strings.Replace(route, `?`, ``, -1)
	route = strings.Replace(route, `(\.json)`, ``, -1)
	route = strings.TrimSuffix(route, `/`)
	for open := strings.Index(route, "{"); open > 0; open = strings.Index(route, "{") {
		close := strings.Index(route, "}")
		if close < 0 {
			panic("malformed route")
		}
		route = route[:open] + `*` + route[close+1:]
	}
	return route
}

// Handler - generic handler func used by the Handlers hooking into the routes
func Handler(routes map[string][]CompiledRoute, catchall http.Handler, db *sqlx.DB, cfg *config.Config, getReqID func() uint64, w http.ResponseWriter, r *http.Request) {
	reqID := getReqID()

	reqIDStr := strconv.FormatUint(reqID, 10)
	log.Infoln(r.Method + " " + r.URL.Path + " handling (reqid " + reqIDStr + ")")
	start := time.Now()
	defer func() {
		log.Infoln(r.Method + " " + r.URL.Path + " handled (reqid " + reqIDStr + ") in " + time.Since(start).String())
	}()

	requested := r.URL.Path[1:]
	mRoutes, ok := routes[r.Method]
	if !ok {
		catchall.ServeHTTP(w, r)
		return
	}

	for _, compiledRoute := range mRoutes {
		match := compiledRoute.Regex.FindStringSubmatch(requested)
		if len(match) == 0 {
			continue
		}
		params := map[string]string{}
		for i, v := range compiledRoute.Params {
			params[v] = match[i+1]
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, api.PathParamsKey, params)
		ctx = context.WithValue(ctx, api.DBContextKey, db)
		ctx = context.WithValue(ctx, api.ConfigContextKey, cfg)
		ctx = context.WithValue(ctx, api.ReqIDContextKey, reqID)
		ctx = context.WithValue(ctx, api.APICapabilityContextKey, compiledRoute.APICapability)
		r = r.WithContext(ctx)
		compiledRoute.Handler(w, r)
		return
	}
	catchall.ServeHTTP(w, r)
}

// RegisterRoutes - parses the routes and registers the handlers with the Go Router
func RegisterRoutes(d ServerData) error {
	routeSlice, rawRoutes, catchall, err := Routes(d)
	if err != nil {
		return err
	}

	authBase := AuthBase{secret: d.Config.Secrets[0], override: nil} //we know d.Config.Secrets is a slice of at least one or start up would fail.
	routes := CreateRouteMap(routeSlice, rawRoutes, authBase, d.RequestTimeout)
	compiledRoutes := CompileRoutes(routes)
	getReqID := nextReqIDGetter()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		Handler(compiledRoutes, catchall, d.DB, &d.Config, getReqID, w, r)
	})
	return nil
}

func use(h http.HandlerFunc, middlewares []Middleware) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- { //apply them in reverse order so they are used in a natural order.
		h = middlewares[i](h)
	}
	return h
}

// nextReqIDGetter returns a function for getting incrementing identifiers. The returned func is safe for calling with multiple goroutines. Note the returned identifiers will not be unique after the max uint64 value.
func nextReqIDGetter() func() uint64 {
	id := uint64(0)
	return func() uint64 {
		return atomic.AddUint64(&id, 1)
	}
}
