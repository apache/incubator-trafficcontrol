package api

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

import "bytes"
import "net/http"
import "regexp"
import "strconv"
//import trops "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang"
import log "github.com/apache/trafficcontrol/lib/go-log"

// todo - make the version string modular and accessible everywhere
const ServerString = "Traffic Operations/3.0.0";

var APIVersions []float64;

// CompiledRoute ...
type CompiledRoute struct {
	Handler http.HandlerFunc
	Regex   *regexp.Regexp
	Params  []string
}


var AllRoutes *map[string][]CompiledRoute;
// Writes a message indicating an internal server error back to the client (in plain text)
func errorResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError);
}

// Handles the disallowed request methods for this endpoint
func AvailableRoutesBadMethodHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "OPTIONS,GET");
	w.Header().Set("Server", ServerString);
	w.WriteHeader(http.StatusMethodNotAllowed);
}

func AvailableVersionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", ServerString);

	// Check for a CORS preflight request (all headers accepted, so immediate response given)
	if r.Header.Get("Origin") != "" &&
	   r.Header.Get("Access-Control-Request-Method") != "" &&
	   r.Header.Get("Access-Control-Request-Headers") != "" {

		w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET");
		w.Header().Set("Access-Control-Allow-Headers", "*");
		w.WriteHeader(http.StatusNoContent);
		return;
	}

	// check that this has been set
	if APIVersions == nil {
		w.WriteHeader(http.StatusNoContent);
		log.Warnln("API versions were requested, but weren't set!");
		return;
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8");

	var body bytes.Buffer;
	for _, version := range APIVersions {
		_, err := body.WriteString(strconv.FormatFloat(version, 'f', 1, 64));
		if err != nil {
			log.Errorf("Unable to append API version to buffer: %s", err.Error());
			errorResponse(w);
			return;
		}
		body.WriteRune('\n');
	}

	w.Write(body.Bytes());
	log.Infof("API versions: '%s'", body.String())
}

// Writes a list of all available API routes in a response to the client in plaintext
// (or writes an error message via errorResponse if something wicked happens)
func AvailableRoutesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", ServerString);

	// if for some reason the variable didn't get set properly, return nothing
	if AllRoutes == nil {
		w.Header().Set("Allow", "OPTIONS,GET");
		w.WriteHeader(http.StatusNoContent);
		log.Warnln("API routes were requested, but weren't set!");
		return;
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8");

	var body bytes.Buffer;
	for method, routes := range *AllRoutes {
		for _, route := range routes {
			_, err := body.WriteString(method);
			if err != nil {
				log.Errorf("Unable to append method to routes buffer: %s", err.Error());
				errorResponse(w);
				return;
			}
			body.WriteRune(' ');

			_, err = body.WriteString(route.Regex.String());
			if err != nil {
				log.Errorf("Unable to append route to routes buffer: %s", err.Error());
				errorResponse(w);
				return;
			}
			body.WriteRune('\n');
		}
	}
	w.Header().Set("Allow", "OPTIONS,GET");
	w.Write(body.Bytes());
}
