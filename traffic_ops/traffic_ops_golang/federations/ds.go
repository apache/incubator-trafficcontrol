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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/lib/pq"
)

func PostDSes(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, []string{"id"}, []string{"id"})
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
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
			inf.HandleErrs(w, r, api.ParseDBError(err))
			return
		}
	}

	if len(post.DSIDs) > 0 {
		// there might be no DSes, if the user is trying to clear the assignments
		if err := insertDSFeds(inf.Tx.Tx, fedID, post.DSIDs); err != nil {
			inf.HandleErrs(w, r, api.ParseDBError(err))
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
	api.APIInfoImpl `json:"-"`
	fedID           *int
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
		"id":   dbhelpers.WhereColumnInfo{"fds.federation", api.IsInt},
		"dsID": dbhelpers.WhereColumnInfo{"fds.deliveryservice", api.IsInt},
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
		{"id", api.GetIntKey},
	}
}

func (v *TOFedDSes) Read(h http.Header, useIMS bool) ([]interface{}, apierrors.Errors, *time.Time) {
	api.DefaultSort(v.APIInfo(), "xmlId")
	return api.GenericRead(h, v, useIMS)
}

func (v *TOFedDSes) Delete() apierrors.Errors {
	errs := apierrors.New()
	dsIDStr, ok := v.APIInfo().Params["dsID"]
	if !ok {
		errs.SetUserError("dsID must be specified for deletion")
		errs.Code = http.StatusBadRequest
		return errs
	}
	dsID, err := strconv.Atoi(dsIDStr)
	if err != nil {
		errs.SetUserError("dsID must be an integer")
		errs.Code = http.StatusBadRequest
		return errs
	}
	v.ID = &dsID

	// Check that we can delete it
	errs = checkFedDSDeletion(v.APIInfo().Tx.Tx, *v.fedID, dsID)
	if errs.Occurred() {
		return errs
	}

	// Actually delete the DS from the Federation
	if err := deleteFedDS(v.APIInfo().Tx.Tx, *v.fedID, dsID); err != nil {
		return api.ParseDBError(err)
	}

	return errs
}

func checkFedDSDeletion(tx *sql.Tx, fedID, dsID int) apierrors.Errors {
	errs := apierrors.New()
	q := `SELECT ARRAY(SELECT deliveryservice FROM federation_deliveryservice WHERE federation=$1)`
	dsIDs := []int64{} // pq.Array does not support int slice needs to be int64
	err := tx.QueryRow(q, fedID).Scan(pq.Array(&dsIDs))
	if err != nil {
		errs.Code = http.StatusInternalServerError
		errs.SystemError = fmt.Errorf("querying federation %v delivery services - %v", fedID, err)
		return errs
	}

	if len(dsIDs) == 0 {
		errs.Code = http.StatusNotFound
		errs.UserError = fmt.Errorf("federation %v not found", fedID)
		return errs
	}

	if len(dsIDs) < 2 {
		errs.Code = http.StatusBadRequest
		errs.UserError = fmt.Errorf("a federation must have at least one delivery service assigned")
		return errs
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
		errs.Code = http.StatusBadRequest
		errs.UserError = fmt.Errorf("delivery service %v is not associated with federation %v", dsID, fedID)
	}
	return errs
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
