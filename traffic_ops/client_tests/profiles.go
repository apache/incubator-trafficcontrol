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

// Static profile fields
var GLOBAL_PROFILE_ID = 1
var GLOBAL_PROFILE_NAME = "GLOBAL"
var GLOBAL_PROFILE_DESCRIPTION = "Global Traffic Ops profile, DO NOT DELETE"
var GLOBAL_PROFILE_CDN_NAME = ALL_CDN
var GLOBAL_PROFILE_CDN_ID = ALL_CDN_ID
var GLOBAL_PROFILE_ROUTING_DISABLED = false
var GLOBAL_PROFILE_TYPE = "UNK_PROFILE"
var TO_PROFILE_ID = 2
var TO_PROFILE_NAME = "TRAFFIC_OPS"
var TO_PROFILE_DESCRIPTION = "Traffic Ops profile"
var TO_PROFILE_CDN_NAME = ALL_CDN
var TO_PROFILE_CDN_ID = ALL_CDN_ID
var TO_PROFILE_ROUTING_DISABLED = false
var TO_PROFILE_TYPE = "UNK_PROFILE"
var TO_DB_PROFILE_ID = 3
var TO_DB_PROFILE_NAME = "TRAFFIC_OPS_DB"
var TO_DB_PROFILE_DESCRIPTION = "Traffic Ops DB profile"
var TO_DB_PROFILE_CDN_NAME = ALL_CDN
var TO_DB_PROFILE_CDN_ID = ALL_CDN_ID
var TO_DB_PROFILE_ROUTING_DISABLED = false
var TO_DB_PROFILE_TYPE = "UNK_PROFILE"
var EDGE_PROFILE_ID = 4
var EDGE_PROFILE_NAME = "ATS_EDGE_TIER_CACHE"
var EDGE_PROFILE_DESCRIPTION = "Edge Cache - Apache Traffic Server"
var EDGE_PROFILE_CDN_NAME = CDN
var EDGE_PROFILE_CDN_ID = CDN_ID
var EDGE_PROFILE_ROUTING_DISABLED = false
var EDGE_PROFILE_TYPE = "ATS_PROFILE"
var MID_PROFILE_ID = 5
var MID_PROFILE_NAME = "ATS_MID_TIER_CACHE"
var MID_PROFILE_DESCRIPTION = "Mid Cache - Apache Traffic Server"
var MID_PROFILE_CDN_NAME = CDN
var MID_PROFILE_CDN_ID = CDN_ID
var MID_PROFILE_ROUTING_DISABLED = false
var MID_PROFILE_TYPE = "ATS_PROFILE"

var PROFILES = []tc.ProfileNullable{
	tc.ProfileNullable{
		ID:              &GLOBAL_PROFILE_ID,
		LastUpdated:     CURRENT_TIME,
		Name:            &GLOBAL_PROFILE_NAME,
		Description:     &GLOBAL_PROFILE_DESCRIPTION,
		CDNName:         &GLOBAL_PROFILE_CDN_NAME,
		CDNID:           &GLOBAL_PROFILE_CDN_ID,
		RoutingDisabled: &GLOBAL_PROFILE_ROUTING_DISABLED,
		Type:            &GLOBAL_PROFILE_TYPE,
		Parameters:      nil,
	},
	tc.ProfileNullable{
		ID:              &TO_PROFILE_ID,
		LastUpdated:     CURRENT_TIME,
		Name:            &TO_PROFILE_NAME,
		Description:     &TO_PROFILE_DESCRIPTION,
		CDNName:         &TO_PROFILE_CDN_NAME,
		CDNID:           &TO_PROFILE_CDN_ID,
		RoutingDisabled: &TO_PROFILE_ROUTING_DISABLED,
		Type:            &TO_PROFILE_TYPE,
		Parameters:      nil,
	},
	tc.ProfileNullable{
		ID:              &TO_DB_PROFILE_ID,
		LastUpdated:     CURRENT_TIME,
		Name:            &TO_DB_PROFILE_NAME,
		Description:     &TO_DB_PROFILE_DESCRIPTION,
		CDNName:         &TO_DB_PROFILE_CDN_NAME,
		CDNID:           &TO_DB_PROFILE_CDN_ID,
		RoutingDisabled: &TO_DB_PROFILE_ROUTING_DISABLED,
		Type:            &TO_DB_PROFILE_TYPE,
		Parameters:      nil,
	},
	tc.ProfileNullable{
		ID:              &EDGE_PROFILE_ID,
		LastUpdated:     CURRENT_TIME,
		Name:            &EDGE_PROFILE_NAME,
		Description:     &EDGE_PROFILE_DESCRIPTION,
		CDNName:         &EDGE_PROFILE_CDN_NAME,
		CDNID:           &EDGE_PROFILE_CDN_ID,
		RoutingDisabled: &EDGE_PROFILE_ROUTING_DISABLED,
		Type:            &EDGE_PROFILE_TYPE,
		Parameters:      nil,
	},
	tc.ProfileNullable{
		ID:              &MID_PROFILE_ID,
		LastUpdated:     CURRENT_TIME,
		Name:            &MID_PROFILE_NAME,
		Description:     &MID_PROFILE_DESCRIPTION,
		CDNName:         &MID_PROFILE_CDN_NAME,
		CDNID:           &MID_PROFILE_CDN_ID,
		RoutingDisabled: &MID_PROFILE_ROUTING_DISABLED,
		Type:            &MID_PROFILE_TYPE,
		Parameters:      nil,
	},
}

func profiles(w http.ResponseWriter, r *http.Request) {
	common(w)
	if r.Method == http.MethodGet {
		api.WriteResp(w, r, PROFILES)
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"alerts":[{"level":"error","text":"This method hasn't yet been implemented."}]}`))
	}
}
