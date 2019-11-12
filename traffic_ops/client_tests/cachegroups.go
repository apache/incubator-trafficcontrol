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

// Static Cache Group fields
var EDGE_CACHEGROUP_ID = 1
var EDGE_CACHEGROUP = "Edge"
var EDGE_CACHEGROUP_SHORT_NAME = "Edge"
var EDGE_CACHEGROUP_LATITUDE = 0.0
var EDGE_CACHEGROUP_LONGITUDE = 0.0
var EDGE_CACHEGROUP_PARENT_NAME = "Mid" // NOTE: This places a hard requirement on the `cachegroups` implementation - must have a `MID_LOC` Cache Group named "Mid"
var EDGE_CACHEGROUP_PARENT_ID = 2       // NOTE: This places a hard requirement on the `cachegroups` implementation - must have a `MID_LOC` Cache Group identified by `2`
var EDGE_CACHEGROUP_FALLBACK_TO_CLOSEST = true
var EDGE_CACHEGROUP_LOCALIZATION_METHODS = []tc.LocalizationMethod{
	tc.LocalizationMethodCZ,
	tc.LocalizationMethodDeepCZ,
	tc.LocalizationMethodGeo,
}

var MID_CACHEGROUP_ID = 2
var MID_CACHEGROUP = "Mid"
var MID_CACHEGROUP_SHORT_NAME = "Mid"
var MID_CACHEGROUP_LATITUDE = 0.0
var MID_CACHEGROUP_LONGITUDE = 0.0
var MID_CACHEGROUP_FALLBACK_TO_CLOSEST = true
var MID_CACHEGROUP_LOCALIZATION_METHODS = []tc.LocalizationMethod{
	tc.LocalizationMethodCZ,
	tc.LocalizationMethodDeepCZ,
	tc.LocalizationMethodGeo,
}

var CACHEGROUPS = []tc.CacheGroupNullable{
	tc.CacheGroupNullable{
		ID:                          &EDGE_CACHEGROUP_ID,
		Name:                        &EDGE_CACHEGROUP,
		ShortName:                   &EDGE_CACHEGROUP_SHORT_NAME,
		Latitude:                    &EDGE_CACHEGROUP_LATITUDE,
		Longitude:                   &EDGE_CACHEGROUP_LONGITUDE,
		ParentName:                  &EDGE_CACHEGROUP_PARENT_NAME,
		ParentCachegroupID:          &EDGE_CACHEGROUP_PARENT_ID,
		SecondaryParentName:         nil,
		SecondaryParentCachegroupID: nil,
		FallbackToClosest:           &EDGE_CACHEGROUP_FALLBACK_TO_CLOSEST,
		LocalizationMethods:         &EDGE_CACHEGROUP_LOCALIZATION_METHODS,
		Type:                        &(TYPE_EDGE_LOC.Name),
		TypeID:                      &(TYPE_EDGE_LOC.ID),
		LastUpdated:                 CURRENT_TIME,
		Fallbacks:                   nil,
	},
	tc.CacheGroupNullable{
		ID:                          &MID_CACHEGROUP_ID,
		Name:                        &MID_CACHEGROUP,
		ShortName:                   &MID_CACHEGROUP_SHORT_NAME,
		Latitude:                    &MID_CACHEGROUP_LATITUDE,
		Longitude:                   &MID_CACHEGROUP_LONGITUDE,
		ParentName:                  nil,
		ParentCachegroupID:          nil,
		SecondaryParentName:         nil,
		SecondaryParentCachegroupID: nil,
		FallbackToClosest:           &MID_CACHEGROUP_FALLBACK_TO_CLOSEST,
		LocalizationMethods:         &MID_CACHEGROUP_LOCALIZATION_METHODS,
		Type:                        &(TYPE_MID_LOC.Name),
		TypeID:                      &(TYPE_MID_LOC.ID),
		LastUpdated:                 CURRENT_TIME,
		Fallbacks:                   nil,
	},
}

func cacheGroups(w http.ResponseWriter, r *http.Request) {
	common(w)
	if r.Method == http.MethodGet {
		api.WriteResp(w, r, CACHEGROUPS)
	} else {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"alerts":[{"level":"error","text":"This method hasn't yet been implemented."}]}`))
	}
}
