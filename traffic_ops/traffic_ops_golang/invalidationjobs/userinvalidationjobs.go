package invalidationjobs

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
	"strconv"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

const userReadQuery = `
SELECT job.agent,
       job.asset_url,
       job.asset_type,
       (
       	SELECT tm_user.username
       	FROM tm_user
       	WHERE tm_user.id=$1
       ) AS username,
       (
       	SELECT deliveryservice.xml_id
       	FROM deliveryservice
       	WHERE deliveryservice.id=job.job_deliveryservice
       ) AS deliveryservice,
       job.entered_time,
       job.id,
       job.keyword,
       job.object_name,
       job.object_type,
       job.parameters
FROM job
WHERE job.job_user=$1
`

type response struct {
	Alerts   []tc.Alert  `json:"alerts"`
	Response interface{} `json:"response"`
}

// Creates a new job for the current user (via POST request to `/user/current/jobs`)
// this uses its own special format encoded in the tc.UserInvalidationJobInput structure
func CreateUserJob(w http.ResponseWriter, r *http.Request) {
	alerts := tc.CreateAlerts(tc.WarnLevel, "This endpoint is deprecated, please use the POST method /jobs instead")

	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		userErr = api.LogErr(r, errCode, userErr, sysErr)
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, errCode, alerts)
		return
	}
	defer inf.Close()

	job := tc.UserInvalidationJobInput{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &job); err != nil {
		userErr = api.LogErr(r, http.StatusBadRequest, err, fmt.Errorf("error parsing jobs POST body: %v", err))
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, http.StatusBadRequest, alerts)
		return
	}

	if ok, err := IsUserAuthorizedToModifyDSID(inf, *job.DSID); err != nil {
		err = fmt.Errorf("Checking user permissions on DS #%d: %v", *job.DSID, err)
		errCode = http.StatusInternalServerError
		userErr = api.LogErr(r, errCode, nil, err)
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, errCode, alerts)
		return
	} else if !ok {
		userErr = api.LogErr(r, http.StatusNotFound, errors.New("No such Delivery Service!"), nil)
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, errCode, alerts)
		return
	}

	resultRow := inf.Tx.Tx.QueryRow(insertQuery,
		job.DSID,
		job.Regex,
		time.Now(),
		job.DSID,
		inf.User.ID,
		fmt.Sprintf("TTL:%dh", *job.TTL),
		job.StartTime.Time)

	result := tc.InvalidationJob{}
	err := resultRow.Scan(&result.AssetURL,
		&result.DeliveryService,
		&result.ID,
		&result.CreatedBy,
		&result.Keyword,
		&result.Parameters,
		&result.StartTime)
	if err != nil {
		userErr, sysErr, code := api.ParseDBError(err)
		userErr = api.LogErr(r, code, userErr, sysErr)
		if err := inf.Tx.Tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Errorln("rolling back transaction: " + err.Error())
		}
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, code, alerts)
		return
	}

	if err := setRevalFlags(*job.DSID, inf.Tx.Tx); err != nil {
		errCode = http.StatusInternalServerError
		alerts.AddNewAlert(tc.ErrorLevel, api.LogErr(r, errCode, nil, fmt.Errorf("setting reval flags: %v", err)).Error())
		if err := inf.Tx.Tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Errorln("rolling back transaction: " + err.Error())
		}
		api.WriteAlerts(w, r, errCode, alerts)
		return
	}

	alerts.AddNewAlert(tc.SuccessLevel, "Invalidation Job creation was successful")
	w.Header().Set(http.CanonicalHeaderKey("location"), inf.Config.URL.Scheme+"://"+r.Host+"/api/1.4/jobs?id="+strconv.FormatUint(uint64(*result.ID), 10))
	api.WriteAlertsObj(w, r, http.StatusOK, alerts, result)
	api.CreateChangeLogRawTx(api.ApiChange, api.Created+"content invalidation job: #"+strconv.FormatUint(*result.ID, 10), inf.User, inf.Tx.Tx)
}

// Gets all jobs that were created by the requesting user, and returns them in
// in a special format encoded in the tc.UserInvalidationJob structure
func GetUserJobs(w http.ResponseWriter, r *http.Request) {
	alerts := tc.CreateAlerts(tc.WarnLevel, "This endpoint is deprecated, please use the 'userId' or 'createdBy' query parameters of a GET request to /jobs instead")

	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		userErr = api.LogErr(r, errCode, userErr, sysErr)
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, errCode, alerts)
		return
	}
	defer inf.Close()

	rows, err := inf.Tx.Query(userReadQuery, inf.User.ID)
	if err != nil {
		userErr = api.LogErr(r, http.StatusInternalServerError, nil, fmt.Errorf("Fetching user jobs: %v", err))
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
		return
	}
	defer rows.Close()

	jobs := []tc.UserInvalidationJob{}
	for rows.Next() {
		var j tc.UserInvalidationJob
		err := rows.Scan(&j.Agent,
			&j.AssetURL,
			&j.AssetType,
			&j.Username,
			&j.DeliveryService,
			&j.EnteredTime,
			&j.ID,
			&j.Keyword,
			&j.ObjectName,
			&j.ObjectType,
			&j.Parameters)

		if err != nil {
			userErr = api.LogErr(r, http.StatusInternalServerError, nil, fmt.Errorf("Parsing user job DB row: %v", err))
			alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
			api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
			return
		}

		jobs = append(jobs, j)
	}

	if err := rows.Err(); err != nil {
		userErr = api.LogErr(r, http.StatusInternalServerError, nil, fmt.Errorf("Parsing user job DB rows: %v", err))
		api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
		return
	}

	// This cannot be done in the scanning loop, because pq will throw an error if you try to make
	// another query before exhausting the rows returned by an earlier query
	filtered := []tc.UserInvalidationJob{}
	for _, j := range jobs {
		ok, err := IsUserAuthorizedToModifyDSXMLID(inf, *j.DeliveryService)
		if err != nil {
			errCode = http.StatusInternalServerError
			userErr = api.LogErr(r, errCode, nil, fmt.Errorf("Checking user permissions for DS %s: %v", *j.DeliveryService, err))
			api.WriteAlerts(w, r, errCode, alerts)
			return
		} else if ok {
			filtered = append(filtered, j)
		}
	}

	api.WriteAlertsObj(w, r, http.StatusOK, alerts, filtered)
}
