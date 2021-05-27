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
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	"github.com/lib/pq"
)

func GetMatches(w http.ResponseWriter, r *http.Request) {
	alerts := tc.CreateAlerts(tc.WarnLevel, "This endpoint is deprecated, please use /deliveryservices_regexes instead")

	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		userErr = api.LogErr(r, errCode, userErr, sysErr)
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, errCode, alerts)
		return
	}
	defer inf.Close()
	matches, err := getUserDSMatches(inf.Tx.Tx, inf.User.TenantID)
	if err != nil {
		userErr = api.LogErr(r, http.StatusInternalServerError, nil, fmt.Errorf("getting delivery service matches: %v", err))
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
		return
	}
	api.WriteAlertsObj(w, r, http.StatusOK, alerts, matches)
}

func getUserDSMatches(tx *sql.Tx, userTenantID int) ([]tc.DeliveryServicePatterns, error) {
	q := `
SELECT ds.xml_id, ds.remap_text, r.pattern
FROM deliveryservice as ds
JOIN deliveryservice_regex as dsr ON dsr.deliveryservice = ds.id
JOIN regex as r ON r.id = dsr.regex
WHERE ds.active = 'ACTIVE'
`
	qParams := []interface{}{}
	tenantIDs, err := tenant.GetUserTenantIDListTx(tx, userTenantID)
	if err != nil {
		return nil, errors.New("getting user tenant ID list: " + err.Error())
	}
	q += `
AND ds.tenant_id = ANY($1)
`
	qParams = append(qParams, pq.Array(tenantIDs))

	q += `
ORDER BY dsr.set_number
`

	rows, err := tx.Query(q, qParams...)
	if err != nil {
		return nil, errors.New("querying delivery service matches: " + err.Error())
	}
	defer rows.Close()

	matches := []tc.DeliveryServicePatterns{}
	matchRegexes := map[tc.DeliveryServiceName][]string{}
	for rows.Next() {
		ds := tc.DeliveryServiceName("")
		remapText := (*string)(nil)
		pattern := ""
		if err := rows.Scan(&ds, &remapText, &pattern); err != nil {
			return nil, errors.New("scanning delivery service matches: " + err.Error())
		}
		if remapText != nil && strings.HasPrefix(*remapText, `regex_map`) {
			matches = append(matches, tc.DeliveryServicePatterns{DSName: dsNameToUnderscores(ds), Patterns: []string{remapToMatch(*remapText)}})
		} else {
			matchRegexes[ds] = append(matchRegexes[ds], regexToMatch(pattern))
		}
	}
	for ds, dsMatches := range matchRegexes {
		matches = append(matches, tc.DeliveryServicePatterns{DSName: dsNameToUnderscores(ds), Patterns: dsMatches})
	}
	return matches, nil
}

// dsNameToUnderscores changes delivery service name (xml_id) hyphens to underscores, to emulate the behavior of the old Perl Traffic Ops API.
func dsNameToUnderscores(ds tc.DeliveryServiceName) tc.DeliveryServiceName {
	return tc.DeliveryServiceName(strings.Replace(string(ds), `-`, `_`, -1))
}

func remapToMatch(regex string) string {
	// TODO: emulates old Perl behavior; verify correctness and usefulness
	regex = strings.TrimPrefix(regex, `regex_map http://`)
	hyphenI := strings.Index(regex, `-`)
	if hyphenI > 0 {
		regex = regex[:hyphenI+1]
	}
	return regex
}

func regexToMatch(remap string) string {
	remap = strings.Replace(remap, `\`, ``, -1)
	remap = strings.Replace(remap, `.*`, ``, -1)
	return remap
}
