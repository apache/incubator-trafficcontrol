package federations

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
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

func GetAll(w http.ResponseWriter, r *http.Request) {
	var maxTime *time.Time
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	feds := []FedInfo{}
	err := error(nil)
	allFederations := []tc.IAllFederation{}

	useIMS := false
	config, e := api.GetConfig(r.Context())
	if e == nil && config != nil {
		useIMS = config.UseIMS
	} else {
		log.Warnf("Couldn't get config %v", e)
	}
	code := http.StatusOK

	if cdnParam, ok := inf.Params["cdnName"]; ok {
		cdnName := tc.CDNName(cdnParam)
		feds, err, code, maxTime = getAllFederationsForCDN(inf.Tx.Tx, cdnName, useIMS, r.Header)
		if code == http.StatusNotModified {
			if maxTime != nil && api.SetLastModifiedHeader(r, useIMS) {
				// RFC1123
				date := maxTime.Format("Mon, 02 Jan 2006 15:04:05 MST")
				w.Header().Add(rfc.LastModified, date)
			}
			w.WriteHeader(code)
			api.WriteResp(w, r, tc.AllFederationCDN{})
			return
		}
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("federations.GetAll getting all federations: "+err.Error()))
			return
		}
		allFederations = append(allFederations, tc.AllFederationCDN{CDNName: &cdnName})
	} else {
		feds, err, code, maxTime = getAllFederations(inf.Tx.Tx, useIMS, r.Header)
		if code == http.StatusNotModified {
			if maxTime != nil && api.SetLastModifiedHeader(r, useIMS) {
				// RFC1123
				date := maxTime.Format("Mon, 02 Jan 2006 15:04:05 MST")
				w.Header().Add(rfc.LastModified, date)
			}
			w.WriteHeader(code)
			api.WriteResp(w, r, tc.AllFederationCDN{})
			return
		}
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("federations.GetAll getting all federations by CDN: "+err.Error()))
			return
		}
	}

	fedsResolvers, err, code, maxTime := getFederationResolvers(inf.Tx.Tx, fedInfoIDs(feds), useIMS, r.Header)
	if code == http.StatusNotModified {
		if maxTime != nil && api.SetLastModifiedHeader(r, useIMS) {
			// RFC1123
			date := maxTime.Format("Mon, 02 Jan 2006 15:04:05 MST")
			w.Header().Add(rfc.LastModified, date)
		}
		w.WriteHeader(code)
		api.WriteResp(w, r, []tc.IAllFederation{})
		return
	}
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("federations.Get getting federations resolvers: "+err.Error()))
		return
	}
	allFederations = addResolvers(allFederations, feds, fedsResolvers)

	api.WriteResp(w, r, allFederations)
}

func getAllFederations(tx *sql.Tx, useIMS bool, header http.Header) ([]FedInfo, error, int, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	qry := `
SELECT
  fds.federation,
  fd.ttl,
  fd.cname,
  ds.xml_id
FROM
  federation_deliveryservice fds
  JOIN deliveryservice ds ON ds.id = fds.deliveryservice
  JOIN federation fd ON fd.id = fds.federation
ORDER BY
  ds.xml_id
`
	imsQuery := `SELECT max(t) from (
		SELECT max(fds.last_updated) as t FROM
  federation_deliveryservice fds
  JOIN deliveryservice ds ON ds.id = fds.deliveryservice
  JOIN federation fd ON fd.id = fds.federation
	UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='federation_deliveryservice') as res`

	if useIMS {
		runSecond, maxTime = tryIfModifiedSinceQuery(header, tx, "", imsQuery)
		if !runSecond {
			log.Debugln("IMS HIT")
			return []FedInfo{}, nil, http.StatusNotModified, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	rows, err := tx.Query(qry)
	if err != nil {
		return nil, errors.New("all federations querying: " + err.Error()), http.StatusInternalServerError, nil
	}
	defer rows.Close()

	feds := []FedInfo{}
	for rows.Next() {
		f := FedInfo{}
		if err := rows.Scan(&f.ID, &f.TTL, &f.CName, &f.DS); err != nil {
			return nil, errors.New("all federations scanning: " + err.Error()), http.StatusInternalServerError, nil
		}
		feds = append(feds, f)
	}
	return feds, nil, http.StatusOK, &maxTime
}

func getAllFederationsForCDN(tx *sql.Tx, cdn tc.CDNName, useIMS bool, header http.Header) ([]FedInfo, error, int, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	qry := `
SELECT
  fds.federation,
  fd.ttl,
  fd.cname,
  ds.xml_id
FROM
  federation_deliveryservice fds
  JOIN deliveryservice ds ON ds.id = fds.deliveryservice
  JOIN federation fd ON fd.id = fds.federation
  JOIN cdn on cdn.id = ds.cdn_id
WHERE
  cdn.name = $1
ORDER BY
  ds.xml_id
`

	imsQuery := `SELECT max(t) from (
		SELECT max(fds.last_updated) as t from federation_deliveryservice fds
	JOIN deliveryservice ds ON ds.id = fds.deliveryservice
	JOIN federation fd ON fd.id = fds.federation
	JOIN cdn on cdn.id = ds.cdn_id
	WHERE cdn.name = $1
	UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='federation_deliveryservice') as res`

	if useIMS {
		runSecond, maxTime = tryIfModifiedSinceQuery(header, tx, string(cdn), imsQuery)
		if !runSecond {
			log.Debugln("IMS HIT")
			return []FedInfo{}, nil, http.StatusNotModified, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	rows, err := tx.Query(qry, cdn)
	if err != nil {
		return nil, errors.New("all federations for cdn querying: " + err.Error()), http.StatusInternalServerError, nil
	}
	defer rows.Close()

	feds := []FedInfo{}
	for rows.Next() {
		f := FedInfo{}
		if err := rows.Scan(&f.ID, &f.TTL, &f.CName, &f.DS); err != nil {
			return nil, errors.New("all federations for cdn scanning: " + err.Error()), http.StatusInternalServerError, nil
		}
		feds = append(feds, f)
	}
	return feds, nil, http.StatusOK, &maxTime
}

func tryIfModifiedSinceQuery(header http.Header, tx *sql.Tx, param string, imsQuery string) (bool, time.Time) {
	var max time.Time
	var imsDate time.Time
	var ok bool
	imsDateHeader := []string{}
	runSecond := true
	dontRunSecond := false
	if header == nil {
		return runSecond, max
	}
	imsDateHeader = header[rfc.IfModifiedSince]
	if len(imsDateHeader) == 0 {
		return runSecond, max
	}
	if imsDate, ok = rfc.ParseHTTPDate(imsDateHeader[0]); !ok {
		log.Warnf("IMS request header date '%s' not parsable", imsDateHeader[0])
		return runSecond, max
	}

	var rows *sql.Rows
	var err error

	if param == "" {
		rows, err = tx.Query(imsQuery)
	} else {
		rows, err = tx.Query(imsQuery, param)
	}

	if err != nil {
		log.Warnf("Couldn't get the max last updated time: %v", err)
		return runSecond, max
	}
	if err == sql.ErrNoRows {
		return dontRunSecond, max
	}
	defer rows.Close()
	// This should only ever contain one row
	if rows.Next() {
		v := tc.TimeNoMod{}
		if err = rows.Scan(&v); err != nil {
			log.Warnf("Failed to parse the max time stamp into a struct %v", err)
			return runSecond, max
		}
		max = v.Time
		// The request IMS time is later than the max of (lastUpdated, deleted_time)
		if imsDate.After(v.Time) {
			return dontRunSecond, max
		}
	}
	return runSecond, max
}
