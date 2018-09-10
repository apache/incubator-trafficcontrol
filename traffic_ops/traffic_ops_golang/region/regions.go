package region

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
	"fmt"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

//we need a type alias to define functions on
type TORegion struct {
	ReqInfo *api.APIInfo `json:"-"`
	tc.Region
}

func GetTypeSingleton() api.CRUDFactory {
	return func(reqInfo *api.APIInfo) api.CRUDer {
		toReturn := TORegion{reqInfo, tc.Region{}}
		return &toReturn
	}
}

func (region TORegion) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"id", api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (region TORegion) GetKeys() (map[string]interface{}, bool) {
	return map[string]interface{}{"id": region.ID}, true
}

func (region *TORegion) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	region.ID = i
}

func (region *TORegion) GetAuditName() string {
	return region.Name
}

func (region *TORegion) GetType() string {
	return "region"
}

func (region *TORegion) Validate() error {
	if len(region.Name) < 1 {
		return errors.New(`Region 'name' is required.`)
	}
	return nil
}

func (region *TORegion) Read(parameters map[string]string) ([]interface{}, []error, tc.ApiErrorType) {
	var rows *sqlx.Rows

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		"name":     dbhelpers.WhereColumnInfo{"r.name", nil},
		"division": dbhelpers.WhereColumnInfo{"r.division", nil},
		"id":       dbhelpers.WhereColumnInfo{"r.id", api.IsInt},
	}
	where, orderBy, queryValues, errs := dbhelpers.BuildWhereAndOrderBy(parameters, queryParamsToQueryCols)
	if len(errs) > 0 {
		return nil, errs, tc.DataConflictError
	}

	query := selectQuery() + where + orderBy
	log.Debugln("Query is ", query)

	rows, err := region.ReqInfo.Txx.NamedQuery(query, queryValues)
	if err != nil {
		log.Errorf("Error querying Regions: %v", err)
		return nil, []error{tc.DBError}, tc.SystemError
	}
	defer rows.Close()

	regions := []interface{}{}
	for rows.Next() {
		var s tc.Region
		if err = rows.StructScan(&s); err != nil {
			log.Errorf("error parsing Region rows: %v", err)
			return nil, []error{tc.DBError}, tc.SystemError
		}
		regions = append(regions, s)
	}

	return regions, []error{}, tc.NoError
}

func selectQuery() string {

	query := `SELECT
r.division,
d.name as divisionname,
r.id,
r.last_updated,
r.name
FROM region r
JOIN division d ON r.division = d.id`
	return query
}

//The TORegion implementation of the Updater interface
//all implementations of Updater should use transactions and return the proper errorType
//ParsePQUniqueConstraintError is used to determine if a region with conflicting values exists
//if so, it will return an errorType of DataConflict and the type should be appended to the
//generic error message returned
func (region *TORegion) Update() (error, tc.ApiErrorType) {
	log.Debugf("about to run exec query: %s with region: %++v", updateQuery(), region)
	resultRows, err := region.ReqInfo.Txx.NamedQuery(updateQuery(), region)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			err, eType := dbhelpers.ParsePQUniqueConstraintError(pqErr)
			if eType == tc.DataConflictError {
				return errors.New("a region with " + err.Error()), eType
			}
			return err, eType
		}
		log.Errorf("received error: %++v from update execution", err)
		return tc.DBError, tc.SystemError
	}
	defer resultRows.Close()

	var lastUpdated tc.TimeNoMod
	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&lastUpdated); err != nil {
			log.Error.Printf("could not scan lastUpdated from insert: %s\n", err)
			return tc.DBError, tc.SystemError
		}
	}
	log.Debugf("lastUpdated: %++v", lastUpdated)
	region.LastUpdated = lastUpdated
	if rowsAffected != 1 {
		if rowsAffected < 1 {
			return errors.New("no region found with this id"), tc.DataMissingError
		}
		return fmt.Errorf("this update affected too many rows: %d", rowsAffected), tc.SystemError
	}

	return nil, tc.NoError
}

//The TORegion implementation of the Creator interface
//all implementations of Creator should use transactions and return the proper errorType
//ParsePQUniqueConstraintError is used to determine if a region with conflicting values exists
//if so, it will return an errorType of DataConflict and the type should be appended to the
//generic error message returned
//The insert sql returns the id and lastUpdated values of the newly inserted region and have
//to be added to the struct
func (region *TORegion) Create() (error, tc.ApiErrorType) {
	resultRows, err := region.ReqInfo.Txx.NamedQuery(insertQuery(), region)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			err, eType := dbhelpers.ParsePQUniqueConstraintError(pqErr)
			if eType == tc.DataConflictError {
				return errors.New("a region with " + err.Error()), eType
			}
			return err, eType
		}
		log.Errorf("received non pq error: %++v from create execution", err)
		return tc.DBError, tc.SystemError
	}
	defer resultRows.Close()

	var id int
	var lastUpdated tc.TimeNoMod
	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&id, &lastUpdated); err != nil {
			log.Error.Printf("could not scan id from insert: %s\n", err)
			return tc.DBError, tc.SystemError
		}
	}
	if rowsAffected == 0 {
		err = errors.New("no region was inserted, no id was returned")
		log.Errorln(err)
		return tc.DBError, tc.SystemError
	}
	if rowsAffected > 1 {
		err = errors.New("too many ids returned from region insert")
		log.Errorln(err)
		return tc.DBError, tc.SystemError
	}
	region.SetKeys(map[string]interface{}{"id": id})
	region.LastUpdated = lastUpdated

	return nil, tc.NoError
}

//The Region implementation of the Deleter interface
//all implementations of Deleter should use transactions and return the proper errorType
func (region *TORegion) Delete() (error, tc.ApiErrorType) {
	log.Debugf("about to run exec query: %s with region: %++v", deleteQuery(), region)
	result, err := region.ReqInfo.Txx.NamedExec(deleteQuery(), region)
	if err != nil {
		log.Errorf("received error: %++v from delete execution", err)
		return tc.DBError, tc.SystemError
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tc.DBError, tc.SystemError
	}
	if rowsAffected < 1 {
		return errors.New("no region with that id found"), tc.DataMissingError
	}
	if rowsAffected > 1 {
		return fmt.Errorf("this create affected too many rows: %d", rowsAffected), tc.SystemError
	}

	return nil, tc.NoError
}

func updateQuery() string {
	query := `UPDATE
region SET
division=:division,
name=:name
WHERE id=:id RETURNING last_updated`
	return query
}

func insertQuery() string {
	query := `INSERT INTO region (
division,
name) VALUES (
:division,
:name) RETURNING id,last_updated`
	return query
}

func deleteQuery() string {
	query := `DELETE FROM region
WHERE id=:id`
	return query
}
