package user

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
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
)

func GetDSes(w http.ResponseWriter, r *http.Request) {
	alt := util.StrPtr("GET deliveryservices?accessibleTo={{tenantId}")
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr, alt)
		return
	}
	defer inf.Close()

	dsUserID := inf.IntParams["id"]
	dses, err := getUserDSes(inf.Tx.Tx, dsUserID)
	if err != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting user delivery services: "+err.Error()), alt)
		return
	}

	dses, err = filterAuthorized(inf.Tx.Tx, dses, inf.User)
	if err != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("filtering user-authorized delivery services: "+err.Error()), alt)
		return
	}
	api.WriteAlertsObj(w, r, http.StatusOK, api.CreateDeprecationAlerts(alt), dses)
}

func GetAvailableDSes(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr, nil)
		return
	}
	defer inf.Close()

	dsUserID := inf.IntParams["id"]
	dses, err := getUserAvailableDSes(inf.Tx.Tx, dsUserID)
	if err != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("getting user delivery services: %v", err), nil)
		return
	}

	dses, err = filterAvailableAuthorized(inf.Tx.Tx, dses, inf.User)
	if err != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("filtering user-authorized delivery services: %v", err), nil)
		return
	}

	alerts := api.CreateDeprecationAlerts(nil)
	api.WriteAlertsObj(w, r, http.StatusOK, alerts, dses)
}

func filterAuthorized(tx *sql.Tx, dses []tc.DeliveryServiceNullable, user *auth.CurrentUser) ([]tc.DeliveryServiceNullable, error) {
	authorizedDSes := []tc.DeliveryServiceNullable{}
	for _, ds := range dses {
		if ds.TenantID == nil {
			continue
		}
		authorized, err := tenant.IsResourceAuthorizedToUserTx(*ds.TenantID, user, tx)
		if err != nil {
			return nil, errors.New("checking delivery service tenancy authorization: " + err.Error())
		}
		if !authorized {
			continue // TODO determine if this is correct - Perl appears to return an error if any DS on the user is unauthorized to the current user
		}
		authorizedDSes = append(authorizedDSes, ds)
	}
	return authorizedDSes, nil
}

func filterAvailableAuthorized(tx *sql.Tx, dses []tc.UserAvailableDS, user *auth.CurrentUser) ([]tc.UserAvailableDS, error) {
	authorizedDSes := []tc.UserAvailableDS{}
	for _, ds := range dses {
		if ds.TenantID == nil {
			continue
		}
		authorized, err := tenant.IsResourceAuthorizedToUserTx(*ds.TenantID, user, tx)
		if err != nil {
			return nil, errors.New("checking delivery service tenancy authorization: " + err.Error())
		}
		if !authorized {
			continue // TODO determine if this is correct - Perl appears to return an error if any DS on the user is unauthorized to the current user
		}
		authorizedDSes = append(authorizedDSes, ds)
	}
	return authorizedDSes, nil
}

func getUserDSes(tx *sql.Tx, userID int) ([]tc.DeliveryServiceNullable, error) {
	q := `
SELECT
ds.active = 'ACTIVE' AS active,
ds.anonymous_blocking_enabled,
ds.cacheurl,
ds.ccr_dns_ttl,
ds.cdn_id,
cdn.name as cdnName,
ds.check_path,
CAST(ds.deep_caching_type AS text) as deep_caching_type,
ds.display_name,
ds.dns_bypass_cname,
ds.dns_bypass_ip,
ds.dns_bypass_ip6,
ds.dns_bypass_ttl,
ds.dscp,
ds.edge_header_rewrite,
ds.geolimit_redirect_url,
ds.geo_limit,
ds.geo_limit_countries,
ds.geo_provider,
ds.global_max_mbps,
ds.global_max_tps,
ds.fq_pacing_rate,
ds.http_bypass_fqdn,
ds.id,
ds.info_url,
ds.initial_dispersion,
ds.ipv6_routing_enabled,
ds.last_updated,
ds.logs_enabled,
ds.long_desc,
ds.long_desc_1,
ds.long_desc_2,
ds.max_dns_answers,
ds.mid_header_rewrite,
COALESCE(ds.miss_lat, 0.0),
COALESCE(ds.miss_long, 0.0),
ds.multi_site_origin,
(SELECT o.protocol::text || '://' || o.fqdn || rtrim(concat(':', o.port::text), ':')
  FROM origin o
  WHERE o.deliveryservice = ds.id
  AND o.is_primary) as org_server_fqdn,
ds.origin_shield,
ds.profile as profileID,
profile.name as profile_name,
profile.description  as profile_description,
ds.protocol,
ds.qstring_ignore,
ds.range_request_handling,
ds.regex_remap,
ds.regional_geo_blocking,
ds.remap_text,
ds.routing_name,
ds.signing_algorithm,
ds.ssl_key_version,
ds.tenant_id,
tenant.name,
ds.tr_request_headers,
ds.tr_response_headers,
type.name,
ds.type as type_id,
ds.xml_id
FROM deliveryservice as ds
JOIN type ON ds.type = type.id
JOIN cdn ON ds.cdn_id = cdn.id
JOIN deliveryservice_tmuser dsu ON ds.id = dsu.deliveryservice
LEFT JOIN profile ON ds.profile = profile.id
LEFT JOIN tenant ON ds.tenant_id = tenant.id
WHERE dsu.tm_user_id = $1
`
	rows, err := tx.Query(q, userID)
	if err != nil {
		return nil, errors.New("querying user delivery services: " + err.Error())
	}
	defer rows.Close()
	dses := []tc.DeliveryServiceNullable{}
	for rows.Next() {
		ds := tc.DeliveryServiceNullable{}
		err := rows.Scan(&ds.Active, &ds.AnonymousBlockingEnabled, &ds.CacheURL, &ds.CCRDNSTTL, &ds.CDNID, &ds.CDNName,
			&ds.CheckPath, &ds.DeepCachingType, &ds.DisplayName, &ds.DNSBypassCNAME, &ds.DNSBypassIP, &ds.DNSBypassIP6,
			&ds.DNSBypassTTL, &ds.DSCP, &ds.EdgeHeaderRewrite, &ds.GeoLimitRedirectURL, &ds.GeoLimit, &ds.GeoLimitCountries,
			&ds.GeoProvider, &ds.GlobalMaxMBPS, &ds.GlobalMaxTPS, &ds.FQPacingRate, &ds.HTTPBypassFQDN, &ds.ID, &ds.InfoURL,
			&ds.InitialDispersion, &ds.IPV6RoutingEnabled, &ds.LastUpdated, &ds.LogsEnabled, &ds.LongDesc, &ds.LongDesc1,
			&ds.LongDesc2, &ds.MaxDNSAnswers, &ds.MidHeaderRewrite, &ds.MissLat, &ds.MissLong, &ds.MultiSiteOrigin, &ds.OrgServerFQDN, &ds.OriginShield, &ds.ProfileID, &ds.ProfileName, &ds.ProfileDesc, &ds.Protocol, &ds.QStringIgnore, &ds.RangeRequestHandling, &ds.RegexRemap, &ds.RegionalGeoBlocking, &ds.RemapText, &ds.RoutingName, &ds.SigningAlgorithm, &ds.SSLKeyVersion, &ds.TenantID, &ds.Tenant, &ds.TRRequestHeaders, &ds.TRResponseHeaders, &ds.Type, &ds.TypeID, &ds.XMLID)
		if err != nil {
			return nil, errors.New("scanning user delivery services : " + err.Error())
		}
		if ds.DeepCachingType != nil {
			*ds.DeepCachingType = tc.DeepCachingTypeFromString(string(*ds.DeepCachingType))
		}
		dses = append(dses, ds)
	}
	return dses, nil
}

func getUserAvailableDSes(tx *sql.Tx, userID int) ([]tc.UserAvailableDS, error) {
	q := `
SELECT
ds.id,
ds.display_name,
ds.xml_id,
ds.tenant_id
FROM deliveryservice as ds
JOIN deliveryservice_tmuser dsu ON ds.id = dsu.deliveryservice
WHERE dsu.tm_user_id = $1
`
	rows, err := tx.Query(q, userID)
	if err != nil {
		return nil, errors.New("querying user available delivery services: " + err.Error())
	}
	defer rows.Close()
	dses := []tc.UserAvailableDS{}
	for rows.Next() {
		ds := tc.UserAvailableDS{}
		err := rows.Scan(&ds.ID, &ds.DisplayName, &ds.XMLID, &ds.TenantID)
		if err != nil {
			return nil, errors.New("scanning user available delivery services : " + err.Error())
		}
		dses = append(dses, ds)
	}
	return dses, nil
}
