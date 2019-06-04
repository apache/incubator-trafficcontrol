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

import "encoding/json"
import "fmt"
import "net/http"
import "strconv"
import "time"

import "github.com/apache/trafficcontrol/lib/go-tc"
import "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"

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

// Creates a new job for the current user (via POST request to `/user/current/jobs`)
// this uses its own special format encoded in the tc.UserInvalidationJobInput structure
func CreateUserJob(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	job := tc.UserInvalidationJobInput{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &job); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, fmt.Errorf("error parsing jobs POST body: %v", err))
		return
	}

	if userErr, sysErr, errCode = IsUserAuthorizedToModifyDSID(inf, *job.DSID); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
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
		api.HandleErr(w, r, inf.Tx.Tx, code, userErr, sysErr)
		return
	}

	if err := setRevalFlags(*job.DSID, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("setting reval flags: %v", err))
	} else {
		resp, err := json.Marshal(apiResponse{[]tc.Alert{{"Invalidation Job creation was successful.", tc.SuccessLevel.String()}}, result})
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("Marshaling JSON: %v", err))
		} else {
			w.Header().Set(http.CanonicalHeaderKey("content-type"), tc.ApplicationJson)
			w.Header().Set(http.CanonicalHeaderKey("location"), inf.Config.URL.Scheme+"://"+r.Host+"/api/1.4/jobs?id="+strconv.FormatUint(uint64(*result.ID), 10))
			w.WriteHeader(http.StatusCreated)
			w.Write(resp)
			w.Write([]byte("\n"))
		}
	}
	api.CreateChangeLogRawTx(api.ApiChange, api.Created+"content invalidation job: #"+strconv.FormatUint(*result.ID, 10), inf.User, inf.Tx.Tx)
}

// Gets all jobs that were created by the requesting user, and returns them in
// in a special format encoded in the tc.UserInvalidationJob structure
func GetUserJobs(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	rows, err := inf.Tx.Query(userReadQuery, inf.User.ID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("Fetching user jobs: %v", err))
		return
	}
	defer rows.Close()

	jobs := []tc.UserInvalidationJob{}
	for rows.Next() {
		var j tc.UserInvalidationJob
		err := rows.Scan(&j.Agent,
			&j.AssetType,
			&j.AssetURL,
			&j.Username,
			&j.DeliveryService,
			&j.EnteredTime,
			&j.ID,
			&j.Keyword,
			&j.ObjectName,
			&j.ObjectType,
			&j.Parameters)

		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("Parsing user job DB row: %v", err))
			return
		}

		jobs = append(jobs, j)
	}

	if err := rows.Err(); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("Parsing user job DB rows: %v", err))
		return
	}

	// This cannot be done in the scanning loop, because pq will throw an error if you try to make
	// another query before exhausting the rows returned by an earlier query
	filtered := []tc.UserInvalidationJob{}
	for _, j := range jobs {
		userErr, sysErr, errCode := IsUserAuthorizedToModifyDSXMLID(inf, *j.DeliveryService)
		if sysErr != nil {
			api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		} else if userErr == nil {
			filtered = append(filtered, j)
		}
	}

	resp, err := json.Marshal(struct {
		Response []tc.UserInvalidationJob `json:"response"`
	}{filtered})
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("encoding user jobs response: %v", err))
	} else {
		w.Header().Set(http.CanonicalHeaderKey("content-type"), tc.ApplicationJson)
		w.Write(resp)
		w.Write([]byte("\n"))
	}
}
