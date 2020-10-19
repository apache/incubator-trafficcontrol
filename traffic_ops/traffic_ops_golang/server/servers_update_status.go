package server

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
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
)

func GetServerUpdateStatusHandler(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, []string{"host_name"}, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	serverUpdateStatus, err := getServerUpdateStatus(inf.Tx.Tx, inf.Config, inf.Params["host_name"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	api.WriteRespRaw(w, r, serverUpdateStatus)
}

func getServerUpdateStatus(tx *sql.Tx, cfg *config.Config, hostName string) ([]tc.ServerUpdateStatus, error) {

	updateStatuses := []tc.ServerUpdateStatus{}

	selectQuery := `
/* topology_ancestors finds the ancestor topology nodes of the topology node for
 * the cachegroup containing server $4.
 */
WITH RECURSIVE topology_ancestors AS (
/* This is the base case of the recursive CTE, the topology node for the
 * cachegroup containing server $4.
 */
	SELECT tcp.child parent, NULL cachegroup, s.id base_server_id
	FROM "server" s
	JOIN cachegroup c ON s.cachegroup = c.id
	JOIN topology_cachegroup tc ON c."name" = tc.cachegroup
	JOIN topology_cachegroup_parents tcp ON tc.id = tcp.child
	WHERE s.host_name = $4
UNION ALL
/* Find all direct topology parent nodes tc of a given topology ancestor ta. */
	SELECT tcp.parent, tc.cachegroup, ta.base_server_id
	FROM topology_ancestors ta, topology_cachegroup_parents tcp
	JOIN topology_cachegroup tc ON tcp.parent = tc.id
	WHERE ta.parent = tcp.child
/* server_topology_ancestors is the set of every server whose cachegroup is an
 * ancestor topology node found by topology_ancestors.
 */
), server_topology_ancestors AS (
SELECT s.id, s.cachegroup, s.cdn_id, s.upd_pending, s.reval_pending, s.status, ta.base_server_id
FROM server s
JOIN cachegroup c ON s.cachegroup = c.id
JOIN topology_ancestors ta ON c."name" = ta.cachegroup
), parentservers AS (
	SELECT ps.id, ps.cachegroup, ps.cdn_id, ps.upd_pending, ps.reval_pending, ps.status
		FROM server ps
	LEFT JOIN status AS pstatus ON pstatus.id = ps.status
	WHERE pstatus.name != $1
), use_reval_pending AS (
	SELECT value::BOOLEAN
	FROM parameter
	WHERE name = $2
	AND config_file = $3
	UNION ALL SELECT FALSE FETCH FIRST 1 ROW ONLY
)
SELECT
	s.id,
	s.host_name,
	type.name AS type,
	(s.reval_pending::BOOLEAN) AS server_reval_pending,
	use_reval_pending.value,
	s.upd_pending,
	status.name AS status,
		/* True if the cachegroup parent or any ancestor topology node has pending updates. */
		TRUE IN (
			SELECT sta.upd_pending FROM server_topology_ancestors sta WHERE sta.base_server_id = s.id
			UNION SELECT COALESCE(BOOL_OR(ps.upd_pending), FALSE)
		) AS parent_upd_pending,
		/* True if the cachegroup parent or any ancestor topology node has pending revalidation. */
		TRUE IN (
			SELECT sta.reval_pending FROM server_topology_ancestors sta WHERE sta.base_server_id = s.id
			UNION SELECT COALESCE(BOOL_OR(ps.reval_pending), FALSE)
		) AS parent_reval_pending
	FROM use_reval_pending,
		 server s
LEFT JOIN status ON s.status = status.id
LEFT JOIN cachegroup cg ON s.cachegroup = cg.id
LEFT JOIN type ON type.id = s.type
LEFT JOIN parentservers ps ON ps.cachegroup = cg.parent_cachegroup_id
	AND ps.cdn_id = s.cdn_id
WHERE s.host_name = $4
GROUP BY s.id, s.host_name, type.name, server_reval_pending, use_reval_pending.value, s.upd_pending, status.name
ORDER BY s.id
`

	rows, err := tx.Query(selectQuery, tc.CacheStatusOffline, tc.UseRevalPendingParameterName, tc.GlobalConfigFileName, hostName)
	if err != nil {
		log.Errorf("could not execute query: %s\n", err)
		return nil, tc.DBError
	}
	defer log.Close(rows, "getServerUpdateStatus(): unable to close db connection")

	for rows.Next() {
		var us tc.ServerUpdateStatus
		var serverType string
		if err := rows.Scan(&us.HostId, &us.HostName, &serverType, &us.RevalPending, &us.UseRevalPending, &us.UpdatePending, &us.Status, &us.ParentPending, &us.ParentRevalPending); err != nil {
			log.Errorf("could not scan server update status: %s\n", err)
			return nil, tc.DBError
		}
		updateStatuses = append(updateStatuses, us)
	}
	return updateStatuses, nil
}

func GetServerUpdateStatusHandlerV2(w http.ResponseWriter, r *http.Request) {
	GetServerUpdateStatusHandlerV1(w, r)
}

func GetServerUpdateStatusHandlerV1(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, []string{"host_name"}, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	serverUpdateStatus, err := getServerUpdateStatusV1(inf.Tx.Tx, inf.Config, inf.Params["host_name"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	api.WriteRespRaw(w, r, serverUpdateStatus)
}

// getServerUpdateStatusV1 supports /servers/all/update_status (believed to be used nowhere) in addition to /servers/{host_name}/update_status.
func getServerUpdateStatusV1(tx *sql.Tx, cfg *config.Config, hostName string) ([]tc.ServerUpdateStatus, error) {
	// language=SQL
	baseSelectStatement := `
WITH parentservers AS (
	SELECT ps.id, ps.cachegroup, ps.cdn_id, ps.upd_pending, ps.reval_pending
	FROM server ps
	LEFT JOIN status AS pstatus ON pstatus.id = ps.status
	WHERE pstatus.name != $1
), use_reval_pending AS (
	SELECT value::BOOLEAN
	FROM parameter
	WHERE name = $2
	AND config_file = $3
	UNION ALL SELECT FALSE FETCH FIRST 1 ROW ONLY
)
SELECT
	s.id,
	s.host_name,
	type.name AS type,
	(s.reval_pending::BOOLEAN) AS server_reval_pending,
	use_reval_pending.value,
	s.upd_pending,
	status.name AS status,
	COALESCE(BOOL_OR(ps.upd_pending), FALSE) AS parent_upd_pending,
	COALESCE(BOOL_OR(ps.reval_pending), FALSE) AS parent_reval_pending
	FROM use_reval_pending,
		 server s
LEFT JOIN status ON s.status = status.id
LEFT JOIN cachegroup cg ON s.cachegroup = cg.id
LEFT JOIN type ON type.id = s.type
LEFT JOIN parentservers ps ON ps.cachegroup = cg.parent_cachegroup_id
	AND ps.cdn_id = s.cdn_id
	AND type.name = 'EDGE'
` // remove the EDGE reference if other server types should have their parents processed

	// language=SQL
	groupBy := `
GROUP BY s.id, s.host_name, type.name, server_reval_pending, use_reval_pending.value, s.upd_pending, status.name
ORDER BY s.id
`

	updateStatuses := []tc.ServerUpdateStatus{}
	var rows *sql.Rows
	var err error
	if hostName == "all" {
		rows, err = tx.Query(baseSelectStatement+groupBy, tc.CacheStatusOffline, tc.UseRevalPendingParameterName, tc.GlobalConfigFileName)
		if err != nil {
			log.Errorf("could not execute select server update status query: %s\n", err)
			return nil, tc.DBError
		}
	} else {
		rows, err = tx.Query(baseSelectStatement+` WHERE s.host_name = $4`+groupBy, tc.CacheStatusOffline, tc.UseRevalPendingParameterName, tc.GlobalConfigFileName, hostName)
		if err != nil {
			log.Errorf("could not execute select server update status by hostname query: %s\n", err)
			return nil, tc.DBError
		}
	}
	defer rows.Close()

	for rows.Next() {
		var us tc.ServerUpdateStatus
		var serverType string
		if err := rows.Scan(&us.HostId, &us.HostName, &serverType, &us.RevalPending, &us.UseRevalPending, &us.UpdatePending, &us.Status, &us.ParentPending, &us.ParentRevalPending); err != nil {
			log.Errorf("could not scan server update status: %s\n", err)
			return nil, tc.DBError
		}
		if hostName == "all" { //if we want to return the parent data for servers when all is used remove this block
			us.ParentRevalPending = false
			us.ParentPending = false
		}
		updateStatuses = append(updateStatuses, us)
	}
	return updateStatuses, nil
}
