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
	"fmt"
	"net/http"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"

	"github.com/jmoiron/sqlx"
)

const RouterProfilePrefix = "CCR"

func selectMaxLastUpdatedQuery() string {
	return `SELECT max(t) from (
		SELECT max(profile.last_updated) as t FROM profile
JOIN cdn ON profile.cdn = cdn.id WHERE profile.name LIKE '` + RouterProfilePrefix + `%'
UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='profile') as res`
}

func getDomainsList(useIMS bool, header http.Header, tx *sqlx.Tx) ([]tc.Domain, error, int, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	domains := []tc.Domain{}

	q := `SELECT p.id, p.name, p.description, domain_name FROM profile AS p
	JOIN cdn ON p.cdn = cdn.id WHERE p.name LIKE '` + RouterProfilePrefix + `%'`

	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(tx, header, nil, selectMaxLastUpdatedQuery())
		if !runSecond {
			log.Debugln("IMS HIT")
			return domains, nil, http.StatusNotModified, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	rows, err := tx.Query(q)
	if err != nil {
		return nil, fmt.Errorf("querying for profile: %s", err), http.StatusInternalServerError, nil
	}
	defer rows.Close()

	for rows.Next() {

		d := tc.Domain{ParameterID: -1}
		err := rows.Scan(&d.ProfileID, &d.ProfileName, &d.ProfileDescription, &d.DomainName)
		if err != nil {
			return nil, fmt.Errorf("getting profile: %s", err), http.StatusInternalServerError, nil
		}
		domains = append(domains, d)
	}

	return domains, nil, http.StatusOK, &maxTime
}

func DomainsHandler(w http.ResponseWriter, r *http.Request) {
	useIMS := false
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	cfg, err := api.GetConfig(r.Context())
	if err != nil {
		log.Warnf("Couldnt get the config %v", err)
	}
	if cfg != nil {
		useIMS = cfg.UseIMS
	}

	domains, err, status, _ := getDomainsList(useIMS, r.Header, inf.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, err, err)
		return
	}
	w.WriteHeader(status)
	api.WriteResp(w, r, domains)
}
