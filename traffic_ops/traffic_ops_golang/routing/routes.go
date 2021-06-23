package routing

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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/about"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/acme"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apicapability"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apitenant"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/asn"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/cachegroup"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/cachegroupparameter"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/cachesstats"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/capabilities"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/cdn"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/cdn_lock"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/cdnfederation"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/cdnnotification"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/coordinate"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/crconfig"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/crstats"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbdump"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice/consistenthash"
	dsrequest "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice/request"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice/request/comment"
	dsserver "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice/servers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservicerequests"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservicesregexes"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/division"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/federation_resolvers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/federations"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/hwinfo"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/invalidationjobs"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/iso"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/login"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/logs"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/origin"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/parameter"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/physlocation"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/ping"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/plugins"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/profile"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/profileparameter"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/region"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/role"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/routing/middleware"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/server"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/servercapability"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/servercheck"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/servercheck/extensions"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/servicecategory"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/staticdnsentry"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/status"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/steering"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/steeringtargets"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/systeminfo"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/topology"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficstats"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/types"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/urisigning"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/user"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/vault"

	"github.com/jmoiron/sqlx"
)

// Authenticated indicates that a route requires authentication for use.
const Authenticated = true

// NoAuth indicates that a route does not require authentication for use.
const NoAuth = false

func handlerToFunc(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}
}

// GetRouteIDMap takes a []int Route IDs and converts it into a map for fast lookup.
func GetRouteIDMap(IDs []int) map[int]struct{} {
	m := make(map[int]struct{}, len(IDs))
	for _, id := range IDs {
		m[id] = struct{}{}
	}
	return m
}

// Routes returns the API routes, raw non-API root level routes, and a catchall route for when no route matches.
func Routes(d ServerData) ([]Route, []RawRoute, http.Handler, error) {
	proxyHandler := rootHandler(d)

	routes := []Route{
		// 1.1 and 1.2 routes are simply a Go replacement for the equivalent Perl route. They may or may not conform with the API guidelines (https://cwiki.apache.org/confluence/display/TC/API+Guidelines).
		// 1.3 routes exist only in Go. There is NO equivalent Perl route. They should conform with the API guidelines (https://cwiki.apache.org/confluence/display/TC/API+Guidelines).

		// 2.x routes exist only in Go. There is NO equivalent Perl route. They should conform with the API guidelines (https://cwiki.apache.org/confluence/display/TC/API+Guidelines).

		// NOTE: Route IDs are immutable and unique. DO NOT change the ID of an existing Route; otherwise, existing
		// configurations may break. New Route IDs can be any integer between 0 and 2147483647 (inclusive), as long as
		// it's unique.

		/**
		 * 4.x API
		 */

		// CDN lock
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdn_locks/?$`, cdn_lock.Read, auth.PrivLevelReadOnly, Authenticated, nil, 4134390561},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `cdn_locks/?$`, cdn_lock.Create, auth.PrivLevelOperations, Authenticated, nil, 4134390562},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `cdn_locks/?$`, cdn_lock.Delete, auth.PrivLevelOperations, Authenticated, nil, 4134390564},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `acme_accounts/providers/?$`, acme.ReadProviders, auth.PrivLevelOperations, Authenticated, nil, 4034390565},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices/sslkeys/generate/acme/?$`, deliveryservice.GenerateAcmeCertificates, auth.PrivLevelOperations, Authenticated, nil, 2534390576},

		// ACME account information
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `acme_accounts/?$`, acme.Read, auth.PrivLevelAdmin, Authenticated, nil, 4034390561},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `acme_accounts/?$`, acme.Create, auth.PrivLevelAdmin, Authenticated, nil, 4034390562},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `acme_accounts/?$`, acme.Update, auth.PrivLevelAdmin, Authenticated, nil, 4034390563},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `acme_accounts/{provider}/{email}/?$`, acme.Delete, auth.PrivLevelAdmin, Authenticated, nil, 4034390564},

		//Delivery service ACME
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices/xmlId/{xmlid}/sslkeys/renew$`, deliveryservice.RenewAcmeCertificate, auth.PrivLevelOperations, Authenticated, nil, 2534390573},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `acme_autorenew/?$`, deliveryservice.RenewCertificates, auth.PrivLevelOperations, Authenticated, nil, 2534390574},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `async_status/{id}$`, api.GetAsyncStatus, auth.PrivLevelOperations, Authenticated, nil, 2534390575},

		// API Capability
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `api_capabilities/?$`, apicapability.GetAPICapabilitiesHandler, auth.PrivLevelReadOnly, Authenticated, nil, 48132065893},

		//ASNs
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `asns/?$`, api.UpdateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 42641723173},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `asns/?$`, api.DeleteHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 402048983},

		//ASN: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `asns/?$`, api.ReadHandler(&asn.TOASNV11{}), auth.PrivLevelReadOnly, Authenticated, nil, 4738777223},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `asns/{id}$`, api.UpdateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 49511986293},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `asns/?$`, api.CreateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 49994921883},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `asns/{id}$`, api.DeleteHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 46725247693},

		// Traffic Stats access
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservice_stats`, trafficstats.GetDSStats, auth.PrivLevelReadOnly, Authenticated, nil, 43195690283},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cache_stats`, trafficstats.GetCacheStats, auth.PrivLevelReadOnly, Authenticated, nil, 44979979063},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `current_stats/?$`, trafficstats.GetCurrentStats, auth.PrivLevelReadOnly, Authenticated, nil, 47854428933},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `caches/stats/?$`, cachesstats.Get, auth.PrivLevelReadOnly, Authenticated, nil, 48132065883},

		//CacheGroup: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cachegroups/?$`, api.ReadHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelReadOnly, Authenticated, nil, 4230791103},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `cachegroups/{id}$`, api.UpdateHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 4129545463},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `cachegroups/?$`, api.CreateHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 429826653},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `cachegroups/{id}$`, api.DeleteHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 4278693653},

		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `cachegroups/{id}/queue_update$`, cachegroup.QueueUpdates, auth.PrivLevelOperations, Authenticated, nil, 40716441103},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `cachegroups/{id}/deliveryservices/?$`, cachegroup.DSPostHandlerV40, auth.PrivLevelOperations, Authenticated, nil, 45202404313},

		//CacheGroup Parameters: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cachegroupparameters/?$`, cachegroupparameter.ReadAllCacheGroupParameters, auth.PrivLevelReadOnly, Authenticated, nil, 4124497243},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `cachegroupparameters/?$`, cachegroupparameter.AddCacheGroupParameters, auth.PrivLevelOperations, Authenticated, nil, 4124497253},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cachegroups/{id}/parameters/?$`, api.ReadHandler(&cachegroupparameter.TOCacheGroupParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 4124497233},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `cachegroupparameters/{cachegroupID}/{parameterId}$`, api.DeleteHandler(&cachegroupparameter.TOCacheGroupParameter{}), auth.PrivLevelOperations, Authenticated, nil, 4124497333},

		//Capabilities
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `capabilities/?$`, capabilities.Read, auth.PrivLevelReadOnly, Authenticated, nil, 40081353},

		//CDN
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/name/{name}/sslkeys/?$`, cdn.GetSSLKeys, auth.PrivLevelAdmin, Authenticated, nil, 42785817723},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/capacity$`, cdn.GetCapacity, auth.PrivLevelReadOnly, Authenticated, nil, 4971852813},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/{name}/health/?$`, cdn.GetNameHealth, auth.PrivLevelReadOnly, Authenticated, nil, 41353481943},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/health/?$`, cdn.GetHealth, auth.PrivLevelReadOnly, Authenticated, nil, 40853811343},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/domains/?$`, cdn.DomainsHandler, auth.PrivLevelReadOnly, Authenticated, nil, 4269025603},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/routing$`, crstats.GetCDNRouting, auth.PrivLevelReadOnly, Authenticated, nil, 467229823},

		//CDN: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `cdns/name/{name}$`, cdn.DeleteName, auth.PrivLevelOperations, Authenticated, nil, 4088049593},

		//CDN: queue updates
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `cdns/{id}/queue_update$`, cdn.Queue, auth.PrivLevelOperations, Authenticated, nil, 4215159803},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `cdns/dnsseckeys/generate?$`, cdn.CreateDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil, 4753363},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `cdns/name/{name}/dnsseckeys?$`, cdn.DeleteDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil, 4711042073},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/name/{name}/dnsseckeys/?$`, cdn.GetDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil, 4790106093},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/dnsseckeys/refresh/?$`, cdn.RefreshDNSSECKeys, auth.PrivLevelOperations, Authenticated, nil, 47719971163},

		//CDN: Monitoring: Traffic Monitor
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/{cdn}/configs/monitoring?$`, crconfig.SnapshotGetMonitoringHandler, auth.PrivLevelReadOnly, Authenticated, nil, 42408478923},

		//Database dumps
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `dbdump/?`, dbdump.DBDump, auth.PrivLevelAdmin, Authenticated, nil, 4240166473},

		//Division: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `divisions/?$`, api.ReadHandler(&division.TODivision{}), auth.PrivLevelReadOnly, Authenticated, nil, 40851815343},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `divisions/{id}$`, api.UpdateHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 4063691403},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `divisions/?$`, api.CreateHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 4537138003},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `divisions/{id}$`, api.DeleteHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 43253822373},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `logs/?$`, logs.Get, auth.PrivLevelReadOnly, Authenticated, nil, 4483405503},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `logs/newcount/?$`, logs.GetNewCount, auth.PrivLevelReadOnly, Authenticated, nil, 44058330123},

		//Content invalidation jobs
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `jobs/?$`, api.ReadHandler(&invalidationjobs.InvalidationJob{}), auth.PrivLevelReadOnly, Authenticated, nil, 49667820413},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `jobs/?$`, invalidationjobs.Delete, auth.PrivLevelPortal, Authenticated, nil, 4167807763},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `jobs/?$`, invalidationjobs.Update, auth.PrivLevelPortal, Authenticated, nil, 4861342263},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `jobs/?`, invalidationjobs.Create, auth.PrivLevelPortal, Authenticated, nil, 404509553},

		//Login
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `user/login/?$`, login.LoginHandler(d.DB, d.Config), 0, NoAuth, nil, 43926708213},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `user/logout/?$`, login.LogoutHandler(d.Config.Secrets[0]), 0, Authenticated, nil, 4434348253},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `user/login/oauth/?$`, login.OauthLoginHandler(d.DB, d.Config), 0, NoAuth, nil, 44158860093},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `user/login/token/?$`, login.TokenLoginHandler(d.DB, d.Config), 0, NoAuth, nil, 4024088413},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `user/reset_password/?$`, login.ResetPassword(d.DB, d.Config), 0, NoAuth, nil, 42929146303},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `users/register/?$`, login.RegisterUser, auth.PrivLevelOperations, Authenticated, nil, 43373},

		//ISO
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `osversions/?$`, iso.GetOSVersions, auth.PrivLevelReadOnly, Authenticated, nil, 4760886573},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `isos/?$`, iso.ISOs, auth.PrivLevelOperations, Authenticated, nil, 4760336573},

		//User: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `users/?$`, api.ReadHandler(&user.TOUser{}), auth.PrivLevelReadOnly, Authenticated, nil, 44919299003},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `users/{id}$`, api.ReadHandler(&user.TOUser{}), auth.PrivLevelReadOnly, Authenticated, nil, 4138099803},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `users/{id}$`, api.UpdateHandler(&user.TOUser{}), auth.PrivLevelOperations, Authenticated, nil, 4354334043},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `users/?$`, api.CreateHandler(&user.TOUser{}), auth.PrivLevelOperations, Authenticated, nil, 4762448163},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `user/current/?$`, user.Current, auth.PrivLevelReadOnly, Authenticated, nil, 46107016143},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `user/current/?$`, user.ReplaceCurrent, auth.PrivLevelReadOnly, Authenticated, nil, 4203},

		//Parameter: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `parameters/?$`, api.ReadHandler(&parameter.TOParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 42125542923},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `parameters/{id}$`, api.UpdateHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 48739361153},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `parameters/?$`, api.CreateHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 46695108593},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `parameters/{id}$`, api.DeleteHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 4262771183},

		//Phys_Location: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `phys_locations/?$`, api.ReadHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelReadOnly, Authenticated, nil, 4204051823},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `phys_locations/{id}$`, api.UpdateHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 4227950213},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `phys_locations/?$`, api.CreateHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 42464566483},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `phys_locations/{id}$`, api.DeleteHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 456142213},

		//Ping
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `ping$`, ping.Handler, 0, NoAuth, nil, 45556615973},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `vault/ping/?$`, ping.Vault, auth.PrivLevelReadOnly, Authenticated, nil, 48840121143},

		//Profile: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `profiles/?$`, api.ReadHandler(&profile.TOProfile{}), auth.PrivLevelReadOnly, Authenticated, nil, 4687585893},

		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `profiles/{id}$`, api.UpdateHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 484391723},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `profiles/?$`, api.CreateHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 45402115563},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `profiles/{id}$`, api.DeleteHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 42055944653},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `profiles/{id}/export/?$`, profile.ExportProfileHandler, auth.PrivLevelReadOnly, Authenticated, nil, 401335173},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `profiles/import/?$`, profile.ImportProfileHandler, auth.PrivLevelOperations, Authenticated, nil, 4061432083},

		// Copy Profile
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `profiles/name/{new_profile}/copy/{existing_profile}`, profile.CopyProfileHandler, auth.PrivLevelOperations, Authenticated, nil, 4061432093},

		//Region: CRUDs
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `regions/?$`, api.ReadHandler(&region.TORegion{}), auth.PrivLevelReadOnly, Authenticated, nil, 4100370853},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `regions/{id}$`, api.UpdateHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 4223082243},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `regions/?$`, api.CreateHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 42883344883},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `regions/?$`, api.DeleteHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 42326267583},

		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `topologies/?$`, api.CreateHandler(&topology.TOTopology{}), auth.PrivLevelOperations, Authenticated, nil, 4871452221},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `topologies/?$`, api.ReadHandler(&topology.TOTopology{}), auth.PrivLevelReadOnly, Authenticated, nil, 4871452222},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `topologies/?$`, api.UpdateHandler(&topology.TOTopology{}), auth.PrivLevelOperations, Authenticated, nil, 4871452223},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `topologies/?$`, api.DeleteHandler(&topology.TOTopology{}), auth.PrivLevelOperations, Authenticated, nil, 4871452224},

		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `topologies/{name}/queue_update$`, topology.QueueUpdateHandler, auth.PrivLevelOperations, Authenticated, nil, 4205351748},

		// get all edge servers associated with a delivery service (from deliveryservice_server table)

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryserviceserver/?$`, dsserver.ReadDSSHandlerV14, auth.PrivLevelReadOnly, Authenticated, nil, 49461450333},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryserviceserver$`, dsserver.GetReplaceHandler, auth.PrivLevelOperations, Authenticated, nil, 4297997883},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `deliveryserviceserver/{dsid}/{serverid}`, dsserver.Delete, auth.PrivLevelOperations, Authenticated, nil, 45321845233},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices/{xml_id}/servers$`, dsserver.GetCreateHandler, auth.PrivLevelOperations, Authenticated, nil, 44281812063},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `servers/{id}/deliveryservices$`, api.ReadHandler(&dsserver.TODSSDeliveryService{}), auth.PrivLevelReadOnly, Authenticated, nil, 4331154113},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `servers/{id}/deliveryservices$`, server.AssignDeliveryServicesToServerHandler, auth.PrivLevelOperations, Authenticated, nil, 4801282533},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices/{id}/servers$`, dsserver.GetReadAssigned, auth.PrivLevelReadOnly, Authenticated, nil, 43451212233},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices/{id}/capacity/?$`, deliveryservice.GetCapacity, auth.PrivLevelReadOnly, Authenticated, nil, 42314091103},
		//Serverchecks
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `servercheck/?$`, servercheck.ReadServerCheck, auth.PrivLevelReadOnly, Authenticated, nil, 47961129223},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `servercheck/?$`, servercheck.CreateUpdateServercheck, auth.PrivLevelInvalid, Authenticated, nil, 47642815683},

		// Servercheck Extensions
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `servercheck/extensions$`, extensions.Create, auth.PrivLevelReadOnly, Authenticated, nil, 4804985993},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `servercheck/extensions$`, extensions.Get, auth.PrivLevelReadOnly, Authenticated, nil, 4834985993},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `servercheck/extensions/{id}$`, extensions.Delete, auth.PrivLevelReadOnly, Authenticated, nil, 4804982993},

		//Server Details
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `servers/details/?$`, server.GetDetailParamHandler, auth.PrivLevelReadOnly, Authenticated, nil, 42612647143},

		//Server status
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `servers/{id}/status$`, server.UpdateStatusHandler, auth.PrivLevelOperations, Authenticated, nil, 4766638513},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `servers/{id}/queue_update$`, server.QueueUpdateHandler, auth.PrivLevelOperations, Authenticated, nil, 41894713},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `servers/{host_name}/update_status$`, server.GetServerUpdateStatusHandler, auth.PrivLevelReadOnly, Authenticated, nil, 4384515993},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `servers/{id-or-name}/update$`, server.UpdateHandler, auth.PrivLevelOperations, Authenticated, nil, 443813233},

		//Server: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `servers/?$`, server.Read, auth.PrivLevelReadOnly, Authenticated, nil, 47209592853},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `servers/{id}$`, server.Update, auth.PrivLevelOperations, Authenticated, nil, 4586341033},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `servers/?$`, server.Create, auth.PrivLevelOperations, Authenticated, nil, 42255580613},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `servers/{id}$`, server.Delete, auth.PrivLevelOperations, Authenticated, nil, 4923222333},

		//Server Capability
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `server_capabilities$`, api.ReadHandler(&servercapability.TOServerCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 4104073913},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `server_capabilities$`, api.CreateHandler(&servercapability.TOServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 40744707083},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `server_capabilities$`, api.UpdateHandler(&servercapability.TOServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 42543770109},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `server_capabilities$`, api.DeleteHandler(&servercapability.TOServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 4364150383},

		//Server Server Capabilities: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `server_server_capabilities/?$`, api.ReadHandler(&server.TOServerServerCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 48002318893},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `server_server_capabilities/?$`, api.CreateHandler(&server.TOServerServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 42931668343},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `server_server_capabilities/?$`, api.DeleteHandler(&server.TOServerServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 40587140583},

		//Status: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `statuses/?$`, api.ReadHandler(&status.TOStatus{}), auth.PrivLevelReadOnly, Authenticated, nil, 42449056563},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `statuses/{id}$`, api.UpdateHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 42079665043},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `statuses/?$`, api.CreateHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 43691236123},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `statuses/{id}$`, api.DeleteHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 4551113603},

		//System
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `system/info/?$`, systeminfo.Get, auth.PrivLevelReadOnly, Authenticated, nil, 4210474753},

		//Type: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `types/?$`, api.ReadHandler(&types.TOType{}), auth.PrivLevelReadOnly, Authenticated, nil, 42267018233},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `types/{id}$`, api.UpdateHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 488601153},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `types/?$`, api.CreateHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 45133081953},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `types/{id}$`, api.DeleteHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 431757733},

		//About
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `about/?$`, about.Handler(), auth.PrivLevelReadOnly, Authenticated, nil, 43175011663},

		//Coordinates
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `coordinates/?$`, api.ReadHandler(&coordinate.TOCoordinate{}), auth.PrivLevelReadOnly, Authenticated, nil, 4967007453},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `coordinates/?$`, api.UpdateHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 4689261743},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `coordinates/?$`, api.CreateHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 44281121573},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `coordinates/?$`, api.DeleteHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 43038498893},

		//CDN notification
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdn_notifications/?$`, cdnnotification.Read, auth.PrivLevelReadOnly, Authenticated, nil, 2221224514},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `cdn_notifications/?$`, cdnnotification.Create, auth.PrivLevelOperations, Authenticated, nil, 2765223513},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `cdn_notifications/?$`, cdnnotification.Delete, auth.PrivLevelOperations, Authenticated, nil, 2722411851},

		//CDN generic handlers:
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/?$`, api.ReadHandler(&cdn.TOCDN{}), auth.PrivLevelReadOnly, Authenticated, nil, 42303186213},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `cdns/{id}$`, api.UpdateHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 43111789343},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `cdns/?$`, api.CreateHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 41605052893},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `cdns/{id}$`, api.DeleteHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 4276946573},

		//Delivery service requests
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservice_requests/?$`, dsrequest.Get, auth.PrivLevelReadOnly, Authenticated, nil, 46811639353},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `deliveryservice_requests/?$`, dsrequest.Put, auth.PrivLevelPortal, Authenticated, nil, 42499079183},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservice_requests/?$`, dsrequest.Post, auth.PrivLevelPortal, Authenticated, nil, 493850393},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `deliveryservice_requests/?$`, dsrequest.Delete, auth.PrivLevelPortal, Authenticated, nil, 42969850253},

		//Delivery service request: Actions
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservice_requests/{id}/assign$`, dsrequest.GetAssignment, auth.PrivLevelOperations, Authenticated, nil, 47031602904},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `deliveryservice_requests/{id}/assign$`, dsrequest.PutAssignment, auth.PrivLevelOperations, Authenticated, nil, 47031602903},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservice_requests/{id}/status$`, dsrequest.GetStatus, auth.PrivLevelPortal, Authenticated, nil, 4684150994},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `deliveryservice_requests/{id}/status$`, dsrequest.PutStatus, auth.PrivLevelPortal, Authenticated, nil, 4684150993},

		//Delivery service request comment: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservice_request_comments/?$`, api.ReadHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelReadOnly, Authenticated, nil, 40326507373},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `deliveryservice_request_comments/?$`, api.UpdateHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 4604878473},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservice_request_comments/?$`, api.CreateHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 4272276723},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `deliveryservice_request_comments/?$`, api.DeleteHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 4995046683},

		//Delivery service uri signing keys: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.GetURIsignkeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 42930785583},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.SaveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 4084663353},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.SaveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 476489693},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.RemoveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 4299254173},

		//Delivery Service Required Capabilities: CRUD
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices_required_capabilities/?$`, api.ReadHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 41585222273},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices_required_capabilities/?$`, api.CreateHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelOperations, Authenticated, nil, 40968739923},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `deliveryservices_required_capabilities/?$`, api.DeleteHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelOperations, Authenticated, nil, 44962893043},

		// Federations by CDN (the actual table for federation)
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/{name}/federations/?$`, api.ReadHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelReadOnly, Authenticated, nil, 4892250323},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `cdns/{name}/federations/?$`, api.CreateHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 49548942193},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `cdns/{name}/federations/{id}$`, api.UpdateHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 4260654663},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `cdns/{name}/federations/{id}$`, api.DeleteHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 44428529023},

		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `cdns/{name}/dnsseckeys/ksk/generate$`, cdn.GenerateKSK, auth.PrivLevelAdmin, Authenticated, nil, 4729242813},

		//Origins
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `origins/?$`, api.ReadHandler(&origin.TOOrigin{}), auth.PrivLevelReadOnly, Authenticated, nil, 4446492563},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `origins/?$`, api.UpdateHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 415677463},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `origins/?$`, api.CreateHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 40995616433},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `origins/?$`, api.DeleteHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 4602732633},

		//Roles
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `roles/?$`, api.ReadHandler(&role.TORole{}), auth.PrivLevelReadOnly, Authenticated, nil, 4870885833},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `roles/?$`, api.UpdateHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 46128974893},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `roles/?$`, api.CreateHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 4306524063},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `roles/?$`, api.DeleteHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 43567059823},

		//Delivery Services Regexes
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices_regexes/?$`, deliveryservicesregexes.Get, auth.PrivLevelReadOnly, Authenticated, nil, 4055014533},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices/{dsid}/regexes/?$`, deliveryservicesregexes.DSGet, auth.PrivLevelReadOnly, Authenticated, nil, 4774327633},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices/{dsid}/regexes/?$`, deliveryservicesregexes.Post, auth.PrivLevelOperations, Authenticated, nil, 4127378003},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `deliveryservices/{dsid}/regexes/{regexid}?$`, deliveryservicesregexes.Put, auth.PrivLevelOperations, Authenticated, nil, 42483396913},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `deliveryservices/{dsid}/regexes/{regexid}?$`, deliveryservicesregexes.Delete, auth.PrivLevelOperations, Authenticated, nil, 42467316633},

		//ServiceCategories
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `service_categories/?$`, api.ReadHandler(&servicecategory.TOServiceCategory{}), auth.PrivLevelReadOnly, Authenticated, nil, 4085181543},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `service_categories/{name}/?$`, servicecategory.Update, auth.PrivLevelOperations, Authenticated, nil, 406369141},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `service_categories/?$`, api.CreateHandler(&servicecategory.TOServiceCategory{}), auth.PrivLevelOperations, Authenticated, nil, 453713801},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `service_categories/{name}$`, api.DeleteHandler(&servicecategory.TOServiceCategory{}), auth.PrivLevelOperations, Authenticated, nil, 4325382238},

		//StaticDNSEntries
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `staticdnsentries/?$`, api.ReadHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelReadOnly, Authenticated, nil, 4289394773},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `staticdnsentries/?$`, api.UpdateHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 4424571113},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `staticdnsentries/?$`, api.CreateHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 46291482383},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `staticdnsentries/?$`, api.DeleteHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 48460311323},

		//ProfileParameters
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `profiles/{id}/parameters/?$`, profileparameter.GetProfileID, auth.PrivLevelReadOnly, Authenticated, nil, 4764649753},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `profiles/name/{name}/parameters/?$`, profileparameter.GetProfileName, auth.PrivLevelReadOnly, Authenticated, nil, 42677378323},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `profiles/name/{name}/parameters/?$`, profileparameter.PostProfileParamsByName, auth.PrivLevelOperations, Authenticated, nil, 43559455823},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `profiles/{id}/parameters/?$`, profileparameter.PostProfileParamsByID, auth.PrivLevelOperations, Authenticated, nil, 4168187083},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `profileparameters/?$`, api.ReadHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 4506098053},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `profileparameters/?$`, api.CreateHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelOperations, Authenticated, nil, 4288096933},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `profileparameter/?$`, profileparameter.PostProfileParam, auth.PrivLevelOperations, Authenticated, nil, 4242753},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `parameterprofile/?$`, profileparameter.PostParamProfile, auth.PrivLevelOperations, Authenticated, nil, 40806108613},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `profileparameters/{profileId}/{parameterId}$`, api.DeleteHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelOperations, Authenticated, nil, 4248395293},

		//Tenants
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `tenants/?$`, api.ReadHandler(&apitenant.TOTenant{}), auth.PrivLevelReadOnly, Authenticated, nil, 46779678143},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `tenants/{id}$`, api.UpdateHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 40941314783},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `tenants/?$`, api.CreateHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 4172480133},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `tenants/{id}$`, api.DeleteHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 4163655583},

		//CRConfig
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/{cdn}/snapshot/?$`, crconfig.SnapshotGetHandler, auth.PrivLevelReadOnly, Authenticated, nil, 49572736953},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `cdns/{cdn}/snapshot/new/?$`, crconfig.Handler, auth.PrivLevelReadOnly, Authenticated, nil, 4767168893},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `snapshot/?$`, crconfig.SnapshotHandler, auth.PrivLevelOperations, Authenticated, nil, 49699118293},

		// Federations
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `federations/all/?$`, federations.GetAll, auth.PrivLevelAdmin, Authenticated, nil, 410599863},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `federations/?$`, federations.Get, auth.PrivLevelFederation, Authenticated, nil, 4549549943},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `federations/?$`, federations.AddFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 48940647423},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `federations/?$`, federations.RemoveFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 420983233},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `federations/?$`, federations.ReplaceFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 42831825163},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `federations/{id}/deliveryservices/?$`, federations.PostDSes, auth.PrivLevelAdmin, Authenticated, nil, 46828635133},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `federations/{id}/deliveryservices/?$`, api.ReadHandler(&federations.TOFedDSes{}), auth.PrivLevelReadOnly, Authenticated, nil, 4537730343},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `federations/{id}/deliveryservices/{dsID}/?$`, api.DeleteHandler(&federations.TOFedDSes{}), auth.PrivLevelAdmin, Authenticated, nil, 44174025703},

		// Federation Resolvers
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `federation_resolvers/?$`, federation_resolvers.Create, auth.PrivLevelAdmin, Authenticated, nil, 41343736613},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `federation_resolvers/?$`, federation_resolvers.Read, auth.PrivLevelReadOnly, Authenticated, nil, 4566087593},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `federations/{id}/federation_resolvers/?$`, federations.AssignFederationResolversToFederationHandler, auth.PrivLevelAdmin, Authenticated, nil, 4566087603},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `federations/{id}/federation_resolvers/?$`, federations.GetFederationFederationResolversHandler, auth.PrivLevelReadOnly, Authenticated, nil, 4566087613},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `federation_resolvers/?$`, federation_resolvers.Delete, auth.PrivLevelAdmin, Authenticated, nil, 40013},

		// Federations Users
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `federations/{id}/users/?$`, federations.PostUsers, auth.PrivLevelAdmin, Authenticated, nil, 47793349303},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `federations/{id}/users/?$`, api.ReadHandler(&federations.TOUsers{}), auth.PrivLevelReadOnly, Authenticated, nil, 4940750153},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `federations/{id}/users/{userID}/?$`, api.DeleteHandler(&federations.TOUsers{}), auth.PrivLevelAdmin, Authenticated, nil, 49491028823},

		////DeliveryServices
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices/?$`, api.ReadHandler(&deliveryservice.TODeliveryService{}), auth.PrivLevelReadOnly, Authenticated, nil, 42383172943},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices/?$`, deliveryservice.CreateV40, auth.PrivLevelOperations, Authenticated, nil, 4064315323},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `deliveryservices/{id}/?$`, deliveryservice.UpdateV40, auth.PrivLevelOperations, Authenticated, nil, 47665675673},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `deliveryservices/{id}/safe/?$`, deliveryservice.UpdateSafe, auth.PrivLevelOperations, Authenticated, nil, 4472109313},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `deliveryservices/{id}/?$`, api.DeleteHandler(&deliveryservice.TODeliveryService{}), auth.PrivLevelOperations, Authenticated, nil, 4226420743},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices/{id}/servers/eligible/?$`, deliveryservice.GetServersEligible, auth.PrivLevelReadOnly, Authenticated, nil, 4747615843},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices/xmlId/{xmlid}/sslkeys$`, deliveryservice.GetSSLKeysByXMLIDV15, auth.PrivLevelAdmin, Authenticated, nil, 41357729073},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices/sslkeys/add$`, deliveryservice.AddSSLKeys, auth.PrivLevelAdmin, Authenticated, nil, 48728785833},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `deliveryservices/xmlId/{xmlid}/sslkeys$`, deliveryservice.DeleteSSLKeys, auth.PrivLevelOperations, Authenticated, nil, 49267343},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices/sslkeys/generate/?$`, deliveryservice.GenerateSSLKeys, auth.PrivLevelOperations, Authenticated, nil, 4534390513},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/copyFromXmlId/{copy-name}/?$`, deliveryservice.CopyURLKeys, auth.PrivLevelOperations, Authenticated, nil, 42625010763},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/generate/?$`, deliveryservice.GenerateURLKeys, auth.PrivLevelOperations, Authenticated, nil, 45304828243},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices/xmlId/{name}/urlkeys/?$`, deliveryservice.GetURLKeysByName, auth.PrivLevelReadOnly, Authenticated, nil, 42027192113},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `deliveryservices/xmlId/{name}/urlkeys/?$`, deliveryservice.DeleteURLKeysByName, auth.PrivLevelOperations, Authenticated, nil, 42027192114},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices/{id}/urlkeys/?$`, deliveryservice.GetURLKeysByID, auth.PrivLevelReadOnly, Authenticated, nil, 4931971143},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `deliveryservices/{id}/urlkeys/?$`, deliveryservice.DeleteURLKeysByID, auth.PrivLevelOperations, Authenticated, nil, 4931971144},

		//Delivery service LetsEncrypt
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `deliveryservices/sslkeys/generate/letsencrypt/?$`, deliveryservice.GenerateLetsEncryptCertificates, auth.PrivLevelOperations, Authenticated, nil, 4534390523},
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `letsencrypt/dnsrecords/?$`, deliveryservice.GetDnsChallengeRecords, auth.PrivLevelOperations, Authenticated, nil, 4534390553},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `letsencrypt/autorenew/?$`, deliveryservice.RenewCertificatesDeprecated, auth.PrivLevelOperations, Authenticated, nil, 4534390563},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices/{id}/health/?$`, deliveryservice.GetHealth, auth.PrivLevelReadOnly, Authenticated, nil, 42345901013},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `deliveryservices/{id}/routing$`, crstats.GetDSRouting, auth.PrivLevelReadOnly, Authenticated, nil, 467339833},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `steering/{deliveryservice}/targets/?$`, api.ReadHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelReadOnly, Authenticated, nil, 45696078243},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `steering/{deliveryservice}/targets/?$`, api.CreateHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 43382163973},
		{api.Version{Major: 4, Minor: 0}, http.MethodPut, `steering/{deliveryservice}/targets/{target}/?$`, api.UpdateHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 44386082953},
		{api.Version{Major: 4, Minor: 0}, http.MethodDelete, `steering/{deliveryservice}/targets/{target}/?$`, api.DeleteHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 42880215153},

		// Stats Summary
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `stats_summary/?$`, trafficstats.GetStatsSummary, auth.PrivLevelReadOnly, Authenticated, nil, 4804985983},
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `stats_summary/?$`, trafficstats.CreateStatsSummary, auth.PrivLevelReadOnly, Authenticated, nil, 4804915983},

		//Pattern based consistent hashing endpoint
		{api.Version{Major: 4, Minor: 0}, http.MethodPost, `consistenthash/?$`, consistenthash.Post, auth.PrivLevelReadOnly, Authenticated, nil, 4607550763},

		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `steering/?$`, steering.Get, auth.PrivLevelSteering, Authenticated, nil, 41748524573},

		// Plugins
		{api.Version{Major: 4, Minor: 0}, http.MethodGet, `plugins/?$`, plugins.Get(d.Plugins), auth.PrivLevelReadOnly, Authenticated, nil, 4834985393},

		/**
		 * 3.x API
		 */
		////DeliveryServices
		{api.Version{Major: 3, Minor: 1}, http.MethodPost, `deliveryservices/?$`, deliveryservice.CreateV31, auth.PrivLevelOperations, Authenticated, nil, 2064315323},
		{api.Version{Major: 3, Minor: 1}, http.MethodPut, `deliveryservices/{id}/?$`, deliveryservice.UpdateV31, auth.PrivLevelOperations, Authenticated, nil, 27665675673},

		// Acme account information
		{api.Version{Major: 3, Minor: 1}, http.MethodGet, `acme_accounts/?$`, acme.Read, auth.PrivLevelAdmin, Authenticated, nil, 2034390561},
		{api.Version{Major: 3, Minor: 1}, http.MethodPost, `acme_accounts/?$`, acme.Create, auth.PrivLevelAdmin, Authenticated, nil, 2034390562},
		{api.Version{Major: 3, Minor: 1}, http.MethodPut, `acme_accounts/?$`, acme.Update, auth.PrivLevelAdmin, Authenticated, nil, 2034390563},
		{api.Version{Major: 3, Minor: 1}, http.MethodDelete, `acme_accounts/{provider}/{email}?$`, acme.Delete, auth.PrivLevelAdmin, Authenticated, nil, 2034390564},

		// API Capability
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `api_capabilities/?$`, apicapability.GetAPICapabilitiesHandler, auth.PrivLevelReadOnly, Authenticated, nil, 28132065893},

		//ASNs
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `asns/?$`, api.UpdateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 22641723173},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `asns/?$`, api.DeleteHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 202048983},

		//ASN: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `asns/?$`, api.ReadHandler(&asn.TOASNV11{}), auth.PrivLevelReadOnly, Authenticated, nil, 2738777223},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `asns/{id}$`, api.UpdateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 29511986293},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `asns/?$`, api.CreateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 29994921883},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `asns/{id}$`, api.DeleteHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 26725247693},

		// Traffic Stats access
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservice_stats`, trafficstats.GetDSStats, auth.PrivLevelReadOnly, Authenticated, nil, 23195690283},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cache_stats`, trafficstats.GetCacheStats, auth.PrivLevelReadOnly, Authenticated, nil, 24979979063},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `current_stats/?$`, trafficstats.GetCurrentStats, auth.PrivLevelReadOnly, Authenticated, nil, 27854428933},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `caches/stats/?$`, cachesstats.Get, auth.PrivLevelReadOnly, Authenticated, nil, 28132065883},

		//CacheGroup: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cachegroups/?$`, api.ReadHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelReadOnly, Authenticated, nil, 2230791103},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `cachegroups/{id}$`, api.UpdateHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 2129545463},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `cachegroups/?$`, api.CreateHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 229826653},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `cachegroups/{id}$`, api.DeleteHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 2278693653},

		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `cachegroups/{id}/queue_update$`, cachegroup.QueueUpdates, auth.PrivLevelOperations, Authenticated, nil, 20716441103},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `cachegroups/{id}/deliveryservices/?$`, cachegroup.DSPostHandlerV31, auth.PrivLevelOperations, Authenticated, nil, 25202404313},

		//CacheGroup Parameters: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cachegroupparameters/?$`, cachegroupparameter.ReadAllCacheGroupParameters, auth.PrivLevelReadOnly, Authenticated, nil, 2124497243},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `cachegroupparameters/?$`, cachegroupparameter.AddCacheGroupParameters, auth.PrivLevelOperations, Authenticated, nil, 2124497253},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cachegroups/{id}/parameters/?$`, api.ReadHandler(&cachegroupparameter.TOCacheGroupParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 2124497233},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `cachegroupparameters/{cachegroupID}/{parameterId}$`, api.DeleteHandler(&cachegroupparameter.TOCacheGroupParameter{}), auth.PrivLevelOperations, Authenticated, nil, 2124497333},

		//Capabilities
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `capabilities/?$`, capabilities.Read, auth.PrivLevelReadOnly, Authenticated, nil, 20081353},

		//CDN
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/name/{name}/sslkeys/?$`, cdn.GetSSLKeys, auth.PrivLevelAdmin, Authenticated, nil, 22785817723},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/capacity$`, cdn.GetCapacity, auth.PrivLevelReadOnly, Authenticated, nil, 2971852813},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/{name}/health/?$`, cdn.GetNameHealth, auth.PrivLevelReadOnly, Authenticated, nil, 21353481943},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/health/?$`, cdn.GetHealth, auth.PrivLevelReadOnly, Authenticated, nil, 20853811343},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/domains/?$`, cdn.DomainsHandler, auth.PrivLevelReadOnly, Authenticated, nil, 2269025603},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/routing$`, crstats.GetCDNRouting, auth.PrivLevelReadOnly, Authenticated, nil, 267229823},

		//CDN: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `cdns/name/{name}$`, cdn.DeleteName, auth.PrivLevelOperations, Authenticated, nil, 2088049593},

		//CDN: queue updates
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `cdns/{id}/queue_update$`, cdn.Queue, auth.PrivLevelOperations, Authenticated, nil, 2215159803},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `cdns/dnsseckeys/generate?$`, cdn.CreateDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil, 2753363},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `cdns/name/{name}/dnsseckeys?$`, cdn.DeleteDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil, 2711042073},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/name/{name}/dnsseckeys/?$`, cdn.GetDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil, 2790106093},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/dnsseckeys/refresh/?$`, cdn.RefreshDNSSECKeys, auth.PrivLevelOperations, Authenticated, nil, 27719971163},

		//CDN: Monitoring: Traffic Monitor
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/{cdn}/configs/monitoring?$`, crconfig.SnapshotGetMonitoringHandler, auth.PrivLevelReadOnly, Authenticated, nil, 22408478923},

		//Database dumps
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `dbdump/?`, dbdump.DBDump, auth.PrivLevelAdmin, Authenticated, nil, 2240166473},

		//Division: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `divisions/?$`, api.ReadHandler(&division.TODivision{}), auth.PrivLevelReadOnly, Authenticated, nil, 20851815343},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `divisions/{id}$`, api.UpdateHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 2063691403},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `divisions/?$`, api.CreateHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 2537138003},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `divisions/{id}$`, api.DeleteHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 23253822373},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `logs/?$`, logs.Get, auth.PrivLevelReadOnly, Authenticated, nil, 2483405503},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `logs/newcount/?$`, logs.GetNewCount, auth.PrivLevelReadOnly, Authenticated, nil, 24058330123},

		//Content invalidation jobs
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `jobs/?$`, api.ReadHandler(&invalidationjobs.InvalidationJob{}), auth.PrivLevelReadOnly, Authenticated, nil, 29667820413},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `jobs/?$`, invalidationjobs.Delete, auth.PrivLevelPortal, Authenticated, nil, 2167807763},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `jobs/?$`, invalidationjobs.Update, auth.PrivLevelPortal, Authenticated, nil, 2861342263},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `jobs/?`, invalidationjobs.Create, auth.PrivLevelPortal, Authenticated, nil, 204509553},

		//Login
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `user/login/?$`, login.LoginHandler(d.DB, d.Config), 0, NoAuth, nil, 23926708213},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `user/logout/?$`, login.LogoutHandler(d.Config.Secrets[0]), 0, Authenticated, nil, 2434348253},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `user/login/oauth/?$`, login.OauthLoginHandler(d.DB, d.Config), 0, NoAuth, nil, 24158860093},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `user/login/token/?$`, login.TokenLoginHandler(d.DB, d.Config), 0, NoAuth, nil, 2024088413},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `user/reset_password/?$`, login.ResetPassword(d.DB, d.Config), 0, NoAuth, nil, 22929146303},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `users/register/?$`, login.RegisterUser, auth.PrivLevelOperations, Authenticated, nil, 23373},

		//ISO
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `osversions/?$`, iso.GetOSVersions, auth.PrivLevelReadOnly, Authenticated, nil, 2760886573},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `isos/?$`, iso.ISOs, auth.PrivLevelOperations, Authenticated, nil, 2760336573},

		//User: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `users/?$`, api.ReadHandler(&user.TOUser{}), auth.PrivLevelReadOnly, Authenticated, nil, 24919299003},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `users/{id}$`, api.ReadHandler(&user.TOUser{}), auth.PrivLevelReadOnly, Authenticated, nil, 2138099803},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `users/{id}$`, api.UpdateHandler(&user.TOUser{}), auth.PrivLevelOperations, Authenticated, nil, 2354334043},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `users/?$`, api.CreateHandler(&user.TOUser{}), auth.PrivLevelOperations, Authenticated, nil, 2762448163},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `user/current/?$`, user.Current, auth.PrivLevelReadOnly, Authenticated, nil, 26107016143},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `user/current/?$`, user.ReplaceCurrent, auth.PrivLevelReadOnly, Authenticated, nil, 2203},

		//Parameter: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `parameters/?$`, api.ReadHandler(&parameter.TOParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 22125542923},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `parameters/{id}$`, api.UpdateHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 28739361153},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `parameters/?$`, api.CreateHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 26695108593},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `parameters/{id}$`, api.DeleteHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 2262771183},

		//Phys_Location: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `phys_locations/?$`, api.ReadHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelReadOnly, Authenticated, nil, 2204051823},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `phys_locations/{id}$`, api.UpdateHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 2227950213},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `phys_locations/?$`, api.CreateHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 22464566483},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `phys_locations/{id}$`, api.DeleteHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 256142213},

		//Ping
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `ping$`, ping.Handler, 0, NoAuth, nil, 25556615973},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `vault/ping/?$`, ping.Vault, auth.PrivLevelReadOnly, Authenticated, nil, 28840121143},

		//Profile: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `profiles/?$`, api.ReadHandler(&profile.TOProfile{}), auth.PrivLevelReadOnly, Authenticated, nil, 2687585893},

		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `profiles/{id}$`, api.UpdateHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 284391723},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `profiles/?$`, api.CreateHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 25402115563},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `profiles/{id}$`, api.DeleteHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 22055944653},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `profiles/{id}/export/?$`, profile.ExportProfileHandler, auth.PrivLevelReadOnly, Authenticated, nil, 201335173},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `profiles/import/?$`, profile.ImportProfileHandler, auth.PrivLevelOperations, Authenticated, nil, 2061432083},

		// Copy Profile
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `profiles/name/{new_profile}/copy/{existing_profile}`, profile.CopyProfileHandler, auth.PrivLevelOperations, Authenticated, nil, 2061432093},

		//Region: CRUDs
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `regions/?$`, api.ReadHandler(&region.TORegion{}), auth.PrivLevelReadOnly, Authenticated, nil, 2100370853},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `regions/{id}$`, api.UpdateHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 2223082243},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `regions/?$`, api.CreateHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 22883344883},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `regions/?$`, api.DeleteHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 22326267583},

		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `topologies/?$`, api.CreateHandler(&topology.TOTopology{}), auth.PrivLevelOperations, Authenticated, nil, 3871452221},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `topologies/?$`, api.ReadHandler(&topology.TOTopology{}), auth.PrivLevelReadOnly, Authenticated, nil, 3871452222},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `topologies/?$`, api.UpdateHandler(&topology.TOTopology{}), auth.PrivLevelOperations, Authenticated, nil, 3871452223},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `topologies/?$`, api.DeleteHandler(&topology.TOTopology{}), auth.PrivLevelOperations, Authenticated, nil, 3871452224},

		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `topologies/{name}/queue_update$`, topology.QueueUpdateHandler, auth.PrivLevelOperations, Authenticated, nil, 3205351748},

		// get all edge servers associated with a delivery service (from deliveryservice_server table)

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryserviceserver/?$`, dsserver.ReadDSSHandlerV14, auth.PrivLevelReadOnly, Authenticated, nil, 29461450333},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryserviceserver$`, dsserver.GetReplaceHandler, auth.PrivLevelOperations, Authenticated, nil, 2297997883},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `deliveryserviceserver/{dsid}/{serverid}`, dsserver.Delete, auth.PrivLevelOperations, Authenticated, nil, 25321845233},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservices/{xml_id}/servers$`, dsserver.GetCreateHandler, auth.PrivLevelOperations, Authenticated, nil, 24281812063},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `servers/{id}/deliveryservices$`, api.ReadHandler(&dsserver.TODSSDeliveryService{}), auth.PrivLevelReadOnly, Authenticated, nil, 2331154113},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `servers/{id}/deliveryservices$`, server.AssignDeliveryServicesToServerHandler, auth.PrivLevelOperations, Authenticated, nil, 2801282533},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices/{id}/servers$`, dsserver.GetReadAssigned, auth.PrivLevelReadOnly, Authenticated, nil, 23451212233},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservices/request`, deliveryservicerequests.Request, auth.PrivLevelPortal, Authenticated, nil, 2408752993},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices/{id}/capacity/?$`, deliveryservice.GetCapacity, auth.PrivLevelReadOnly, Authenticated, nil, 22314091103},
		//Serverchecks
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `servercheck/?$`, servercheck.ReadServerCheck, auth.PrivLevelReadOnly, Authenticated, nil, 27961129223},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `servercheck/?$`, servercheck.CreateUpdateServercheck, auth.PrivLevelInvalid, Authenticated, nil, 27642815683},

		// Servercheck Extensions
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `servercheck/extensions$`, extensions.Create, auth.PrivLevelReadOnly, Authenticated, nil, 2804985993},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `servercheck/extensions$`, extensions.Get, auth.PrivLevelReadOnly, Authenticated, nil, 2834985993},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `servercheck/extensions/{id}$`, extensions.Delete, auth.PrivLevelReadOnly, Authenticated, nil, 2804982993},

		//Server Details
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `servers/details/?$`, server.GetDetailParamHandler, auth.PrivLevelReadOnly, Authenticated, nil, 22612647143},

		//Server status
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `servers/{id}/status$`, server.UpdateStatusHandler, auth.PrivLevelOperations, Authenticated, nil, 2766638513},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `servers/{id}/queue_update$`, server.QueueUpdateHandler, auth.PrivLevelOperations, Authenticated, nil, 21894713},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `servers/{host_name}/update_status$`, server.GetServerUpdateStatusHandler, auth.PrivLevelReadOnly, Authenticated, nil, 2384515993},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `servers/{id-or-name}/update$`, server.UpdateHandler, auth.PrivLevelOperations, Authenticated, nil, 143813233},

		//Server: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `servers/?$`, server.Read, auth.PrivLevelReadOnly, Authenticated, nil, 27209592853},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `servers/{id}$`, server.Update, auth.PrivLevelOperations, Authenticated, nil, 2586341033},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `servers/?$`, server.Create, auth.PrivLevelOperations, Authenticated, nil, 22255580613},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `servers/{id}$`, server.Delete, auth.PrivLevelOperations, Authenticated, nil, 2923222333},

		//Server Capability
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `server_capabilities$`, api.ReadHandler(&servercapability.TOServerCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 2104073913},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `server_capabilities$`, api.CreateHandler(&servercapability.TOServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 20744707083},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `server_capabilities$`, api.DeleteHandler(&servercapability.TOServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 2364150383},

		//Server Server Capabilities: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `server_server_capabilities/?$`, api.ReadHandler(&server.TOServerServerCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 28002318893},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `server_server_capabilities/?$`, api.CreateHandler(&server.TOServerServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 22931668343},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `server_server_capabilities/?$`, api.DeleteHandler(&server.TOServerServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 20587140583},

		//Status: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `statuses/?$`, api.ReadHandler(&status.TOStatus{}), auth.PrivLevelReadOnly, Authenticated, nil, 22449056563},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `statuses/{id}$`, api.UpdateHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 22079665043},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `statuses/?$`, api.CreateHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 23691236123},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `statuses/{id}$`, api.DeleteHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 2551113603},

		//System
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `system/info/?$`, systeminfo.Get, auth.PrivLevelReadOnly, Authenticated, nil, 2210474753},

		//Type: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `types/?$`, api.ReadHandler(&types.TOType{}), auth.PrivLevelReadOnly, Authenticated, nil, 22267018233},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `types/{id}$`, api.UpdateHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 288601153},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `types/?$`, api.CreateHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 25133081953},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `types/{id}$`, api.DeleteHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 231757733},

		//About
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `about/?$`, about.Handler(), auth.PrivLevelReadOnly, Authenticated, nil, 23175011663},

		//Coordinates
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `coordinates/?$`, api.ReadHandler(&coordinate.TOCoordinate{}), auth.PrivLevelReadOnly, Authenticated, nil, 2967007453},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `coordinates/?$`, api.UpdateHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 2689261743},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `coordinates/?$`, api.CreateHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 24281121573},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `coordinates/?$`, api.DeleteHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 23038498893},

		//CDN generic handlers:
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/?$`, api.ReadHandler(&cdn.TOCDN{}), auth.PrivLevelReadOnly, Authenticated, nil, 22303186213},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `cdns/{id}$`, api.UpdateHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 23111789343},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `cdns/?$`, api.CreateHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 21605052893},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `cdns/{id}$`, api.DeleteHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 2276946573},

		//Delivery service requests
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservice_requests/?$`, dsrequest.Get, auth.PrivLevelReadOnly, Authenticated, nil, 26811639353},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `deliveryservice_requests/?$`, dsrequest.Put, auth.PrivLevelPortal, Authenticated, nil, 22499079183},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservice_requests/?$`, dsrequest.Post, auth.PrivLevelPortal, Authenticated, nil, 293850393},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `deliveryservice_requests/?$`, dsrequest.Delete, auth.PrivLevelPortal, Authenticated, nil, 22969850253},

		//Delivery service request: Actions
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `deliveryservice_requests/{id}/assign$`, dsrequest.PutAssignment, auth.PrivLevelOperations, Authenticated, nil, 27031602903},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `deliveryservice_requests/{id}/status$`, dsrequest.PutStatus, auth.PrivLevelPortal, Authenticated, nil, 2684150993},

		//Delivery service request comment: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservice_request_comments/?$`, api.ReadHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelReadOnly, Authenticated, nil, 20326507373},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `deliveryservice_request_comments/?$`, api.UpdateHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 2604878473},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservice_request_comments/?$`, api.CreateHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 2272276723},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `deliveryservice_request_comments/?$`, api.DeleteHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 2995046683},

		//Delivery service uri signing keys: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.GetURIsignkeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 22930785583},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.SaveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 2084663353},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.SaveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 276489693},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.RemoveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 2299254173},

		//Delivery Service Required Capabilities: CRUD
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices_required_capabilities/?$`, api.ReadHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 21585222273},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservices_required_capabilities/?$`, api.CreateHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelOperations, Authenticated, nil, 20968739923},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `deliveryservices_required_capabilities/?$`, api.DeleteHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelOperations, Authenticated, nil, 24962893043},

		// Federations by CDN (the actual table for federation)
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/{name}/federations/?$`, api.ReadHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelReadOnly, Authenticated, nil, 2892250323},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `cdns/{name}/federations/?$`, api.CreateHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 29548942193},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `cdns/{name}/federations/{id}$`, api.UpdateHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 2260654663},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `cdns/{name}/federations/{id}$`, api.DeleteHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 24428529023},

		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `cdns/{name}/dnsseckeys/ksk/generate$`, cdn.GenerateKSK, auth.PrivLevelAdmin, Authenticated, nil, 2729242813},

		//Origins
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `origins/?$`, api.ReadHandler(&origin.TOOrigin{}), auth.PrivLevelReadOnly, Authenticated, nil, 2446492563},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `origins/?$`, api.UpdateHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 215677463},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `origins/?$`, api.CreateHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 20995616433},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `origins/?$`, api.DeleteHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 2602732633},

		//Roles
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `roles/?$`, api.ReadHandler(&role.TORole{}), auth.PrivLevelReadOnly, Authenticated, nil, 2870885833},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `roles/?$`, api.UpdateHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 26128974893},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `roles/?$`, api.CreateHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 2306524063},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `roles/?$`, api.DeleteHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 23567059823},

		//Delivery Services Regexes
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices_regexes/?$`, deliveryservicesregexes.Get, auth.PrivLevelReadOnly, Authenticated, nil, 2055014533},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices/{dsid}/regexes/?$`, deliveryservicesregexes.DSGet, auth.PrivLevelReadOnly, Authenticated, nil, 2774327633},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservices/{dsid}/regexes/?$`, deliveryservicesregexes.Post, auth.PrivLevelOperations, Authenticated, nil, 2127378003},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `deliveryservices/{dsid}/regexes/{regexid}?$`, deliveryservicesregexes.Put, auth.PrivLevelOperations, Authenticated, nil, 22483396913},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `deliveryservices/{dsid}/regexes/{regexid}?$`, deliveryservicesregexes.Delete, auth.PrivLevelOperations, Authenticated, nil, 22467316633},

		//ServiceCategories
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `service_categories/?$`, api.ReadHandler(&servicecategory.TOServiceCategory{}), auth.PrivLevelReadOnly, Authenticated, nil, 1085181543},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `service_categories/{name}/?$`, servicecategory.Update, auth.PrivLevelOperations, Authenticated, nil, 306369141},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `service_categories/?$`, api.CreateHandler(&servicecategory.TOServiceCategory{}), auth.PrivLevelOperations, Authenticated, nil, 553713801},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `service_categories/{name}$`, api.DeleteHandler(&servicecategory.TOServiceCategory{}), auth.PrivLevelOperations, Authenticated, nil, 1325382238},

		//StaticDNSEntries
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `staticdnsentries/?$`, api.ReadHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelReadOnly, Authenticated, nil, 2289394773},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `staticdnsentries/?$`, api.UpdateHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 2424571113},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `staticdnsentries/?$`, api.CreateHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 26291482383},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `staticdnsentries/?$`, api.DeleteHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 28460311323},

		//ProfileParameters
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `profiles/{id}/parameters/?$`, profileparameter.GetProfileID, auth.PrivLevelReadOnly, Authenticated, nil, 2764649753},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `profiles/name/{name}/parameters/?$`, profileparameter.GetProfileName, auth.PrivLevelReadOnly, Authenticated, nil, 22677378323},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `profiles/name/{name}/parameters/?$`, profileparameter.PostProfileParamsByName, auth.PrivLevelOperations, Authenticated, nil, 23559455823},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `profiles/{id}/parameters/?$`, profileparameter.PostProfileParamsByID, auth.PrivLevelOperations, Authenticated, nil, 2168187083},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `profileparameters/?$`, api.ReadHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 2506098053},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `profileparameters/?$`, api.CreateHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelOperations, Authenticated, nil, 2288096933},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `profileparameter/?$`, profileparameter.PostProfileParam, auth.PrivLevelOperations, Authenticated, nil, 2242753},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `parameterprofile/?$`, profileparameter.PostParamProfile, auth.PrivLevelOperations, Authenticated, nil, 20806108613},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `profileparameters/{profileId}/{parameterId}$`, api.DeleteHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelOperations, Authenticated, nil, 2248395293},

		//Tenants
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `tenants/?$`, api.ReadHandler(&apitenant.TOTenant{}), auth.PrivLevelReadOnly, Authenticated, nil, 26779678143},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `tenants/{id}$`, api.UpdateHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 20941314783},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `tenants/?$`, api.CreateHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 2172480133},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `tenants/{id}$`, api.DeleteHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 2163655583},

		//CRConfig
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/{cdn}/snapshot/?$`, crconfig.SnapshotGetHandler, auth.PrivLevelReadOnly, Authenticated, nil, 29572736953},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `cdns/{cdn}/snapshot/new/?$`, crconfig.Handler, auth.PrivLevelReadOnly, Authenticated, nil, 2767168893},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `snapshot/?$`, crconfig.SnapshotHandler, auth.PrivLevelOperations, Authenticated, nil, 29699118293},

		// Federations
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `federations/all/?$`, federations.GetAll, auth.PrivLevelAdmin, Authenticated, nil, 210599863},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `federations/?$`, federations.Get, auth.PrivLevelFederation, Authenticated, nil, 2549549943},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `federations/?$`, federations.AddFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 28940647423},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `federations/?$`, federations.RemoveFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 220983233},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `federations/?$`, federations.ReplaceFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 22831825163},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `federations/{id}/deliveryservices/?$`, federations.PostDSes, auth.PrivLevelAdmin, Authenticated, nil, 26828635133},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `federations/{id}/deliveryservices/?$`, api.ReadHandler(&federations.TOFedDSes{}), auth.PrivLevelReadOnly, Authenticated, nil, 2537730343},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `federations/{id}/deliveryservices/{dsID}/?$`, api.DeleteHandler(&federations.TOFedDSes{}), auth.PrivLevelAdmin, Authenticated, nil, 24174025703},

		// Federation Resolvers
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `federation_resolvers/?$`, federation_resolvers.Create, auth.PrivLevelAdmin, Authenticated, nil, 21343736613},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `federation_resolvers/?$`, federation_resolvers.Read, auth.PrivLevelReadOnly, Authenticated, nil, 2566087593},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `federations/{id}/federation_resolvers/?$`, federations.AssignFederationResolversToFederationHandler, auth.PrivLevelAdmin, Authenticated, nil, 2566087603},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `federations/{id}/federation_resolvers/?$`, federations.GetFederationFederationResolversHandler, auth.PrivLevelReadOnly, Authenticated, nil, 2566087613},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `federation_resolvers/?$`, federation_resolvers.Delete, auth.PrivLevelAdmin, Authenticated, nil, 20013},

		// Federations Users
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `federations/{id}/users/?$`, federations.PostUsers, auth.PrivLevelAdmin, Authenticated, nil, 27793349303},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `federations/{id}/users/?$`, api.ReadHandler(&federations.TOUsers{}), auth.PrivLevelReadOnly, Authenticated, nil, 2940750153},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `federations/{id}/users/{userID}/?$`, api.DeleteHandler(&federations.TOUsers{}), auth.PrivLevelAdmin, Authenticated, nil, 29491028823},

		////DeliveryServices
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices/?$`, api.ReadHandler(&deliveryservice.TODeliveryService{}), auth.PrivLevelReadOnly, Authenticated, nil, 22383172943},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservices/?$`, deliveryservice.CreateV30, auth.PrivLevelOperations, Authenticated, nil, 2064314323},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `deliveryservices/{id}/?$`, deliveryservice.UpdateV30, auth.PrivLevelOperations, Authenticated, nil, 27665675273},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `deliveryservices/{id}/safe/?$`, deliveryservice.UpdateSafe, auth.PrivLevelOperations, Authenticated, nil, 2472109313},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `deliveryservices/{id}/?$`, api.DeleteHandler(&deliveryservice.TODeliveryService{}), auth.PrivLevelOperations, Authenticated, nil, 2226420743},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices/{id}/servers/eligible/?$`, deliveryservice.GetServersEligible, auth.PrivLevelReadOnly, Authenticated, nil, 2747615843},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices/xmlId/{xmlid}/sslkeys$`, deliveryservice.GetSSLKeysByXMLIDV15, auth.PrivLevelAdmin, Authenticated, nil, 21357729073},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservices/sslkeys/add$`, deliveryservice.AddSSLKeys, auth.PrivLevelAdmin, Authenticated, nil, 28728785833},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `deliveryservices/xmlId/{xmlid}/sslkeys$`, deliveryservice.DeleteSSLKeys, auth.PrivLevelOperations, Authenticated, nil, 29267343},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservices/sslkeys/generate/?$`, deliveryservice.GenerateSSLKeys, auth.PrivLevelOperations, Authenticated, nil, 2534390513},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/copyFromXmlId/{copy-name}/?$`, deliveryservice.CopyURLKeys, auth.PrivLevelOperations, Authenticated, nil, 22625010763},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/generate/?$`, deliveryservice.GenerateURLKeys, auth.PrivLevelOperations, Authenticated, nil, 25304828243},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices/xmlId/{name}/urlkeys/?$`, deliveryservice.GetURLKeysByName, auth.PrivLevelReadOnly, Authenticated, nil, 22027192113},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices/{id}/urlkeys/?$`, deliveryservice.GetURLKeysByID, auth.PrivLevelReadOnly, Authenticated, nil, 2931971143},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `vault/bucket/{bucket}/key/{key}/values/?$`, vault.GetBucketKeyDeprecated, auth.PrivLevelAdmin, Authenticated, nil, 22205108013},

		//Delivery service LetsEncrypt
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `deliveryservices/sslkeys/generate/letsencrypt/?$`, deliveryservice.GenerateLetsEncryptCertificates, auth.PrivLevelOperations, Authenticated, nil, 2534390523},
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `letsencrypt/dnsrecords/?$`, deliveryservice.GetDnsChallengeRecords, auth.PrivLevelOperations, Authenticated, nil, 2534390553},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `letsencrypt/autorenew/?$`, deliveryservice.RenewCertificates, auth.PrivLevelOperations, Authenticated, nil, 2534390563},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices/{id}/health/?$`, deliveryservice.GetHealth, auth.PrivLevelReadOnly, Authenticated, nil, 22345901013},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `deliveryservices/{id}/routing$`, crstats.GetDSRouting, auth.PrivLevelReadOnly, Authenticated, nil, 667339833},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `steering/{deliveryservice}/targets/?$`, api.ReadHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelReadOnly, Authenticated, nil, 25696078243},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `steering/{deliveryservice}/targets/?$`, api.CreateHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 23382163973},
		{api.Version{Major: 3, Minor: 0}, http.MethodPut, `steering/{deliveryservice}/targets/{target}/?$`, api.UpdateHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 24386082953},
		{api.Version{Major: 3, Minor: 0}, http.MethodDelete, `steering/{deliveryservice}/targets/{target}/?$`, api.DeleteHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 22880215153},

		// Stats Summary
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `stats_summary/?$`, trafficstats.GetStatsSummary, auth.PrivLevelReadOnly, Authenticated, nil, 2804985983},
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `stats_summary/?$`, trafficstats.CreateStatsSummary, auth.PrivLevelReadOnly, Authenticated, nil, 2804915983},

		//Pattern based consistent hashing endpoint
		{api.Version{Major: 3, Minor: 0}, http.MethodPost, `consistenthash/?$`, consistenthash.Post, auth.PrivLevelReadOnly, Authenticated, nil, 2607550763},

		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `steering/?$`, steering.Get, auth.PrivLevelSteering, Authenticated, nil, 21748524573},

		// Plugins
		{api.Version{Major: 3, Minor: 0}, http.MethodGet, `plugins/?$`, plugins.Get(d.Plugins), auth.PrivLevelReadOnly, Authenticated, nil, 2834985393},

		/**
		 * 2.x API
		 */
		// API Capability
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `api_capabilities/?$`, apicapability.GetAPICapabilitiesHandler, auth.PrivLevelReadOnly, Authenticated, nil, 2813206589},

		//ASNs
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `asns/?$`, api.UpdateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 2264172317},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `asns/?$`, api.DeleteHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 20204898},

		//ASN: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `asns/?$`, api.ReadHandler(&asn.TOASNV11{}), auth.PrivLevelReadOnly, Authenticated, nil, 273877722},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `asns/{id}$`, api.UpdateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 2951198629},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `asns/?$`, api.CreateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 2999492188},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `asns/{id}$`, api.DeleteHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 2672524769},

		// Traffic Stats access
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservice_stats`, trafficstats.GetDSStats, auth.PrivLevelReadOnly, Authenticated, nil, 2319569028},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cache_stats`, trafficstats.GetCacheStats, auth.PrivLevelReadOnly, Authenticated, nil, 2497997906},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `current_stats/?$`, trafficstats.GetCurrentStats, auth.PrivLevelReadOnly, Authenticated, nil, 2785442893},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `caches/stats/?$`, cachesstats.Get, auth.PrivLevelReadOnly, Authenticated, nil, 2813206588},

		//CacheGroup: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cachegroups/?$`, api.ReadHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelReadOnly, Authenticated, nil, 223079110},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `cachegroups/{id}$`, api.UpdateHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 212954546},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `cachegroups/?$`, api.CreateHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 22982665},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `cachegroups/{id}$`, api.DeleteHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 227869365},

		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `cachegroups/{id}/queue_update$`, cachegroup.QueueUpdates, auth.PrivLevelOperations, Authenticated, nil, 2071644110},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `cachegroups/{id}/deliveryservices/?$`, cachegroup.DSPostHandlerV31, auth.PrivLevelOperations, Authenticated, nil, 2520240431},

		//CacheGroup Parameters: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cachegroupparameters/?$`, cachegroupparameter.ReadAllCacheGroupParameters, auth.PrivLevelReadOnly, Authenticated, nil, 212449724},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `cachegroupparameters/?$`, cachegroupparameter.AddCacheGroupParameters, auth.PrivLevelOperations, Authenticated, nil, 212449725},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cachegroups/{id}/parameters/?$`, api.ReadHandler(&cachegroupparameter.TOCacheGroupParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 212449723},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `cachegroupparameters/{cachegroupID}/{parameterId}$`, api.DeleteHandler(&cachegroupparameter.TOCacheGroupParameter{}), auth.PrivLevelOperations, Authenticated, nil, 212449733},

		//Capabilities
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `capabilities/?$`, capabilities.Read, auth.PrivLevelReadOnly, Authenticated, nil, 2008135},

		//CDN
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/name/{name}/sslkeys/?$`, cdn.GetSSLKeys, auth.PrivLevelAdmin, Authenticated, nil, 2278581772},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/capacity$`, cdn.GetCapacity, auth.PrivLevelReadOnly, Authenticated, nil, 297185281},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/{name}/health/?$`, cdn.GetNameHealth, auth.PrivLevelReadOnly, Authenticated, nil, 2135348194},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/health/?$`, cdn.GetHealth, auth.PrivLevelReadOnly, Authenticated, nil, 2085381134},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/domains/?$`, cdn.DomainsHandler, auth.PrivLevelReadOnly, Authenticated, nil, 226902560},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/routing$`, crstats.GetCDNRouting, auth.PrivLevelReadOnly, Authenticated, nil, 26722982},

		//CDN: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `cdns/name/{name}$`, cdn.DeleteName, auth.PrivLevelOperations, Authenticated, nil, 208804959},

		//CDN: queue updates
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `cdns/{id}/queue_update$`, cdn.Queue, auth.PrivLevelOperations, Authenticated, nil, 221515980},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `cdns/dnsseckeys/generate?$`, cdn.CreateDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil, 275336},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `cdns/name/{name}/dnsseckeys?$`, cdn.DeleteDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil, 271104207},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/name/{name}/dnsseckeys/?$`, cdn.GetDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil, 279010609},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/dnsseckeys/refresh/?$`, cdn.RefreshDNSSECKeys, auth.PrivLevelOperations, Authenticated, nil, 2771997116},

		//CDN: Monitoring: Traffic Monitor
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/{cdn}/configs/monitoring?$`, crconfig.SnapshotGetMonitoringLegacyHandler, auth.PrivLevelReadOnly, Authenticated, nil, 2240847892},

		//Database dumps
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `dbdump/?`, dbdump.DBDump, auth.PrivLevelAdmin, Authenticated, nil, 224016647},

		//Division: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `divisions/?$`, api.ReadHandler(&division.TODivision{}), auth.PrivLevelReadOnly, Authenticated, nil, 2085181534},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `divisions/{id}$`, api.UpdateHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 206369140},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `divisions/?$`, api.CreateHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 253713800},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `divisions/{id}$`, api.DeleteHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 2325382237},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `logs/?$`, logs.Get, auth.PrivLevelReadOnly, Authenticated, nil, 248340550},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `logs/newcount/?$`, logs.GetNewCount, auth.PrivLevelReadOnly, Authenticated, nil, 2405833012},

		//Content invalidation jobs
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `jobs/?$`, api.ReadHandler(&invalidationjobs.InvalidationJob{}), auth.PrivLevelReadOnly, Authenticated, nil, 2966782041},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `jobs/?$`, invalidationjobs.Delete, auth.PrivLevelPortal, Authenticated, nil, 216780776},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `jobs/?$`, invalidationjobs.Update, auth.PrivLevelPortal, Authenticated, nil, 286134226},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `jobs/?`, invalidationjobs.Create, auth.PrivLevelPortal, Authenticated, nil, 20450955},

		//Login
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `user/login/?$`, login.LoginHandler(d.DB, d.Config), 0, NoAuth, nil, 2392670821},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `user/logout/?$`, login.LogoutHandler(d.Config.Secrets[0]), 0, Authenticated, nil, 243434825},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `user/login/oauth/?$`, login.OauthLoginHandler(d.DB, d.Config), 0, NoAuth, nil, 2415886009},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `user/login/token/?$`, login.TokenLoginHandler(d.DB, d.Config), 0, NoAuth, nil, 202408841},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `user/reset_password/?$`, login.ResetPassword(d.DB, d.Config), 0, NoAuth, nil, 2292914630},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `users/register/?$`, login.RegisterUser, auth.PrivLevelOperations, Authenticated, nil, 2337},

		//ISO
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `osversions/?$`, iso.GetOSVersions, auth.PrivLevelReadOnly, Authenticated, nil, 276088657},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `isos/?$`, iso.ISOs, auth.PrivLevelOperations, Authenticated, nil, 276033657},

		//User: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `users/?$`, api.ReadHandler(&user.TOUser{}), auth.PrivLevelReadOnly, Authenticated, nil, 2491929900},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `users/{id}$`, api.ReadHandler(&user.TOUser{}), auth.PrivLevelReadOnly, Authenticated, nil, 213809980},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `users/{id}$`, api.UpdateHandler(&user.TOUser{}), auth.PrivLevelOperations, Authenticated, nil, 235433404},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `users/?$`, api.CreateHandler(&user.TOUser{}), auth.PrivLevelOperations, Authenticated, nil, 276244816},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `user/current/?$`, user.Current, auth.PrivLevelReadOnly, Authenticated, nil, 2610701614},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `user/current/?$`, user.ReplaceCurrent, auth.PrivLevelReadOnly, Authenticated, nil, 220},

		//Parameter: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `parameters/?$`, api.ReadHandler(&parameter.TOParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 2212554292},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `parameters/{id}$`, api.UpdateHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 2873936115},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `parameters/?$`, api.CreateHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 2669510859},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `parameters/{id}$`, api.DeleteHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 226277118},

		//Phys_Location: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `phys_locations/?$`, api.ReadHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelReadOnly, Authenticated, nil, 220405182},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `phys_locations/{id}$`, api.UpdateHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 222795021},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `phys_locations/?$`, api.CreateHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 2246456648},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `phys_locations/{id}$`, api.DeleteHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 25614221},

		//Ping
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `ping$`, ping.Handler, 0, NoAuth, nil, 2555661597},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `vault/ping/?$`, ping.Vault, auth.PrivLevelReadOnly, Authenticated, nil, 2884012114},

		//Profile: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `profiles/?$`, api.ReadHandler(&profile.TOProfile{}), auth.PrivLevelReadOnly, Authenticated, nil, 268758589},

		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `profiles/{id}$`, api.UpdateHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 28439172},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `profiles/?$`, api.CreateHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 2540211556},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `profiles/{id}$`, api.DeleteHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 2205594465},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `profiles/{id}/export/?$`, profile.ExportProfileHandler, auth.PrivLevelReadOnly, Authenticated, nil, 20133517},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `profiles/import/?$`, profile.ImportProfileHandler, auth.PrivLevelOperations, Authenticated, nil, 206143208},

		// Copy Profile
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `profiles/name/{new_profile}/copy/{existing_profile}`, profile.CopyProfileHandler, auth.PrivLevelOperations, Authenticated, nil, 206143209},

		//Region: CRUDs
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `regions/?$`, api.ReadHandler(&region.TORegion{}), auth.PrivLevelReadOnly, Authenticated, nil, 210037085},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `regions/{id}$`, api.UpdateHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 222308224},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `regions/?$`, api.CreateHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 2288334488},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `regions/?$`, api.DeleteHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 2232626758},

		// get all edge servers associated with a delivery service (from deliveryservice_server table)

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryserviceserver/?$`, dsserver.ReadDSSHandlerV14, auth.PrivLevelReadOnly, Authenticated, nil, 2946145033},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryserviceserver$`, dsserver.GetReplaceHandler, auth.PrivLevelOperations, Authenticated, nil, 229799788},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `deliveryserviceserver/{dsid}/{serverid}`, dsserver.Delete, auth.PrivLevelOperations, Authenticated, nil, 2532184523},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservices/{xml_id}/servers$`, dsserver.GetCreateHandler, auth.PrivLevelOperations, Authenticated, nil, 2428181206},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `servers/{id}/deliveryservices$`, api.ReadHandler(&dsserver.TODSSDeliveryService{}), auth.PrivLevelReadOnly, Authenticated, nil, 233115411},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `servers/{id}/deliveryservices$`, server.AssignDeliveryServicesToServerHandler, auth.PrivLevelOperations, Authenticated, nil, 280128253},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices/{id}/servers$`, dsserver.GetReadAssigned, auth.PrivLevelReadOnly, Authenticated, nil, 2345121223},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservices/request`, deliveryservicerequests.Request, auth.PrivLevelPortal, Authenticated, nil, 240875299},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices/{id}/capacity/?$`, deliveryservice.GetCapacity, auth.PrivLevelReadOnly, Authenticated, nil, 2231409110},
		//Serverchecks
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `servercheck/?$`, servercheck.ReadServerCheck, auth.PrivLevelReadOnly, Authenticated, nil, 2796112922},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `servercheck/?$`, servercheck.CreateUpdateServercheck, auth.PrivLevelInvalid, Authenticated, nil, 2764281568},

		// Servercheck Extensions
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `servercheck/extensions$`, extensions.Create, auth.PrivLevelReadOnly, Authenticated, nil, 280498599},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `servercheck/extensions$`, extensions.Get, auth.PrivLevelReadOnly, Authenticated, nil, 283498599},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `servercheck/extensions/{id}$`, extensions.Delete, auth.PrivLevelReadOnly, Authenticated, nil, 280498299},

		//Server Details
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `servers/details/?$`, server.GetDetailParamHandler, auth.PrivLevelReadOnly, Authenticated, nil, 2261264714},

		//Server status
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `servers/{id}/status$`, server.UpdateStatusHandler, auth.PrivLevelOperations, Authenticated, nil, 276663851},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `servers/{id}/queue_update$`, server.QueueUpdateHandler, auth.PrivLevelOperations, Authenticated, nil, 2189471},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `servers/{host_name}/update_status$`, server.GetServerUpdateStatusHandler, auth.PrivLevelReadOnly, Authenticated, nil, 238451599},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `servers/{id-or-name}/update$`, server.UpdateHandler, auth.PrivLevelOperations, Authenticated, nil, 14381323},

		//Server: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `servers/?$`, server.Read, auth.PrivLevelReadOnly, Authenticated, nil, 2720959285},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `servers/{id}$`, server.Update, auth.PrivLevelOperations, Authenticated, nil, 258634103},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `servers/?$`, server.Create, auth.PrivLevelOperations, Authenticated, nil, 2225558061},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `servers/{id}$`, server.Delete, auth.PrivLevelOperations, Authenticated, nil, 292322233},

		//Server Capability
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `server_capabilities$`, api.ReadHandler(&servercapability.TOServerCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 210407391},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `server_capabilities$`, api.CreateHandler(&servercapability.TOServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 2074470708},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `server_capabilities$`, api.DeleteHandler(&servercapability.TOServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 236415038},

		//Server Server Capabilities: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `server_server_capabilities/?$`, api.ReadHandler(&server.TOServerServerCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 2800231889},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `server_server_capabilities/?$`, api.CreateHandler(&server.TOServerServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 2293166834},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `server_server_capabilities/?$`, api.DeleteHandler(&server.TOServerServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 2058714058},

		//Status: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `statuses/?$`, api.ReadHandler(&status.TOStatus{}), auth.PrivLevelReadOnly, Authenticated, nil, 2244905656},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `statuses/{id}$`, api.UpdateHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 2207966504},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `statuses/?$`, api.CreateHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 2369123612},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `statuses/{id}$`, api.DeleteHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 255111360},

		//System
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `system/info/?$`, systeminfo.Get, auth.PrivLevelReadOnly, Authenticated, nil, 221047475},

		//Type: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `types/?$`, api.ReadHandler(&types.TOType{}), auth.PrivLevelReadOnly, Authenticated, nil, 2226701823},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `types/{id}$`, api.UpdateHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 28860115},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `types/?$`, api.CreateHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 2513308195},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `types/{id}$`, api.DeleteHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 23175773},

		//About
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `about/?$`, about.Handler(), auth.PrivLevelReadOnly, Authenticated, nil, 2317501166},

		//Coordinates
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `coordinates/?$`, api.ReadHandler(&coordinate.TOCoordinate{}), auth.PrivLevelReadOnly, Authenticated, nil, 296700745},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `coordinates/?$`, api.UpdateHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 268926174},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `coordinates/?$`, api.CreateHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 2428112157},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `coordinates/?$`, api.DeleteHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 2303849889},

		//CDN generic handlers:
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/?$`, api.ReadHandler(&cdn.TOCDN{}), auth.PrivLevelReadOnly, Authenticated, nil, 2230318621},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `cdns/{id}$`, api.UpdateHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 2311178934},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `cdns/?$`, api.CreateHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 2160505289},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `cdns/{id}$`, api.DeleteHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 227694657},

		//Delivery service requests
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservice_requests/?$`, dsrequest.Get, auth.PrivLevelReadOnly, Authenticated, nil, 2681163935},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `deliveryservice_requests/?$`, dsrequest.Put, auth.PrivLevelPortal, Authenticated, nil, 2249907918},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservice_requests/?$`, dsrequest.Post, auth.PrivLevelPortal, Authenticated, nil, 29385039},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `deliveryservice_requests/?$`, dsrequest.Delete, auth.PrivLevelPortal, Authenticated, nil, 2296985025},

		//Delivery service request: Actions
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `deliveryservice_requests/{id}/assign$`, dsrequest.PutAssignment, auth.PrivLevelOperations, Authenticated, nil, 2703160290},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `deliveryservice_requests/{id}/status$`, dsrequest.PutStatus, auth.PrivLevelPortal, Authenticated, nil, 268415099},

		//Delivery service request comment: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservice_request_comments/?$`, api.ReadHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelReadOnly, Authenticated, nil, 2032650737},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `deliveryservice_request_comments/?$`, api.UpdateHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 260487847},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservice_request_comments/?$`, api.CreateHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 227227672},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `deliveryservice_request_comments/?$`, api.DeleteHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 299504668},

		//Delivery service uri signing keys: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.GetURIsignkeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 2293078558},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.SaveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 208466335},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.SaveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 27648969},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.RemoveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 229925417},

		//Delivery Service Required Capabilities: CRUD
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices_required_capabilities/?$`, api.ReadHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 2158522227},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservices_required_capabilities/?$`, api.CreateHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelOperations, Authenticated, nil, 2096873992},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `deliveryservices_required_capabilities/?$`, api.DeleteHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelOperations, Authenticated, nil, 2496289304},

		// Federations by CDN (the actual table for federation)
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/{name}/federations/?$`, api.ReadHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelReadOnly, Authenticated, nil, 289225032},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `cdns/{name}/federations/?$`, api.CreateHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 2954894219},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `cdns/{name}/federations/{id}$`, api.UpdateHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 226065466},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `cdns/{name}/federations/{id}$`, api.DeleteHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 2442852902},

		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `cdns/{name}/dnsseckeys/ksk/generate$`, cdn.GenerateKSK, auth.PrivLevelAdmin, Authenticated, nil, 272924281},

		//Origins
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `origins/?$`, api.ReadHandler(&origin.TOOrigin{}), auth.PrivLevelReadOnly, Authenticated, nil, 244649256},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `origins/?$`, api.UpdateHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 21567746},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `origins/?$`, api.CreateHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 2099561643},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `origins/?$`, api.DeleteHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 260273263},

		//Roles
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `roles/?$`, api.ReadHandler(&role.TORole{}), auth.PrivLevelReadOnly, Authenticated, nil, 287088583},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `roles/?$`, api.UpdateHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 2612897489},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `roles/?$`, api.CreateHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 230652406},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `roles/?$`, api.DeleteHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 2356705982},

		//Delivery Services Regexes
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices_regexes/?$`, deliveryservicesregexes.Get, auth.PrivLevelReadOnly, Authenticated, nil, 205501453},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices/{dsid}/regexes/?$`, deliveryservicesregexes.DSGet, auth.PrivLevelReadOnly, Authenticated, nil, 277432763},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservices/{dsid}/regexes/?$`, deliveryservicesregexes.Post, auth.PrivLevelOperations, Authenticated, nil, 212737800},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `deliveryservices/{dsid}/regexes/{regexid}?$`, deliveryservicesregexes.Put, auth.PrivLevelOperations, Authenticated, nil, 2248339691},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `deliveryservices/{dsid}/regexes/{regexid}?$`, deliveryservicesregexes.Delete, auth.PrivLevelOperations, Authenticated, nil, 2246731663},

		//StaticDNSEntries
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `staticdnsentries/?$`, api.ReadHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelReadOnly, Authenticated, nil, 228939477},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `staticdnsentries/?$`, api.UpdateHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 242457111},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `staticdnsentries/?$`, api.CreateHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 2629148238},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `staticdnsentries/?$`, api.DeleteHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 2846031132},

		//ProfileParameters
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `profiles/{id}/parameters/?$`, profileparameter.GetProfileID, auth.PrivLevelReadOnly, Authenticated, nil, 276464975},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `profiles/name/{name}/parameters/?$`, profileparameter.GetProfileName, auth.PrivLevelReadOnly, Authenticated, nil, 2267737832},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `profiles/name/{name}/parameters/?$`, profileparameter.PostProfileParamsByName, auth.PrivLevelOperations, Authenticated, nil, 2355945582},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `profiles/{id}/parameters/?$`, profileparameter.PostProfileParamsByID, auth.PrivLevelOperations, Authenticated, nil, 216818708},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `profileparameters/?$`, api.ReadHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 250609805},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `profileparameters/?$`, api.CreateHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelOperations, Authenticated, nil, 228809693},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `profileparameter/?$`, profileparameter.PostProfileParam, auth.PrivLevelOperations, Authenticated, nil, 224275},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `parameterprofile/?$`, profileparameter.PostParamProfile, auth.PrivLevelOperations, Authenticated, nil, 2080610861},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `profileparameters/{profileId}/{parameterId}$`, api.DeleteHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelOperations, Authenticated, nil, 224839529},

		//Tenants
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `tenants/?$`, api.ReadHandler(&apitenant.TOTenant{}), auth.PrivLevelReadOnly, Authenticated, nil, 2677967814},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `tenants/{id}$`, api.UpdateHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 2094131478},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `tenants/?$`, api.CreateHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 217248013},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `tenants/{id}$`, api.DeleteHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 216365558},

		//CRConfig
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/{cdn}/snapshot/?$`, crconfig.SnapshotGetHandler, auth.PrivLevelReadOnly, Authenticated, nil, 2957273695},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `cdns/{cdn}/snapshot/new/?$`, crconfig.Handler, auth.PrivLevelReadOnly, Authenticated, nil, 276716889},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `snapshot/?$`, crconfig.SnapshotHandler, auth.PrivLevelOperations, Authenticated, nil, 2969911829},

		// Federations
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `federations/all/?$`, federations.GetAll, auth.PrivLevelAdmin, Authenticated, nil, 21059986},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `federations/?$`, federations.Get, auth.PrivLevelFederation, Authenticated, nil, 254954994},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `federations/?$`, federations.AddFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 2894064742},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `federations/?$`, federations.RemoveFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 22098323},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `federations/?$`, federations.ReplaceFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 2283182516},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `federations/{id}/deliveryservices/?$`, federations.PostDSes, auth.PrivLevelAdmin, Authenticated, nil, 2682863513},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `federations/{id}/deliveryservices/?$`, api.ReadHandler(&federations.TOFedDSes{}), auth.PrivLevelReadOnly, Authenticated, nil, 253773034},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `federations/{id}/deliveryservices/{dsID}/?$`, api.DeleteHandler(&federations.TOFedDSes{}), auth.PrivLevelAdmin, Authenticated, nil, 2417402570},

		// Federation Resolvers
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `federation_resolvers/?$`, federation_resolvers.Create, auth.PrivLevelAdmin, Authenticated, nil, 2134373661},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `federation_resolvers/?$`, federation_resolvers.Read, auth.PrivLevelReadOnly, Authenticated, nil, 256608759},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `federations/{id}/federation_resolvers/?$`, federations.AssignFederationResolversToFederationHandler, auth.PrivLevelAdmin, Authenticated, nil, 256608760},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `federations/{id}/federation_resolvers/?$`, federations.GetFederationFederationResolversHandler, auth.PrivLevelReadOnly, Authenticated, nil, 256608761},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `federation_resolvers/?$`, federation_resolvers.Delete, auth.PrivLevelAdmin, Authenticated, nil, 2001},

		// Federations Users
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `federations/{id}/users/?$`, federations.PostUsers, auth.PrivLevelAdmin, Authenticated, nil, 2779334930},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `federations/{id}/users/?$`, api.ReadHandler(&federations.TOUsers{}), auth.PrivLevelReadOnly, Authenticated, nil, 294075015},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `federations/{id}/users/{userID}/?$`, api.DeleteHandler(&federations.TOUsers{}), auth.PrivLevelAdmin, Authenticated, nil, 2949102882},

		////DeliveryServices
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices/?$`, api.ReadHandler(&deliveryservice.TODeliveryService{}), auth.PrivLevelReadOnly, Authenticated, nil, 2238317294},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservices/?$`, deliveryservice.CreateV15, auth.PrivLevelOperations, Authenticated, nil, 206431432},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `deliveryservices/{id}/?$`, deliveryservice.UpdateV15, auth.PrivLevelOperations, Authenticated, nil, 2766567527},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `deliveryservices/{id}/safe/?$`, deliveryservice.UpdateSafe, auth.PrivLevelOperations, Authenticated, nil, 247210931},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `deliveryservices/{id}/?$`, api.DeleteHandler(&deliveryservice.TODeliveryService{}), auth.PrivLevelOperations, Authenticated, nil, 222642074},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices/{id}/servers/eligible/?$`, deliveryservice.GetServersEligible, auth.PrivLevelReadOnly, Authenticated, nil, 274761584},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices/xmlId/{xmlid}/sslkeys$`, deliveryservice.GetSSLKeysByXMLIDV15, auth.PrivLevelAdmin, Authenticated, nil, 2135772907},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservices/sslkeys/add$`, deliveryservice.AddSSLKeys, auth.PrivLevelAdmin, Authenticated, nil, 2872878583},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `deliveryservices/xmlId/{xmlid}/sslkeys$`, deliveryservice.DeleteSSLKeys, auth.PrivLevelOperations, Authenticated, nil, 2926734},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservices/sslkeys/generate/?$`, deliveryservice.GenerateSSLKeys, auth.PrivLevelOperations, Authenticated, nil, 253439051},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/copyFromXmlId/{copy-name}/?$`, deliveryservice.CopyURLKeys, auth.PrivLevelOperations, Authenticated, nil, 2262501076},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/generate/?$`, deliveryservice.GenerateURLKeys, auth.PrivLevelOperations, Authenticated, nil, 2530482824},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices/xmlId/{name}/urlkeys/?$`, deliveryservice.GetURLKeysByName, auth.PrivLevelReadOnly, Authenticated, nil, 2202719211},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices/{id}/urlkeys/?$`, deliveryservice.GetURLKeysByID, auth.PrivLevelReadOnly, Authenticated, nil, 293197114},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `vault/bucket/{bucket}/key/{key}/values/?$`, vault.GetBucketKeyDeprecated, auth.PrivLevelAdmin, Authenticated, nil, 2220510801},

		//Delivery service LetsEncrypt
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `deliveryservices/sslkeys/generate/letsencrypt/?$`, deliveryservice.GenerateLetsEncryptCertificates, auth.PrivLevelOperations, Authenticated, nil, 253439052},
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `letsencrypt/dnsrecords/?$`, deliveryservice.GetDnsChallengeRecords, auth.PrivLevelOperations, Authenticated, nil, 253439055},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `letsencrypt/autorenew/?$`, deliveryservice.RenewCertificates, auth.PrivLevelOperations, Authenticated, nil, 253439056},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices/{id}/health/?$`, deliveryservice.GetHealth, auth.PrivLevelReadOnly, Authenticated, nil, 2234590101},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `deliveryservices/{id}/routing$`, crstats.GetDSRouting, auth.PrivLevelReadOnly, Authenticated, nil, 66733983},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `steering/{deliveryservice}/targets/?$`, api.ReadHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelReadOnly, Authenticated, nil, 2569607824},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `steering/{deliveryservice}/targets/?$`, api.CreateHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 2338216397},
		{api.Version{Major: 2, Minor: 0}, http.MethodPut, `steering/{deliveryservice}/targets/{target}/?$`, api.UpdateHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 2438608295},
		{api.Version{Major: 2, Minor: 0}, http.MethodDelete, `steering/{deliveryservice}/targets/{target}/?$`, api.DeleteHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 2288021515},

		// Stats Summary
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `stats_summary/?$`, trafficstats.GetStatsSummary, auth.PrivLevelReadOnly, Authenticated, nil, 280498598},
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `stats_summary/?$`, trafficstats.CreateStatsSummary, auth.PrivLevelReadOnly, Authenticated, nil, 280491598},

		//Pattern based consistent hashing endpoint
		{api.Version{Major: 2, Minor: 0}, http.MethodPost, `consistenthash/?$`, consistenthash.Post, auth.PrivLevelReadOnly, Authenticated, nil, 260755076},

		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `steering/?$`, steering.Get, auth.PrivLevelSteering, Authenticated, nil, 2174852457},

		// Plugins
		{api.Version{Major: 2, Minor: 0}, http.MethodGet, `plugins/?$`, plugins.Get(d.Plugins), auth.PrivLevelReadOnly, Authenticated, nil, 283498539},

		// API 1.x routes will be deprecated.  Please add all new routes to 2.x.
		// API Capability
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `api_capabilities/?(\.json)?$`, apicapability.GetAPICapabilitiesHandler, auth.PrivLevelReadOnly, Authenticated, nil, 1813206589},

		//ASN: CRUD
		{api.Version{Major: 1, Minor: 2}, http.MethodGet, `asns/?(\.json)?$`, api.ReadHandler(&asn.TOASNV11{}), auth.PrivLevelReadOnly, Authenticated, nil, 473877722},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `asns/?(\.json)?$`, asn.V11ReadAll, auth.PrivLevelReadOnly, Authenticated, nil, 570341929},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `asns/{id}$`, api.DeprecatedReadHandler(&asn.TOASNV11{}, util.StrPtr("GET /asns")), auth.PrivLevelReadOnly, Authenticated, nil, 123008984},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `asns/{id}$`, api.UpdateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 1951198629},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `asns/?$`, api.CreateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 1999492188},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `asns/{id}$`, api.DeleteHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 1672524769},

		// Traffic Stats access
		{api.Version{Major: 1, Minor: 2}, http.MethodGet, `deliveryservice_stats`, trafficstats.GetDSStats, auth.PrivLevelReadOnly, Authenticated, nil, 1319569028},
		{api.Version{Major: 1, Minor: 2}, http.MethodGet, `cache_stats`, trafficstats.GetCacheStats, auth.PrivLevelReadOnly, Authenticated, nil, 1497997906},
		{api.Version{Major: 1, Minor: 2}, http.MethodGet, `current_stats/?(\.json)?$`, trafficstats.GetCurrentStats, auth.PrivLevelReadOnly, Authenticated, nil, 1785442893},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `caches/stats/?(\.json)?$`, cachesstats.Get, auth.PrivLevelReadOnly, Authenticated, nil, 1813206588},

		//CacheGroup: CRUD
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cachegroups/trimmed/?(\.json)?$`, cachegroup.GetTrimmed, auth.PrivLevelReadOnly, Authenticated, nil, 329527916},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cachegroups/?(\.json)?$`, api.ReadHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelReadOnly, Authenticated, nil, 123079110},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cachegroups/{id}$`, api.DeprecatedReadHandler(&cachegroup.TOCacheGroup{}, util.StrPtr("GET /cachegroups with query parameter id")), auth.PrivLevelReadOnly, Authenticated, nil, 691886338},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `cachegroups/{id}$`, api.UpdateHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 112954546},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `cachegroups/?$`, api.CreateHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 32982665},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `cachegroups/{id}$`, api.DeleteHandler(&cachegroup.TOCacheGroup{}), auth.PrivLevelOperations, Authenticated, nil, 257869365},

		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `cachegroups/{id}/queue_update$`, cachegroup.QueueUpdates, auth.PrivLevelOperations, Authenticated, nil, 1071644110},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `cachegroups/{id}/deliveryservices/?$`, cachegroup.DSPostHandlerV31, auth.PrivLevelOperations, Authenticated, nil, 1520240431},

		//CacheGroup Parameters: CRUD
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cachegroupparameters/?(\.json)?$`, cachegroupparameter.ReadAllCacheGroupParameters, auth.PrivLevelReadOnly, Authenticated, nil, 912449724},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `cachegroupparameters/?(\.json)?$`, cachegroupparameter.AddCacheGroupParameters, auth.PrivLevelOperations, Authenticated, nil, 912449725},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cachegroups/{id}/parameters/?(\.json)?$`, api.ReadHandler(&cachegroupparameter.TOCacheGroupParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 912449723},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cachegroups/{id}/unassigned_parameters/?(\.json)?$`, api.DeprecatedReadHandler(&cachegroupparameter.TOCacheGroupUnassignedParameter{}, util.StrPtr("GET /cachegroupparameters & GET /parameters")), auth.PrivLevelReadOnly, Authenticated, nil, 1457339250},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `cachegroupparameters/{cachegroupID}/{parameterId}$`, api.DeleteHandler(&cachegroupparameter.TOCacheGroupParameter{}), auth.PrivLevelOperations, Authenticated, nil, 912449733},

		//Capabilities
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `capabilities(/|\.json)?$`, capabilities.Read, auth.PrivLevelReadOnly, Authenticated, nil, 8008135},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `capabilities(/|\.json)?$`, capabilities.Create, auth.PrivLevelOperations, Authenticated, nil, -1},

		//CDN
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/name/{name}/sslkeys/?(\.json)?$`, cdn.GetSSLKeys, auth.PrivLevelAdmin, Authenticated, nil, 1278581772},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/metric_types`, notImplementedHandler, 0, NoAuth, nil, 683165463}, // MUST NOT end in $, because the 1.x route is longer

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/capacity$`, cdn.GetCapacity, auth.PrivLevelReadOnly, Authenticated, nil, 697185281},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/configs/?(\.json)?$`, api.DeprecatedReadHandler(&cdn.TOCDNConf{}, util.StrPtr("GET /cdns")), auth.PrivLevelReadOnly, Authenticated, nil, 1768437852},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{name}/health/?(\.json)?$`, cdn.GetNameHealth, auth.PrivLevelReadOnly, Authenticated, nil, 1135348194},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/health/?(\.json)?$`, cdn.GetHealth, auth.PrivLevelReadOnly, Authenticated, nil, 1085381134},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/domains/?(\.json)?$`, cdn.DomainsHandler, auth.PrivLevelReadOnly, Authenticated, nil, 296902560},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/routing$`, crstats.GetCDNRouting, auth.PrivLevelReadOnly, Authenticated, nil, 66722982},

		//CDN: CRUD
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/name/{name}/?(\.json)?$`, api.DeprecatedReadHandler(&cdn.TOCDN{}, util.StrPtr("GET /cdns with query parameter name")), auth.PrivLevelReadOnly, Authenticated, nil, 2135233288},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `cdns/name/{name}$`, cdn.DeleteName, auth.PrivLevelOperations, Authenticated, nil, 408804959},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/?(\.json)?$`, api.ReadHandler(&cdn.TOCDN{}), auth.PrivLevelReadOnly, Authenticated, nil, 1345914650},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{id}$`, api.DeprecatedReadHandler(&cdn.TOCDN{}, util.StrPtr("GET /cdns with query parameter id")), auth.PrivLevelReadOnly, Authenticated, nil, 2122954075},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `cdns/{id}$`, api.UpdateHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 549326357},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `cdns/?$`, api.CreateHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 24013912},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `cdns/{id}$`, api.DeleteHandler(&cdn.TOCDN{}), auth.PrivLevelOperations, Authenticated, nil, 1595587002},

		//CDN: queue updates
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `cdns/{id}/queue_update$`, cdn.Queue, auth.PrivLevelOperations, Authenticated, nil, 271515980},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `cdns/dnsseckeys/generate(\.json)?$`, cdn.CreateDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil, 675336},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/name/{name}/dnsseckeys/delete/?(\.json)?$`, cdn.DeleteDNSSECKeysDeprecated, auth.PrivLevelAdmin, Authenticated, nil, 571104207},
		{api.Version{Major: 1, Minor: 4}, http.MethodGet, `cdns/name/{name}/dnsseckeys/?(\.json)?$`, cdn.GetDNSSECKeys, auth.PrivLevelAdmin, Authenticated, nil, 479010609},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/name/{name}/dnsseckeys/?(\.json)?$`, cdn.GetDNSSECKeysV11, auth.PrivLevelAdmin, Authenticated, nil, 1427173311},

		{api.Version{Major: 1, Minor: 4}, http.MethodGet, `cdns/dnsseckeys/refresh/?(\.json)?$`, cdn.RefreshDNSSECKeys, auth.PrivLevelOperations, Authenticated, nil, 1771997116},

		//CDN: Monitoring: Traffic Monitor
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{cdn}/configs/monitoring(\.json)?$`, crconfig.SnapshotGetMonitoringLegacyHandler, auth.PrivLevelReadOnly, Authenticated, nil, 2140847892},

		//Database dumps
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `dbdump/?`, dbdump.DBDump, auth.PrivLevelAdmin, Authenticated, nil, 274016647},

		//Division: CRUD
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `divisions/?(\.json)?$`, api.ReadHandler(&division.TODivision{}), auth.PrivLevelReadOnly, Authenticated, nil, 1085181534},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `divisions/{id}$`, api.DeprecatedReadHandler(&division.TODivision{}, util.StrPtr("GET /divisions with the 'id' parameter")), auth.PrivLevelReadOnly, Authenticated, nil, 1241497902},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `divisions/{id}$`, api.UpdateHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 306369140},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `divisions/?$`, api.CreateHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 553713800},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `divisions/{id}$`, api.DeleteHandler(&division.TODivision{}), auth.PrivLevelOperations, Authenticated, nil, 1325382237},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `divisions/name/{name}/?(\.json)?$`, api.DeprecatedReadHandler(&division.TODivision{}, util.StrPtr("GET /divisions with the 'name' parameter")), auth.PrivLevelReadOnly, Authenticated, nil, 1211408769},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `logs/?(\.json)?$`, logs.Get, auth.PrivLevelReadOnly, Authenticated, nil, 848340550},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `logs/{days}/days/?(\.json)?$`, logs.GetDeprecated, auth.PrivLevelReadOnly, Authenticated, nil, 1192414145},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `logs/newcount/?(\.json)?$`, logs.GetNewCount, auth.PrivLevelReadOnly, Authenticated, nil, 1405833012},

		//HWInfo
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `hwinfo/?(\.json)?$`, hwinfo.Get, auth.PrivLevelReadOnly, Authenticated, nil, 621685998},

		//Content invalidation jobs
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `jobs(/|\.json/?)?$`, api.ReadHandler(&invalidationjobs.InvalidationJob{}), auth.PrivLevelReadOnly, Authenticated, nil, 1966782041},
		{api.Version{Major: 1, Minor: 4}, http.MethodDelete, `jobs/?$`, invalidationjobs.Delete, auth.PrivLevelPortal, Authenticated, nil, 616780776},
		{api.Version{Major: 1, Minor: 4}, http.MethodPut, `jobs/?$`, invalidationjobs.Update, auth.PrivLevelPortal, Authenticated, nil, 186134226},
		{api.Version{Major: 1, Minor: 4}, http.MethodPost, `jobs/?`, invalidationjobs.Create, auth.PrivLevelPortal, Authenticated, nil, 80450955},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `jobs/{id}(/|\.json/?)?$`, api.DeprecatedReadHandler(&invalidationjobs.InvalidationJob{}, util.StrPtr("GET /jobs with the 'id' parameter")), auth.PrivLevelReadOnly, Authenticated, nil, 2085189426},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `user/current/jobs(/|\.json/?)?$`, invalidationjobs.CreateUserJob, auth.PrivLevelPortal, Authenticated, nil, 611328688},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `user/current/jobs(/|\.json/?)?$`, invalidationjobs.GetUserJobs, auth.PrivLevelReadOnly, Authenticated, nil, 349163540},

		//Login
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `users/{id}/deliveryservices/?(\.json)?$`, user.GetDSes, auth.PrivLevelReadOnly, Authenticated, nil, 988787789},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `user/{id}/deliveryservices/available/?(\.json)?$`, user.GetAvailableDSes, auth.PrivLevelReadOnly, Authenticated, nil, 757082995},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `user/login/?$`, login.LoginHandler(d.DB, d.Config), 0, NoAuth, nil, 1392670821},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `user/logout(/|\.json)?$`, login.LogoutHandler(d.Config.Secrets[0]), 0, Authenticated, nil, 443434825},
		{api.Version{Major: 1, Minor: 4}, http.MethodPost, `user/login/oauth/?$`, login.OauthLoginHandler(d.DB, d.Config), 0, NoAuth, nil, 1415886009},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `user/login/token(/|\.json)?$`, login.TokenLoginHandler(d.DB, d.Config), 0, NoAuth, nil, 402408841},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `user/reset_password(/|\.json)?$`, login.ResetPassword(d.DB, d.Config), 0, NoAuth, nil, 2092914630},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `users/register(/|\.json)?$`, login.RegisterUser, auth.PrivLevelOperations, Authenticated, nil, 1337},

		//ISO
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `osversions(/|\.json)?$`, iso.GetOSVersions, auth.PrivLevelReadOnly, Authenticated, nil, 576088657},

		//User: CRUD
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `users/?(\.json)?$`, api.ReadHandler(&user.TOUser{}), auth.PrivLevelReadOnly, Authenticated, nil, 1491929900},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `users/{id}$`, api.ReadHandler(&user.TOUser{}), auth.PrivLevelReadOnly, Authenticated, nil, 713809980},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `users/{id}$`, api.UpdateHandler(&user.TOUser{}), auth.PrivLevelOperations, Authenticated, nil, 135433404},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `users/?(\.json)?$`, api.CreateHandler(&user.TOUser{}), auth.PrivLevelOperations, Authenticated, nil, 876244816},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `user/current/?(\.json)?$`, user.Current, auth.PrivLevelReadOnly, Authenticated, nil, 1610701614},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `user/current(/|\.json)?$`, user.ReplaceCurrent, auth.PrivLevelReadOnly, Authenticated, nil, 420},

		//Parameter: CRUD
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `parameters/?(\.json)?$`, api.ReadHandler(&parameter.TOParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 2012554292},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `parameters/{id}$`, api.DeprecatedReadHandler(&parameter.TOParameter{}, util.StrPtr("GET /parameters with query parameter id")), auth.PrivLevelReadOnly, Authenticated, nil, 1221666841},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `parameters/{id}$`, api.UpdateHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 1873936115},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `parameters/?$`, api.CreateHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 1669510859},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `parameters/{id}$`, api.DeleteHandler(&parameter.TOParameter{}), auth.PrivLevelOperations, Authenticated, nil, 276277118},

		//Phys_Location: CRUD
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `phys_locations/?(\.json)?$`, api.ReadHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelReadOnly, Authenticated, nil, 120405182},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `phys_locations/trimmed/?(\.json)?$`, physlocation.GetTrimmed, auth.PrivLevelReadOnly, Authenticated, nil, 1097221000},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `phys_locations/{id}$`, api.DeprecatedReadHandler(&physlocation.TOPhysLocation{}, util.StrPtr("GET /phys_locations with query parameter id")), auth.PrivLevelReadOnly, Authenticated, nil, 1554216025},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `phys_locations/{id}$`, api.UpdateHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 226795021},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `phys_locations/?$`, api.CreateHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 2146456648},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `phys_locations/{id}$`, api.DeleteHandler(&physlocation.TOPhysLocation{}), auth.PrivLevelOperations, Authenticated, nil, 15614221},

		//Ping
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `ping$`, ping.Handler, 0, NoAuth, nil, 1555661597},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `riak/ping/?(\.json)?$`, ping.Riak, auth.PrivLevelReadOnly, Authenticated, nil, 1884012114},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `keys/ping/?(\.json)?$`, ping.Keys, auth.PrivLevelReadOnly, Authenticated, nil, 318416022},

		//Profile: CRUD
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/?(\.json)?$`, api.ReadHandler(&profile.TOProfile{}), auth.PrivLevelReadOnly, Authenticated, nil, 668758589},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/trimmed/?(\.json)?$`, profile.Trimmed, auth.PrivLevelReadOnly, Authenticated, nil, 644942941},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{id}$`, api.DeprecatedReadHandler(&profile.TOProfile{}, util.StrPtr("GET /profiles with query parameter id")), auth.PrivLevelReadOnly, Authenticated, nil, 1570260672},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `profiles/{id}$`, api.UpdateHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 98439172},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `profiles/?$`, api.CreateHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 1540211556},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `profiles/{id}$`, api.DeleteHandler(&profile.TOProfile{}), auth.PrivLevelOperations, Authenticated, nil, 2005594465},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{id}/export/?(\.json)?$`, profile.ExportProfileHandler, auth.PrivLevelReadOnly, Authenticated, nil, 30133517},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `profiles/import/?(\.json)?$`, profile.ImportProfileHandler, auth.PrivLevelOperations, Authenticated, nil, 806143208},

		// Copy Profile
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `profiles/name/{new_profile}/copy/{existing_profile}`, profile.CopyProfileHandler, auth.PrivLevelOperations, Authenticated, nil, 806143209},

		//Region: CRUDs
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `regions/?(\.json)?$`, api.ReadHandler(&region.TORegion{}), auth.PrivLevelReadOnly, Authenticated, nil, 410037085},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `regions/{id}$`, api.DeprecatedReadHandler(&region.TORegion{}, util.StrPtr("GET /regions with query parameter id")), auth.PrivLevelReadOnly, Authenticated, nil, 2024440051},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `regions/name/{name}/?(\.json)?$`, region.GetName, auth.PrivLevelReadOnly, Authenticated, nil, 503583197},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `regions/{id}$`, api.UpdateHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 226308224},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `regions/?$`, api.CreateHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 1288334488},
		{api.Version{Major: 1, Minor: 5}, http.MethodDelete, `regions/?$`, api.DeleteHandler(&region.TORegion{}), auth.PrivLevelOperations, Authenticated, nil, 2032626758},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `regions/name/{name}$`, handlerToFunc(proxyHandler), 0, NoAuth, nil, 1925881096},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `regions/{id}$`, api.DeprecatedDeleteHandler(&region.TORegion{}, util.StrPtr("DELETE /regions with query parameter id")), auth.PrivLevelOperations, Authenticated, nil, 1181575271},

		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `deliveryservice_server/{dsid}/{serverid}`, dsserver.DeleteDeprecated, auth.PrivLevelOperations, Authenticated, nil, 1532184523},

		// get all edge servers associated with a delivery service (from deliveryservice_server table)

		{api.Version{Major: 1, Minor: 4}, http.MethodGet, `deliveryserviceserver/?(\.json)?$`, dsserver.ReadDSSHandlerV14, auth.PrivLevelReadOnly, Authenticated, nil, 1946145033},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryserviceserver/?(\.json)?$`, dsserver.ReadDSSHandler, auth.PrivLevelReadOnly, Authenticated, nil, 1928775049},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `deliveryserviceserver$`, dsserver.GetReplaceHandler, auth.PrivLevelOperations, Authenticated, nil, 429799788},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `deliveryservices/{xml_id}/servers$`, dsserver.GetCreateHandler, auth.PrivLevelOperations, Authenticated, nil, 1428181206},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/{id}/deliveryservices?(\/.json)?$`, api.ReadHandler(&dsserver.TODSSDeliveryService{}), auth.PrivLevelReadOnly, Authenticated, nil, 133115411},
		{api.Version{Major: 1, Minor: 3}, http.MethodPost, `servers/{id}/deliveryservices$`, server.AssignDeliveryServicesToServerHandler, auth.PrivLevelOperations, Authenticated, nil, 880128253},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/{id}/servers$`, dsserver.GetReadAssigned, auth.PrivLevelReadOnly, Authenticated, nil, 1345121223},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/{id}/unassigned_servers$`, dsserver.GetReadUnassigned, auth.PrivLevelReadOnly, Authenticated, nil, 2023944221},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `deliveryservices/request`, deliveryservicerequests.Request, auth.PrivLevelPortal, Authenticated, nil, 740875299},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservice_matches/?(\.json)?$`, deliveryservice.GetMatches, auth.PrivLevelReadOnly, Authenticated, nil, 1191301170},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/{id}/capacity/?(\.json)?$`, deliveryservice.GetCapacity, auth.PrivLevelReadOnly, Authenticated, nil, 1231409110},

		//Server
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/status$`, server.GetServersStatusCountsHandler, auth.PrivLevelReadOnly, Authenticated, nil, 2052786293},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/totals$`, handlerToFunc(proxyHandler), 0, NoAuth, []middleware.Middleware{}, 2037840835},

		//Serverchecks
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/checks(/|\.json)?$`, servercheck.DeprecatedReadServersChecks, auth.PrivLevelReadOnly, Authenticated, nil, 1796112922},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `servercheck/?(\.json)?$`, servercheck.CreateUpdateServercheck, auth.PrivLevelInvalid, Authenticated, nil, 1764281568},

		//Server Details
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/details/?(\.json)?$`, server.GetDetailParamHandler, auth.PrivLevelReadOnly, Authenticated, nil, 1261264714},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/hostname/{hostName}/details/?(\.json)?$`, server.GetDetailHandler, auth.PrivLevelReadOnly, Authenticated, nil, 372366128},

		//Server status
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `servers/{id}/status$`, server.UpdateStatusHandler, auth.PrivLevelOperations, Authenticated, nil, 776663851},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `servers/{id}/queue_update$`, server.QueueUpdateHandler, auth.PrivLevelOperations, Authenticated, nil, 9189471},
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `servers/{host_name}/update_status$`, server.GetServerUpdateStatusHandler, auth.PrivLevelReadOnly, Authenticated, nil, 438451599},

		//Server: CRUD
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/?(\.json)?$`, server.Read, auth.PrivLevelReadOnly, Authenticated, nil, 1720959285},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/{id}$`, server.ReadID, auth.PrivLevelReadOnly, Authenticated, nil, 1543122028},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `servers/{id}$`, server.Update, auth.PrivLevelOperations, Authenticated, nil, 958634103},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `servers/?$`, server.Create, auth.PrivLevelOperations, Authenticated, nil, 2025558061},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `servers/{id}$`, server.Delete, auth.PrivLevelOperations, Authenticated, nil, 192322233},

		//Server Capability
		{api.Version{Major: 1, Minor: 4}, http.MethodGet, `server_capabilities$`, api.ReadHandler(&servercapability.TOServerCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 610407391},
		{api.Version{Major: 1, Minor: 4}, http.MethodPost, `server_capabilities$`, api.CreateHandler(&servercapability.TOServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 1074470708},
		{api.Version{Major: 1, Minor: 4}, http.MethodDelete, `server_capabilities$`, api.DeleteHandler(&servercapability.TOServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 736415038},

		//Server Server Capabilities: CRUD
		{api.Version{Major: 1, Minor: 4}, http.MethodGet, `server_server_capabilities/?$`, api.ReadHandler(&server.TOServerServerCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 1800231889},
		{api.Version{Major: 1, Minor: 4}, http.MethodPost, `server_server_capabilities/?$`, api.CreateHandler(&server.TOServerServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 2093166834},
		{api.Version{Major: 1, Minor: 4}, http.MethodDelete, `server_server_capabilities/?$`, api.DeleteHandler(&server.TOServerServerCapability{}), auth.PrivLevelOperations, Authenticated, nil, 1058714058},

		//Status: CRUD
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `statuses/?(\.json)?$`, api.ReadHandler(&status.TOStatus{}), auth.PrivLevelReadOnly, Authenticated, nil, 2044905656},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `statuses/{id}$`, api.DeprecatedReadHandler(&status.TOStatus{}, util.StrPtr("GET /statuses with query parameter id")), auth.PrivLevelReadOnly, Authenticated, nil, 1899095947},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `statuses/{id}$`, api.UpdateHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 1207966504},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `statuses/?$`, api.CreateHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 1369123612},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `statuses/{id}$`, api.DeleteHandler(&status.TOStatus{}), auth.PrivLevelOperations, Authenticated, nil, 755111360},

		//System
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `system/info/?(\.json)?$`, systeminfo.Get, auth.PrivLevelReadOnly, Authenticated, nil, 211047475},

		//Type: CRUD
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `types/trimmed/?(\.json)?$`, handlerToFunc(proxyHandler), 0, NoAuth, []middleware.Middleware{}, 666},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `types/?(\.json)?$`, api.ReadHandler(&types.TOType{}), auth.PrivLevelReadOnly, Authenticated, nil, 2026701823},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `types/{id}$`, api.DeprecatedReadHandler(&types.TOType{}, util.StrPtr("GET /types with the 'id' query parameter")), auth.PrivLevelReadOnly, Authenticated, nil, 86037256},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `types/{id}$`, api.UpdateHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 68860115},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `types/?$`, api.CreateHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 1513308195},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `types/{id}$`, api.DeleteHandler(&types.TOType{}), auth.PrivLevelOperations, Authenticated, nil, 93175773},

		//About
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `about/?(\.json)?$`, about.Handler(), auth.PrivLevelReadOnly, Authenticated, nil, 1317501166},

		//Coordinates
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `coordinates/?(\.json)?$`, api.ReadHandler(&coordinate.TOCoordinate{}), auth.PrivLevelReadOnly, Authenticated, nil, 696700745},
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `coordinates/?$`, api.ReadHandler(&coordinate.TOCoordinate{}), auth.PrivLevelReadOnly, Authenticated, nil, 244546706},
		{api.Version{Major: 1, Minor: 3}, http.MethodPut, `coordinates/?$`, api.UpdateHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 368926174},
		{api.Version{Major: 1, Minor: 3}, http.MethodPost, `coordinates/?$`, api.CreateHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 1428112157},
		{api.Version{Major: 1, Minor: 3}, http.MethodDelete, `coordinates/?$`, api.DeleteHandler(&coordinate.TOCoordinate{}), auth.PrivLevelOperations, Authenticated, nil, 1303849889},

		//ASNs
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `asns/?(\.json)?$`, api.ReadHandler(&asn.TOASNV11{}), auth.PrivLevelReadOnly, Authenticated, nil, 2017162392},
		{api.Version{Major: 1, Minor: 3}, http.MethodPut, `asns/?$`, api.UpdateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 2064172317},
		{api.Version{Major: 1, Minor: 3}, http.MethodPost, `asns/?$`, api.CreateHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 859114392},
		{api.Version{Major: 1, Minor: 3}, http.MethodDelete, `asns/?$`, api.DeleteHandler(&asn.TOASNV11{}), auth.PrivLevelOperations, Authenticated, nil, 680204898},

		//Delivery service requests
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `deliveryservice_requests/?(\.json)?$`, dsrequest.Get, auth.PrivLevelReadOnly, Authenticated, nil, 1681163935},
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `deliveryservice_requests/?$`, dsrequest.Get, auth.PrivLevelReadOnly, Authenticated, nil, 286812311},
		{api.Version{Major: 1, Minor: 3}, http.MethodPut, `deliveryservice_requests/?$`, dsrequest.Put, auth.PrivLevelPortal, Authenticated, nil, 2049907918},
		{api.Version{Major: 1, Minor: 3}, http.MethodPost, `deliveryservice_requests/?$`, dsrequest.Post, auth.PrivLevelPortal, Authenticated, nil, 59385039},
		{api.Version{Major: 1, Minor: 3}, http.MethodDelete, `deliveryservice_requests/?$`, dsrequest.Delete, auth.PrivLevelPortal, Authenticated, nil, 1296985025},

		//Delivery service request: Actions
		{api.Version{Major: 1, Minor: 3}, http.MethodPut, `deliveryservice_requests/{id}/assign$`, dsrequest.PutAssignment, auth.PrivLevelOperations, Authenticated, nil, 1703160290},
		{api.Version{Major: 1, Minor: 3}, http.MethodPut, `deliveryservice_requests/{id}/status$`, dsrequest.PutStatus, auth.PrivLevelPortal, Authenticated, nil, 668415099},

		//Delivery service request comment: CRUD
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `deliveryservice_request_comments/?(\.json)?$`, api.ReadHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelReadOnly, Authenticated, nil, 1032650737},
		{api.Version{Major: 1, Minor: 3}, http.MethodPut, `deliveryservice_request_comments/?$`, api.UpdateHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 860487847},
		{api.Version{Major: 1, Minor: 3}, http.MethodPost, `deliveryservice_request_comments/?$`, api.CreateHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 727227672},
		{api.Version{Major: 1, Minor: 3}, http.MethodDelete, `deliveryservice_request_comments/?$`, api.DeleteHandler(&comment.TODeliveryServiceRequestComment{}), auth.PrivLevelPortal, Authenticated, nil, 199504668},

		//Delivery service uri signing keys: CRUD
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.GetURIsignkeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 1293078558},
		{api.Version{Major: 1, Minor: 3}, http.MethodPost, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.SaveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 508466335},
		{api.Version{Major: 1, Minor: 3}, http.MethodPut, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.SaveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 47648969},
		{api.Version{Major: 1, Minor: 3}, http.MethodDelete, `deliveryservices/{xmlID}/urisignkeys$`, urisigning.RemoveDeliveryServiceURIKeysHandler, auth.PrivLevelAdmin, Authenticated, nil, 429925417},

		//Delivery Service Required Capabilities: CRUD
		{api.Version{Major: 1, Minor: 4}, http.MethodGet, `deliveryservices_required_capabilities/?$`, api.ReadHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelReadOnly, Authenticated, nil, 1158522227},
		{api.Version{Major: 1, Minor: 4}, http.MethodPost, `deliveryservices_required_capabilities/?$`, api.CreateHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelOperations, Authenticated, nil, 1096873992},
		{api.Version{Major: 1, Minor: 4}, http.MethodDelete, `deliveryservices_required_capabilities/?$`, api.DeleteHandler(&deliveryservice.RequiredCapability{}), auth.PrivLevelOperations, Authenticated, nil, 1496289304},

		// Federations by CDN (the actual table for federation)
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{name}/federations/?(\.json)?$`, api.ReadHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelReadOnly, Authenticated, nil, 989225032},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{name}/federations/{id}$`, api.DeprecatedReadHandler(&cdnfederation.TOCDNFederation{}, util.StrPtr("GET /cdns/{name}/federations with query parameter id")), auth.PrivLevelReadOnly, Authenticated, nil, 21850599},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `cdns/{name}/federations/?(\.json)?$`, api.CreateHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 1954894219},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `cdns/{name}/federations/{id}$`, api.UpdateHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 2106065466},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `cdns/{name}/federations/{id}$`, api.DeleteHandler(&cdnfederation.TOCDNFederation{}), auth.PrivLevelAdmin, Authenticated, nil, 1442852902},

		{api.Version{Major: 1, Minor: 4}, http.MethodPost, `cdns/{name}/dnsseckeys/ksk/generate$`, cdn.GenerateKSK, auth.PrivLevelAdmin, Authenticated, nil, 872924281},

		//Origins
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `origins/?(\.json)?$`, api.ReadHandler(&origin.TOOrigin{}), auth.PrivLevelReadOnly, Authenticated, nil, 844649256},
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `origins/?$`, api.ReadHandler(&origin.TOOrigin{}), auth.PrivLevelReadOnly, Authenticated, nil, 1945936793},
		{api.Version{Major: 1, Minor: 3}, http.MethodPut, `origins/?$`, api.UpdateHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 141567746},
		{api.Version{Major: 1, Minor: 3}, http.MethodPost, `origins/?$`, api.CreateHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 1099561643},
		{api.Version{Major: 1, Minor: 3}, http.MethodDelete, `origins/?$`, api.DeleteHandler(&origin.TOOrigin{}), auth.PrivLevelOperations, Authenticated, nil, 460273263},

		//Roles
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `roles/?(\.json)?$`, api.ReadHandler(&role.TORole{}), auth.PrivLevelReadOnly, Authenticated, nil, 187088583},
		{api.Version{Major: 1, Minor: 3}, http.MethodPut, `roles/?$`, api.UpdateHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 1612897489},
		{api.Version{Major: 1, Minor: 3}, http.MethodPost, `roles/?$`, api.CreateHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 430652406},
		{api.Version{Major: 1, Minor: 3}, http.MethodDelete, `roles/?$`, api.DeleteHandler(&role.TORole{}), auth.PrivLevelAdmin, Authenticated, nil, 1356705982},

		//Delivery Services Regexes
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices_regexes/?(\.json)?$`, deliveryservicesregexes.Get, auth.PrivLevelReadOnly, Authenticated, nil, 605501453},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/{dsid}/regexes/?(\.json)?$`, deliveryservicesregexes.DSGet, auth.PrivLevelReadOnly, Authenticated, nil, 577432763},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/{dsid}/regexes/{regexid}?(\.json)?$`, deliveryservicesregexes.DSGetID, auth.PrivLevelReadOnly, Authenticated, nil, 1044974567},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `deliveryservices/{dsid}/regexes/?(\.json)?$`, deliveryservicesregexes.Post, auth.PrivLevelOperations, Authenticated, nil, 412737800},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `deliveryservices/{dsid}/regexes/{regexid}?(\.json)?$`, deliveryservicesregexes.Put, auth.PrivLevelOperations, Authenticated, nil, 1248339691},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `deliveryservices/{dsid}/regexes/{regexid}?(\.json)?$`, deliveryservicesregexes.Delete, auth.PrivLevelOperations, Authenticated, nil, 2046731663},

		//StaticDNSEntries
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `staticdnsentries/?(\.json)?$`, api.ReadHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelReadOnly, Authenticated, nil, 258939477},
		{api.Version{Major: 1, Minor: 3}, http.MethodGet, `staticdnsentries/?$`, api.ReadHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelReadOnly, Authenticated, nil, 1116932668},
		{api.Version{Major: 1, Minor: 3}, http.MethodPut, `staticdnsentries/?$`, api.UpdateHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 142457111},
		{api.Version{Major: 1, Minor: 3}, http.MethodPost, `staticdnsentries/?$`, api.CreateHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 1629148238},
		{api.Version{Major: 1, Minor: 3}, http.MethodDelete, `staticdnsentries/?$`, api.DeleteHandler(&staticdnsentry.TOStaticDNSEntry{}), auth.PrivLevelOperations, Authenticated, nil, 1846031132},

		//ProfileParameters
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{id}/parameters/?(\.json)?$`, profileparameter.GetProfileID, auth.PrivLevelReadOnly, Authenticated, nil, 876464975},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{id}/unassigned_parameters/?(\.json)?$`, profileparameter.GetUnassigned, auth.PrivLevelReadOnly, Authenticated, nil, 574429262},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/name/{name}/parameters/?(\.json)?$`, profileparameter.GetProfileName, auth.PrivLevelReadOnly, Authenticated, nil, 2067737832},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `parameters/profile/{name}/?(\.json)?$`, profileparameter.GetProfileNameDeprecated, auth.PrivLevelReadOnly, Authenticated, nil, 1802599194},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `profiles/name/{name}/parameters/?$`, profileparameter.PostProfileParamsByName, auth.PrivLevelOperations, Authenticated, nil, 1355945582},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `profiles/{id}/parameters/?$`, profileparameter.PostProfileParamsByID, auth.PrivLevelOperations, Authenticated, nil, 316818708},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profileparameters/?(\.json)?$`, api.ReadHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelReadOnly, Authenticated, nil, 850609805},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `profileparameters/?$`, api.CreateHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelOperations, Authenticated, nil, 218809693},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `profileparameter/?$`, profileparameter.PostProfileParam, auth.PrivLevelOperations, Authenticated, nil, 234275},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `parameterprofile/?$`, profileparameter.PostParamProfile, auth.PrivLevelOperations, Authenticated, nil, 1080610861},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `profileparameters/{profileId}/{parameterId}$`, api.DeleteHandler(&profileparameter.TOProfileParameter{}), auth.PrivLevelOperations, Authenticated, nil, 254839529},

		//Tenants
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `tenants/?(\.json)?$`, api.ReadHandler(&apitenant.TOTenant{}), auth.PrivLevelReadOnly, Authenticated, nil, 1677967814},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `tenants/{id}$`, api.DeprecatedReadHandler(&apitenant.TOTenant{}, util.StrPtr("GET /tenants with query parameter id")), auth.PrivLevelReadOnly, Authenticated, nil, 171544338},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `tenants/{id}$`, api.UpdateHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 1094131478},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `tenants/?$`, api.CreateHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 917248013},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `tenants/{id}$`, api.DeleteHandler(&apitenant.TOTenant{}), auth.PrivLevelOperations, Authenticated, nil, 516365558},

		//CRConfig
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{cdn}/snapshot/?$`, crconfig.SnapshotGetHandler, auth.PrivLevelReadOnly, Authenticated, nil, 1957273695},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{cdn}/snapshot/new/?$`, crconfig.Handler, auth.PrivLevelReadOnly, Authenticated, nil, 676716889},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `cdns/{id}/snapshot/?$`, crconfig.SnapshotHandlerDeprecated, auth.PrivLevelOperations, Authenticated, nil, 854424150},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `snapshot/{cdn}/?$`, crconfig.SnapshotHandlerDeprecated, auth.PrivLevelOperations, Authenticated, nil, 1969911829},

		// Legacy Configfile routes
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/{server-name-or-id}/configfiles/ats/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1755842214},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{cdn-name-or-id}/configfiles/ats/regex_revalidate\.config/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1810067775},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{cdn-name-or-id}/configfiles/ats/hdr_rw_mid_{xml-id}\.config/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 658322121},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{cdn-name-or-id}/configfiles/ats/hdr_rw_{xml-id}\.config/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1894063777},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{cdn-name-or-id}/configfiles/ats/bg_fetch\.config/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 160404036},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{cdn-name-or-id}/configfiles/ats/cacheurl{filename}\.config/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1373111113},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{cdn-name-or-id}/configfiles/ats/regex_remap_{ds-name}\.config/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1283602930},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{cdn-name-or-id}/configfiles/ats/set_dscp_{dscp}\.config/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1889993740},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `cdns/{cdn-name-or-id}/configfiles/ats/ssl_multicert\.config/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1113687166},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/12M_facts/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 2146608231},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/50-ats\.rules/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1101032000},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/astats\.config/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1362661662},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/cache\.config/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 292387870},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/drop_qstring\.config/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1097869291},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/logging\.config/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 172702063},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/logging\.yaml/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 453568059},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/logs_xml\.config/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1309053227},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/plugin\.config/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 274047559},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/records\.config/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 469014057},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/storage\.config/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 121977329},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/sysctl\.conf/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1202950646},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/url_sig_{file}\.config/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 448450070},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/uri_signing_{file}\.config/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 125995582},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/volume\.config/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 792704719},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `profiles/{profile-name-or-id}/configfiles/ats/{file}/?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1651257268},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/{id-or-host}/configfiles/ats/cache\.config/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 34686861},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/{id-or-host}/configfiles/ats/hosting\.config/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1387459113},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/{id-or-host}/configfiles/ats/packages/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 245024839},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/{id-or-host}/configfiles/ats/chkconfig/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 1012457987},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `servers/{id-or-host}/configfiles/ats/{file}/?(\.json)?$`, api.GoneHandler, auth.PrivLevelOperations, Authenticated, nil, 322079218},

		// Federations
		{api.Version{Major: 1, Minor: 4}, http.MethodGet, `federations/all/?(\.json)?$`, federations.GetAll, auth.PrivLevelAdmin, Authenticated, nil, 61059986},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `federations/?(\.json)?$`, federations.Get, auth.PrivLevelFederation, Authenticated, nil, 154954994},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `federations(/|\.json)?$`, federations.AddFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 1894064742},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `federations(/|\.json)?$`, federations.RemoveFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 592098323},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `federations(/|\.json)?$`, federations.ReplaceFederationResolverMappingsForCurrentUser, auth.PrivLevelFederation, Authenticated, nil, 1283182516},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `federations/{id}/deliveryservices/?(\.json)?$`, federations.PostDSes, auth.PrivLevelAdmin, Authenticated, nil, 1682863513},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `federations/{id}/deliveryservices/?(\.json)?$`, api.ReadHandler(&federations.TOFedDSes{}), auth.PrivLevelReadOnly, Authenticated, nil, 353773034},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `federations/{id}/deliveryservices/{dsID}/?(\.json)?$`, api.DeleteHandler(&federations.TOFedDSes{}), auth.PrivLevelAdmin, Authenticated, nil, 1417402570},

		// Federation Resolvers
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `federation_resolvers(/|\.json)?$`, federation_resolvers.Create, auth.PrivLevelAdmin, Authenticated, nil, 1134373661},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `federation_resolvers(/|\.json)?$`, federation_resolvers.Read, auth.PrivLevelReadOnly, Authenticated, nil, 556608759},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `federations/{id}/federation_resolvers(/|\.json)?$`, federations.AssignFederationResolversToFederationHandler, auth.PrivLevelAdmin, Authenticated, nil, 556608760},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `federations/{id}/federation_resolvers(/|\.json)?$`, federations.GetFederationFederationResolversHandler, auth.PrivLevelReadOnly, Authenticated, nil, 556608761},
		{api.Version{Major: 1, Minor: 5}, http.MethodDelete, `federation_resolvers(/|\.json)?$`, federation_resolvers.Delete, auth.PrivLevelAdmin, Authenticated, nil, 9001},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `federation_resolvers/{id}(/|\.json)?$`, federation_resolvers.DeleteByID, auth.PrivLevelAdmin, Authenticated, nil, 42},

		// Federations Users
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `federations/{id}/users/?(\.json)?$`, federations.PostUsers, auth.PrivLevelAdmin, Authenticated, nil, 1779334930},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `federations/{id}/users/?(\.json)?$`, api.ReadHandler(&federations.TOUsers{}), auth.PrivLevelReadOnly, Authenticated, nil, 394075015},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `federations/{id}/users/{userID}/?(\.json)?$`, api.DeleteHandler(&federations.TOUsers{}), auth.PrivLevelAdmin, Authenticated, nil, 1949102882},

		////DeliveryServices
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/?(\.json)?$`, api.ReadHandler(&deliveryservice.TODeliveryService{}), auth.PrivLevelReadOnly, Authenticated, nil, 1238317294},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/{id}/?(\.json)?$`, api.DeprecatedReadHandler(&deliveryservice.TODeliveryService{}, util.StrPtr("GET deliveryservices/ with the id query parameter")), auth.PrivLevelReadOnly, Authenticated, nil, 444348195},

		{api.Version{Major: 1, Minor: 5}, http.MethodPost, `deliveryservices/?(\.json)?$`, deliveryservice.CreateV15, auth.PrivLevelOperations, Authenticated, nil, 506431432},
		{api.Version{Major: 1, Minor: 4}, http.MethodPost, `deliveryservices/?(\.json)?$`, deliveryservice.CreateV14, auth.PrivLevelOperations, Authenticated, nil, 506431431},
		{api.Version{Major: 1, Minor: 3}, http.MethodPost, `deliveryservices/?(\.json)?$`, deliveryservice.CreateV13, auth.PrivLevelOperations, Authenticated, nil, 1705681904},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `deliveryservices/?(\.json)?$`, deliveryservice.CreateV12, auth.PrivLevelOperations, Authenticated, nil, 652813412},

		{api.Version{Major: 1, Minor: 5}, http.MethodPut, `deliveryservices/{id}/?(\.json)?$`, deliveryservice.UpdateV15, auth.PrivLevelOperations, Authenticated, nil, 1766567527},
		{api.Version{Major: 1, Minor: 4}, http.MethodPut, `deliveryservices/{id}/?(\.json)?$`, deliveryservice.UpdateV14, auth.PrivLevelOperations, Authenticated, nil, 1766567526},
		{api.Version{Major: 1, Minor: 3}, http.MethodPut, `deliveryservices/{id}/?(\.json)?$`, deliveryservice.UpdateV13, auth.PrivLevelOperations, Authenticated, nil, 1559124565},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `deliveryservices/{id}/?(\.json)?$`, deliveryservice.UpdateV12, auth.PrivLevelOperations, Authenticated, nil, 597160536},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `deliveryservices/{id}/safe/?(\.json)?$`, deliveryservice.UpdateSafe, auth.PrivLevelOperations, Authenticated, nil, 547210931},

		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `deliveryservices/{id}/?(\.json)?$`, api.DeleteHandler(&deliveryservice.TODeliveryService{}), auth.PrivLevelOperations, Authenticated, nil, 242642074},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/{id}/servers/eligible/?(\.json)?$`, deliveryservice.GetServersEligible, auth.PrivLevelReadOnly, Authenticated, nil, 474761584},

		{api.Version{Major: 1, Minor: 5}, http.MethodGet, `deliveryservices/{id}/routing$`, crstats.GetDSRouting, auth.PrivLevelReadOnly, Authenticated, nil, 66733982},

		{api.Version{Major: 1, Minor: 5}, http.MethodGet, `deliveryservices/xmlId/{xmlid}/sslkeys$`, deliveryservice.GetSSLKeysByXMLIDV15, auth.PrivLevelAdmin, Authenticated, nil, 1135772906},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/xmlId/{xmlid}/sslkeys$`, deliveryservice.GetSSLKeysByXMLID, auth.PrivLevelAdmin, Authenticated, nil, 1135772907},
		{api.Version{Major: 1, Minor: 5}, http.MethodGet, `deliveryservices/hostname/{hostname}/sslkeys$`, deliveryservice.GetSSLKeysByHostNameV15, auth.PrivLevelAdmin, Authenticated, nil, 2105792224},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/hostname/{hostname}/sslkeys$`, deliveryservice.GetSSLKeysByHostName, auth.PrivLevelAdmin, Authenticated, nil, 2105792225},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `deliveryservices/sslkeys/add$`, deliveryservice.AddSSLKeys, auth.PrivLevelAdmin, Authenticated, nil, 1872878583},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/xmlId/{xmlid}/sslkeys/delete$`, deliveryservice.DeleteSSLKeysDeprecated, auth.PrivLevelOperations, Authenticated, nil, 1926734},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `deliveryservices/sslkeys/generate/?(\.json)?$`, deliveryservice.GenerateSSLKeys, auth.PrivLevelOperations, Authenticated, nil, 753439051},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/copyFromXmlId/{copy-name}/?(\.json)?$`, deliveryservice.CopyURLKeys, auth.PrivLevelOperations, Authenticated, nil, 1262501076},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `deliveryservices/xmlId/{name}/urlkeys/generate/?(\.json)?$`, deliveryservice.GenerateURLKeys, auth.PrivLevelOperations, Authenticated, nil, 1530482824},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/xmlId/{name}/urlkeys/?(\.json)?$`, deliveryservice.GetURLKeysByName, auth.PrivLevelReadOnly, Authenticated, nil, 2102719211},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/{id}/urlkeys/?(\.json)?$`, deliveryservice.GetURLKeysByID, auth.PrivLevelReadOnly, Authenticated, nil, 393197114},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `riak/bucket/{bucket}/key/{key}/values/?(\.json)?$`, vault.GetBucketKeyDeprecated, auth.PrivLevelAdmin, Authenticated, nil, 2020510801},

		//Delivery service LetsEncrypt
		{api.Version{Major: 1, Minor: 5}, http.MethodPost, `deliveryservices/sslkeys/generate/letsencrypt/?(\.json)?$`, deliveryservice.GenerateLetsEncryptCertificates, auth.PrivLevelOperations, Authenticated, nil, 753439052},
		{api.Version{Major: 1, Minor: 5}, http.MethodGet, `letsencrypt/dnsrecords/?(\.json)?$`, deliveryservice.GetDnsChallengeRecords, auth.PrivLevelOperations, Authenticated, nil, 753439055},
		{api.Version{Major: 1, Minor: 5}, http.MethodPost, `letsencrypt/autorenew/?(\.json)?$`, deliveryservice.RenewCertificates, auth.PrivLevelOperations, Authenticated, nil, 753439056},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `deliveryservices/{id}/health/?(\.json)?$`, deliveryservice.GetHealth, auth.PrivLevelReadOnly, Authenticated, nil, 2034590101},

		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `steering/{deliveryservice}/targets/?(\.json)?$`, api.ReadHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelReadOnly, Authenticated, nil, 1569607824},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `steering/{deliveryservice}/targets/{target}$`, api.DeprecatedReadHandler(&steeringtargets.TOSteeringTargetV11{}, util.StrPtr("GET steering/{deliveryservice}/targets with the query parameter target")), auth.PrivLevelReadOnly, Authenticated, nil, 105995849},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `steering/{deliveryservice}/targets/?(\.json)?$`, api.CreateHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 1338216397},
		{api.Version{Major: 1, Minor: 1}, http.MethodPut, `steering/{deliveryservice}/targets/{target}/?(\.json)?$`, api.UpdateHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 1438608295},
		{api.Version{Major: 1, Minor: 1}, http.MethodDelete, `steering/{deliveryservice}/targets/{target}/?(\.json)?$`, api.DeleteHandler(&steeringtargets.TOSteeringTargetV11{}), auth.PrivLevelSteering, Authenticated, nil, 2088021515},

		// Stats Summary
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `stats_summary/?(\.json)?$`, trafficstats.GetStatsSummary, auth.PrivLevelReadOnly, Authenticated, nil, 380498598},
		{api.Version{Major: 1, Minor: 5}, http.MethodPost, `stats_summary/?(\.json)?$`, trafficstats.CreateStatsSummary, auth.PrivLevelReadOnly, Authenticated, nil, 380491598},

		// TO Extensions
		{api.Version{Major: 1, Minor: 5}, http.MethodPost, `to_extensions$`, extensions.Create, auth.PrivLevelReadOnly, Authenticated, nil, 380498599},
		{api.Version{Major: 1, Minor: 1}, http.MethodGet, `to_extensions$`, extensions.Get, auth.PrivLevelReadOnly, Authenticated, nil, 383498599},
		{api.Version{Major: 1, Minor: 1}, http.MethodPost, `to_extensions/{id}/delete$`, extensions.Delete, auth.PrivLevelReadOnly, Authenticated, nil, 385118599},

		//Pattern based consistent hashing endpoint
		{api.Version{Major: 1, Minor: 4}, http.MethodPost, `consistenthash/?$`, consistenthash.Post, auth.PrivLevelReadOnly, Authenticated, nil, 1960755076},

		{api.Version{Major: 1, Minor: 4}, http.MethodGet, `steering/?(\.json)?$`, steering.Get, auth.PrivLevelSteering, Authenticated, nil, 1174852457},
	}

	// sanity check to make sure all Route IDs are unique
	knownRouteIDs := make(map[int]struct{}, len(routes))
	for _, r := range routes {
		if _, found := knownRouteIDs[r.ID]; !found {
			knownRouteIDs[r.ID] = struct{}{}
		} else {
			return nil, nil, nil, fmt.Errorf("route ID %d is already taken. Please give it a unique Route ID", r.ID)
		}
	}

	// check for unknown route IDs in cdn.conf
	disabledRoutes := GetRouteIDMap(d.DisabledRoutes)
	unknownRouteIDs := []string{}
	for _, routeMap := range []map[int]struct{}{disabledRoutes} {
		for routeID := range routeMap {
			if _, known := knownRouteIDs[routeID]; !known {
				unknownRouteIDs = append(unknownRouteIDs, fmt.Sprintf("%d", routeID))
			}
		}
	}
	if len(unknownRouteIDs) > 0 {
		msg := "unknown route IDs in routing_blacklist: " + strings.Join(unknownRouteIDs, ", ")
		if d.IgnoreUnknownRoutes {
			log.Warnln(msg)
		} else {
			return nil, nil, nil, errors.New(msg)
		}
	}

	// rawRoutes are served at the root path. These should be almost exclusively old Perl pre-API routes, which have yet to be converted in all clients. New routes should be in the versioned API path.
	rawRoutes := []RawRoute{
		// DEPRECATED - use PUT /api/1.2/snapshot/{cdn}
		{http.MethodGet, `tools/write_crconfig/{cdn}/?$`, crconfig.SnapshotOldGUIHandler, auth.PrivLevelOperations, Authenticated, nil},
		// DEPRECATED - use GET /api/1.2/cdns/{cdn}/snapshot
		{http.MethodGet, `CRConfig-Snapshots/{cdn}/CRConfig.json?$`, crconfig.SnapshotOldGetHandler, auth.PrivLevelReadOnly, Authenticated, nil},
	}

	return routes, rawRoutes, proxyHandler, nil
}

func MemoryStatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := runtime.MemStats{}
		runtime.ReadMemStats(&stats)

		bytes, err := json.Marshal(stats)
		if err != nil {
			err = fmt.Errorf("unable to marshal stats: %w", err)
			api.HandleErr(w, r, nil, http.StatusInternalServerError, errors.New("marshalling error"), err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

func DBStatsHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := db.DB.Stats()

		bytes, err := json.Marshal(stats)
		if err != nil {
			err = fmt.Errorf("unable to marshal stats: %v", err)
			api.HandleErr(w, r, nil, http.StatusInternalServerError, errors.New("marshalling error"), err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

type root struct {
	Handler http.Handler
}

func (root) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	alerts := tc.CreateAlerts(tc.ErrorLevel, fmt.Sprintf("The requested path '%s' does not exist.", r.URL.Path))
	api.WriteAlerts(w, r, http.StatusNotFound, alerts)
}

// rootHandler returns the / handler for the service, which simply returns a "not found" response.
func rootHandler(d ServerData) http.Handler {
	return root{}
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
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) >= 3 {
		version, err := strconv.ParseFloat(pathParts[2], 64)
		if err == nil && version >= 2 { // do not default to Perl for versions over 2.x
			api.WriteRespAlertNotFound(w, r)
			return
		}
	}

	m.ReqChan <- struct{}{}
	defer func() { <-m.ReqChan }()
	m.Handler.ServeHTTP(w, r)
}
