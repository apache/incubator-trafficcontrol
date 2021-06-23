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
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/lib/pq"
)

func PostDSes(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	fedID := inf.IntParams["id"]
	fedName, ok, err := getFedNameByID(inf.Tx.Tx, fedID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("getting federation cname from ID '%v': %v", fedID, err))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, fmt.Errorf("federation %v not found", fedID), nil)
		return
	}

	post := tc.FederationDSPost{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &post); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("parse error: "+err.Error()), nil)
		return
	}

	if post.Replace != nil && *post.Replace {
		if len(post.DSIDs) < 1 {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("A federation must have at least one delivery service assigned"), nil)
			return
		}
		if err := deleteDSFeds(inf.Tx.Tx, fedID); err != nil {
			userErr, sysErr, errCode := api.ParseDBError(err)
			api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
	}

	if len(post.DSIDs) > 0 {
		// there might be no DSes, if the user is trying to clear the assignments
		if err := insertDSFeds(inf.Tx.Tx, fedID, post.DSIDs); err != nil {
			userErr, sysErr, errCode := api.ParseDBError(err)
			api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
	}
	api.CreateChangeLogRawTx(api.ApiChange, fmt.Sprintf("FEDERATION: %v, ID: %v, ACTION: Assign DSes to federation", fedName, fedID), inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, strconv.Itoa(len(post.DSIDs))+" delivery service(s) were assigned to the federation "+strconv.Itoa(fedID), post)
}

func deleteDSFeds(tx *sql.Tx, fedID int) error {
	qry := `DELETE FROM federation_deliveryservice WHERE federation = $1`
	_, err := tx.Exec(qry, fedID)
	return err
}

func deleteFedDS(tx *sql.Tx, fedID, dsID int) error {
	qry := `DELETE FROM federation_deliveryservice WHERE federation = $1 AND deliveryservice = $2`
	_, err := tx.Exec(qry, fedID, dsID)
	return err
}

func insertDSFeds(tx *sql.Tx, fedID int, dsIDs []int) error {
	qry := `
INSERT INTO federation_deliveryservice (federation, deliveryservice)
VALUES ($1, unnest($2::integer[]))
`
	_, err := tx.Exec(qry, fedID, pq.Array(dsIDs))
	return err
}

// getFedNameFromID returns the federations name and whether or not one with the given ID exists, or an error
func getFedNameByID(tx *sql.Tx, id int) (string, bool, error) {
	name := ""
	if err := tx.QueryRow(`select cname from federation where id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("Error querying federation cname: " + err.Error())
	}
	return name, true, nil
}

// TOFedDSes data structure to use on read/delete of federation deliveryservices
type TOFedDSes struct {
	api.InfoerImpl `json:"-"`
	fedID          *int
	tc.FederationDeliveryServiceNullable
}

func (v *TOFedDSes) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(fds.last_updated) as t FROM federation_deliveryservice fds
RIGHT JOIN deliveryservice ds ON fds.deliveryservice = ds.id
JOIN cdn c ON ds.cdn_id = c.id
JOIN type t ON ds.type = t.id ` + where + orderBy + pagination +
		` UNION ALL
select max(last_updated) as t from last_deleted l where l.table_name='federation_deliveryservice') as res`
}
func (v *TOFedDSes) NewReadObj() interface{} { return &tc.FederationDeliveryServiceNullable{} }
func (v *TOFedDSes) SelectQuery() string     { return selectQuery() }
func (v *TOFedDSes) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"id":   dbhelpers.WhereColumnInfo{Column: "fds.federation", Checker: api.IsInt},
		"dsID": dbhelpers.WhereColumnInfo{Column: "fds.deliveryservice", Checker: api.IsInt},
	}
}
func (v *TOFedDSes) GetType() string {
	return "federation deliveryservice"
}

func (v *TOFedDSes) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int)
	v.fedID = &i
}

func (v *TOFedDSes) GetKeys() (map[string]interface{}, bool) {
	if v.fedID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *v.fedID}, true
}

func (v *TOFedDSes) GetAuditName() string {
	if v.XMLID != nil {
		return *v.XMLID
	}
	return strconv.Itoa(*v.ID)
}

func (v *TOFedDSes) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{
		{Field: "id", Func: api.GetIntKey},
	}
}

func (v *TOFedDSes) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	api.DefaultSort(v.Info(), "xmlId")
	return api.GenericRead(h, v, useIMS)
}

func (v *TOFedDSes) Delete() (error, error, int) {
	dsIDStr, ok := v.Info().Params["dsID"]
	if !ok {
		return errors.New("dsID must be specified for deletion"), nil, http.StatusBadRequest
	}
	dsID, err := strconv.Atoi(dsIDStr)
	if err != nil {
		return errors.New("dsID must be an integer"), nil, http.StatusBadRequest
	}
	v.ID = &dsID

	// Check that we can delete it
	if respCode, usrErr, sysErr := checkFedDSDeletion(v.Info().Tx.Tx, *v.fedID, dsID); usrErr != nil || sysErr != nil {
		if usrErr != nil {
			return usrErr, sysErr, respCode
		}
		return usrErr, sysErr, respCode
	}

	// Actually delete the DS from the Federation
	if err := deleteFedDS(v.Info().Tx.Tx, *v.fedID, dsID); err != nil {
		return api.ParseDBError(err)
	}

	return nil, nil, http.StatusOK
}

func checkFedDSDeletion(tx *sql.Tx, fedID, dsID int) (int, error, error) {

	q := `SELECT ARRAY(SELECT deliveryservice FROM federation_deliveryservice WHERE federation=$1)`
	dsIDs := []int64{} // pq.Array does not support int slice needs to be int64
	err := tx.QueryRow(q, fedID).Scan(pq.Array(&dsIDs))
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("querying federation %v delivery services - %v", fedID, err)
	}

	if len(dsIDs) == 0 {
		return http.StatusNotFound, fmt.Errorf("federation %v not found", fedID), nil
	}

	if len(dsIDs) < 2 {
		return http.StatusBadRequest, fmt.Errorf("a federation must have at least one delivery service assigned"), nil
	}
	found := false
	dsID64 := int64(dsID) // need in order to compare
	for _, id := range dsIDs {
		if id == dsID64 {
			found = true
			break
		}
	}
	if !found {
		return http.StatusBadRequest, fmt.Errorf("delivery service %v is not associated with federation %v", dsID, fedID), nil
	}
	return http.StatusOK, nil, nil
}

func selectQuery() string {

	query := `SELECT
ds.id,
ds.xml_id,
c.name AS cdn,
t.name as type
FROM federation_deliveryservice fds
RIGHT JOIN deliveryservice ds ON fds.deliveryservice = ds.id
JOIN cdn c ON ds.cdn_id = c.id
JOIN type t ON ds.type = t.id`
	return query
}
