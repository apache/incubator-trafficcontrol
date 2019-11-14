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

import "database/sql"
import "encoding/json"
import "errors"
import "fmt"
import "net/http"
import "strconv"
import "strings"
import "time"

import "github.com/apache/trafficcontrol/lib/go-tc"
import "github.com/apache/trafficcontrol/lib/go-log"
import "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
import "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
import "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

type InvalidationJob struct {
	api.APIInfoImpl `json:"-"`
	tc.InvalidationJob
}

const readQuery = `
SELECT job.id,
       keyword,
       parameters,
       asset_url,
       start_time,
       u.username AS createdBy,
       ds.xml_id AS dsId
FROM job
JOIN tm_user u ON job.job_user = u.id
JOIN deliveryservice ds  ON job.job_deliveryservice = ds.id
`

const insertQuery = `
INSERT INTO job (
	agent,
	asset_type,
	asset_url,
	entered_time,
	job_deliveryservice,
	job_user,
	keyword,
	parameters,
	start_time,
	status)
VALUES (
	1::bigint,
	'file',
	(
		SELECT o.protocol::text || '://' || o.fqdn || rtrim(concat(':', o.port::text), ':')
		FROM origin o
		WHERE o.deliveryservice = $1
		AND o.is_primary
	) || $2,
	$3,
	$4,
	$5,
	'PURGE',
	$6,
	$7,
	1::bigint
)
RETURNING
	asset_url,
	(SELECT deliveryservice.xml_id
	 FROM deliveryservice
	 WHERE deliveryservice.id=job_deliveryservice) AS deliveryservice,
	id,
	(SELECT tm_user.username
	 FROM tm_user
	 WHERE tm_user.id=job_user) AS createdBy,
	keyword,
	parameters,
	start_time
`

const revalQuery = `
UPDATE server SET %s=TRUE
WHERE server.status NOT IN (
                             SELECT status.id
                             FROM status
                             WHERE name IN ('OFFLINE', 'PRE_PROD')
                           )
     AND server.profile IN (
                             SELECT profile_parameter.profile
                             FROM profile_parameter
                             WHERE profile_parameter.parameter IN (
                                                                    SELECT parameter.id
                                                                    FROM parameter
                                                                    WHERE parameter.name='location'
                                                                     AND parameter.config_file='regex_revalidate.config'
                                                                  )
                           )
     AND server.cdn_id  =  (
                             SELECT deliveryservice.cdn_id
                             FROM deliveryservice
                             WHERE deliveryservice.%s=$1
                           )
`

const updateQuery = `
UPDATE job
SET asset_url=$1,
    keyword=$2,
    parameters=$3,
    start_time=$4
WHERE job.id=$5
RETURNING job.asset_url,
          (
           SELECT tm_user.username
           FROM tm_user
           WHERE tm_user.id=job.job_user
          ) AS created_by,
          (
           SELECT deliveryservice.xml_id
           FROM deliveryservice
           WHERE deliveryservice.id=job.job_deliveryservice
          ) AS delivery_service,
          job.id,
          job.keyword,
          job.parameters,
          job.start_time
`

const putInfoQuery = `
SELECT job.id AS id,
       tm_user.username AS createdBy,
       job.job_user AS createdByID,
       job.job_deliveryservice AS dsid,
       deliveryservice.xml_id AS dsxmlid,
       job.asset_url AS assetURL,
       job.parameters,
       job.start_time AS start_time,
       origin.protocol || '://' || origin.fqdn || rtrim(concat(':', origin.port), ':') AS OFQDN
FROM job
INNER JOIN origin ON origin.deliveryservice=job.job_deliveryservice AND origin.is_primary
INNER JOIN tm_user ON tm_user.id=job.job_user
INNER JOIN deliveryservice ON deliveryservice.id=job.job_deliveryservice
WHERE job.id=$1
`

const deleteQuery = `
DELETE
FROM job
WHERE job.id=$1
RETURNING job.asset_url,
          (
           SELECT tm_user.username
           FROM tm_user
           WHERE tm_user.id=job.job_user
          ) AS created_by,
          (
           SELECT deliveryservice.xml_id
           FROM deliveryservice
           WHERE deliveryservice.id=job.job_deliveryservice
          ) AS deliveryservice,
          job.id,
          job.keyword,
          job.parameters,
          job.start_time
`

func (job InvalidationJob) GetType() string {
	return "invalidation_job"
}

func (job InvalidationJob) GetAuditName() string {
	if job.ID != nil {
		return strconv.Itoa(int(*job.ID))
	}
	return "unknown"
}

func (job InvalidationJob) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"id", api.GetIntKey}}
}

func (job InvalidationJob) GetKeys() (map[string]interface{}, bool) {
	if job.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *job.ID}, true
}

func (job *InvalidationJob) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	ii := uint64(i)
	job.ID = &ii
}

type apiResponse struct {
	Alerts   []tc.Alert         `json:"alerts,omitempty"`
	Response tc.InvalidationJob `json:"response,omitempty"`
}

// Used by GET requests to `/jobs`, simply returns a filtered list of
// content invalidation jobs according to the provided query parameters.
func (job *InvalidationJob) Read() ([]interface{}, error, error, int) {
	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"id":              dbhelpers.WhereColumnInfo{"job.id", api.IsInt},
		"keyword":         dbhelpers.WhereColumnInfo{"job.keyword", nil},
		"assetUrl":        dbhelpers.WhereColumnInfo{"job.asset_url", nil},
		"userId":          dbhelpers.WhereColumnInfo{"job.job_user", api.IsInt},
		"createdBy":       dbhelpers.WhereColumnInfo{`(SELECT tm_user.username FROM tm_user WHERE tm_user.id=job.job_user)`, nil},
		"deliveryService": dbhelpers.WhereColumnInfo{`(SELECT deliveryservice.xml_id FROM deliveryservice WHERE deliveryservice.id=job.job_deliveryservice)`, nil},
		"dsId":            dbhelpers.WhereColumnInfo{"job.job_deliveryservice", api.IsInt},
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(job.APIInfo().Params, queryParamsToSQLCols)
	if len(errs) > 0 {
		var b strings.Builder
		b.WriteString("Reading jobs:")
		for _, err := range errs {
			b.WriteString("\n\t")
			b.WriteString(err.Error())
		}

		return nil, nil, errors.New(b.String()), http.StatusInternalServerError
	}

	// TODO: check tenancy here
	query := readQuery + where + orderBy + pagination

	log.Debugln("generated job query: " + query)
	log.Debugf("executing with values: %++v\n", queryValues)

	returnable := []tc.InvalidationJob{}
	rows, err := job.APIInfo().Tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, nil, fmt.Errorf("querying: %v", err), http.StatusInternalServerError
	}
	defer rows.Close()

	for rows.Next() {
		j := tc.InvalidationJob{}
		err := rows.Scan(&j.ID,
			&j.Keyword,
			&j.Parameters,
			&j.AssetURL,
			&j.StartTime,
			&j.CreatedBy,
			&j.DeliveryService)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing db response: %v", err), http.StatusInternalServerError
		}

		returnable = append(returnable, j)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("Parsing db responses: %v", err), http.StatusInternalServerError
	}

	// This cannot be done in the scanning loop, because pq will throw an error if you try to make
	// another query before exhausting the rows returned by an earlier query
	filtered := []interface{}{}
	for _, r := range returnable {
		ok, err := IsUserAuthorizedToModifyDSXMLID(job.APIInfo(), *r.DeliveryService)
		if err != nil {
			return nil, nil, err, http.StatusInternalServerError
		} else if ok {
			filtered = append(filtered, r)
		}
	}

	return filtered, nil, nil, http.StatusOK
}

// Used by POST requests to `/jobs`, creates a new content invalidation job
// from the provided request body.
func Create(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	job := tc.InvalidationJobInput{}
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("Unable to parse Invalidation Job"), fmt.Errorf("parsing jobs/ POST: %v", err))
		return
	}

	w.Header().Set(tc.ContentType, tc.ApplicationJson)
	if err := job.Validate(inf.Tx.Tx); err != nil {
		response := tc.Alerts{
			[]tc.Alert{
				tc.Alert{
					Text:  err.Error(),
					Level: tc.ErrorLevel.String(),
				},
			},
		}

		resp, err := json.Marshal(response)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("Encoding bad request response: %v", err))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write(append(resp, '\n'))
		return
	}

	// Validate() would have already checked for deliveryservice existence and
	// parsed the ttl, so if either of these throws an error now, something
	// weird has happened
	dsid, err := job.DSID(nil)
	if err != nil {
		sysErr = fmt.Errorf("retrieving parsed DSID: %v", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	}
	var ttl uint
	if ttl, err = job.TTLHours(); err != nil {
		sysErr = fmt.Errorf("retrieving parsed TTL: %v", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	}

	if ok, err := IsUserAuthorizedToModifyDSID(inf, dsid); err != nil {
		sysErr = fmt.Errorf("Checking current user permissions for DS #%d: %v", dsid, err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	} else if !ok {
		userErr = fmt.Errorf("No such Delivery Service!")
		errCode = http.StatusNotFound
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, nil)
		return
	}

	row := inf.Tx.Tx.QueryRow(insertQuery,
		dsid,
		*job.Regex,
		time.Now(),
		dsid,
		inf.User.ID,
		fmt.Sprintf("TTL:%dh", ttl),
		(*job.StartTime).Time)

	result := tc.InvalidationJob{}
	err = row.Scan(&result.AssetURL,
		&result.DeliveryService,
		&result.ID,
		&result.CreatedBy,
		&result.Keyword,
		&result.Parameters,
		&result.StartTime)
	if err != nil {
		userErr, sysErr, errCode = api.ParseDBError(err)
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	if err := setRevalFlags(dsid, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("setting reval flags: %v", err))
		return
	}

	resp, err := json.Marshal(apiResponse{[]tc.Alert{{"Invalidation Job creation was successful", tc.SuccessLevel.String()}}, result})
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("Marshaling JSON: %v", err))
		return
	}

	w.Header().Set(http.CanonicalHeaderKey("location"), inf.Config.URL.Scheme+"://"+r.Host+"/api/1.4/jobs?id="+strconv.FormatUint(uint64(*result.ID), 10))
	w.WriteHeader(http.StatusOK)
	w.Write(append(resp, '\n'))

	api.CreateChangeLogRawTx(api.ApiChange, api.Created+"content invalidation job: #"+strconv.FormatUint(*result.ID, 10), inf.User, inf.Tx.Tx)
}

// Used by PUT requests to `/jobs`, replaces an existing content invalidation job
// with the provided request body.
func Update(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	var oFQDN string
	var dsid uint
	var uid uint
	job := tc.InvalidationJob{}
	row := inf.Tx.Tx.QueryRow(putInfoQuery, inf.Params["id"])
	err := row.Scan(&job.ID,
		&job.CreatedBy,
		&uid,
		&dsid,
		&job.DeliveryService,
		&job.AssetURL,
		&job.Parameters,
		&job.StartTime,
		&oFQDN)
	if err != nil {
		if err == sql.ErrNoRows {
			userErr = fmt.Errorf("No job by id '%s'!", inf.Params["id"])
			errCode = http.StatusNotFound
		} else {
			sysErr = fmt.Errorf("fetching job update info: %v", err)
			errCode = http.StatusInternalServerError
		}
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	if ok, err := IsUserAuthorizedToModifyDSID(inf, dsid); err != nil {
		sysErr = fmt.Errorf("Checking user permissions on DS #%d: %v", dsid, err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	} else if !ok {
		userErr = errors.New("No such Delivery Service!")
		errCode = http.StatusNotFound
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, nil)
		return
	}

	if ok, err := IsUserAuthorizedToModifyJobsMadeByUsername(inf, *job.CreatedBy); err != nil {
		sysErr = fmt.Errorf("Checking user permissions against user %s: %v", *job.CreatedBy, err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	} else if !ok {
		userErr = fmt.Errorf("No job by id '%s'!", inf.Params["id"])
		errCode = http.StatusNotFound
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, nil)
		return
	}

	input := tc.InvalidationJob{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		userErr = fmt.Errorf("Unable to parse input: %v", err)
		sysErr = fmt.Errorf("parsing input to PUT jobs?id=%s: %v", inf.Params["id"], err)
		errCode = http.StatusBadRequest
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	if err := input.Validate(); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
		return
	}

	if !strings.HasPrefix(*input.AssetURL, oFQDN) {
		userErr = fmt.Errorf("Cannot set asset URL that does not start with Delivery Service origin URL: %s", oFQDN)
		errCode = http.StatusBadRequest
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, nil)
		return
	}

	if job.StartTime.Before(time.Now()) {
		userErr = errors.New("Cannot modify a job that has already started!")
		errCode = http.StatusMethodNotAllowed
		w.Header().Set(http.CanonicalHeaderKey("allow"), "GET,HEAD,DELETE")
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, nil)
		return
	}

	if *job.DeliveryService != *input.DeliveryService {
		userErr = errors.New("Cannot change 'deliveryService' of existing invalidation job!")
		errCode = http.StatusConflict
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, nil)
		return
	}

	if *job.CreatedBy != *input.CreatedBy {
		userErr = errors.New("Cannot change 'createdBy' of existing invalidation jobs!")
		errCode = http.StatusConflict
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, nil)
		return
	}

	if *job.ID != *input.ID {
		userErr = errors.New("Cannot change an invalidation job 'id'!")
		errCode = http.StatusConflict
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, nil)
		return
	}

	row = inf.Tx.Tx.QueryRow(updateQuery,
		input.AssetURL,
		input.Keyword,
		input.Parameters,
		input.StartTime.Time,
		*job.ID)
	err = row.Scan(&job.AssetURL,
		&job.CreatedBy,
		&job.DeliveryService,
		&job.ID,
		&job.Keyword,
		&job.Parameters,
		&job.StartTime)
	if err != nil {
		sysErr = fmt.Errorf("Updating a job: %v", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	}

	if err = setRevalFlags(*job.DeliveryService, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("Setting reval flags: %v", err))
		return
	}

	response := apiResponse{
		[]tc.Alert{
			tc.Alert{
				"Content invalidation job updated",
				tc.SuccessLevel.String(),
			},
		},
		job,
	}

	resp, err := json.Marshal(response)
	if err != nil {
		sysErr = fmt.Errorf("encoding response: %v", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	}

	w.Header().Set(http.CanonicalHeaderKey("content-type"), tc.ApplicationJson)
	w.Write(append(resp, '\n'))

	api.CreateChangeLogRawTx(api.ApiChange, api.Updated+"content invalidation job: #"+strconv.FormatUint(*job.ID, 10), inf.User, inf.Tx.Tx)
}

// Used by DELETE requests to `/jobs`, deletes an existing content invalidation job
func Delete(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	var dsid uint
	var createdBy uint
	row := inf.Tx.Tx.QueryRow(`SELECT job_deliveryservice, job_user FROM job WHERE id=$1`, inf.Params["id"])
	if err := row.Scan(&dsid, &createdBy); err != nil {
		if err == sql.ErrNoRows {
			userErr = fmt.Errorf("No job by id '%s'!", inf.Params["id"])
			errCode = http.StatusNotFound
		} else {
			sysErr = fmt.Errorf("Getting info for job #%s: %v", inf.Params["id"], err)
			errCode = http.StatusInternalServerError
		}
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	if ok, err := IsUserAuthorizedToModifyDSID(inf, dsid); err != nil {
		sysErr = fmt.Errorf("Checking user permissions on DS #%d: %v", dsid, err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	} else if !ok {
		userErr = errors.New("No such Delivery Service!")
		errCode = http.StatusNotFound
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, nil)
		return
	}

	if ok, err := IsUserAuthorizedToModifyJobsMadeByUserID(inf, createdBy); err != nil {
		sysErr = fmt.Errorf("Checking user permissions against user %v: %v", createdBy, err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	} else if !ok {
		userErr = fmt.Errorf("No job by id '%s'!", inf.Params["id"])
		errCode = http.StatusNotFound
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, nil)
		return
	}

	result := tc.InvalidationJob{}
	row = inf.Tx.Tx.QueryRow(deleteQuery, inf.Params["id"])
	err := row.Scan(&result.AssetURL,
		&result.CreatedBy,
		&result.DeliveryService,
		&result.ID,
		&result.Keyword,
		&result.Parameters,
		&result.StartTime)
	if err != nil {
		sysErr = fmt.Errorf("deleting job #%s: %v", inf.Params["id"], err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	}

	if err = setRevalFlags(dsid, inf.Tx.Tx); err != nil {
		sysErr = fmt.Errorf("setting reval_pending after deleting job #%s: %v", inf.Params["id"], err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	}

	response := apiResponse{[]tc.Alert{tc.Alert{"Content invalidation job was deleted", tc.SuccessLevel.String()}}, result}
	resp, err := json.Marshal(response)
	if err != nil {
		sysErr = fmt.Errorf("encoding response: %v", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	}

	w.Header().Set(http.CanonicalHeaderKey("content-type"), tc.ApplicationJson)
	w.Write(append(resp, '\n'))

	api.CreateChangeLogRawTx(api.ApiChange, api.Deleted+"content invalidation job: #"+strconv.FormatUint(*result.ID, 10), inf.User, inf.Tx.Tx)
}

func setRevalFlags(d interface{}, tx *sql.Tx) error {
	var useReval string
	row := tx.QueryRow(`SELECT value FROM parameter WHERE name='use_reval_pending' AND config_file='global'`)
	if err := row.Scan(&useReval); err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		useReval = "0"
	}

	col := "reval_pending"
	if useReval == "0" {
		col = "upd_pending"
	}

	var q string
	switch t := d.(type) {
	case uint:
		q = fmt.Sprintf(revalQuery, col, "id")
	case string:
		q = fmt.Sprintf(revalQuery, col, "xml_id")
	default:
		return fmt.Errorf("Invalid type passed to 'setRevalFlags': %v", t)
	}

	row = tx.QueryRow(q, d)
	if err := row.Scan(); err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}

// Checks if the current user's (identified in the APIInfo) tenant has permissions to
// edit a Delivery Service. `ds` is expected to be the integral, unique identifer of the
// Delivery Service in question.
//
// This returns, in order, a boolean that indicates whether or not the current user
// has the required tenancy to modify the indicated Delivery Service, and an error
// indicating what, if anything, went wrong during the check.
// returned errors is not nil, otherwise its value is undefined.
//
// Note: If no such delivery service exists, the return values shall indicate that the
// user isn't authorized.
func IsUserAuthorizedToModifyDSID(inf *api.APIInfo, ds uint) (bool, error) {
	var t uint
	row := inf.Tx.Tx.QueryRow(`SELECT tenant_id FROM deliveryservice where id=$1`, ds)
	if err := row.Scan(&t); err != nil {
		if err == sql.ErrNoRows {
			return false, nil //I do this to conceal the existence of DSes for which the user has no permission to see
		}
		return false, err
	}

	return tenant.IsResourceAuthorizedToUserTx(int(t), inf.User, inf.Tx.Tx)
}

// Checks if the current user's (identified in the APIInfo) tenant has permissions to
// edit a Delivery Service. `ds` is expected to be the "xml_id" of the
// Delivery Service in question.
//
// This returns, in order, a boolean that indicates whether or not the current user
// has the required tenancy to modify the indicated Delivery Service, and an error
// indicating what, if anything, went wrong during the check.
// returned errors is not nil, otherwise its value is undefined.
//
// Note: If no such delivery service exists, the return values shall indicate that the
// user isn't authorized.
func IsUserAuthorizedToModifyDSXMLID(inf *api.APIInfo, ds string) (bool, error) {
	var t uint
	row := inf.Tx.Tx.QueryRow(`SELECT tenant_id FROM deliveryservice where xml_id=$1`, ds)
	if err := row.Scan(&t); err != nil {
		if err == sql.ErrNoRows {
			return false, nil //I do this to conceal the existence of DSes for which the user has no permission to see
		}
		return false, err
	}

	return tenant.IsResourceAuthorizedToUserTx(int(t), inf.User, inf.Tx.Tx)
}

// Checks if the current user's (identified in the APIInfo) tenant has permissions to
// edit on par with the user identified by `u`. `u` is expected to be the integral,
// unique identifer of the user in question (not the current, requesting user).
//
// This returns, in order, a boolean that indicates whether or not the current user
// has the required tenancy to modify the indicated Delivery Service, and an error
// indicating what, if anything, went wrong during the check.
// returned errors is not nil, otherwise its value is undefined.
//
// Note: If no such delivery service exists, the return values shall indicate that the
// user isn't authorized.
func IsUserAuthorizedToModifyJobsMadeByUserID(inf *api.APIInfo, u uint) (bool, error) {
	var t uint
	row := inf.Tx.Tx.QueryRow(`SELECT tenant_id FROM tm_user where id=$1`, u)
	if err := row.Scan(&t); err != nil {
		if err == sql.ErrNoRows {
			return false, nil //I do this to conceal the existence of DSes for which the user has no permission to see
		}
		return false, err
	}

	return tenant.IsResourceAuthorizedToUserTx(int(t), inf.User, inf.Tx.Tx)
}

// Checks if the current user's (identified in the APIInfo) tenant has permissions to
// edit on par with the user identified by `u`. `u` is expected to be the username of
// the user in question (not the current, requesting user).
//
// This returns, in order, a boolean that indicates whether or not the current user
// has the required tenancy to modify the indicated Delivery Service, and an error
// indicating what, if anything, went wrong during the check.
// returned errors is not nil, otherwise its value is undefined.
//
// Note: If no such delivery service exists, the return values shall indicate that the
// user isn't authorized.
func IsUserAuthorizedToModifyJobsMadeByUsername(inf *api.APIInfo, u string) (bool, error) {
	var t uint
	row := inf.Tx.Tx.QueryRow(`SELECT tenant_id FROM tm_user where username=$1`, u)
	if err := row.Scan(&t); err != nil {
		if err == sql.ErrNoRows {
			return false, nil //I do this to conceal the existence of DSes for which the user has no permission to see
		}
		return false, err
	}

	return tenant.IsResourceAuthorizedToUserTx(int(t), inf.User, inf.Tx.Tx)
}
