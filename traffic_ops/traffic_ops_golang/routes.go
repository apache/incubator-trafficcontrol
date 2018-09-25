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
	"database/sql"
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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/steeringtargets"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/systeminfo"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tocookie"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/types"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/user"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/origin"
	"github.com/basho/riak-go-client"
	"github.com/jmoiron/sqlx"
)

// these constants are convenience aliases, to make the Routes more readable.

const Authenticated = true
const NoAuth = false

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
		{1.2, http.MethodGet, `asns/?(\.json)?$`, api.ReadHandler(asn.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `asns/?(\.json)?$`, asn.V11ReadAll, Authenticated, nil},
		{1.1, http.MethodGet, `asns/{id}$`, api.ReadHandler(asn.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `asns/{id}$`, api.UpdateHandler(asn.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `asns/?$`, api.CreateHandler(asn.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `asns/{id}$`, api.DeleteHandler(asn.GetTypeSingleton()), Authenticated, nil},

		//CacheGroup: CRUD
		{1.1, http.MethodGet, `cachegroups/trimmed/?(\.json)?$`, cachegroup.GetTrimmed, Authenticated, nil},
		{1.1, http.MethodGet, `cachegroups/?(\.json)?$`, api.ReadHandler(cachegroup.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `cachegroups/{id}$`, api.ReadHandler(cachegroup.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `cachegroups/{id}$`, api.UpdateHandler(cachegroup.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `cachegroups/?$`, api.CreateHandler(cachegroup.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `cachegroups/{id}$`, api.DeleteHandler(cachegroup.GetTypeSingleton()), Authenticated, nil},

		{1.1, http.MethodPost, `cachegroups/{id}/queue_update$`, cachegroup.QueueUpdates, Authenticated, nil},
		{1.1, http.MethodPost, `cachegroups/{id}/deliveryservices/?$`, cachegroup.DSPostHandler, Authenticated, nil},

		//CDN
		{1.1, http.MethodGet, `cdns/name/{name}/sslkeys/?(\.json)?$`, cdn.GetSSLKeys, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/metric_types`, notImplementedHandler, NoAuth, nil}, // MUST NOT end in $, because the 1.x route is longer
		{1.1, http.MethodGet, `cdns/capacity$`, handlerToFunc(proxyHandler), NoAuth, []Middleware{}},
		{1.1, http.MethodGet, `cdns/configs/?(\.json)?$`, cdn.GetConfigs, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/domains/?(\.json)?$`, cdn.DomainsHandler, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/health$`, handlerToFunc(proxyHandler), NoAuth, []Middleware{}},
		{1.1, http.MethodGet, `cdns/routing$`, handlerToFunc(proxyHandler), NoAuth, []Middleware{}},

		//CDN: CRUD
		{1.1, http.MethodGet, `cdns/?(\.json)?$`, api.ReadHandler(cdn.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `cdns/{id}$`, api.ReadHandler(cdn.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `cdns/name/{name}/?(\.json)?$`, api.ReadHandler(cdn.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `cdns/{id}$`, api.UpdateHandler(cdn.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `cdns/?$`, api.CreateHandler(cdn.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `cdns/{id}$`, api.DeleteHandler(cdn.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `cdns/name/{name}$`, cdn.DeleteName, Authenticated, nil},

		//CDN: queue updates
		{1.1, http.MethodPost, `cdns/{id}/queue_update$`, cdn.Queue, Authenticated, nil},
		{1.1, http.MethodPost, `cdns/dnsseckeys/generate(\.json)?$`, cdn.CreateDNSSECKeys, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/name/{name}/dnsseckeys/delete/?(\.json)?$`, cdn.DeleteDNSSECKeys, Authenticated, nil},

		//CDN: Monitoring: Traffic Monitor
		{1.1, http.MethodGet, `cdns/{cdn}/configs/monitoring(\.json)?$`, monitoring.Get, Authenticated, nil},

		//Division: CRUD
		{1.1, http.MethodGet, `divisions/?(\.json)?$`, api.ReadHandler(division.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `divisions/{id}$`, api.ReadHandler(division.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `divisions/{id}$`, api.UpdateHandler(division.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `divisions/?$`, api.CreateHandler(division.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `divisions/{id}$`, api.DeleteHandler(division.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `divisions/name/{name}/?(\.json)?$`, api.ReadHandler(division.GetTypeSingleton()), Authenticated, nil},

		//HWInfo
		{1.1, http.MethodGet, `hwinfo-wip/?(\.json)?$`, hwinfo.Get, Authenticated, nil},

		//Login
		{1.1, http.MethodGet, `users/{id}/deliveryservices/?(\.json)?$`, user.GetDSes, Authenticated, nil},
		{1.1, http.MethodGet, `user/{id}/deliveryservices/available/?(\.json)?$`, user.GetAvailableDSes, Authenticated, nil},
		{1.1, http.MethodPost, `user/login/?$`, login.LoginHandler(d.DB, d.Config), NoAuth, nil},

		{1.1, http.MethodGet, `user/current/?(\.json)?$`, user.Current, Authenticated, nil},

		//Parameter: CRUD
		{1.1, http.MethodGet, `parameters/?(\.json)?$`, api.ReadHandler(parameter.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `parameters/{id}$`, api.ReadHandler(parameter.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `parameters/{id}$`, api.UpdateHandler(parameter.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `parameters/?$`, api.CreateHandler(parameter.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `parameters/{id}$`, api.DeleteHandler(parameter.GetTypeSingleton()), Authenticated, nil},

		//Phys_Location: CRUD
		{1.1, http.MethodGet, `phys_locations/?(\.json)?$`, api.ReadHandler(physlocation.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `phys_locations/trimmed/?(\.json)?$`, physlocation.GetTrimmed, Authenticated, nil},
		{1.1, http.MethodGet, `phys_locations/{id}$`, api.ReadHandler(physlocation.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `phys_locations/{id}$`, api.UpdateHandler(physlocation.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `phys_locations/?$`, api.CreateHandler(physlocation.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `phys_locations/{id}$`, api.DeleteHandler(physlocation.GetTypeSingleton()), Authenticated, nil},

		//Ping
		{1.1, http.MethodGet, `ping$`, ping.PingHandler(), NoAuth, nil},
		{1.1, http.MethodGet, `riak/ping/?(\.json)?$`, ping.Riak, Authenticated, nil},
		{1.1, http.MethodGet, `keys/ping/?(\.json)?$`, ping.Keys, Authenticated, nil},

		//Profile: CRUD
		{1.1, http.MethodGet, `profiles/?(\.json)?$`, api.ReadHandler(profile.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `profiles/trimmed/?(\.json)?$`, profile.Trimmed, Authenticated, nil},

		{1.1, http.MethodGet, `profiles/{id}$`, api.ReadHandler(profile.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `profiles/{id}$`, api.UpdateHandler(profile.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `profiles/?$`, api.CreateHandler(profile.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `profiles/{id}$`, api.DeleteHandler(profile.GetTypeSingleton()), Authenticated, nil},

		//Region: CRUDs
		{1.1, http.MethodGet, `regions/?(\.json)?$`, api.ReadHandler(region.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `regions/{id}$`, api.ReadHandler(region.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `regions/name/{name}/?(\.json)?$`, region.GetName, Authenticated, nil},
		{1.1, http.MethodPut, `regions/{id}$`, api.UpdateHandler(region.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `regions/?$`, api.CreateHandler(region.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `regions/{id}$`, api.DeleteHandler(region.GetTypeSingleton()), Authenticated, nil},

		{1.1, http.MethodDelete, `deliveryservice_server/{dsid}/{serverid}`, dsserver.Delete, Authenticated, nil},

		// get all edge servers associated with a delivery service (from deliveryservice_server table)

		{1.1, http.MethodGet, `deliveryserviceserver$`, dsserver.ReadDSSHandler, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryserviceserver$`, dsserver.GetReplaceHandler, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/{xml_id}/servers$`, dsserver.GetCreateHandler, Authenticated, nil},
		{1.1, http.MethodGet, `servers/{id}/deliveryservices$`, api.ReadOnlyHandler(dsserver.TypeSingleton), Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{id}/servers$`, dsserver.GetReadAssigned, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{id}/unassigned_servers$`, dsserver.GetReadUnassigned, Authenticated, nil},
		//{1.1, http.MethodGet, `deliveryservices/{id}/servers/eligible$`, dsserver.GetReadHandler(d.Tx, tc.Eligible),Authenticated, nil},

		{1.1, http.MethodGet, `deliveryservice_matches/?(\.json)?$`, deliveryservice.GetMatches, Authenticated, nil},

		//Server
		{1.1, http.MethodGet, `servers/checks$`, handlerToFunc(proxyHandler), NoAuth, []Middleware{}},
		{1.1, http.MethodGet, `servers/status$`, handlerToFunc(proxyHandler), NoAuth, []Middleware{}},
		{1.1, http.MethodGet, `servers/totals$`, handlerToFunc(proxyHandler), NoAuth, []Middleware{}},

		//Server Details
		{1.2, http.MethodGet, `servers/details/?(\.json)?$`, server.GetDetailParamHandler, Authenticated, nil},
		{1.2, http.MethodGet, `servers/hostname/{hostName}/details/?(\.json)?$`, server.GetDetailHandler, Authenticated, nil},

		//Server: CRUD
		{1.1, http.MethodGet, `servers/?(\.json)?$`, api.ReadHandler(server.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `servers/{id}$`, api.ReadHandler(server.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `servers/{id}$`, api.UpdateHandler(server.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `servers/?$`, api.CreateHandler(server.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `servers/{id}$`, api.DeleteHandler(server.GetTypeSingleton()), Authenticated, nil},

		//Status: CRUD
		{1.1, http.MethodGet, `statuses/?(\.json)?$`, api.ReadHandler(status.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `statuses/{id}$`, api.ReadHandler(status.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `statuses/{id}$`, api.UpdateHandler(status.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `statuses/?$`, api.CreateHandler(status.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `statuses/{id}$`, api.DeleteHandler(status.GetTypeSingleton()), Authenticated, nil},

		//System
		{1.1, http.MethodGet, `system/info/?(\.json)?$`, systeminfo.Get, Authenticated, nil},

		//Type: CRUD
		{1.1, http.MethodGet, `types/?(\.json)?$`, api.ReadHandler(types.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `types/{id}$`, api.ReadHandler(types.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `types/{id}$`, api.UpdateHandler(types.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `types/?$`, api.CreateHandler(types.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `types/{id}$`, api.DeleteHandler(types.GetTypeSingleton()), Authenticated, nil},

		//About
		{1.3, http.MethodGet, `about/?(\.json)?$`, about.Handler(), Authenticated, nil},

		//Coordinates
		{1.3, http.MethodGet, `coordinates/?(\.json)?$`, api.ReadHandler(coordinate.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodGet, `coordinates/?$`, api.ReadHandler(coordinate.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPut, `coordinates/?$`, api.UpdateHandler(coordinate.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPost, `coordinates/?$`, api.CreateHandler(coordinate.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodDelete, `coordinates/?$`, api.DeleteHandler(coordinate.GetTypeSingleton()), Authenticated, nil},

		//Servers
		// explicitly passed to legacy system until fully implemented.  Auth handled by legacy system.
		{1.2, http.MethodGet, `servers/checks$`, handlerToFunc(proxyHandler), NoAuth, []Middleware{}},
		{1.2, http.MethodGet, `servers/details$`, handlerToFunc(proxyHandler), NoAuth, []Middleware{}},
		{1.2, http.MethodGet, `servers/status$`, handlerToFunc(proxyHandler), NoAuth, []Middleware{}},
		{1.2, http.MethodGet, `servers/totals$`, handlerToFunc(proxyHandler), NoAuth, []Middleware{}},

		//Monitoring
		{1.2, http.MethodGet, `cdns/{name}/configs/monitoring(\.json)?$`, monitoring.Get, Authenticated, nil},

		//ASNs
		{1.3, http.MethodGet, `asns/?(\.json)?$`, api.ReadHandler(asn.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPut, `asns/?$`, api.UpdateHandler(asn.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPost, `asns/?$`, api.CreateHandler(asn.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodDelete, `asns/?$`, api.DeleteHandler(asn.GetTypeSingleton()), Authenticated, nil},

		//CDN generic handlers:
		{1.3, http.MethodGet, `cdns/?(\.json)?$`, api.ReadHandler(cdn.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodGet, `cdns/{id}$`, api.ReadHandler(cdn.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPut, `cdns/{id}$`, api.UpdateHandler(cdn.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPost, `cdns/?$`, api.CreateHandler(cdn.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodDelete, `cdns/{id}$`, api.DeleteHandler(cdn.GetTypeSingleton()), Authenticated, nil},

		//Delivery service requests
		{1.3, http.MethodGet, `deliveryservice_requests/?(\.json)?$`, api.ReadHandler(dsrequest.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodGet, `deliveryservice_requests/?$`, api.ReadHandler(dsrequest.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPut, `deliveryservice_requests/?$`, api.UpdateHandler(dsrequest.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPost, `deliveryservice_requests/?$`, api.CreateHandler(dsrequest.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodDelete, `deliveryservice_requests/?$`, api.DeleteHandler(dsrequest.GetTypeSingleton()), Authenticated, nil},

		//Delivery service request: Actions
		{1.3, http.MethodPut, `deliveryservice_requests/{id}/assign$`, api.UpdateHandler(dsrequest.GetAssignmentTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPut, `deliveryservice_requests/{id}/status$`, api.UpdateHandler(dsrequest.GetStatusTypeSingleton()), Authenticated, nil},

		//Delivery service request comment: CRUD
		{1.3, http.MethodGet, `deliveryservice_request_comments/?(\.json)?$`, api.ReadHandler(comment.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPut, `deliveryservice_request_comments/?$`, api.UpdateHandler(comment.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPost, `deliveryservice_request_comments/?$`, api.CreateHandler(comment.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodDelete, `deliveryservice_request_comments/?$`, api.DeleteHandler(comment.GetTypeSingleton()), Authenticated, nil},

		//Delivery service uri signing keys: CRUD
		{1.3, http.MethodGet, `deliveryservices/{xmlID}/urisignkeys$`, getURIsignkeysHandler, Authenticated, nil},
		{1.3, http.MethodPost, `deliveryservices/{xmlID}/urisignkeys$`, saveDeliveryServiceURIKeysHandler, Authenticated, nil},
		{1.3, http.MethodPut, `deliveryservices/{xmlID}/urisignkeys$`, saveDeliveryServiceURIKeysHandler, Authenticated, nil},
		{1.3, http.MethodDelete, `deliveryservices/{xmlID}/urisignkeys$`, removeDeliveryServiceURIKeysHandler, Authenticated, nil},

		// Federations by CDN (the actual table for federation)
		{1.1, http.MethodGet, `cdns/{name}/federations/?(\.json)?$`, api.ReadHandler(cdnfederation.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `cdns/{name}/federations/{id}$`, api.ReadHandler(cdnfederation.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `cdns/{name}/federations/?(\.json)?$`, api.CreateHandler(cdnfederation.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `cdns/{name}/federations/{id}$`, api.UpdateHandler(cdnfederation.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `cdns/{name}/federations/{id}$`, api.DeleteHandler(cdnfederation.GetTypeSingleton()), Authenticated, nil},

		//Origins
		{1.3, http.MethodGet, `origins/?(\.json)?$`, api.ReadHandler(origin.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodGet, `origins/?$`, api.ReadHandler(origin.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPut, `origins/?$`, api.UpdateHandler(origin.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPost, `origins/?$`, api.CreateHandler(origin.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodDelete, `origins/?$`, api.DeleteHandler(origin.GetTypeSingleton()), Authenticated, nil},

		//Roles
		{1.3, http.MethodGet, `roles/?(\.json)?$`, api.ReadHandler(role.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPut, `roles/?$`, api.UpdateHandler(role.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPost, `roles/?$`, api.CreateHandler(role.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodDelete, `roles/?$`, api.DeleteHandler(role.GetTypeSingleton()), Authenticated, nil},

		//Delivery Services Regexes
		{1.1, http.MethodGet, `deliveryservices_regexes/?(\.json)?$`, deliveryservicesregexes.Get, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{dsid}/regexes/?(\.json)?$`, deliveryservicesregexes.DSGet, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{dsid}/regexes/{regexid}?(\.json)?$`, deliveryservicesregexes.DSGetID, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/{dsid}/regexes/?(\.json)?$`, deliveryservicesregexes.Post, Authenticated, nil},
		{1.1, http.MethodPut, `deliveryservices/{dsid}/regexes/{regexid}?(\.json)?$`, deliveryservicesregexes.Put, Authenticated, nil},
		{1.1, http.MethodDelete, `deliveryservices/{dsid}/regexes/{regexid}?(\.json)?$`, deliveryservicesregexes.Delete, Authenticated, nil},

		//Servers
		{1.3, http.MethodPost, `servers/{id}/deliveryservices$`, server.AssignDeliveryServicesToServerHandler, Authenticated, nil},
		{1.3, http.MethodGet, `servers/{host_name}/update_status$`, server.GetServerUpdateStatusHandler, Authenticated, nil},

		//StaticDNSEntries
		{1.1, http.MethodGet, `staticdnsentries/?(\.json)?$`, api.ReadHandler(staticdnsentry.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodGet, `staticdnsentries/?$`, api.ReadHandler(staticdnsentry.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPut, `staticdnsentries/?$`, api.UpdateHandler(staticdnsentry.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodPost, `staticdnsentries/?$`, api.CreateHandler(staticdnsentry.GetTypeSingleton()), Authenticated, nil},
		{1.3, http.MethodDelete, `staticdnsentries/?$`, api.DeleteHandler(staticdnsentry.GetTypeSingleton()), Authenticated, nil},

		//ProfileParameters
		{1.1, http.MethodGet, `profiles/{id}/parameters/?(\.json)?$`, profileparameter.GetProfileID, Authenticated, nil},
		{1.1, http.MethodGet, `profiles/{id}/unassigned_parameters/?(\.json)?$`, profileparameter.GetUnassigned, Authenticated, nil},
		{1.1, http.MethodGet, `profiles/name/{name}/parameters/?(\.json)?$`, profileparameter.GetProfileName, Authenticated, nil},
		{1.1, http.MethodGet, `parameters/profile/{name}/?(\.json)?$`, profileparameter.GetProfileName, Authenticated, nil},
		{1.1, http.MethodPost, `profiles/name/{name}/parameters/?$`, profileparameter.PostProfileParamsByName, Authenticated, nil},
		{1.1, http.MethodPost, `profiles/{id}/parameters/?$`, profileparameter.PostProfileParamsByID, Authenticated, nil},
		{1.1, http.MethodGet, `profileparameters/?(\.json)?$`, api.ReadHandler(profileparameter.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `profileparameters/?$`, api.CreateHandler(profileparameter.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `profileparameter/?$`, profileparameter.PostProfileParam, Authenticated, nil},
		{1.1, http.MethodPost, `parameterprofile/?$`, profileparameter.PostParamProfile, Authenticated, nil},
		{1.1, http.MethodDelete, `profileparameters/{profileId}/{parameterId}$`, api.DeleteHandler(profileparameter.GetTypeSingleton()), Authenticated, nil},

		//Tenants
		{1.1, http.MethodGet, `tenants/?(\.json)?$`, api.ReadHandler(apitenant.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodGet, `tenants/{id}$`, api.ReadHandler(apitenant.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPut, `tenants/{id}$`, api.UpdateHandler(apitenant.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodPost, `tenants/?$`, api.CreateHandler(apitenant.GetTypeSingleton()), Authenticated, nil},
		{1.1, http.MethodDelete, `tenants/{id}$`, api.DeleteHandler(apitenant.GetTypeSingleton()), Authenticated, nil},

		//CRConfig
		{1.1, http.MethodGet, `cdns/{cdn}/snapshot/?$`, crconfig.SnapshotGetHandler, Authenticated, nil},
		{1.1, http.MethodGet, `cdns/{cdn}/snapshot/new/?$`, crconfig.Handler, Authenticated, nil},
		{1.1, http.MethodPut, `cdns/{id}/snapshot/?$`, crconfig.SnapshotHandler, Authenticated, nil},
		{1.1, http.MethodPut, `snapshot/{cdn}/?$`, crconfig.SnapshotHandler, Authenticated, nil},

		//SSLKeys deliveryservice endpoints here that are marked  marked as '-wip' need to have tenancy checks added

		{1.3, http.MethodGet, `deliveryservices-wip/xmlId/{xmlID}/sslkeys$`, deliveryservice.GetSSLKeysByXMLID, Authenticated, nil},
		{1.3, http.MethodGet, `deliveryservices-wip/hostname/{hostName}/sslkeys$`, deliveryservice.GetSSLKeysByHostName, Authenticated, nil},
		{1.3, http.MethodPost, `deliveryservices-wip/hostname/{hostName}/sslkeys/add$`, deliveryservice.AddSSLKeys, Authenticated, nil},
		{1.3, http.MethodGet, `deliveryservices/xmlId/{name}/sslkeys/delete$`, deliveryservice.DeleteSSLKeys, Authenticated, nil},

		////DeliveryServices
		{1.3, http.MethodGet, `deliveryservices/?(\.json)?$`, api.ReadHandler(deliveryservice.GetTypeV13Factory()), Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/?(\.json)?$`, api.ReadHandler(deliveryservice.GetTypeV12Factory()), Authenticated, nil},
		{1.3, http.MethodGet, `deliveryservices/{id}/?(\.json)?$`, api.ReadHandler(deliveryservice.GetTypeV13Factory()), Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{id}/?(\.json)?$`, api.ReadHandler(deliveryservice.GetTypeV12Factory()), Authenticated, nil},
		{1.3, http.MethodPost, `deliveryservices/?(\.json)?$`, deliveryservice.CreateV13, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/?(\.json)?$`, deliveryservice.CreateV12, Authenticated, nil},
		{1.3, http.MethodPut, `deliveryservices/{id}/?(\.json)?$`, deliveryservice.UpdateV13, Authenticated, nil},
		{1.1, http.MethodPut, `deliveryservices/{id}/?(\.json)?$`, deliveryservice.UpdateV12, Authenticated, nil},
		{1.3, http.MethodDelete, `deliveryservices/{id}/?(\.json)?$`, api.DeleteHandler(deliveryservice.GetTypeV13Factory()), Authenticated, nil},
		{1.1, http.MethodDelete, `deliveryservices/{id}/?(\.json)?$`, api.DeleteHandler(deliveryservice.GetTypeV12Factory()), Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{id}/servers/eligible/?(\.json)?$`, deliveryservice.GetServersEligible, Authenticated, nil},

		{1.1, http.MethodPost, `deliveryservices/sslkeys/generate/?(\.json)?$`, deliveryservice.GenerateSSLKeys, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/copyFromXmlId/{copy-name}/?(\.json)?$`, deliveryservice.CopyURLKeys, Authenticated, nil},
		{1.1, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/generate/?(\.json)?$`, deliveryservice.GenerateURLKeys, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/xmlId/{name}/urlkeys/?(\.json)?$`, deliveryservice.GetURLKeysByName, Authenticated, nil},
		{1.1, http.MethodGet, `deliveryservices/{id}/urlkeys/?(\.json)?$`, deliveryservice.GetURLKeysByID, Authenticated, nil},
		{1.1, http.MethodGet, `riak/bucket/{bucket}/key/{key}/values/?(\.json)?$`, apiriak.GetBucketKey, Authenticated, nil},

		{1.1, http.MethodGet, `steering/{deliveryservice}/targets/?(\.json)?$`, api.ReadHandler(steeringtargets.TypeFactory), Authenticated, nil},
		{1.1, http.MethodGet, `steering/{deliveryservice}/targets/{target}$`, api.ReadHandler(steeringtargets.TypeFactory), Authenticated, nil},
		{1.1, http.MethodPost, `steering/{deliveryservice}/targets/?(\.json)?$`, api.CreateHandler(steeringtargets.TypeFactory), Authenticated, nil},
		{1.1, http.MethodPut, `steering/{deliveryservice}/targets/{target}/?(\.json)?$`, api.UpdateHandler(steeringtargets.TypeFactory), Authenticated, nil},
		{1.1, http.MethodDelete, `steering/{deliveryservice}/targets/{target}/?(\.json)?$`, api.DeleteHandler(steeringtargets.TypeFactory), Authenticated, nil},
	}

	// rawRoutes are served at the root path. These should be almost exclusively old Perl pre-API routes, which have yet to be converted in all clients. New routes should be in the versioned API path.
	rawRoutes := []RawRoute{
		// DEPRECATED - use PUT /api/1.2/snapshot/{cdn}
		{http.MethodGet, `tools/write_crconfig/{cdn}/?$`, crconfig.SnapshotOldGUIHandler, Authenticated, nil},
		// DEPRECATED - use GET /api/1.2/cdns/{cdn}/snapshot
		{http.MethodGet, `CRConfig-Snapshots/{cdn}/CRConfig.json?$`, crconfig.SnapshotOldGetHandler, Authenticated, nil},

		// These handlers force /api/* calls which don't match routes to be denied (via a `` api_capability).
		// This prevents nonmatching calls to be routed to Perl, which doesn't respect capabilities.
		// These should be removed when Perl TO (and the reverse-proxy to it) are completely removed.
		{http.MethodGet, `api`, handlerToFunc(proxyHandler), Authenticated, nil},
		{http.MethodPost, `api`, handlerToFunc(proxyHandler), Authenticated, nil},
		{http.MethodPut, `api`, handlerToFunc(proxyHandler), Authenticated, nil},
		{http.MethodDelete, `api`, handlerToFunc(proxyHandler), Authenticated, nil},
	}

	for _, r := range PerlRoutes(d) {
		routes = append(routes, r)
	}

	return routes, rawRoutes, proxyHandler, nil
}

// PerlRoutes returns the API routes handled by the Perl Traffic Ops via reverse-proxy.
func PerlRoutes(d ServerData) []Route {
	perlAPIHandler := getPerlAPIHandler(d)
	return []Route{
		{1.1, http.MethodGet, `caches/stats/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `cachegroup_fallbacks/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `cachegroup_fallbacks/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPut, `cachegroup_fallbacks/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodDelete, `cachegroup_fallbacks/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `cachegroups/{id}/deliveryservices/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `cdns/{name}/health/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `cdns/name/{name}/dnsseckeys/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `cdns/{name}/configs/routing/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `logs/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `logs/{days}/days/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `logs/newcount/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `servers/{id}/configfiles/ats/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `profiles/{id}/configfiles/ats/{filename}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `servers/{id}/configfiles/ats/{filename}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `cdns/{id}/configfiles/ats/{filename}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `dbdump/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPut, `deliveryservices/{id}/safe/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `deliveryservices/{id}/health/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `deliveryservices/{id}/capacity/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `deliveryservices/{id}/routing/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `deliveryservices/{id}/state/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `deliveryservices/request/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `deliveryservices/xmlId/{xmlid}/sslkeys/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `deliveryservices/hostname/{hostname}/sslkeys/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `deliveryservices/sslkeys/add/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodDelete, `divisions/name/{name}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `federations/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `federations/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPut, `federations/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodDelete, `federations/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `federations/{fedId}/users/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `federations/{fedId}/users/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodDelete, `federations/{fedId}/users/{userId}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `federations/{fedId}/deliveryservices/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `federations/{fedId}/deliveryservices/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodDelete, `federations/{fedId}/deliveryservices/{dsId}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `federations/{fedId}/federation_resolvers/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `federations/{fedId}/federation_resolvers/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `federation_resolvers/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodDelete, `federation_resolvers/{id}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `hwinfo/dtdata/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `hwinfo/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `osversions/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `isos/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `jobs/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `jobs/{id}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `user/current/jobs/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `user/current/jobs/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `parameters/validate/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `cachegroups/{id}/parameters/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `cachegroups/{id}/unassigned_parameters/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `cachegroup/{parameter_id}/parameter/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `cachegroupparameters/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `cachegroupparameters/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodDelete, `cachegroupparameters/{cachegroup_id}/{parameter_id}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `cachegroups/{parameter_id}/parameter/available/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `regions/{region_name}/phys_locations/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `profiles/name/{profile_name}/copy/{profile_copy_from}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `profiles/{id}/export/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `profiles/import/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `parameters/{id}/profiles/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `parameters/{id}/unassigned_profiles/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `divisions/{division_name}/regions/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodDelete, `regions/name/{name}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `capabilities/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `capabilities/{name}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPut, `capabilities/{name}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `capabilities/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodDelete, `capabilities/{name}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `api_capabilities/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `api_capabilities/{id}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPut, `api_capabilities/{id}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `api_capabilities/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodDelete, `api_capabilities/{id}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `servers/{id}/queue_update/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPut, `servers/{id}/status/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `servercheck/aadata/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `servercheck/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `stats_summary/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `stats_summary/create/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `types/trimmed/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `users/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `users/{id}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPut, `users/{id}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `users/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `users/register/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `deliveryservice_user/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodDelete, `deliveryservice_user/{dsId}/{userId}/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPut, `user/current/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `user/current/update/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `user/login/token/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `user/logout/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `user/reset_password/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `riak/stats/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `to_extensions/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `to_extensions/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodPost, `to_extensions/{id}/delete/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
		{1.1, http.MethodGet, `traffic_monitor/stats/?(\.json)?$`, perlAPIHandler, Authenticated, []Middleware{}},
	}
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

var ErrUnauthorized = errors.New("Unauthorized, please log in.")

// getPerlAPIHandler checks the user's capabilities, and reverse-proxies to Perl iff the user's role has a capability with the "api capability" of the requested route.
func getPerlAPIHandler(d ServerData) http.HandlerFunc {
	rootHandle := rootHandler(d)
	return func(w http.ResponseWriter, r *http.Request) {
		noTx := (*sql.Tx)(nil)
		// TODO make func
		cookie, err := r.Cookie(tocookie.Name)
		if err != nil {
			api.HandleErr(w, r, noTx, http.StatusUnauthorized, ErrUnauthorized, errors.New("Getting cookie: "+err.Error()))
			return
		}
		if cookie == nil {
			api.HandleErr(w, r, noTx, http.StatusUnauthorized, ErrUnauthorized, nil)
			return
		}
		toCookie, err := tocookie.Parse(d.Config.Secrets[0], cookie.Value)
		if err != nil {
			api.HandleErr(w, r, noTx, http.StatusUnauthorized, ErrUnauthorized, errors.New("Parsing cookie: "+err.Error()))
			return
		}
		username := toCookie.AuthData
		if username == "" {
			api.HandleErr(w, r, noTx, http.StatusUnauthorized, ErrUnauthorized, nil)
			return
		}

		dbTimeout := time.Duration(d.Config.DBQueryTimeoutSeconds) * time.Second
		// MUST check the db, even though we only need the username, because the cookie could lie
		user, userErr, sysErr, code := auth.GetCurrentUserFromDB(d.DB, username, dbTimeout)
		if userErr != nil || sysErr != nil {
			api.HandleErr(w, r, noTx, code, userErr, sysErr)
			return
		}

		apiCapability, err := api.GetAPICapability(r.Context())
		if err != nil {
			api.HandleErr(w, r, noTx, http.StatusInternalServerError, nil, errors.New("No capability found in request context!"))
			return
		}
		userErr, sysErr, errCode := CheckAPICapability(d.DB.DB, dbTimeout, &user, r.Method, apiCapability)
		if userErr != nil || sysErr != nil {
			api.HandleErr(w, r, noTx, errCode, userErr, sysErr)
		}
		rootHandle.ServeHTTP(w, r)
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
