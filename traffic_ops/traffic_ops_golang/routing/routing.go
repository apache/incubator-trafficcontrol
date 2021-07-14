// Package routing defines the HTTP routes for Traffic Ops and provides tools to
// register those routes with appropriate middleware.
package routing

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
	"errors"
	"fmt"
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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/plugin"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/routing/middleware"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault"

	"github.com/jmoiron/sqlx"
)

// RoutePrefix ...
const RoutePrefix = "^api" // TODO config?

// DoCache is a constant to make routes more readable.
const DoCache = true

// NoCache is a constant to make routes more readable.
const NoCache = false

// Route ...
type Route struct {
	// Order matters! Do not reorder this! Routes() uses positional construction for readability.
	Version           api.Version
	Method            string
	Path              string
	Handler           http.HandlerFunc
	RequiredPrivLevel int
	Authenticated     bool
	Middlewares       []middleware.Middleware
	Cache             bool
	ID                int // unique ID for referencing this Route
}

func (r Route) String() string {
	return fmt.Sprintf("id=%d\tmethod=%s\tversion=%d.%d\tpath=%s", r.ID, r.Method, r.Version.Major, r.Version.Minor, r.Path)
}

// RawRoute is an HTTP route to be served at the root, rather than under /api/version. Raw Routes should be rare, and almost exclusively converted old Perl routes which have yet to be moved to an API path.
type RawRoute struct {
	// Order matters! Do not reorder this! Routes() uses positional construction for readability.
	Method            string
	Path              string
	Handler           http.HandlerFunc
	RequiredPrivLevel int
	Authenticated     bool
	Middlewares       []middleware.Middleware
}

// ServerData ...
type ServerData struct {
	config.Config
	DB           *sqlx.DB
	Profiling    *bool // Yes this is a field in the config but we want to live reload this value and NOT the entire config
	Plugins      plugin.Plugins
	TrafficVault trafficvault.TrafficVault
}

// CompiledRoute ...
type CompiledRoute struct {
	Handler http.HandlerFunc
	Regex   *regexp.Regexp
	Params  []string
	ID      int
}

func getSortedRouteVersions(rs []Route) []api.Version {
	majorsToMinors := map[uint64][]uint64{}
	majors := map[uint64]struct{}{}
	for _, r := range rs {
		majors[r.Version.Major] = struct{}{}
		if _, ok := majorsToMinors[r.Version.Major]; ok {
			previouslyIncluded := false
			for _, prevMinor := range majorsToMinors[r.Version.Major] {
				if prevMinor == r.Version.Minor {
					previouslyIncluded = true
				}
			}
			if !previouslyIncluded {
				majorsToMinors[r.Version.Major] = append(majorsToMinors[r.Version.Major], r.Version.Minor)
			}
		} else {
			majorsToMinors[r.Version.Major] = []uint64{r.Version.Minor}
		}
	}

	sortedMajors := []uint64{}
	for major := range majors {
		sortedMajors = append(sortedMajors, major)
	}
	sort.Slice(sortedMajors, func(i, j int) bool { return sortedMajors[i] < sortedMajors[j] })

	versions := []api.Version{}
	for _, major := range sortedMajors {
		sort.Slice(majorsToMinors[major], func(i, j int) bool { return majorsToMinors[major][i] < majorsToMinors[major][j] })
		for _, minor := range majorsToMinors[major] {
			version := api.Version{Major: major, Minor: minor}
			versions = append(versions, version)
		}
	}
	return versions
}

func indexOfApiVersion(versions []api.Version, desiredVersion api.Version) int {
	for i, v := range versions {
		if v.Major > desiredVersion.Major {
			return i
		}
		if v.Major == desiredVersion.Major && v.Minor >= desiredVersion.Minor {
			return i
		}
	}
	return len(versions) - 1
}

// PathHandler ...
type PathHandler struct {
	Path    string
	Handler http.HandlerFunc
	ID      int
}

// CreateRouteMap returns a map of methods to a slice of paths and handlers; wrapping the handlers in the appropriate middleware. Uses Semantic Versioning: routes are added to every subsequent minor version, but not subsequent major versions. For example, a 1.2 route is added to 1.3 but not 2.1. Also truncates '2.0' to '2', creating succinct major versions.
// Returns the map of routes, and a map of API versions served.
func CreateRouteMap(rs []Route, rawRoutes []RawRoute, disabledRouteIDs []int, perlHandler http.HandlerFunc, authBase middleware.AuthBase, reqTimeOutSeconds int, cacheTime time.Duration, disableReadWhileWriter bool) (map[string][]PathHandler, map[api.Version]struct{}) {
	// TODO strong types for method, path
	versions := getSortedRouteVersions(rs)
	requestTimeout := middleware.DefaultRequestTimeout
	if reqTimeOutSeconds > 0 {
		requestTimeout = time.Second * time.Duration(reqTimeOutSeconds)
	}
	disabledRoutes := GetRouteIDMap(disabledRouteIDs)
	m := map[string][]PathHandler{}
	for _, r := range rs {
		versionI := indexOfApiVersion(versions, r.Version)
		nextMajorVer := r.Version.Major + 1
		_, isDisabledRoute := disabledRoutes[r.ID]
		for _, version := range versions[versionI:] {
			if version.Major >= nextMajorVer {
				break
			}
			vstr := strconv.FormatUint(version.Major, 10) + "." + strconv.FormatUint(version.Minor, 10)
			path := RoutePrefix + "/" + vstr + "/" + r.Path

			routeDisableReadWhileWriter := disableReadWhileWriter
			routeCacheTime := cacheTime
			if !r.Cache {
				routeDisableReadWhileWriter = true
				routeCacheTime = 0
			}

			middlewares := getRouteMiddleware(r.Middlewares, authBase, r.Authenticated, r.RequiredPrivLevel, requestTimeout, routeCacheTime, routeDisableReadWhileWriter)

			if isDisabledRoute {
				m[r.Method] = append(m[r.Method], PathHandler{Path: path, Handler: middleware.WrapAccessLog(authBase.Secret, middleware.DisabledRouteHandler()), ID: r.ID})
			} else {
				m[r.Method] = append(m[r.Method], PathHandler{Path: path, Handler: middleware.Use(r.Handler, middlewares), ID: r.ID})
			}
			log.Infof("adding route %v %v\n", r.Method, path)
		}
	}
	for _, r := range rawRoutes {
		middlewares := getRouteMiddleware(r.Middlewares, authBase, r.Authenticated, r.RequiredPrivLevel, requestTimeout, 0, true)
		m[r.Method] = append(m[r.Method], PathHandler{Path: r.Path, Handler: middleware.Use(r.Handler, middlewares)})
		log.Infof("adding raw route %v %v\n", r.Method, r.Path)
	}

	versionSet := map[api.Version]struct{}{}
	for _, version := range versions {
		versionSet[version] = struct{}{}
	}
	return m, versionSet
}

func getRouteMiddleware(middlewares []middleware.Middleware, authBase middleware.AuthBase, authenticated bool, privLevel int, requestTimeout time.Duration, cacheTime time.Duration, disableReadWhileWriter bool) []middleware.Middleware {
	if middlewares == nil {
		middlewares = middleware.GetDefault(authBase.Secret, requestTimeout)
	}

	// CacheWrapper has to be here, rather than middleware.GetDefault, because otherwise the user won't be in the context.
	// DEBUG the above statment is false

	if cacheTime > 0 || !disableReadWhileWriter {
		// DEBUG: Prepend. Issues?
		middlewares = append([]middleware.Middleware{middleware.SmallCacheWrapper(cacheTime, disableReadWhileWriter)}, middlewares...)
		// middlewares = append(middlewares, middleware.SmallCacheWrapper(cacheTime, disableReadWhileWriter))
	}

	if authenticated { // a privLevel of zero is an unauthenticated endpoint.
		authWrapper := authBase.GetWrapper(privLevel)
		// DEBUG: Prepend. Issues?
		middlewares = append([]middleware.Middleware{authWrapper}, middlewares...)
		// middlewares = append(middlewares, authWrapper)
	}

	return middlewares
}

// CompileRoutes - takes a map of methods to paths and handlers, and returns a map of methods to CompiledRoutes
func CompileRoutes(routes map[string][]PathHandler) map[string][]CompiledRoute {
	compiledRoutes := map[string][]CompiledRoute{}
	for method, mRoutes := range routes {
		for _, pathHandler := range mRoutes {
			route := pathHandler.Path
			handler := pathHandler.Handler
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
			regex := regexp.MustCompile(route)
			id := pathHandler.ID
			compiledRoutes[method] = append(compiledRoutes[method], CompiledRoute{Handler: handler, Regex: regex, Params: params, ID: id})
		}
	}
	return compiledRoutes
}

// Handler - generic handler func used by the Handlers hooking into the routes
func Handler(
	routes map[string][]CompiledRoute,
	versions map[api.Version]struct{},
	catchall http.Handler,
	db *sqlx.DB,
	cfg *config.Config,
	getReqID func() uint64,
	plugins plugin.Plugins,
	tv trafficvault.TrafficVault,
	w http.ResponseWriter,
	r *http.Request,
) {
	reqID := getReqID()

	reqIDStr := strconv.FormatUint(reqID, 10)
	log.Infoln(r.Method + " " + r.URL.Path + "?" + r.URL.RawQuery + " handling (reqid " + reqIDStr + ")")
	start := time.Now()
	defer func() {
		log.Infoln(r.Method + " " + r.URL.Path + "?" + r.URL.RawQuery + " handled (reqid " + reqIDStr + ") in " + time.Since(start).String())
	}()

	ctx := r.Context()
	ctx = context.WithValue(ctx, api.DBContextKey, db)
	ctx = context.WithValue(ctx, api.ConfigContextKey, cfg)
	ctx = context.WithValue(ctx, api.ReqIDContextKey, reqID)
	ctx = context.WithValue(ctx, api.TrafficVaultContextKey, tv)

	// plugins have no pre-parsed path params, but add an empty map so they can use the api helper funcs that require it.
	pluginCtx := context.WithValue(ctx, api.PathParamsKey, map[string]string{})
	pluginReq := r.WithContext(pluginCtx)

	onReqData := plugin.OnRequestData{Data: plugin.Data{RequestID: reqID, AppCfg: *cfg}, W: w, R: pluginReq}
	if handled := plugins.OnRequest(onReqData); handled {
		return
	}

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

		routeCtx := context.WithValue(ctx, api.PathParamsKey, params)
		r = r.WithContext(routeCtx)
		r.Header.Add(middleware.RouteID, strconv.Itoa(compiledRoute.ID))
		compiledRoute.Handler(w, r)
		return
	}
	if IsRequestAPIAndUnknownVersion(r, versions) {
		h := middleware.WrapAccessLog(cfg.Secrets[0], middleware.NotImplementedHandler())
		h.ServeHTTP(w, r)
		return
	}

	catchall.ServeHTTP(w, r)
}

// IsRequestAPIAndUnknownVersion returns true if the request starts with `/api` and is a version not in the list of versions.
func IsRequestAPIAndUnknownVersion(req *http.Request, versions map[api.Version]struct{}) bool {
	pathParts := strings.Split(req.URL.Path, "/")
	if len(pathParts) < 2 {
		return false // path doesn't start with `/api`, so it's not an api request
	}
	if strings.ToLower(pathParts[1]) != "api" {
		return false // path doesn't start with `/api`, so it's not an api request
	}
	if len(pathParts) < 3 {
		return true // path starts with `/api` but not `/api/{version}`, so it's an api request, and an unknown/nonexistent version.
	}

	version, err := stringVersionToApiVersion(pathParts[2])
	if err != nil {
		return true // path starts with `/api`, and version isn't a number, so it's an unknown/nonexistent version
	}
	if _, versionExists := versions[version]; versionExists {
		return false // path starts with `/api` and version exists, so it's API but a known version
	}
	return true // path starts with `/api`, and version is unknown
}

func stringVersionToApiVersion(version string) (api.Version, error) {
	versionParts := strings.Split(version, ".")
	if len(versionParts) < 2 {
		return api.Version{}, errors.New("error parsing version " + version)
	}
	major, err := strconv.ParseUint(versionParts[0], 10, 64)
	if err != nil {
		return api.Version{}, errors.New("error parsing version " + version)
	}
	minor, err := strconv.ParseUint(versionParts[1], 10, 64)
	if err != nil {
		return api.Version{}, errors.New("error parsing version " + version)
	}
	return api.Version{Major: major, Minor: minor}, nil
}

// RegisterRoutes - parses the routes and registers the handlers with the Go Router
func RegisterRoutes(d ServerData) error {
	routeSlice, rawRoutes, catchall, err := Routes(d)
	if err != nil {
		return err
	}

	authBase := middleware.AuthBase{Secret: d.Config.Secrets[0], Override: nil} //we know d.Config.Secrets is a slice of at least one or start up would fail.
	routes, versions := CreateRouteMap(routeSlice, rawRoutes, d.DisabledRoutes, handlerToFunc(catchall), authBase, d.RequestTimeout, time.Duration(*d.CacheMS)*time.Millisecond, d.DisableReadWhileWriter)

	compiledRoutes := CompileRoutes(routes)
	getReqID := nextReqIDGetter()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		Handler(compiledRoutes, versions, catchall, d.DB, &d.Config, getReqID, d.Plugins, d.TrafficVault, w, r)
	})
	return nil
}

// nextReqIDGetter returns a function for getting incrementing identifiers. The returned func is safe for calling with multiple goroutines. Note the returned identifiers will not be unique after the max uint64 value.
func nextReqIDGetter() func() uint64 {
	id := uint64(0)
	return func() uint64 {
		return atomic.AddUint64(&id, 1)
	}
}
