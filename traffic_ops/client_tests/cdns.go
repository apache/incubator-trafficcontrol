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

// Static CDN Fields
var ALL_CDN = "ALL"
var ALL_CDN_ID = 1
var ALL_CDN_DOMAINNAME = "-"
var ALL_CDN_DNSSEC_ENABLED = false
var CDN = "Mock-CDN"
var CDN_ID = 2
var CDN_DOMAIN_NAME = "mock.cdn.test"
var CDN_DNSSEC_ENABLED = false

var CDNS = []tc.CDNNullable{
	tc.CDNNullable{
		DNSSECEnabled: &ALL_CDN_DNSSEC_ENABLED,
		DomainName:    &ALL_CDN_DOMAINNAME,
		ID:            &ALL_CDN_ID,
		LastUpdated:   CURRENT_TIME,
		Name:          &ALL_CDN,
	},
	tc.CDNNullable{
		DNSSECEnabled: &CDN_DNSSEC_ENABLED,
		DomainName:    &CDN_DOMAIN_NAME,
		ID:            &CDN_ID,
		Name:          &CDN,
	},
}

func CDNs(w http.ResponseWriter, r *http.Request) {
	common(w)
	if r.Method == http.MethodGet {
		api.WriteResp(w, r, CDNS)
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"alerts":[{"level":"error","text":"This method hasn't yet been implemented."}]}`))
	}
}
