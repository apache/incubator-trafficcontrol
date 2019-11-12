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
import "net/http"

import "github.com/apache/trafficcontrol/lib/go-tc"
import "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"

var DIVISION = tc.Division{
	ID:          1,
	LastUpdated: *CURRENT_TIME,
	Name:        "Mock",
}
var DIVISIONS = []tc.Division{DIVISION}

func divisions(w http.ResponseWriter, r *http.Request) {
	common(w)
	if r.Method == http.MethodGet {
		api.WriteResp(w, r, DIVISIONS)
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"alerts":[{"level":"error","text":"This method hasn't yet been implemented."}]}`))
	}
}

var REGION = tc.Region{
	DivisionName: DIVISION.Name,
	Division:     DIVISION.ID,
	ID:           1,
	LastUpdated:  *CURRENT_TIME,
	Name:         "Mock",
}
var REGIONS = []tc.Region{REGION}

func regions(w http.ResponseWriter, r *http.Request) {
	common(w)
	if r.Method == http.MethodGet {
		api.WriteResp(w, r, REGIONS)
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"alerts":[{"level":"error","text":"This method hasn't yet been implemented."}]}`))
	}
}

var LOCATION = tc.PhysLocation{
	Address:     "555 Mock Lane",
	City:        "Mockville",
	Comments:    "This isn't a real place",
	Email:       "admin@cdn.test",
	ID:          1,
	LastUpdated: *CURRENT_TIME,
	Name:        "Mock",
	Phone:       "1-555-555-5555",
	POC:         "Nobody",
	RegionID:    REGION.ID,
	RegionName:  REGION.Name,
	ShortName:   "Mock",
	State:       "Denial",
	Zip:         "0",
}
var LOCATIONS = []tc.PhysLocation{LOCATION}

func locations(w http.ResponseWriter, r *http.Request) {
	common(w)
	if r.Method == http.MethodGet {
		api.WriteResp(w, r, LOCATIONS)
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"alerts":[{"level":"error","text":"This method hasn't yet been implemented."}]}`))
	}
}
