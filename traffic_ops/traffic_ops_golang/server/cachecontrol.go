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
	"strings"
	"time"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/jmoiron/sqlx"
)

func serverModified(tx *sqlx.Tx, lastModified time.Time, where string, whereParams map[string]interface{}) bool {
	return dbhelpers.ResourceModified(tx, lastModified, makeServerModifiedQry(where), whereParams)
}

func makeServerModifiedQry(where string) string {
	maybeDSSLastUpdated := ``
	if strings.Contains(where, "deliveryservice_server") {
		maybeDSSLastUpdated += `dss.last_updated,`
	}
	return `
SELECT
  MAX(GREATEST(
    s.last_updated,
    cg.last_updated,
    cdn.last_updated,
    pl.last_updated,
    pr.last_updated,
    st.last_updated,
    ` + maybeDSSLastUpdated + `
    tp.last_updated
  )) AS last_updated_any
FROM
  server s
  JOIN cachegroup cg on cg.id = s.cachegroup
  JOIN cdn cdn on cdn.id = s.cdn_id
  JOIN phys_location pl on pl.id = s.phys_location
  JOIN profile pr on pr.id = s.profile
  JOIN status st on st.id = s.status
  JOIN type tp on tp.id = s.type
` + where
}
