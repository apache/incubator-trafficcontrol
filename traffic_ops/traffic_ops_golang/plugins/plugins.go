package plugins

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
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-util"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/plugin"
)

// Get handler for getting enabled TO Plugins.
func Get(p plugin.Plugins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inf, sysErr, userErr, errCode := api.NewInfo(w, r, nil, nil)
		if sysErr != nil || userErr != nil {
			api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
		// Add plugins
		plugins := []tc.Plugin{}
		for _, pi := range p.GetInfo() {
			plugins = append(plugins, tc.Plugin{
				Name:        util.StrPtr(pi.Name),
				Version:     util.StrPtr(pi.Version),
				Description: util.StrPtr(pi.Description),
			})
		}
		api.WriteResp(w, r, plugins)
	}
}
