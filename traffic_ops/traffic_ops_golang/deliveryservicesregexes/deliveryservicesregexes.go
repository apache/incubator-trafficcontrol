package deliveryservicesregexes

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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/lib/pq"
)

func Get(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	q := `
SELECT ds.xml_id, dsr.set_number, r.pattern, rt.name as type
FROM deliveryservice_regex as dsr
JOIN deliveryservice as ds ON dsr.deliveryservice = ds.id
JOIN regex as r ON dsr.regex = r.id
JOIN type as rt ON r.type = rt.id
WHERE ds.tenant_id = ANY($1)
`

	accessibleTenants, err := tenant.GetUserTenantIDListTx(inf.Tx.Tx, inf.User.TenantID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("getting accessible tenants for user - %v", err))
	}
	rows, err := inf.Tx.Tx.Query(q, pq.Array(accessibleTenants))
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("querying deliveryserviceregexes: "+err.Error()))
		return
	}
	defer rows.Close()
	dsRegexes := map[tc.DeliveryServiceName][]tc.DeliveryServiceRegex{}
	for rows.Next() {
		dsName := ""
		setNumber := 0
		pattern := ""
		typeName := ""
		if err = rows.Scan(&dsName, &setNumber, &pattern, &typeName); err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("scanning deliveryserviceregexes: "+err.Error()))
			return
		}
		dsRegexes[tc.DeliveryServiceName(dsName)] = append(dsRegexes[tc.DeliveryServiceName(dsName)], tc.DeliveryServiceRegex{
			Type:      typeName,
			SetNumber: setNumber,
			Pattern:   pattern,
		})
	}

	respRegexes := []tc.DeliveryServiceRegexes{}
	for dsName, regexes := range dsRegexes {
		respRegexes = append(respRegexes, tc.DeliveryServiceRegexes{DSName: string(dsName), Regexes: regexes})
	}
	api.WriteResp(w, r, respRegexes)
}

func DSGet(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"dsid"}, []string{"dsid"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	q := `
SELECT ds.tenant_id, dsr.set_number, r.id, r.pattern, rt.id as type, rt.name as type_name
FROM deliveryservice_regex as dsr
JOIN deliveryservice as ds ON dsr.deliveryservice = ds.id
JOIN regex as r ON dsr.regex = r.id
JOIN type as rt ON r.type = rt.id
`
	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		"dsid": dbhelpers.WhereColumnInfo{Column: "ds.ID", Checker: api.IsInt},
		"id":   dbhelpers.WhereColumnInfo{Column: "r.id", Checker: api.IsInt}}
	where, _, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(inf.Params, queryParamsToQueryCols)
	if len(errs) > 0 {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, util.JoinErrs(errs), nil)
		return
	}

	accessibleTenants, err := tenant.GetUserTenantIDListTx(inf.Tx.Tx, inf.User.TenantID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("getting accessible tenants for user - %v", err))
	}
	if len(where) > 0 {
		where += " AND ds.tenant_id = ANY(:tenants) "
	} else {
		where = dbhelpers.BaseWhere + " ds.tenant_id = ANY(:tenants) "
	}
	queryValues["tenants"] = pq.Array(accessibleTenants)

	query := q + where + " ORDER BY dsr.set_number ASC" + pagination

	rows, err := inf.Tx.NamedQuery(query, queryValues)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("querying deliveryserviceregexes get: "+err.Error()))
		return
	}
	defer rows.Close()
	regexes := []tc.DeliveryServiceIDRegex{}
	dsTenants := map[int]*int{}
	for rows.Next() {
		dsTenantID := util.IntPtr(0)
		rx := tc.DeliveryServiceIDRegex{}
		if err = rows.Scan(&dsTenantID, &rx.SetNumber, &rx.ID, &rx.Pattern, &rx.Type, &rx.TypeName); err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("scanning deliveryserviceregexes get: "+err.Error()))
			return
		}
		regexes = append(regexes, rx)
		dsTenants[rx.ID] = dsTenantID
	}
	api.WriteResp(w, r, regexes)
}

func DSGetID(w http.ResponseWriter, r *http.Request) {
	alerts := tc.CreateAlerts(tc.WarnLevel, "This endpoint is deprecated, please use GET '/deliveryservices/{dsid}/regexes' with query parameter id instead")

	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"dsid", "regexid"}, []string{"dsid", "regexid"})
	if userErr != nil || sysErr != nil {
		userErr = api.LogErr(r, errCode, userErr, sysErr)
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, errCode, alerts)
		return
	}
	defer inf.Close()

	q := `
SELECT dsr.set_number, r.id, r.pattern, rt.id as type, rt.name as type_name
FROM deliveryservice_regex as dsr
JOIN deliveryservice as ds ON dsr.deliveryservice = ds.id
JOIN regex as r ON dsr.regex = r.id
JOIN type as rt ON r.type = rt.id
WHERE ds.ID = $1 AND ds.tenant_id = ANY($2)
AND r.ID = $3
ORDER BY dsr.set_number ASC
`
	accessibleTenants, err := tenant.GetUserTenantIDListTx(inf.Tx.Tx, inf.User.TenantID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("getting accessible tenants for user - %v", err))
	}
	rows, err := inf.Tx.Tx.Query(q, inf.IntParams["dsid"], pq.Array(accessibleTenants), inf.IntParams["regexid"])
	if err != nil {
		userErr = api.LogErr(r, errCode, userErr, errors.New("querying deliveryserviceregexes getid: "+err.Error()))
		alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
		api.WriteAlerts(w, r, errCode, alerts)
		return
	}
	defer rows.Close()
	regexes := []tc.DeliveryServiceIDRegex{}
	for rows.Next() {
		rx := tc.DeliveryServiceIDRegex{}
		if err = rows.Scan(&rx.SetNumber, &rx.ID, &rx.Pattern, &rx.Type, &rx.TypeName); err != nil {
			userErr = api.LogErr(r, errCode, userErr, errors.New("scanning deliveryserviceregexes getid: "+err.Error()))
			alerts.AddNewAlert(tc.ErrorLevel, userErr.Error())
			api.WriteAlerts(w, r, errCode, alerts)
			return
		}
		regexes = append(regexes, rx)
	}
	api.WriteAlertsObj(w, r, http.StatusOK, alerts, regexes)
}

func Post(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"dsid"}, []string{"dsid"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	tx := inf.Tx.Tx

	dsTenantID := 0
	if err := tx.QueryRow(`SELECT tenant_id from deliveryservice where id = $1`, inf.IntParams["dsid"]).Scan(&dsTenantID); err != nil {
		if err == sql.ErrNoRows {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
			return
		}
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("querying deliveryserviceregexes post: "+err.Error()))
		return
	}
	if ok, err := tenant.IsResourceAuthorizedToUserTx(dsTenantID, inf.User, tx); !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusUnauthorized, nil, nil)
		return
	} else if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking tenancy: "+err.Error()))
		return
	}
	dsr := tc.DeliveryServiceRegexPost{}
	if err := json.NewDecoder(r.Body).Decode(&dsr); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}

	if err := validateDSRegex(tx, dsr, inf.IntParams["dsid"], true); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
		return
	}

	_, cdnName, _, err := dbhelpers.GetDSNameAndCDNFromID(inf.Tx.Tx, inf.IntParams["dsid"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	userErr, sysErr, errCode = dbhelpers.CheckIfCurrentUserCanModifyCDN(inf.Tx.Tx, string(cdnName), inf.User.UserName)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	regexID := 0
	if err := tx.QueryRow(`INSERT INTO regex (pattern, type) VALUES ($1, $2) RETURNING id`, dsr.Pattern, dsr.Type).Scan(&regexID); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("inserting deliveryserviceregex regex: "+err.Error()))
		return
	}

	if _, err := tx.Exec(`INSERT INTO deliveryservice_regex (deliveryservice, regex, set_number) values ($1, $2, $3)`, inf.IntParams["dsid"], regexID, dsr.SetNumber); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("inserting deliveryserviceregex: "+err.Error()))
		return
	}

	typeName := ""
	if err := tx.QueryRow(`SELECT name from type where id = $1`, dsr.Type).Scan(&typeName); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("querying deliveryserviceregex type: "+err.Error()))
		return
	}

	dsID := inf.IntParams["dsid"]
	dsName, _, err := dbhelpers.GetDSNameFromID(inf.Tx.Tx, dsID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting delivery service name from id: "+err.Error()))
		return
	}

	respObj := tc.DeliveryServiceIDRegex{
		ID:        regexID,
		Pattern:   dsr.Pattern,
		Type:      dsr.Type,
		TypeName:  typeName,
		SetNumber: dsr.SetNumber,
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+string(dsName)+", ID: "+strconv.Itoa(dsID)+", ACTION: Created a regular expression ("+dsr.Pattern+") in position "+strconv.Itoa(dsr.SetNumber), inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Delivery service regex creation was successful.", respObj)
}

func getCurrentDetails(tx *sql.Tx, dsID int, regexID int) error {
	var setNumber int
	var typeName string
	err := tx.QueryRow(`
select dsr.set_number, t.name 
from deliveryservice_regex as dsr 
join regex as r on dsr.regex = r.id 
join type as t on t.id = r.type 
where dsr.deliveryservice=$1 and r.id=$2`,
		dsID, regexID).Scan(&setNumber, &typeName)
	if err != nil {
		return err
	}
	if setNumber == 0 && typeName == "HOST_REGEXP" {
		return errors.New("cannot change/ delete a regex with an order of 0 and type name of HOST_REGEXP")
	}
	return nil
}

func Put(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"dsid", "regexid"}, []string{"dsid", "regexid"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	tx := inf.Tx.Tx

	dsID := inf.IntParams["dsid"]
	dsName, ok, err := dbhelpers.GetDSNameFromID(inf.Tx.Tx, dsID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting delivery service name from id: "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
		return
	}
	regexID := inf.IntParams["regexid"]
	dsTenantID := 0
	if err := tx.QueryRow(`SELECT tenant_id from deliveryservice where id = $1`, dsID).Scan(&dsTenantID); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("querying deliveryserviceregex tenant: "+err.Error()))
		return
	}
	if ok, err := tenant.IsResourceAuthorizedToUserTx(dsTenantID, inf.User, tx); !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusUnauthorized, nil, nil)
		return
	} else if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryserviceregex put checking tenancy: "+err.Error()))
		return
	}
	dsr := tc.DeliveryServiceRegexPost{} // PUT uses same format as POST
	if err := json.NewDecoder(r.Body).Decode(&dsr); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	_, cdnName, _, err := dbhelpers.GetDSNameAndCDNFromID(inf.Tx.Tx, inf.IntParams["dsid"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	userErr, sysErr, errCode = dbhelpers.CheckIfCurrentUserCanModifyCDN(inf.Tx.Tx, string(cdnName), inf.User.UserName)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	// Get current details to make sure that you're not trying to change a regex that has set number = 0 and type = HOST_REGEXP
	if err := getCurrentDetails(tx, dsID, regexID); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
		return
	}
	if err := validateDSRegex(tx, dsr, inf.IntParams["dsid"], false); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
		return
	}

	if _, err := tx.Exec(`UPDATE regex SET pattern=$1, type=$2 WHERE id=$3`, dsr.Pattern, dsr.Type, regexID); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservicesregexes.Put: updating regex: "+err.Error()))
		return
	}
	if _, err := tx.Exec(`UPDATE deliveryservice_regex SET set_number=$1 WHERE deliveryservice=$2 AND regex=$3`, dsr.SetNumber, dsID, regexID); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservicesregexes.Put: updating ds_regex: "+err.Error()))
		return
	}
	typeName := ""
	if err := tx.QueryRow(`SELECT name from type where id = $1`, dsr.Type).Scan(&typeName); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting ds regex type: "+err.Error()))
		return
	}
	respObj := tc.DeliveryServiceIDRegex{
		ID:        regexID,
		Pattern:   dsr.Pattern,
		Type:      dsr.Type,
		TypeName:  typeName,
		SetNumber: dsr.SetNumber,
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+string(dsName)+", ID: "+strconv.Itoa(dsID)+", ACTION: Updated a regular expression ("+dsr.Pattern+") in position "+strconv.Itoa(dsr.SetNumber), inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Delivery service regex creation was successful.", respObj)
}

// canUpdate checks to see if the current regex can be updated. If the current regex has a set number of 0, and a type of HOST_REGEXP, it cannot be updated.
func canUpdate(tx *sql.Tx, dsr tc.DeliveryServiceRegexPost) error {
	var name string
	err := tx.QueryRow(`
select name from type as t 
where t.id=$1`,
		dsr.Type).Scan(&name)
	if err != nil {
		return err
	}
	// Cannot have more than one regex with typename as "HOST_REGEXP" and set number as 0
	if name == "HOST_REGEXP" && dsr.SetNumber == 0 {
		return errors.New("cannot update regex with set number 0 and type HOST_REGEXP")
	}
	return nil
}

// Validate POST/PUT regex struct
func validateDSRegex(tx *sql.Tx, dsr tc.DeliveryServiceRegexPost, dsID int, new bool) error {
	var ds int
	var setNumberErr error
	if dsr.SetNumber < 0 {
		return errors.New("cannot add regex with order < 0")
	}
	err := tx.QueryRow(`
select deliveryservice from deliveryservice_regex
where deliveryservice = $1 and set_number = $2`,
		dsID, dsr.SetNumber).Scan(&ds)
	if new {
		// If its a new regex, then another shouldn't exist with the same set number
		if err == nil {
			setNumberErr = errors.New("cannot add regex, another regex with the same order exists")
		} else {
			setNumberErr = nil
		}
	} else {
		// Cannot update a regex, to set the new set number = 0 and type = HOST_REGEXP
		e := canUpdate(tx, dsr)
		if e != nil {
			setNumberErr = e
		} else {
			setNumberErr = err
		}
	}
	_, typeErr := tc.ValidateTypeID(tx, &dsr.Type, "regex")

	errs := validation.Errors{
		"type":      typeErr,
		"setNumber": setNumberErr,
		"pattern":   validation.Validate(dsr.Pattern, validation.Required)}

	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func Delete(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"dsid", "regexid"}, []string{"dsid", "regexid"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	dsID := inf.IntParams["dsid"]
	dsName, cdnName, ok, err := dbhelpers.GetDSNameAndCDNFromID(inf.Tx.Tx, inf.IntParams["dsid"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
		return
	}
	userErr, sysErr, errCode = dbhelpers.CheckIfCurrentUserCanModifyCDN(inf.Tx.Tx, string(cdnName), inf.User.UserName)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	regexID := inf.IntParams["regexid"]

	// Get current details to make sure that you're not trying to delete a regex that has set number = 0 and type = HOST_REGEXP
	if err := getCurrentDetails(inf.Tx.Tx, dsID, regexID); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("cannot delete regex: "+err.Error()), nil)
		return
	}
	count := 0
	if err := inf.Tx.Tx.QueryRow(`SELECT count(*) from deliveryservice_regex where deliveryservice = $1`, dsID).Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
			return
		}
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting deliveryservice regex count: "+err.Error()))
		return
	}
	if count < 2 {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("a delivery service must have at least one regex"), nil)
		return
	}

	dsTenantID := 0
	if err := inf.Tx.Tx.QueryRow(`SELECT tenant_id from deliveryservice where id = $1`, dsID).Scan(&dsTenantID); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting deliveryservice name: "+err.Error()))
		return
	}
	if ok, err := tenant.IsResourceAuthorizedToUserTx(dsTenantID, inf.User, inf.Tx.Tx); !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusUnauthorized, nil, nil)
		return
	} else if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("checking delete ds regexes tenancy : "+err.Error()))
		return
	}

	dsrSetNumber := 0
	if err := inf.Tx.Tx.QueryRow(`SELECT set_number FROM deliveryservice_regex WHERE regex = $1`, regexID).Scan(&dsrSetNumber); err != nil {
		if err == sql.ErrNoRows {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("regex not found for this delivery service"), nil)
			return
		}
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservicesregexes.Delete finding set number: "+err.Error()))
		return
	}

	dsrType, dsrPattern := 0, ""
	if err := inf.Tx.Tx.QueryRow(`SELECT type, pattern FROM regex WHERE id = $1`, regexID).Scan(&dsrType, &dsrPattern); err != nil {
		if err == sql.ErrNoRows {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("regex not found"), nil)
			return
		}
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservicesregexes.Delete finding type and pattern: "+err.Error()))
		return
	}

	result, err := inf.Tx.Tx.Exec(`DELETE FROM deliveryservice_regex WHERE deliveryservice = $1 and regex = $2`, dsID, regexID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservicesregexes.Delete deleting delivery service regexes: "+err.Error()))
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservicesregexes.Delete delete error: "+err.Error()))
		return
	}
	if rowsAffected < 1 {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
		return
	}
	if rowsAffected > 1 {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("this create affected too many rows: %d", rowsAffected))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+string(dsName)+", ID: "+strconv.Itoa(dsID)+", ACTION: Deleted a regular expression ("+dsrPattern+") in position "+strconv.Itoa(dsrSetNumber), inf.User, inf.Tx.Tx)
	api.WriteRespAlert(w, r, tc.SuccessLevel, "deliveryservice_regex was deleted.")
}
