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

	"github.com/lib/pq"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
)

func GetServerUpdateStatusHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"host_name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	serverUpdateStatus, err := getServerUpdateStatus(inf.Tx.Tx, inf.Config, inf.Params["host_name"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	if inf.Version.Major < 4 {
		api.WriteRespRaw(w, r, serverUpdateStatus)
	} else {
		api.WriteResp(w, r, serverUpdateStatus)
	}
}

func getServerUpdateStatus(tx *sql.Tx, cfg *config.Config, hostName string) ([]tc.ServerUpdateStatus, error) {

	updateStatuses := []tc.ServerUpdateStatus{}

	selectQuery := `
/* topology_ancestors finds the ancestor topology nodes of the topology node for
 * the cachegroup containing server $5.
 */
WITH RECURSIVE topology_ancestors AS (
/* This is the base case of the recursive CTE, the topology node for the
 * cachegroup containing server $5.
 */
	SELECT tcp.child parent, NULL cachegroup, s.id base_server_id
	FROM "server" s
	JOIN cachegroup c ON s.cachegroup = c.id
	JOIN topology_cachegroup tc ON c."name" = tc.cachegroup
	JOIN topology_cachegroup_parents tcp ON tc.id = tcp.child
	WHERE s.host_name = $5
UNION ALL
/* Find all direct topology parent nodes tc of a given topology ancestor ta. */
	SELECT tcp.parent, tc.cachegroup, ta.base_server_id
	FROM topology_ancestors ta, topology_cachegroup_parents tcp
	JOIN topology_cachegroup tc ON tcp.parent = tc.id
	JOIN cachegroup c ON tc.cachegroup = c."name"
	JOIN "type" t ON c."type" = t.id
	WHERE ta.parent = tcp.child
	AND t."name" LIKE ANY($4::TEXT[])
/* server_topology_ancestors is the set of every server whose cachegroup is an
 * ancestor topology node found by topology_ancestors.
 */
), server_topology_ancestors AS (
SELECT s.id, s.cachegroup, s.cdn_id, s.upd_pending, s.reval_pending, s.status, ta.base_server_id
	FROM server s
	JOIN cachegroup c ON s.cachegroup = c.id
	JOIN topology_ancestors ta ON c."name" = ta.cachegroup
	JOIN status ON status.id = s.status
	WHERE status.name = ANY($1::TEXT[])
), parentservers AS (
SELECT ps.id, ps.cachegroup, ps.cdn_id, ps.upd_pending, ps.reval_pending, ps.status
		FROM server ps
	LEFT JOIN status AS pstatus ON pstatus.id = ps.status
	LEFT JOIN type t ON ps."type" = t.id
	WHERE pstatus.name = ANY($1::TEXT[])
	AND t."name" LIKE ANY($4::TEXT[])
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
		SELECT sta.upd_pending FROM server_topology_ancestors sta
		WHERE sta.base_server_id = s.id
		AND sta.cdn_id = s.cdn_id
		UNION SELECT COALESCE(BOOL_OR(ps.upd_pending), FALSE)
	) AS parent_upd_pending,
	/* True if the cachegroup parent or any ancestor topology node has pending revalidation. */
	TRUE IN (
		SELECT sta.reval_pending FROM server_topology_ancestors sta
		WHERE sta.base_server_id = s.id
		AND sta.cdn_id = s.cdn_id
		UNION SELECT COALESCE(BOOL_OR(ps.reval_pending), FALSE)
	) AS parent_reval_pending
	FROM use_reval_pending,
		 server s
LEFT JOIN status ON s.status = status.id
LEFT JOIN cachegroup cg ON s.cachegroup = cg.id
LEFT JOIN type ON type.id = s.type
LEFT JOIN parentservers ps ON ps.cachegroup = cg.parent_cachegroup_id
	AND ps.cdn_id = s.cdn_id
WHERE s.host_name = $5
GROUP BY s.id, s.host_name, type.name, server_reval_pending, use_reval_pending.value, s.upd_pending, status.name
ORDER BY s.id
`

	cacheStatusesToCheck := []tc.CacheStatus{tc.CacheStatusOnline, tc.CacheStatusReported, tc.CacheStatusAdminDown}
	cacheGroupTypes := []string{tc.EdgeTypePrefix + "%", tc.MidTypePrefix + "%"}
	rows, err := tx.Query(selectQuery, pq.Array(cacheStatusesToCheck), tc.UseRevalPendingParameterName, tc.GlobalConfigFileName, pq.Array(cacheGroupTypes), hostName)
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
