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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	"github.com/lib/pq"
)

func DSPostHandlerV31(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	req := tc.CachegroupPostDSReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}

	resp, vals, userErr, sysErr, errCode := postDSes(inf.Tx.Tx, inf.User, inf.IntParams["id"], req.DeliveryServices)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	if err := updateDSParam(inf.Tx.Tx, req.DeliveryServices, "cacheurl_", "cacheurl"); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("updating delivery service parameters: "+err.Error()))
		return
	}

	if err, errCode := writeChangeLog(inf.Tx.Tx, inf.User, inf.IntParams["id"]); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, err)
		return
	}

	api.WriteRespVals(w, r, resp, vals)
}

func DSPostHandlerV40(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	req := tc.CachegroupPostDSReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}

	resp, vals, userErr, sysErr, errCode := postDSes(inf.Tx.Tx, inf.User, inf.IntParams["id"], req.DeliveryServices)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	if err, errCode := writeChangeLog(inf.Tx.Tx, inf.User, inf.IntParams["id"]); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, err)
	}

	api.WriteRespVals(w, r, resp, vals)
}

func writeChangeLog(tx *sql.Tx, user *auth.CurrentUser, cgID int) (error, int) {
	cgName, ok, err := dbhelpers.GetCacheGroupNameFromID(tx, cgID)
	if err != nil {
		return fmt.Errorf("getting cachegroup name from ID %d: %s", cgID, err.Error()), http.StatusInternalServerError
	} else if !ok {
		return fmt.Errorf("cachegroup %d does not exist", cgID), http.StatusNotFound
	}
	api.CreateChangeLogRawTx(api.ApiChange, "CACHEGROUP: "+string(cgName)+", ID: "+strconv.Itoa(cgID)+", ACTION: Assign DSes to CacheGroup servers", user, tx)
	return nil, 0
}

// postDSes returns the post response, any user error, any system error, and the HTTP status code to be returned in the event of an error.
func postDSes(tx *sql.Tx, user *auth.CurrentUser, cgID int, dsIDs []int) (tc.CacheGroupPostDSResp, map[string]interface{}, error, error, int) {
	cdnName, usrErr, sysErr, errCode := getCachegroupCDN(tx, cgID)
	if sysErr != nil {
		sysErr = errors.New("getting cachegroup CDN: " + sysErr.Error())
	}
	if usrErr != nil || sysErr != nil {
		return tc.CacheGroupPostDSResp{}, nil, usrErr, sysErr, errCode
	}

	tenantIDs, err := getDSTenants(tx, dsIDs)
	if err != nil {
		return tc.CacheGroupPostDSResp{}, nil, nil, errors.New("getting delivery service tenant IDs: " + err.Error()), http.StatusInternalServerError
	}
	for _, tenantID := range tenantIDs {
		ok, err := tenant.IsResourceAuthorizedToUserTx(int(tenantID), user, tx)
		if err != nil {
			return tc.CacheGroupPostDSResp{}, nil, nil, errors.New("checking tenancy: " + err.Error()), http.StatusInternalServerError
		}
		if !ok {
			return tc.CacheGroupPostDSResp{}, nil, fmt.Errorf("not authorized for delivery service tenant %d", tenantID), nil, http.StatusForbidden
		}
	}

	topologyDSes, err := dbhelpers.GetDeliveryServicesWithTopologies(tx, dsIDs)
	if err != nil {
		return tc.CacheGroupPostDSResp{}, nil, nil, errors.New("getting delivery services with topologies: " + err.Error()), http.StatusInternalServerError
	}
	if len(topologyDSes) > 0 {
		return tc.CacheGroupPostDSResp{}, nil, fmt.Errorf("delivery services %v are already assigned to a topology", topologyDSes), nil, http.StatusBadRequest
	}

	if err := verifyDSesCDN(tx, dsIDs, cdnName); err != nil {
		return tc.CacheGroupPostDSResp{}, nil, nil, errors.New("verifying delivery service CDNs match cachegroup server CDNs: " + err.Error()), http.StatusInternalServerError
	}
	cgServers, err := getCachegroupServers(tx, cgID)
	if err != nil {
		return tc.CacheGroupPostDSResp{}, nil, nil, errors.New("getting cachegroup server names " + err.Error()), http.StatusInternalServerError
	}
	if err := insertCachegroupDSes(tx, cgID, dsIDs); err != nil {
		return tc.CacheGroupPostDSResp{}, nil, nil, errors.New("inserting cachegroup delivery services: " + err.Error()), http.StatusInternalServerError
	}

	if err := updateParams(tx, dsIDs); err != nil {
		return tc.CacheGroupPostDSResp{}, nil, nil, errors.New("updating delivery service parameters: " + err.Error()), http.StatusInternalServerError
	}
	vals := map[string]interface{}{
		"alerts": tc.CreateAlerts(tc.SuccessLevel, "Delivery services successfully assigned to all the servers of cache group "+strconv.Itoa(cgID)+".").Alerts,
	}
	return tc.CacheGroupPostDSResp{ID: util.JSONIntStr(cgID), ServerNames: cgServers, DeliveryServices: dsIDs}, vals, nil, nil, http.StatusOK
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

func getCachegroupCDN(tx *sql.Tx, cgID int) (string, error, error, int) {
	q := `
SELECT cdn.name
FROM cdn
JOIN server on server.cdn_id = cdn.id
JOIN type on server.type = type.id
WHERE server.cachegroup = $1
AND (type.name LIKE 'EDGE%' OR type.name LIKE 'ORG%')
`
	rows, err := tx.Query(q, cgID)
	if err != nil {
		return "", nil, errors.New("selecting cachegroup CDNs: " + err.Error()), http.StatusInternalServerError
	}
	defer rows.Close()
	cdn := ""
	for rows.Next() {
		serverCDN := ""
		if err := rows.Scan(&serverCDN); err != nil {
			return "", nil, errors.New("scanning cachegroup CDN: " + err.Error()), http.StatusInternalServerError
		}
		if cdn == "" {
			cdn = serverCDN
		}
		if cdn != serverCDN {
			return "", nil, errors.New("cachegroup servers have different CDNs '" + cdn + "' and '" + serverCDN + "'"), http.StatusInternalServerError
		}
	}
	if cdn == "" {
		return "", fmt.Errorf("no edge or origin servers found on cachegroup %d", cgID), nil, http.StatusBadRequest
	}
	return cdn, nil, nil, http.StatusOK
}

// updateParams updated the header rewrite, and regex remap params for the given edge caches, on the given delivery services. NOTE it does not update Mid params.
func updateParams(tx *sql.Tx, dsIDs []int) error {
	if err := updateDSParam(tx, dsIDs, "hdr_rw_", "edge_header_rewrite"); err != nil {
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
