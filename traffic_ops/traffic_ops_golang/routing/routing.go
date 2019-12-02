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

	"github.com/jmoiron/sqlx"
)

// RoutePrefix ...
const RoutePrefix = "^api" // TODO config?

// Middleware ...
type Middleware func(handlerFunc http.HandlerFunc) http.HandlerFunc

// Route ...
type Route struct {
	// Order matters! Do not reorder this! Routes() uses positional construction for readability.
	Version           float64
	Method            string
	Path              string
	Handler           http.HandlerFunc
	RequiredPrivLevel int
	Authenticated     bool
	Middlewares       []Middleware
}

// RawRoute is an HTTP route to be served at the root, rather than under /api/version. Raw Routes should be rare, and almost exclusively converted old Perl routes which have yet to be moved to an API path.
type RawRoute struct {
	// Order matters! Do not reorder this! Routes() uses positional construction for readability.
	Method            string
	Path              string
	Handler           http.HandlerFunc
	RequiredPrivLevel int
	Authenticated     bool
	Middlewares       []Middleware
}

func getDefaultMiddleware(secret string, requestTimeout time.Duration) []Middleware {
	return []Middleware{getWrapAccessLog(secret), timeOutWrapper(requestTimeout), wrapHeaders, wrapPanicRecover}
}

// ServerData ...
type ServerData struct {
	config.Config
	DB        *sqlx.DB
	Profiling *bool // Yes this is a field in the config but we want to live reload this value and NOT the entire config
	Plugins   plugin.Plugins
}

// CompiledRoute ...
type CompiledRoute struct {
	Handler http.HandlerFunc
	Regex   *regexp.Regexp
	Params  []string
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
	Path    string
	Handler http.HandlerFunc
}

// CreateRouteMap returns a map of methods to a slice of paths and handlers; wrapping the handlers in the appropriate middleware. Uses Semantic Versioning: routes are added to every subsequent minor version, but not subsequent major versions. For example, a 1.2 route is added to 1.3 but not 2.1. Also truncates '2.0' to '2', creating succinct major versions.
// Returns the map of routes, and a map of API versions served.
func CreateRouteMap(rs []Route, rawRoutes []RawRoute, authBase AuthBase, reqTimeOutSeconds int) (map[string][]PathHandler, map[float64]struct{}) {
	// TODO strong types for method, path
	versions := getSortedRouteVersions(rs)
	api.APIVersions = versions
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
			middlewares := getRouteMiddleware(r.Middlewares, authBase, r.Authenticated, r.RequiredPrivLevel, requestTimeout)
			m[r.Method] = append(m[r.Method], PathHandler{Path: path, Handler: use(r.Handler, middlewares)})
			log.Infof("adding route %v %v\n", r.Method, path)
		}
	}
	for _, r := range rawRoutes {
		middlewares := getRouteMiddleware(r.Middlewares, authBase, r.Authenticated, r.RequiredPrivLevel, requestTimeout)
		m[r.Method] = append(m[r.Method], PathHandler{Path: r.Path, Handler: use(r.Handler, middlewares)})
		log.Infof("adding raw route %v %v\n", r.Method, r.Path)
	}

	versionSet := map[float64]struct{}{}
	for _, version := range versions {
		versionSet[version] = struct{}{}
	}

	return m, versionSet
}

func getRouteMiddleware(middlewares []Middleware, authBase AuthBase, authenticated bool, privLevel int, requestTimeout time.Duration) []Middleware {
	if middlewares == nil {
		middlewares = getDefaultMiddleware(authBase.secret, requestTimeout)
	}
	if authenticated { // a privLevel of zero is an unauthenticated endpoint.
		authWrapper := authBase.GetWrapper(privLevel)
		middlewares = append(middlewares, authWrapper)
	}
	return middlewares
}

// CompileRoutes - takes a map of methods to paths and handlers, and returns a map of methods to CompiledRoutes
func CompileRoutes(routes map[string][]PathHandler) map[string][]api.CompiledRoute {
	compiledRoutes := map[string][]api.CompiledRoute{}
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
			compiledRoutes[method] = append(compiledRoutes[method], api.CompiledRoute{Handler: handler, Regex: regex, Params: params})
		}
	}
	return compiledRoutes
}

// Handler - generic handler func used by the Handlers hooking into the routes
func Handler(
	routes map[string][]api.CompiledRoute,
	versions map[float64]struct{},
	catchall http.Handler,
	db *sqlx.DB,
	cfg *config.Config,
	getReqID func() uint64,
	plugins plugin.Plugins,
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
		compiledRoute.Handler(w, r)
		return
	}

	if IsRequestAPIAndUnknownVersion(r, versions) {
		h := wrapAccessLog(cfg.Secrets[0], NotImplementedHandler())
		h.ServeHTTP(w, r)
		return
	}

	catchall.ServeHTTP(w, r)
}

// IsRequestAPIAndUnknownVersion returns true if the request starts with `/api` and is a version not in the list of versions.
func IsRequestAPIAndUnknownVersion(req *http.Request, versions map[float64]struct{}) bool {
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

	version, err := strconv.ParseFloat(pathParts[2], 64)
	if err != nil {
		return true // path starts with `/api`, and version isn't a number, so it's an unknown/nonexistent version
	}
	if _, versionExists := versions[version]; versionExists {
		return false // path starts with `/api` and version exists, so it's API but a known version
	}
	return true // path starts with `/api`, and version is unknown
}

// RegisterRoutes - parses the routes and registers the handlers with the Go Router
func RegisterRoutes(d ServerData) error {
	routeSlice, rawRoutes, catchall, err := Routes(d)
	if err != nil {
		return err
	}

	authBase := AuthBase{secret: d.Config.Secrets[0], override: nil} //we know d.Config.Secrets is a slice of at least one or start up would fail.
	routes, versions := CreateRouteMap(routeSlice, rawRoutes, authBase, d.RequestTimeout)
	compiledRoutes := CompileRoutes(routes)
	getReqID := nextReqIDGetter()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		Handler(compiledRoutes, versions, catchall, d.DB, &d.Config, getReqID, d.Plugins, w, r)
	})

	api.AllRoutes = &compiledRoutes;
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
