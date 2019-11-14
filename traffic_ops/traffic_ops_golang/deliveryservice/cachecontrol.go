package deliveryservice

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
	"time"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/jmoiron/sqlx"
)

func dsModified(tx *sqlx.Tx, lastModified time.Time, where string, whereParams map[string]interface{}) bool {
	return dbhelpers.ResourceModified(tx, lastModified, makeDSModifiedQry(where), whereParams)
}

func makeDSModifiedQry(where string) string {
	return `
SELECT
  MAX(GREATEST(
    ds.last_updated,
    type.last_updated,
    cdn.last_updated,
    profile.last_updated,
    tenant.last_updated,
    o.last_updated
  )) as last_updated
FROM
  deliveryservice AS ds
  JOIN type ON ds.type = type.id
  JOIN cdn ON ds.cdn_id = cdn.id
  LEFT JOIN profile ON ds.profile = profile.id
  LEFT JOIN tenant ON ds.tenant_id = tenant.id
  LEFT JOIN origin o ON (o.deliveryservice = ds.id AND o.is_primary)
` + where
}
