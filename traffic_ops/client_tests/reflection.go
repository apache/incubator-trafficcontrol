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

var STATUS_OFFLINE = tc.Status{
	Description: "Server is Offline. Not active in any configuration.",
	ID:          1,
	LastUpdated: *CURRENT_TIME,
	Name:        "OFFLINE",
}
var STATUS_ONLINE = tc.Status{
	Description: "Server is online.",
	ID:          2,
	LastUpdated: *CURRENT_TIME,
	Name:        "ONLINE",
}
var STATUS_REPORTED = tc.Status{
	Description: "Server is online and reported in the health protocol.",
	ID:          3,
	LastUpdated: *CURRENT_TIME,
	Name:        "REPORTED",
}
var STATUS_ADMIN_DOWN = tc.Status{
	Description: "Sever is administrative down and does not receive traffic.",
	ID:          4,
	LastUpdated: *CURRENT_TIME,
	Name:        "ADMIN_DOWN",
}
var STATUS_CCR_IGNORE = tc.Status{
	Description: "Server is ignored by traffic router.",
	ID:          5,
	LastUpdated: *CURRENT_TIME,
	Name:        "CCR_IGNORE",
}
var STATUS_PRE_PROD = tc.Status{
	Description: "Pre Production. Not active in any configuration.",
	ID:          6,
	LastUpdated: *CURRENT_TIME,
	Name:        "PRE_PROD",
}

var STATUSES = []tc.Status{STATUS_OFFLINE, STATUS_ONLINE, STATUS_REPORTED, STATUS_ADMIN_DOWN, STATUS_CCR_IGNORE, STATUS_PRE_PROD}

func statuses(w http.ResponseWriter, r *http.Request) {
	common(w)
	if r.Method == http.MethodGet {
		api.WriteResp(w, r, STATUSES)
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"alerts":[{"level":"error","text":"This method hasn't yet been implemented."}]}`))
	}
}

var TYPE_HTTP = tc.Type{
	ID:          1,
	LastUpdated: *CURRENT_TIME,
	Name:        "HTTP",
	Description: "HTTP Content Routing",
	UseInTable:  "deliveryservice",
}
var TYPE_HTTP_NO_CACHE = tc.Type{
	ID:          2,
	LastUpdated: *CURRENT_TIME,
	Name:        "HTTP_NO_CACHE",
	Description: "HTTP Content Routing, no caching",
	UseInTable:  "deliveryservice",
}
var TYPE_HTTP_LIVE = tc.Type{
	ID:          3,
	LastUpdated: *CURRENT_TIME,
	Name:        "HTTP_LIVE",
	Description: "HTTP Content routing cache in RAM",
	UseInTable:  "deliveryservice",
}
var TYPE_HTTP_LIVE_NATNL = tc.Type{
	ID:          4,
	LastUpdated: *CURRENT_TIME,
	Name:        "HTTP_LIVE_NATNL",
	Description: "HTTP Content routing, RAM cache, National",
	UseInTable:  "deliveryservice",
}
var TYPE_DNS = tc.Type{
	ID:          5,
	LastUpdated: *CURRENT_TIME,
	Name:        "DNS",
	Description: "DNS Content Routing",
	UseInTable:  "deliveryservice",
}
var TYPE_DNS_LIVE = tc.Type{
	ID:          6,
	LastUpdated: *CURRENT_TIME,
	Name:        "DNS_LIVE",
	Description: "DNS Content routing, RAM cache, Local",
	UseInTable:  "deliveryservice",
}
var TYPE_DNS_LIVE_NATNL = tc.Type{
	ID:          7,
	LastUpdated: *CURRENT_TIME,
	Name:        "DNS_LIVE_NATNL",
	Description: "DNS Content routing, RAM cache, National",
	UseInTable:  "deliveryservice",
}
var TYPE_ANY_MAP = tc.Type{
	ID:          8,
	LastUpdated: *CURRENT_TIME,
	Name:        "ANY_MAP",
	Description: "No Content Routing - arbitrary remap at the edge, no Traffic Router config",
	UseInTable:  "deliveryservice",
}
var TYPE_STEERING = tc.Type{
	ID:          9,
	LastUpdated: *CURRENT_TIME,
	Name:        "STEERING",
	Description: "Steering Delivery Service",
	UseInTable:  "deliveryservice",
}
var TYPE_CLIENT_STEERING = tc.Type{
	ID:          10,
	LastUpdated: *CURRENT_TIME,
	Name:        "CLIENT_STEERING",
	Description: "Client-Controlled Steering Delivery Service",
	UseInTable:  "deliveryservice",
}
var TYPE_EDGE = tc.Type{
	ID:          11,
	LastUpdated: *CURRENT_TIME,
	Name:        "EDGE",
	Description: "Edge Cache",
	UseInTable:  "server",
}
var TYPE_MID = tc.Type{
	ID:          12,
	LastUpdated: *CURRENT_TIME,
	Name:        "MID",
	Description: "Mid Tier Cache",
	UseInTable:  "server",
}
var TYPE_ORG = tc.Type{
	ID:          13,
	LastUpdated: *CURRENT_TIME,
	Name:        "ORG",
	Description: "Origin",
	UseInTable:  "server",
}
var TYPE_CCR = tc.Type{
	ID:          14,
	LastUpdated: *CURRENT_TIME,
	Name:        "CCR",
	Description: "Traffic Router",
	UseInTable:  "server",
}
var TYPE_RASCAL = tc.Type{
	ID:          15,
	LastUpdated: *CURRENT_TIME,
	Name:        "RASCAL",
	Description: "Traffic Monitor",
	UseInTable:  "server",
}
var TYPE_RIAK = tc.Type{
	ID:          16,
	LastUpdated: *CURRENT_TIME,
	Name:        "RIAK",
	Description: "Riak keystore",
	UseInTable:  "server",
}
var TYPE_INFLUXDB = tc.Type{
	ID:          17,
	LastUpdated: *CURRENT_TIME,
	Name:        "INFLUXDB",
	Description: "influxDb server",
	UseInTable:  "server",
}
var TYPE_TRAFFIC_ANALYTICS = tc.Type{
	ID:          18,
	LastUpdated: *CURRENT_TIME,
	Name:        "TRAFFIC_ANALYTICS",
	Description: "traffic analytics server",
	UseInTable:  "server",
}
var TYPE_TRAFFIC_OPS = tc.Type{
	ID:          19,
	LastUpdated: *CURRENT_TIME,
	Name:        "TRAFFIC_OPS",
	Description: "traffic ops server",
	UseInTable:  "server",
}
var TYPE_TRAFFIC_OPS_DB = tc.Type{
	ID:          20,
	LastUpdated: *CURRENT_TIME,
	Name:        "TRAFFIC_OPS_DB",
	Description: "traffic ops DB server",
	UseInTable:  "server",
}
var TYPE_TRAFFIC_PORTAL = tc.Type{
	ID:          21,
	LastUpdated: *CURRENT_TIME,
	Name:        "TRAFFIC_PORTAL",
	Description: "traffic portal server",
	UseInTable:  "server",
}
var TYPE_TRAFFIC_STATS = tc.Type{
	ID:          22,
	LastUpdated: *CURRENT_TIME,
	Name:        "TRAFFIC_STATS",
	Description: "traffic stats server",
	UseInTable:  "server",
}
var TYPE_EDGE_LOC = tc.Type{
	ID:          23,
	LastUpdated: *CURRENT_TIME,
	Name:        "EDGE_LOC",
	Description: "Edge Logical Location",
	UseInTable:  "cachegroup",
}
var TYPE_MID_LOC = tc.Type{
	ID:          24,
	LastUpdated: *CURRENT_TIME,
	Name:        "MID_LOC",
	Description: "Mid Logical Location",
	UseInTable:  "cachegroup",
}
var TYPE_ORG_LOC = tc.Type{
	ID:          25,
	LastUpdated: *CURRENT_TIME,
	Name:        "ORG_LOC",
	Description: "Origin Logical Site",
	UseInTable:  "cachegroup",
}
var TYPE_TR_LOC = tc.Type{
	ID:          26,
	LastUpdated: *CURRENT_TIME,
	Name:        "TR_LOC",
	Description: "Traffic Router Logical Location",
	UseInTable:  "cachegroup",
}
var TYPE_CHECK_EXTENSION_BOOL = tc.Type{
	ID:          27,
	LastUpdated: *CURRENT_TIME,
	Name:        "CHECK_EXTENSION_BOOL",
	Description: "Extension for checkmark in Server Check",
	UseInTable:  "to_extension",
}
var TYPE_CHECK_EXTENSION_NUM = tc.Type{
	ID:          28,
	LastUpdated: *CURRENT_TIME,
	Name:        "CHECK_EXTENSION_NUM",
	Description: "Extension for int value in Server Check",
	UseInTable:  "to_extension",
}
var TYPE_CHECK_EXTENSION_OPEN_SLOT = tc.Type{
	ID:          29,
	LastUpdated: *CURRENT_TIME,
	Name:        "CHECK_EXTENSION_OPEN_SLOT",
	Description: "Open slot for check in Server Status",
	UseInTable:  "to_extension",
}
var TYPE_CONFIG_EXTENSION = tc.Type{
	ID:          30,
	LastUpdated: *CURRENT_TIME,
	Name:        "CONFIG_EXTENSION",
	Description: "Extension for additional configuration file",
	UseInTable:  "to_extension",
}
var TYPE_STATISTIC_EXTENSION = tc.Type{
	ID:          31,
	LastUpdated: *CURRENT_TIME,
	Name:        "STATISTIC_EXTENSION",
	Description: "Extension source for 12M graphs",
	UseInTable:  "to_extension",
}
var TYPE_HOST_REGEXP = tc.Type{
	ID:          32,
	LastUpdated: *CURRENT_TIME,
	Name:        "HOST_REGEXP",
	Description: "Host header regular expression",
	UseInTable:  "regex",
}
var TYPE_HEADER_REGEXP = tc.Type{
	ID:          33,
	LastUpdated: *CURRENT_TIME,
	Name:        "HEADER_REGEXP",
	Description: "HTTP header regular expression",
	UseInTable:  "regex",
}
var TYPE_PATH_REGEXP = tc.Type{
	ID:          34,
	LastUpdated: *CURRENT_TIME,
	Name:        "PATH_REGEXP",
	Description: "URL path regular expression",
	UseInTable:  "regex",
}
var TYPE_STEERING_REGEXP = tc.Type{
	ID:          35,
	LastUpdated: *CURRENT_TIME,
	Name:        "STEERING_REGEXP",
	Description: "Steering target filter regular expression",
	UseInTable:  "regex",
}
var TYPE_RESOLVE4 = tc.Type{
	ID:          36,
	LastUpdated: *CURRENT_TIME,
	Name:        "RESOLVE4",
	Description: "federation type resolve4",
	UseInTable:  "federation",
}
var TYPE_RESOLVE6 = tc.Type{
	ID:          37,
	LastUpdated: *CURRENT_TIME,
	Name:        "RESOLVE6",
	Description: "federation type resolve6",
	UseInTable:  "federation",
}
var TYPE_A_RECORD = tc.Type{
	ID:          38,
	LastUpdated: *CURRENT_TIME,
	Name:        "A_RECORD",
	Description: "Static DNS A entry",
	UseInTable:  "staticdnsentry",
}
var TYPE_AAAA_RECORD = tc.Type{
	ID:          39,
	LastUpdated: *CURRENT_TIME,
	Name:        "AAAA_RECORD",
	Description: "Static DNS AAAA entry",
	UseInTable:  "staticdnsentry",
}
var TYPE_CNAME_RECORD = tc.Type{
	ID:          40,
	LastUpdated: *CURRENT_TIME,
	Name:        "CNAME_RECORD",
	Description: "Static DNS CNAME entry",
	UseInTable:  "staticdnsentry",
}
var TYPE_TXT_RECORD = tc.Type{
	ID:          41,
	LastUpdated: *CURRENT_TIME,
	Name:        "TXT_RECORD",
	Description: "Static DNS TXT entry",
	UseInTable:  "staticdnsentry",
}
var TYPE_STEERING_WEIGHT = tc.Type{
	ID:          42,
	LastUpdated: *CURRENT_TIME,
	Name:        "STEERING_WEIGHT",
	Description: "Weighted steering target",
	UseInTable:  "steering_target",
}
var TYPE_STEERING_ORDER = tc.Type{
	ID:          43,
	LastUpdated: *CURRENT_TIME,
	Name:        "STEERING_ORDER",
	Description: "Ordered steering target",
	UseInTable:  "steering_target",
}
var TYPE_STEERING_GEO_ORDER = tc.Type{
	ID:          44,
	LastUpdated: *CURRENT_TIME,
	Name:        "STEERING_GEO_ORDER",
	Description: "Geo-ordered steering target",
	UseInTable:  "steering_target",
}
var TYPE_STEERING_GEO_WEIGHT = tc.Type{
	ID:          45,
	LastUpdated: *CURRENT_TIME,
	Name:        "STEERING_GEO_WEIGHT",
	Description: "Geo-weighted steering target",
	UseInTable:  "steering_target",
}

var TYPES = []tc.Type{
	TYPE_HTTP,
	TYPE_HTTP_NO_CACHE,
	TYPE_HTTP_LIVE,
	TYPE_HTTP_LIVE_NATNL,
	TYPE_DNS,
	TYPE_DNS_LIVE,
	TYPE_DNS_LIVE_NATNL,
	TYPE_ANY_MAP,
	TYPE_STEERING,
	TYPE_CLIENT_STEERING,
	TYPE_EDGE,
	TYPE_MID,
	TYPE_ORG,
	TYPE_CCR,
	TYPE_RASCAL,
	TYPE_RIAK,
	TYPE_INFLUXDB,
	TYPE_TRAFFIC_ANALYTICS,
	TYPE_TRAFFIC_OPS,
	TYPE_TRAFFIC_OPS_DB,
	TYPE_TRAFFIC_PORTAL,
	TYPE_TRAFFIC_STATS,
	TYPE_EDGE_LOC,
	TYPE_MID_LOC,
	TYPE_ORG_LOC,
	TYPE_TR_LOC,
	TYPE_CHECK_EXTENSION_BOOL,
	TYPE_CHECK_EXTENSION_NUM,
	TYPE_CHECK_EXTENSION_OPEN_SLOT,
	TYPE_CONFIG_EXTENSION,
	TYPE_STATISTIC_EXTENSION,
	TYPE_HOST_REGEXP,
	TYPE_HEADER_REGEXP,
	TYPE_PATH_REGEXP,
	TYPE_STEERING_REGEXP,
	TYPE_RESOLVE4,
	TYPE_RESOLVE6,
	TYPE_A_RECORD,
	TYPE_AAAA_RECORD,
	TYPE_CNAME_RECORD,
	TYPE_TXT_RECORD,
	TYPE_STEERING_WEIGHT,
	TYPE_STEERING_ORDER,
	TYPE_STEERING_GEO_ORDER,
	TYPE_STEERING_GEO_WEIGHT,
}

func types(w http.ResponseWriter, r *http.Request) {
	common(w)
	if r.Method == http.MethodGet {
		api.WriteResp(w, r, TYPES)
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"alerts":[{"level":"error","text":"This method hasn't yet been implemented."}]}`))
	}
}
