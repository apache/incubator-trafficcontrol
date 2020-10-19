package cachegroup

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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	"github.com/lib/pq"
)

func DSPostHandler(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, []string{"id"}, []string{"id"})
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	req := tc.CachegroupPostDSReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	vals := map[string]interface{}{
		"alerts": tc.CreateAlerts(tc.SuccessLevel, "Delivery services successfully assigned to all the servers of cache group "+strconv.Itoa(inf.IntParams["id"])+".").Alerts,
	}

	resp, errs := postDSes(inf.Tx.Tx, inf.User, inf.IntParams["id"], req.DeliveryServices)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	api.WriteRespVals(w, r, resp, vals)
}

// postDSes returns the post response, any user error, any system error, and the HTTP status code to be returned in the event of an error.
func postDSes(tx *sql.Tx, user *auth.CurrentUser, cgID int, dsIDs []int) (tc.CacheGroupPostDSResp, apierrors.Errors) {
	var resp tc.CacheGroupPostDSResp
	cdnName, errs := getCachegroupCDN(tx, cgID)
	if errs.Occurred() {
		if errs.SystemError != nil {
			errs.SetSystemError("getting cachegroup CDN: " + errs.SystemError.Error())
		}
		return resp, errs
	}

	tenantIDs, err := getDSTenants(tx, dsIDs)
	if err != nil {
		errs.SetSystemError("getting delivery service tenant IDs: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return resp, errs
	}
	for _, tenantID := range tenantIDs {
		ok, err := tenant.IsResourceAuthorizedToUserTx(int(tenantID), user, tx)
		if err != nil {
			errs.SetSystemError("checking tenancy: " + err.Error())
			errs.Code = http.StatusInternalServerError
			return resp, errs
		}
		if !ok {
			errs.UserError = fmt.Errorf("not authorized for delivery service tenant %d", tenantID)
			errs.Code = http.StatusForbidden
			return resp, errs
		}
	}

	cgName, ok, err := dbhelpers.GetCacheGroupNameFromID(tx, cgID)
	if err != nil {
		errs.SystemError = fmt.Errorf("getting cachegroup name from ID %d: %s", cgID, err.Error())
		errs.Code = http.StatusInternalServerError
		return resp, errs
	} else if !ok {
		errs.UserError = fmt.Errorf("cachegroup %d does not exist", cgID)
		errs.Code = http.StatusNotFound
		return resp, errs
	}

	topologyDSes, err := dbhelpers.GetDeliveryServicesWithTopologies(tx, dsIDs)
	if err != nil {
		errs.SetSystemError("getting delivery services with topologies: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return resp, errs
	}
	if len(topologyDSes) > 0 {
		errs.UserError = fmt.Errorf("delivery services %v are already assigned to a topology", topologyDSes)
		errs.Code = http.StatusBadRequest
		return resp, errs
	}

	if err := verifyDSesCDN(tx, dsIDs, cdnName); err != nil {
		errs.SetSystemError("verifying delivery service CDNs match cachegroup server CDNs: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return resp, errs
	}
	cgServers, err := getCachegroupServers(tx, cgID)
	if err != nil {
		errs.SetSystemError("getting cachegroup server names " + err.Error())
		errs.Code = http.StatusInternalServerError
		return resp, errs
	}
	if err := insertCachegroupDSes(tx, cgID, dsIDs); err != nil {
		errs.SetSystemError("inserting cachegroup delivery services: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return resp, errs
	}

	if err := updateParams(tx, dsIDs); err != nil {
		errs.SetSystemError("updating delivery service parameters: " + err.Error())
		errs.Code = http.StatusInternalServerError
	}
	api.CreateChangeLogRawTx(api.ApiChange, "CACHEGROUP: "+string(cgName)+", ID: "+strconv.Itoa(cgID)+", ACTION: Assign DSes to CacheGroup servers", user, tx)
	resp = tc.CacheGroupPostDSResp{
		ID:               util.JSONIntStr(cgID),
		ServerNames:      cgServers,
		DeliveryServices: dsIDs,
	}
	return resp, errs
}

func insertCachegroupDSes(tx *sql.Tx, cgID int, dsIDs []int) error {
	_, err := tx.Exec(`
INSERT INTO deliveryservice_server (deliveryservice, server) (
  SELECT unnest($1::int[]), server.id
  FROM server
  JOIN type on type.id = server.type
  WHERE server.cachegroup = $2
  AND (type.name LIKE 'EDGE%' OR type.name LIKE 'ORG%')
) ON CONFLICT DO NOTHING
`, pq.Array(dsIDs), cgID)
	if err != nil {
		return errors.New("inserting cachegroup servers: " + err.Error())
	}
	return nil
}

func getCachegroupServers(tx *sql.Tx, cgID int) ([]tc.CacheName, error) {
	q := `
SELECT server.host_name FROM server
JOIN type on type.id = server.type
WHERE server.cachegroup = $1
AND (type.name LIKE 'EDGE%' OR type.name LIKE 'ORG%')
`
	rows, err := tx.Query(q, cgID)
	if err != nil {
		return nil, errors.New("selecting cachegroup servers: " + err.Error())
	}
	defer rows.Close()
	names := []tc.CacheName{}
	for rows.Next() {
		name := ""
		if err := rows.Scan(&name); err != nil {
			return nil, errors.New("querying cachegroup server names: " + err.Error())
		}
		names = append(names, tc.CacheName(name))
	}
	return names, nil
}

func verifyDSesCDN(tx *sql.Tx, dsIDs []int, cdn string) error {
	q := `
SELECT count(cdn.name)
FROM cdn
JOIN deliveryservice as ds on ds.cdn_id = cdn.id
WHERE ds.id = ANY($1::bigint[])
AND cdn.name <> $2::text
`
	count := 0
	if err := tx.QueryRow(q, pq.Array(dsIDs), cdn).Scan(&count); err != nil {
		return errors.New("querying cachegroup CDNs: " + err.Error())
	}
	if count > 0 {
		return errors.New("servers/deliveryservices do not belong to same cdn '" + cdn + "'")
	}
	return nil
}

func getCachegroupCDN(tx *sql.Tx, cgID int) (string, apierrors.Errors) {
	q := `
SELECT cdn.name
FROM cdn
JOIN server on server.cdn_id = cdn.id
JOIN type on server.type = type.id
WHERE server.cachegroup = $1
AND (type.name LIKE 'EDGE%' OR type.name LIKE 'ORG%')
`
	errs := apierrors.New()
	rows, err := tx.Query(q, cgID)
	if err != nil {
		errs.SetSystemError("selecting cachegroup CDNs: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return "", errs
	}
	defer rows.Close()
	cdn := ""
	for rows.Next() {
		serverCDN := ""
		if err := rows.Scan(&serverCDN); err != nil {
			errs.SetSystemError("scanning cachegroup CDN: " + err.Error())
			errs.Code = http.StatusInternalServerError
		}
		if cdn == "" {
			cdn = serverCDN
		}
		if cdn != serverCDN {
			errs.SetSystemError("cachegroup servers have different CDNs '" + cdn + "' and '" + serverCDN + "'")
			errs.Code = http.StatusInternalServerError
			return "", errs
		}
	}
	if cdn == "" {
		errs.UserError = fmt.Errorf("no edge or origin servers found on cachegroup %d", cgID)
		errs.Code = http.StatusBadRequest
	}
	return cdn, errs
}

// updateParams updated the header rewrite, cacheurl, and regex remap params for the given edge caches, on the given delivery services. NOTE it does not update Mid params.
func updateParams(tx *sql.Tx, dsIDs []int) error {
	if err := updateDSParam(tx, dsIDs, "hdr_rw_", "edge_header_rewrite"); err != nil {
		return err
	}
	if err := updateDSParam(tx, dsIDs, "cacheurl_", "cacheurl"); err != nil {
		return err
	}
	if err := updateDSParam(tx, dsIDs, "regex_remap_", "regex_remap"); err != nil {
		return err
	}
	return nil
}

func updateDSParam(tx *sql.Tx, dsIDs []int, paramPrefix string, dsField string) error {
	_, err := tx.Exec(`
DELETE FROM parameter
WHERE name = 'location'
AND config_file IN (
  SELECT CONCAT('`+paramPrefix+`', xml_id, '.config')
  FROM deliveryservice as ds
  WHERE ds.id = ANY($1)
  AND (ds.`+dsField+` IS NULL OR ds.`+dsField+` = '')
)
`, pq.Array(dsIDs))
	if err != nil {
		return err
	}

	rows, err := tx.Query(`
WITH ats_config_location AS (
  SELECT TRIM(TRAILING '/' FROM value) as v FROM parameter WHERE name = 'location' AND config_file = 'remap.config'
)
INSERT INTO parameter (name, config_file, value) (
  SELECT
    'location' as name,
    CONCAT('`+paramPrefix+`', xml_id, '.config'),
    (select v from ats_config_location)
  FROM deliveryservice WHERE id = ANY($1)
) ON CONFLICT (name, config_file, value) DO UPDATE SET name = EXCLUDED.name RETURNING id
`, pq.Array(dsIDs))
	if err != nil {
		return errors.New("inserting parameters: " + err.Error())
	}
	ids := []int{}
	for rows.Next() {
		id := 0
		if err := rows.Scan(&id); err != nil {
			return errors.New("scanning inserted parameters: " + err.Error())
		}
		ids = append(ids, id)
	}

	_, err = tx.Exec(`
INSERT INTO profile_parameter (parameter, profile) (
  SELECT UNNEST($1::int[]), server.profile
  FROM server
  JOIN deliveryservice_server as dss ON dss.server = server.id
  JOIN deliveryservice as ds ON ds.id = dss.deliveryservice
  WHERE ds.id = ANY($2)
) ON CONFLICT DO NOTHING
`, pq.Array(ids), pq.Array(dsIDs))
	if err != nil {
		return errors.New("inserting profile parameters: " + err.Error())
	}
	return nil
}

func getDSTenants(tx *sql.Tx, dsIDs []int) ([]int, error) {
	q := `
SELECT COALESCE(tenant_id, 0) FROM deliveryservice
WHERE deliveryservice.id = ANY($1)
`
	rows, err := tx.Query(q, pq.Array(dsIDs))
	if err != nil {
		return nil, errors.New("selecting delivery service tenants: " + err.Error())
	}
	defer rows.Close()
	tenantIDs := []int{}
	for rows.Next() {
		id := 0
		if err := rows.Scan(&id); err != nil {
			return nil, errors.New("querying cachegroup delivery service tenants: " + err.Error())
		}
		tenantIDs = append(tenantIDs, id)
	}
	return tenantIDs, nil
}
