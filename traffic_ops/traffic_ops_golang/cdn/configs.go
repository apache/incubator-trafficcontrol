package cdn

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
	"net/http"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

// TOCDNConf used as a type alias to define functions on to satisfy shared API REST interfaces.
type TOCDNConf struct {
	api.InfoerImpl `json:"-"`
}

func (v *TOCDNConf) NewReadObj() interface{} { return &tc.CDNConfig{} }
func (v *TOCDNConf) SelectQuery() string     { return cdnConfSelectQuery() }
func (v *TOCDNConf) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{}
}

func cdnConfSelectQuery() string {
	return `SELECT
name,
id
FROM cdn`
}

func (v *TOCDNConf) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	return api.GenericRead(h, v, useIMS)
}
func (v *TOCDNConf) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(last_updated) as t from federation ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='federation') as res`
}
func (v TOCDNConf) GetType() string {
	return "cdn_configs"
}
