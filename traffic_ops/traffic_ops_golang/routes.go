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
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"runtime"
	"time"

	tclog "github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/about"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apiriak"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apitenant"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/asn"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/cachegroup"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/cdn"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/cdnfederation"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/coordinate"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/crconfig"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
	dsrequest "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice/request"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice/request/comment"
	dsserver "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice/servers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservicesregexes"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/division"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/federations"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/hwinfo"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/login"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/monitoring"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/parameter"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/physlocation"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/ping"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/profile"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/profileparameter"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/region"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/role"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/server"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/staticdnsentry"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/status"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/steering"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/steeringtargets"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/systeminfo"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/types"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/user"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/origin"
	"github.com/basho/riak-go-client"
	"github.com/jmoiron/sqlx"
)

// Authenticated ...
var Authenticated = true

// NoAuth ...
var NoAuth = false

func handlerToFunc(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}
}

// Routes returns the API routes, raw non-API root level routes, and a catchall route for when no route matches.
func Routes(d ServerData) ([]Route, []RawRoute, http.Handler, error) {
	proxyHandler := rootHandler(d)

	routes := []Route{
		// 1.1 and 1.2 routes are simply a Go replacement for the equivalent Perl route. They may or may not conform with the API guidelines (https://cwiki.apache.org/confluence/display/TC/API+Guidelines).
		// 1.3 routes exist only in a Go. There is NO equivalent Perl route. They should conform with the API guidelines (https://cwiki.apache.org/confluence/display/TC/API+Guidelines).

		//ASN: CRUD
		{1.2, http.MethodGet, `asns/?(\.json)?$`, api.ReadHandler(asn.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `asns/?(\.json)?$`, asn.V11ReadAll, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `asns/{id}$`, api.ReadHandler(asn.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `asns/{id}$`, api.UpdateHandler(asn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `asns/?$`, api.CreateHandler(asn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `asns/{id}$`, api.DeleteHandler(asn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//CacheGroup: CRUD
		{1.1, http.MethodGet, `cachegroups/trimmed/?(\.json)?$`, cachegroup.GetTrimmed, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `cachegroups/?(\.json)?$`, api.ReadHandler(cachegroup.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `cachegroups/{id}$`, api.ReadHandler(cachegroup.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `cachegroups/{id}$`, api.UpdateHandler(cachegroup.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `cachegroups/?$`, api.CreateHandler(cachegroup.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `cachegroups/{id}$`, api.DeleteHandler(cachegroup.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		{1.1, http.MethodPost, `cachegroups/{id}/queue_update$`, cachegroup.QueueUpdates, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `cachegroups/{id}/deliveryservices/?$`, cachegroup.DSPostHandler, auth.PrivLevelOperations, Authenticated, nil},

		//CDN
		{1.1, http.MethodGet, `cdns/name/{name}/sslkeys/?(\.json)?$`, cdn.GetSSLKeys, auth.PrivLevelAdmin, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/metric_types`, notImplementedHandler, 0, NoAuth, nil}, // MUST NOT end in $, because the 1.x route is longer
		{1.1, http.MethodGet, `cdns/capacity$`, handlerToFunc(proxyHandler), 0, NoAuth, []Middleware{}},
		{1.1, http.MethodGet, `cdns/configs/?(\.json)?$`, cdn.GetConfigs, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/domains/?(\.json)?$`, cdn.DomainsHandler, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/health$`, handlerToFunc(proxyHandler), 0, NoAuth, []Middleware{}},
		{1.1, http.MethodGet, `cdns/routing$`, handlerToFunc(proxyHandler), 0, NoAuth, []Middleware{}},

		//CDN: CRUD
		{1.1, http.MethodGet, `cdns/?(\.json)?$`, api.ReadHandler(cdn.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/{id}$`, api.ReadHandler(cdn.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/name/{name}/?(\.json)?$`, api.ReadHandler(cdn.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `cdns/{id}$`, api.UpdateHandler(cdn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `cdns/?$`, api.CreateHandler(cdn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `cdns/{id}$`, api.DeleteHandler(cdn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `cdns/name/{name}$`, cdn.DeleteName, auth.PrivLevelOperations, Authenticated, nil},

		//CDN: queue updates
		{1.1, http.MethodPost, `cdns/{id}/queue_update$`, cdn.Queue, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `cdns/dnsseckeys/generate(\.json)?$`, cdn.CreateDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/name/{name}/dnsseckeys/delete/?(\.json)?$`, cdn.DeleteDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil},

		{1.4, http.MethodGet, `cdns/dnsseckeys/refresh/?(\.json)?$`, cdn.RefreshDNSSECKeys, auth.PrivLevelOperations, Authenticated, nil},

		//CDN: Monitoring: Traffic Monitor
		{1.1, http.MethodGet, `cdns/{cdn}/configs/monitoring(\.json)?$`, monitoring.Get, auth.PrivLevelReadOnly, Authenticated, nil},

		//Division: CRUD
		{1.1, http.MethodGet, `divisions/?(\.json)?$`, api.ReadHandler(division.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `divisions/{id}$`, api.ReadHandler(division.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `divisions/{id}$`, api.UpdateHandler(division.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `divisions/?$`, api.CreateHandler(division.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `divisions/{id}$`, api.DeleteHandler(division.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodGet, `divisions/name/{name}/?(\.json)?$`, api.ReadHandler(division.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},

		//HWInfo
		{1.1, http.MethodGet, `hwinfo-wip/?(\.json)?$`, hwinfo.Get, auth.PrivLevelReadOnly, Authenticated, nil},

		//Login
		{1.1, http.MethodGet, `users/{id}/deliveryservices/?(\.json)?$`, user.GetDSes, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `user/{id}/deliveryservices/available/?(\.json)?$`, user.GetAvailableDSes, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPost, `user/login/?$`, login.LoginHandler(d.DB, d.Config), 0, NoAuth, nil},

		//User: CRUD (1.4 then 1.1)
		//Incrementing version for users because change to Nullable struct. Delete is completely new.
		{1.4, http.MethodGet, `users/?(\.json)?$`, api.ReadHandler(user.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.4, http.MethodGet, `users/{id}?(\.json)?$`, api.ReadHandler(user.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.4, http.MethodPut, `users/{id}?(\.json)?$`, api.UpdateHandler(user.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.4, http.MethodPost, `users/?(\.json)?$`, api.CreateHandler(user.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		//{1.4, http.MethodDelete, `users/{id}?(\.json)?$`, api.DeleteHandler(user.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		{1.1, http.MethodGet, `users/?(\.json)?$`, handlerToFunc(proxyHandler), auth.PrivLevelReadOnly, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `users/{id}?(\.json)?$`, handlerToFunc(proxyHandler), auth.PrivLevelReadOnly, Authenticated, []Middleware{}},
		{1.1, http.MethodPut, `users/{id}?(\.json)?$`, handlerToFunc(proxyHandler), auth.PrivLevelReadOnly, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `users/?(\.json)?$`, handlerToFunc(proxyHandler), auth.PrivLevelReadOnly, Authenticated, []Middleware{}},

		{1.1, http.MethodGet, `user/current/?(\.json)?$`, user.Current, auth.PrivLevelReadOnly, Authenticated, nil},

		//Parameter: CRUD
		{1.1, http.MethodGet, `parameters/?(\.json)?$`, api.ReadHandler(parameter.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `parameters/{id}$`, api.ReadHandler(parameter.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `parameters/{id}$`, api.UpdateHandler(parameter.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `parameters/?$`, api.CreateHandler(parameter.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `parameters/{id}$`, api.DeleteHandler(parameter.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//Phys_Location: CRUD
		{1.1, http.MethodGet, `phys_locations/?(\.json)?$`, api.ReadHandler(physlocation.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `phys_locations/trimmed/?(\.json)?$`, physlocation.GetTrimmed, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `phys_locations/{id}$`, api.ReadHandler(physlocation.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `phys_locations/{id}$`, api.UpdateHandler(physlocation.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `phys_locations/?$`, api.CreateHandler(physlocation.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `phys_locations/{id}$`, api.DeleteHandler(physlocation.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//Ping
		{1.1, http.MethodGet, `ping$`, ping.PingHandler(), 0, NoAuth, nil},
		{1.1, http.MethodGet, `riak/ping/?(\.json)?$`, ping.Riak, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `keys/ping/?(\.json)?$`, ping.Keys, auth.PrivLevelReadOnly, Authenticated, nil},

		//Profile: CRUD
		{1.1, http.MethodGet, `profiles/?(\.json)?$`, api.ReadHandler(profile.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `profiles/trimmed/?(\.json)?$`, profile.Trimmed, auth.PrivLevelReadOnly, Authenticated, nil},

		{1.1, http.MethodGet, `profiles/{id}$`, api.ReadHandler(profile.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `profiles/{id}$`, api.UpdateHandler(profile.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `profiles/?$`, api.CreateHandler(profile.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `profiles/{id}$`, api.DeleteHandler(profile.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//Region: CRUDs
		{1.1, http.MethodGet, `regions/?(\.json)?$`, api.ReadHandler(region.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `regions/{id}$`, api.ReadHandler(region.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `regions/name/{name}/?(\.json)?$`, region.GetName, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `regions/{id}$`, api.UpdateHandler(region.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `regions/?$`, api.CreateHandler(region.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `regions/{id}$`, api.DeleteHandler(region.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		{1.1, http.MethodDelete, `deliveryservice_server/{dsid}/{serverid}`, dsserver.Delete, auth.PrivLevelReadOnly, Authenticated, nil},

		// get all edge servers associated with a delivery service (from deliveryservice_server table)

		{1.1, http.MethodGet, `deliveryserviceserver$`, dsserver.ReadDSSHandler, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryserviceserver$`, dsserver.GetReplaceHandler, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/{xml_id}/servers$`, dsserver.GetCreateHandler, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodGet, `servers/{id}/deliveryservices$`, api.ReadOnlyHandler(dsserver.TypeSingleton), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{id}/servers$`, dsserver.GetReadAssigned, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{id}/unassigned_servers$`, dsserver.GetReadUnassigned, auth.PrivLevelReadOnly, Authenticated, nil},
		//{1.1, http.MethodGet, `deliveryservices/{id}/servers/eligible$`, dsserver.GetReadHandler(d.Tx, tc.Eligible),auth.PrivLevelReadOnly, Authenticated, nil},

		{1.1, http.MethodGet, `deliveryservice_matches/?(\.json)?$`, deliveryservice.GetMatches, auth.PrivLevelReadOnly, Authenticated, nil},

		//Server
		{1.1, http.MethodGet, `servers/checks$`, handlerToFunc(proxyHandler), 0, NoAuth, []Middleware{}},
		{1.1, http.MethodGet, `servers/status$`, handlerToFunc(proxyHandler), 0, NoAuth, []Middleware{}},
		{1.1, http.MethodGet, `servers/totals$`, handlerToFunc(proxyHandler), 0, NoAuth, []Middleware{}},

		//Server Details
		{1.2, http.MethodGet, `servers/details/?(\.json)?$`, server.GetDetailParamHandler, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.2, http.MethodGet, `servers/hostname/{hostName}/details/?(\.json)?$`, server.GetDetailHandler, auth.PrivLevelReadOnly, Authenticated, nil},

		//Server: CRUD
		{1.1, http.MethodGet, `servers/?(\.json)?$`, api.ReadHandler(server.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `servers/{id}$`, api.ReadHandler(server.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `servers/{id}$`, api.UpdateHandler(server.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `servers/?$`, api.CreateHandler(server.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `servers/{id}$`, api.DeleteHandler(server.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//Status: CRUD
		{1.1, http.MethodGet, `statuses/?(\.json)?$`, api.ReadHandler(status.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `statuses/{id}$`, api.ReadHandler(status.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `statuses/{id}$`, api.UpdateHandler(status.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `statuses/?$`, api.CreateHandler(status.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `statuses/{id}$`, api.DeleteHandler(status.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//System
		{1.1, http.MethodGet, `system/info/?(\.json)?$`, systeminfo.Get, auth.PrivLevelReadOnly, Authenticated, nil},

		//Type: CRUD
		{1.1, http.MethodGet, `types/?(\.json)?$`, api.ReadHandler(types.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `types/{id}$`, api.ReadHandler(types.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `types/{id}$`, api.UpdateHandler(types.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `types/?$`, api.CreateHandler(types.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `types/{id}$`, api.DeleteHandler(types.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//About
		{1.3, http.MethodGet, `about/?(\.json)?$`, about.Handler(), auth.PrivLevelReadOnly, Authenticated, nil},

		//Coordinates
		{1.3, http.MethodGet, `coordinates/?(\.json)?$`, api.ReadHandler(coordinate.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodGet, `coordinates/?$`, api.ReadHandler(coordinate.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodPut, `coordinates/?$`, api.UpdateHandler(coordinate.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodPost, `coordinates/?$`, api.CreateHandler(coordinate.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodDelete, `coordinates/?$`, api.DeleteHandler(coordinate.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//Servers
		// explicitly passed to legacy system until fully implemented.  Auth handled by legacy system.
		{1.2, http.MethodGet, `servers/checks$`, handlerToFunc(proxyHandler), 0, NoAuth, []Middleware{}},
		{1.2, http.MethodGet, `servers/details$`, handlerToFunc(proxyHandler), 0, NoAuth, []Middleware{}},
		{1.2, http.MethodGet, `servers/status$`, handlerToFunc(proxyHandler), 0, NoAuth, []Middleware{}},
		{1.2, http.MethodGet, `servers/totals$`, handlerToFunc(proxyHandler), 0, NoAuth, []Middleware{}},

		//Monitoring
		{1.2, http.MethodGet, `cdns/{name}/configs/monitoring(\.json)?$`, monitoring.Get, auth.PrivLevelReadOnly, Authenticated, nil},

		//ASNs
		{1.3, http.MethodGet, `asns/?(\.json)?$`, api.ReadHandler(asn.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodPut, `asns/?$`, api.UpdateHandler(asn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodPost, `asns/?$`, api.CreateHandler(asn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodDelete, `asns/?$`, api.DeleteHandler(asn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//CDN generic handlers:
		{1.3, http.MethodGet, `cdns/?(\.json)?$`, api.ReadHandler(cdn.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodGet, `cdns/{id}$`, api.ReadHandler(cdn.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodPut, `cdns/{id}$`, api.UpdateHandler(cdn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodPost, `cdns/?$`, api.CreateHandler(cdn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodDelete, `cdns/{id}$`, api.DeleteHandler(cdn.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//Delivery service requests
		{1.3, http.MethodGet, `deliveryservice_requests/?(\.json)?$`, api.ReadHandler(dsrequest.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodGet, `deliveryservice_requests/?$`, api.ReadHandler(dsrequest.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodPut, `deliveryservice_requests/?$`, api.UpdateHandler(dsrequest.GetTypeSingleton()), auth.PrivLevelPortal, Authenticated, nil},
		{1.3, http.MethodPost, `deliveryservice_requests/?$`, api.CreateHandler(dsrequest.GetTypeSingleton()), auth.PrivLevelPortal, Authenticated, nil},
		{1.3, http.MethodDelete, `deliveryservice_requests/?$`, api.DeleteHandler(dsrequest.GetTypeSingleton()), auth.PrivLevelPortal, Authenticated, nil},

		//Delivery service request: Actions
		{1.3, http.MethodPut, `deliveryservice_requests/{id}/assign$`, api.UpdateHandler(dsrequest.GetAssignmentTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodPut, `deliveryservice_requests/{id}/status$`, api.UpdateHandler(dsrequest.GetStatusTypeSingleton()), auth.PrivLevelPortal, Authenticated, nil},

		//Delivery service request comment: CRUD
		{1.3, http.MethodGet, `deliveryservice_request_comments/?(\.json)?$`, api.ReadHandler(comment.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodPut, `deliveryservice_request_comments/?$`, api.UpdateHandler(comment.GetTypeSingleton()), auth.PrivLevelPortal, Authenticated, nil},
		{1.3, http.MethodPost, `deliveryservice_request_comments/?$`, api.CreateHandler(comment.GetTypeSingleton()), auth.PrivLevelPortal, Authenticated, nil},
		{1.3, http.MethodDelete, `deliveryservice_request_comments/?$`, api.DeleteHandler(comment.GetTypeSingleton()), auth.PrivLevelPortal, Authenticated, nil},

		//Delivery service uri signing keys: CRUD
		{1.3, http.MethodGet, `deliveryservices/{xmlID}/urisignkeys$`, getURIsignkeysHandler, auth.PrivLevelAdmin, Authenticated, nil},
		{1.3, http.MethodPost, `deliveryservices/{xmlID}/urisignkeys$`, saveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil},
		{1.3, http.MethodPut, `deliveryservices/{xmlID}/urisignkeys$`, saveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil},
		{1.3, http.MethodDelete, `deliveryservices/{xmlID}/urisignkeys$`, removeDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil},

		// Federations by CDN (the actual table for federation)
		{1.1, http.MethodGet, `cdns/{name}/federations/?(\.json)?$`, api.ReadHandler(cdnfederation.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/{name}/federations/{id}$`, api.ReadHandler(cdnfederation.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPost, `cdns/{name}/federations/?(\.json)?$`, api.CreateHandler(cdnfederation.GetTypeSingleton()), auth.PrivLevelAdmin, Authenticated, nil},
		{1.1, http.MethodPut, `cdns/{name}/federations/{id}$`, api.UpdateHandler(cdnfederation.GetTypeSingleton()), auth.PrivLevelAdmin, Authenticated, nil},
		{1.1, http.MethodDelete, `cdns/{name}/federations/{id}$`, api.DeleteHandler(cdnfederation.GetTypeSingleton()), auth.PrivLevelAdmin, Authenticated, nil},

		{1.4, http.MethodPost, `cdns/{name}/dnsseckeys/ksk/generate$`, cdn.GenerateKSK, auth.PrivLevelAdmin, Authenticated, nil},

		//Origins
		{1.3, http.MethodGet, `origins/?(\.json)?$`, api.ReadHandler(origin.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodGet, `origins/?$`, api.ReadHandler(origin.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodPut, `origins/?$`, api.UpdateHandler(origin.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodPost, `origins/?$`, api.CreateHandler(origin.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodDelete, `origins/?$`, api.DeleteHandler(origin.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//Roles
		{1.3, http.MethodGet, `roles/?(\.json)?$`, api.ReadHandler(role.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodPut, `roles/?$`, api.UpdateHandler(role.GetTypeSingleton()), auth.PrivLevelAdmin, Authenticated, nil},
		{1.3, http.MethodPost, `roles/?$`, api.CreateHandler(role.GetTypeSingleton()), auth.PrivLevelAdmin, Authenticated, nil},
		{1.3, http.MethodDelete, `roles/?$`, api.DeleteHandler(role.GetTypeSingleton()), auth.PrivLevelAdmin, Authenticated, nil},

		//Delivery Services Regexes
		{1.1, http.MethodGet, `deliveryservices_regexes/?(\.json)?$`, deliveryservicesregexes.Get, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{dsid}/regexes/?(\.json)?$`, deliveryservicesregexes.DSGet, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{dsid}/regexes/{regexid}?(\.json)?$`, deliveryservicesregexes.DSGetID, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/{dsid}/regexes/?(\.json)?$`, deliveryservicesregexes.Post, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPut, `deliveryservices/{dsid}/regexes/{regexid}?(\.json)?$`, deliveryservicesregexes.Put, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `deliveryservices/{dsid}/regexes/{regexid}?(\.json)?$`, deliveryservicesregexes.Delete, auth.PrivLevelOperations, Authenticated, nil},

		//Servers
		{1.3, http.MethodPost, `servers/{id}/deliveryservices$`, server.AssignDeliveryServicesToServerHandler, auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodGet, `servers/{host_name}/update_status$`, server.GetServerUpdateStatusHandler, auth.PrivLevelReadOnly, Authenticated, nil},

		//StaticDNSEntries
		{1.1, http.MethodGet, `staticdnsentries/?(\.json)?$`, api.ReadHandler(staticdnsentry.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodGet, `staticdnsentries/?$`, api.ReadHandler(staticdnsentry.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodPut, `staticdnsentries/?$`, api.UpdateHandler(staticdnsentry.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodPost, `staticdnsentries/?$`, api.CreateHandler(staticdnsentry.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodDelete, `staticdnsentries/?$`, api.DeleteHandler(staticdnsentry.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//ProfileParameters
		{1.1, http.MethodGet, `profiles/{id}/parameters/?(\.json)?$`, profileparameter.GetProfileID, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `profiles/{id}/unassigned_parameters/?(\.json)?$`, profileparameter.GetUnassigned, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `profiles/name/{name}/parameters/?(\.json)?$`, profileparameter.GetProfileName, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `parameters/profile/{name}/?(\.json)?$`, profileparameter.GetProfileName, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPost, `profiles/name/{name}/parameters/?$`, profileparameter.PostProfileParamsByName, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `profiles/{id}/parameters/?$`, profileparameter.PostProfileParamsByID, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodGet, `profileparameters/?(\.json)?$`, api.ReadHandler(profileparameter.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPost, `profileparameters/?$`, api.CreateHandler(profileparameter.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `profileparameter/?$`, profileparameter.PostProfileParam, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `parameterprofile/?$`, profileparameter.PostParamProfile, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `profileparameters/{profileId}/{parameterId}$`, api.DeleteHandler(profileparameter.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//Tenants
		{1.1, http.MethodGet, `tenants/?(\.json)?$`, api.ReadHandler(apitenant.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `tenants/{id}$`, api.ReadHandler(apitenant.GetTypeSingleton()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `tenants/{id}$`, api.UpdateHandler(apitenant.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `tenants/?$`, api.CreateHandler(apitenant.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `tenants/{id}$`, api.DeleteHandler(apitenant.GetTypeSingleton()), auth.PrivLevelOperations, Authenticated, nil},

		//CRConfig
		{1.1, http.MethodGet, `cdns/{cdn}/snapshot/?$`, crconfig.SnapshotGetHandler, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/{cdn}/snapshot/new/?$`, crconfig.Handler, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPut, `cdns/{id}/snapshot/?$`, crconfig.SnapshotHandler, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPut, `snapshot/{cdn}/?$`, crconfig.SnapshotHandler, auth.PrivLevelOperations, Authenticated, nil},

		// Federations
		{1.4, http.MethodGet, `federations/all/?(\.json)?$`, federations.GetAll, auth.PrivLevelAdmin, Authenticated, nil},
		{1.1, http.MethodGet, `federations/?(\.json)?$`, federations.Get, auth.PrivLevelFederation, Authenticated, nil},
		{1.1, http.MethodPost, `federations/{id}/deliveryservices?(\.json)?$`, federations.PostDSes, auth.PrivLevelAdmin, Authenticated, nil},

		////DeliveryServices
		{1.3, http.MethodGet, `deliveryservices/?(\.json)?$`, api.ReadHandler(deliveryservice.GetTypeV13Factory()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/?(\.json)?$`, api.ReadHandler(deliveryservice.GetTypeV12Factory()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodGet, `deliveryservices/{id}/?(\.json)?$`, api.ReadHandler(deliveryservice.GetTypeV13Factory()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{id}/?(\.json)?$`, api.ReadHandler(deliveryservice.GetTypeV12Factory()), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.3, http.MethodPost, `deliveryservices/?(\.json)?$`, deliveryservice.CreateV13, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/?(\.json)?$`, deliveryservice.CreateV12, auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodPut, `deliveryservices/{id}/?(\.json)?$`, deliveryservice.UpdateV13, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPut, `deliveryservices/{id}/?(\.json)?$`, deliveryservice.UpdateV12, auth.PrivLevelOperations, Authenticated, nil},
		{1.3, http.MethodDelete, `deliveryservices/{id}/?(\.json)?$`, api.DeleteHandler(deliveryservice.GetTypeV13Factory()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `deliveryservices/{id}/?(\.json)?$`, api.DeleteHandler(deliveryservice.GetTypeV12Factory()), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{id}/servers/eligible/?(\.json)?$`, deliveryservice.GetServersEligible, auth.PrivLevelReadOnly, Authenticated, nil},

		{1.1, http.MethodGet, `deliveryservices/xmlId/{xmlid}/sslkeys$`, deliveryservice.GetSSLKeysByXMLID, auth.PrivLevelAdmin, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/hostname/{hostname}/sslkeys$`, deliveryservice.GetSSLKeysByHostName, auth.PrivLevelAdmin, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/sslkeys/add$`, deliveryservice.AddSSLKeys, auth.PrivLevelAdmin, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/xmlId/{xmlid}/sslkeys/delete$`, deliveryservice.DeleteSSLKeys, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/sslkeys/generate/?(\.json)?$`, deliveryservice.GenerateSSLKeys, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/copyFromXmlId/{copy-name}/?(\.json)?$`, deliveryservice.CopyURLKeys, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/generate/?(\.json)?$`, deliveryservice.GenerateURLKeys, auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/xmlId/{name}/urlkeys/?(\.json)?$`, deliveryservice.GetURLKeysByName, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{id}/urlkeys/?(\.json)?$`, deliveryservice.GetURLKeysByID, auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `riak/bucket/{bucket}/key/{key}/values/?(\.json)?$`, apiriak.GetBucketKey, auth.PrivLevelAdmin, Authenticated, nil},

		{1.1, http.MethodGet, `steering/{deliveryservice}/targets/?(\.json)?$`, api.ReadHandler(steeringtargets.TypeFactory), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodGet, `steering/{deliveryservice}/targets/{target}$`, api.ReadHandler(steeringtargets.TypeFactory), auth.PrivLevelReadOnly, Authenticated, nil},
		{1.1, http.MethodPost, `steering/{deliveryservice}/targets/?(\.json)?$`, api.CreateHandler(steeringtargets.TypeFactory), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodPut, `steering/{deliveryservice}/targets/{target}/?(\.json)?$`, api.UpdateHandler(steeringtargets.TypeFactory), auth.PrivLevelOperations, Authenticated, nil},
		{1.1, http.MethodDelete, `steering/{deliveryservice}/targets/{target}/?(\.json)?$`, api.DeleteHandler(steeringtargets.TypeFactory), auth.PrivLevelOperations, Authenticated, nil},

		{1.4, http.MethodGet, `steering/?(\.json)?$`, steering.Get, auth.PrivLevelSteering, Authenticated, nil},
	}

	// rawRoutes are served at the root path. These should be almost exclusively old Perl pre-API routes, which have yet to be converted in all clients. New routes should be in the versioned API path.
	rawRoutes := []RawRoute{
		// DEPRECATED - use PUT /api/1.2/snapshot/{cdn}
		{http.MethodGet, `tools/write_crconfig/{cdn}/?$`, crconfig.SnapshotOldGUIHandler, auth.PrivLevelOperations, Authenticated, nil},
		// DEPRECATED - use GET /api/1.2/cdns/{cdn}/snapshot
		{http.MethodGet, `CRConfig-Snapshots/{cdn}/CRConfig.json?$`, crconfig.SnapshotOldGetHandler, auth.PrivLevelReadOnly, Authenticated, nil},

		// The '/api' endpoint, that tells clients what routes are available under /api/1.x
		{http.MethodGet, `api/?$`, api.AvailableRoutesHandler, 0, false, nil},
		{http.MethodHead, `api/?$`, api.AvailableRoutesBadMethodHandler, 0, false, nil},
		{http.MethodPost, `api/?$`, api.AvailableRoutesBadMethodHandler, 0, false, nil},
		{http.MethodPut, `api/?$`, api.AvailableRoutesBadMethodHandler, 0, false, nil},
		{http.MethodPatch, `api/?$`, api.AvailableRoutesBadMethodHandler, 0, false, nil},
		{http.MethodDelete, `api/?$`, api.AvailableRoutesBadMethodHandler, 0, false, nil},
		{http.MethodConnect, `api/?$`, api.AvailableRoutesBadMethodHandler, 0, false, nil},
		{http.MethodTrace, `api/?$`, api.AvailableRoutesBadMethodHandler, 0, false, nil},
		{http.MethodOptions, `api/?$`, api.AvailableVersionsHandler, 0, false, nil},
	}

	return routes, rawRoutes, proxyHandler, nil
}

func memoryStatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleErrs := tc.GetHandleErrorsFunc(w, r)
		stats := runtime.MemStats{}
		runtime.ReadMemStats(&stats)

		bytes, err := json.Marshal(stats)
		if err != nil {
			tclog.Errorln("unable to marshal stats: " + err.Error())
			handleErrs(http.StatusInternalServerError, errors.New("marshalling error"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

func dbStatsHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleErrs := tc.GetHandleErrorsFunc(w, r)
		stats := db.DB.Stats()

		bytes, err := json.Marshal(stats)
		if err != nil {
			tclog.Errorln("unable to marshal stats: " + err.Error())
			handleErrs(http.StatusInternalServerError, errors.New("marshalling error"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

// RootHandler returns the / handler for the service, which reverse-proxies the old Perl Traffic Ops
func rootHandler(d ServerData) http.Handler {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(d.Config.ProxyTimeout) * time.Second,
			KeepAlive: time.Duration(d.Config.ProxyKeepAlive) * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   time.Duration(d.Config.ProxyTLSTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(d.Config.ProxyReadHeaderTimeout) * time.Second,
		//IdleConnTimeout: time.Duration(d.Config.ProxyIdleConnTimeout) * time.Second,
		//Other knobs we can turn: ExpectContinueTimeout,IdleConnTimeout
	}
	rp := httputil.NewSingleHostReverseProxy(d.URL)
	rp.Transport = tr

	var errorLogger interface{}
	errorLogger, err := tclog.GetLogWriter(d.Config.ErrorLog())
	if err != nil {
		tclog.Errorln("could not create error log writer for proxy: ", err)
	}
	if errorLogger != nil {
		rp.ErrorLog = log.New(errorLogger.(io.Writer), "proxy error: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC) //if we don't provide a logger to the reverse proxy it logs to stdout/err and is lost when ran by a script.
		riak.SetErrorLogger(log.New(errorLogger.(io.Writer), "riak error: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC))
	}
	var infoLogger interface{}
	infoLogger, err = tclog.GetLogWriter(d.Config.InfoLog())
	if err != nil {
		tclog.Errorln("could not create info log writer for proxy: ", err)
	}
	if infoLogger != nil {
		riak.SetLogger(log.New(infoLogger.(io.Writer), "riak info: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC))
	}
	tclog.Debugf("our reverseProxy: %++v\n", rp)
	tclog.Debugf("our reverseProxy's transport: %++v\n", tr)
	loggingProxyHandler := wrapAccessLog(d.Secrets[0], rp)

	managerHandler := CreateThrottledHandler(loggingProxyHandler, d.BackendMaxConnections["mojolicious"])
	return managerHandler
}

// notImplementedHandler returns a 501 Not Implemented to the client. This should be used very rarely, and primarily for old API Perl routes which were broken long ago, which we don't have the resources to rewrite in Go for the time being.
func notImplementedHandler(w http.ResponseWriter, r *http.Request) {
	code := http.StatusNotImplemented
	w.WriteHeader(code)
	w.Write([]byte(http.StatusText(code)))
}

//CreateThrottledHandler takes a handler, and a max and uses a channel to insure the handler is used concurrently by only max number of routines
func CreateThrottledHandler(handler http.Handler, maxConcurrentCalls int) ThrottledHandler {
	return ThrottledHandler{handler, make(chan struct{}, maxConcurrentCalls)}
}

// ThrottledHandler ...
type ThrottledHandler struct {
	Handler http.Handler
	ReqChan chan struct{}
}

func (m ThrottledHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.ReqChan <- struct{}{}
	defer func() { <-m.ReqChan }()
	m.Handler.ServeHTTP(w, r)
}
