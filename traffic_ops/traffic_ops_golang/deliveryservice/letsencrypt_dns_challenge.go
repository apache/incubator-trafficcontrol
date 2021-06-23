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
	"errors"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/jmoiron/sqlx"
)

type DnsRecord struct {
	Fqdn   *string `json:"fqdn" db:"fqdn"`
	Record *string `json:"record" db:"record"`
}

func GetDnsChallengeRecords(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	getQuery := `SELECT fqdn, record FROM dnschallenges`

	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		"fqdn": dbhelpers.WhereColumnInfo{Column: "fqdn"},
	}

	where, _, _, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(inf.Params, queryParamsToQueryCols)
	if len(errs) > 0 {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, util.JoinErrs(errs))
		return
	}
	getQuery += where

	dnsRecord, err := getDnsRecords(inf.Tx, getQuery, queryValues)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking dns records: "+err.Error()))
		return
	}
	api.WriteResp(w, r, dnsRecord)
}

func getDnsRecords(tx *sqlx.Tx, getQuery string, queryValues map[string]interface{}) ([]DnsRecord, error) {
	records := []DnsRecord{}
	rows, err := tx.NamedQuery(getQuery, queryValues)
	if err != nil {
		return nil, errors.New("getting dns challenge records: " + err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		record := DnsRecord{}
		if err := rows.Scan(&record.Fqdn, &record.Record); err != nil {
			return nil, errors.New("scanning dns challenge records: " + err.Error())
		}
		records = append(records, record)
	}

	return records, nil
}
