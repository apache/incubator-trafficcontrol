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
	"database/sql"
	"net/http"
	"regexp"
	"strings"
)

type ServerData struct {
	Config
	DB *sql.DB
}

type ParamMap map[string]string

type RegexHandlerFunc func(w http.ResponseWriter, r *http.Request, params ParamMap)

type CompiledRoute struct {
	Handler RegexHandlerFunc
	Regex   *regexp.Regexp
	Params  []string
}

func CompileRoutes(routes *map[string]RegexHandlerFunc) map[string]CompiledRoute {
	compiledRoutes := map[string]CompiledRoute{}
	for route, handler := range *routes {
		originalRoute := route
		var params []string
		for open := strings.Index(route, "{"); open > 0; open = strings.Index(route, "{") {
			close := strings.Index(route, "}")
			if close < 0 {
				panic("malformed route")
			}
			param := route[open+1 : close]

			params = append(params, param)
			route = route[:open] + `(.+)` + route[close+1:]
		}
		regex := regexp.MustCompile(route)
		compiledRoutes[originalRoute] = CompiledRoute{Handler: handler, Regex: regex, Params: params}
	}
	return compiledRoutes
}

func Handler(routes map[string]CompiledRoute, catchall http.Handler, w http.ResponseWriter, r *http.Request) {
	requested := r.URL.Path[1:]

	for _, compiledRoute := range routes {
		match := compiledRoute.Regex.FindStringSubmatch(requested)
		if len(match) == 0 {
			continue
		}

		params := map[string]string{}
		for i, v := range compiledRoute.Params {
			params[v] = match[i+1]
		}
		compiledRoute.Handler(w, r, params)
		return
	}
	catchall.ServeHTTP(w, r)
}

func RegisterRoutes(d ServerData) error {
	routes, catchall, err := GetRoutes(d)
	if err != nil {
		return err
	}

	compiledRoutes := CompileRoutes(&routes)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		Handler(compiledRoutes, catchall, w, r)
	})

	return nil
}
