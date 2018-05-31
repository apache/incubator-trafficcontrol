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
	"database/sql"
	"errors"
	"net/http"

	tc "github.com/apache/incubator-trafficcontrol/lib/go-tc/v13"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

func GetConfigs(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api.RespWriter(w, r)(getConfigs(db))
	}
}

func getConfigs(db *sql.DB) ([]tc.CDNConfig, error) {
	rows, err := db.Query(`SELECT name, id FROM cdn`)
	if err != nil {
		return nil, errors.New("querying cdn configs: " + err.Error())
	}
	cdns := []tc.CDNConfig{}
	defer rows.Close()
	for rows.Next() {
		c := tc.CDNConfig{}
		if err := rows.Scan(&c.Name, &c.ID); err != nil {
			return nil, errors.New("scanning cdn config: " + err.Error())
		}
		cdns = append(cdns, c)
	}
	return cdns, nil
}
