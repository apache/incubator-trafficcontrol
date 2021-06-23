package origin

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

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/jmoiron/sqlx"
)

//we need a type alias to define functions on
type TOOrigin struct {
	api.InfoerImpl `json:"-"`
	tc.Origin
}

func (origin *TOOrigin) SetID(i int) {
	origin.ID = &i
}

func (origin TOOrigin) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (origin TOOrigin) GetKeys() (map[string]interface{}, bool) {
	if origin.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *origin.ID}, true
}

func (origin *TOOrigin) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	origin.ID = &i
}

func (origin *TOOrigin) GetAuditName() string {
	if origin.Name != nil {
		return *origin.Name
	}
	if origin.ID != nil {
		return strconv.Itoa(*origin.ID)
	}
	return "unknown"
}

func (origin *TOOrigin) GetType() string {
	return "origin"
}

func (origin *TOOrigin) Validate() error {

	noSpaces := validation.NewStringRule(tovalidate.NoSpaces, "cannot contain spaces")
	validProtocol := validation.NewStringRule(tovalidate.IsOneOfStringICase("http", "https"), "must be http or https")
	portErr := "must be a valid integer between 1 and 65535"

	validateErrs := validation.Errors{
		"cachegroupId":      validation.Validate(origin.CachegroupID, validation.Min(1)),
		"coordinateId":      validation.Validate(origin.CoordinateID, validation.Min(1)),
		"deliveryServiceId": validation.Validate(origin.DeliveryServiceID, validation.NotNil),
		"fqdn":              validation.Validate(origin.FQDN, validation.Required, is.DNSName),
		"ip6Address":        validation.Validate(origin.IP6Address, validation.NilOrNotEmpty, is.IPv6),
		"ipAddress":         validation.Validate(origin.IPAddress, validation.NilOrNotEmpty, is.IPv4),
		"name":              validation.Validate(origin.Name, validation.Required, noSpaces),
		"port":              validation.Validate(origin.Port, validation.NilOrNotEmpty.Error(portErr), validation.Min(1).Error(portErr), validation.Max(65535).Error(portErr)),
		"profileId":         validation.Validate(origin.ProfileID, validation.Min(1)),
		"protocol":          validation.Validate(origin.Protocol, validation.Required, validProtocol),
		"tenantId":          validation.Validate(origin.TenantID, validation.Min(1)),
	}
	return util.JoinErrs(tovalidate.ToErrors(validateErrs))
}

// GetTenantID returns a pointer to the Origin's tenant ID from the Tx, whether or not the Origin exists, and any error encountered
func (origin *TOOrigin) GetTenantID(tx *sqlx.Tx) (*int, bool, error) {
	if origin.ID != nil {
		var tenantID *int
		if err := tx.QueryRow(`SELECT tenant FROM origin where id = $1`, *origin.ID).Scan(&tenantID); err != nil {
			if err == sql.ErrNoRows {
				return nil, false, nil
			}
			return nil, false, fmt.Errorf("querying tenant ID for origin ID '%v': %v", *origin.ID, err)
		}
		return tenantID, true, nil
	}
	return nil, false, nil
}

func (origin *TOOrigin) IsTenantAuthorized(user *auth.CurrentUser) (bool, error) {
	currentTenantID, originExists, err := origin.GetTenantID(origin.ReqInfo.Tx)
	if !originExists {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return tenant.IsResourceAuthorizedToUserTx(*currentTenantID, user, origin.ReqInfo.Tx.Tx)
}

func (origin *TOOrigin) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	returnable := []interface{}{}
	origins, userErr, sysErr, errCode, maxTime := getOrigins(h, origin.ReqInfo.Params, origin.ReqInfo.Tx, origin.ReqInfo.User, useIMS)
	if userErr != nil || sysErr != nil {
		return nil, userErr, sysErr, errCode, nil
	}

	for _, origin := range origins {
		returnable = append(returnable, origin)
	}

	return returnable, nil, nil, http.StatusOK, maxTime
}

func getOrigins(h http.Header, params map[string]string, tx *sqlx.Tx, user *auth.CurrentUser, useIMS bool) ([]tc.Origin, error, error, int, *time.Time) {
	var rows *sqlx.Rows
	var err error
	var maxTime time.Time
	var runSecond bool

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"cachegroup":      dbhelpers.WhereColumnInfo{Column: "o.cachegroup", Checker: api.IsInt},
		"coordinate":      dbhelpers.WhereColumnInfo{Column: "o.coordinate", Checker: api.IsInt},
		"deliveryservice": dbhelpers.WhereColumnInfo{Column: "o.deliveryservice", Checker: api.IsInt},
		"id":              dbhelpers.WhereColumnInfo{Column: "o.id", Checker: api.IsInt},
		"name":            dbhelpers.WhereColumnInfo{Column: "o.name"},
		"primary":         dbhelpers.WhereColumnInfo{Column: "o.is_primary", Checker: api.IsBool},
		"profileId":       dbhelpers.WhereColumnInfo{Column: "o.profile", Checker: api.IsInt},
		"tenant":          dbhelpers.WhereColumnInfo{Column: "o.tenant", Checker: api.IsInt},
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(params, queryParamsToSQLCols)
	if len(errs) > 0 {
		return nil, util.JoinErrs(errs), nil, http.StatusBadRequest, nil
	}
	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(tx, h, queryValues, selectMaxLastUpdatedQuery(where))
		if !runSecond {
			log.Debugln("IMS HIT")
			return []tc.Origin{}, nil, nil, http.StatusNotModified, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	tenantIDs, err := tenant.GetUserTenantIDListTx(tx.Tx, user.TenantID)
	if err != nil {
		log.Errorln("received error querying for user's tenants: " + err.Error())
		return nil, nil, tc.DBError, http.StatusInternalServerError, nil
	}
	where, queryValues = dbhelpers.AddTenancyCheck(where, queryValues, "o.tenant", tenantIDs)

	query := selectQuery() + where + orderBy + pagination
	log.Debugln("Query is ", query)

	rows, err = tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, nil, fmt.Errorf("querying: %v", err), http.StatusInternalServerError, nil
	}
	defer rows.Close()

	origins := []tc.Origin{}

	for rows.Next() {
		var s tc.Origin
		if err = rows.StructScan(&s); err != nil {
			return nil, nil, fmt.Errorf("getting origins: %v", err), http.StatusInternalServerError, nil
		}
		origins = append(origins, s)
	}
	return origins, nil, nil, http.StatusOK, &maxTime
}

func selectMaxLastUpdatedQuery(where string) string {
	return `SELECT max(t) from (
		SELECT max(o.last_updated) as t from origin as o
	JOIN deliveryservice d ON o.deliveryservice = d.id
LEFT JOIN cachegroup cg ON o.cachegroup = cg.id
LEFT JOIN coordinate c ON o.coordinate = c.id
LEFT JOIN profile p ON o.profile = p.id
LEFT JOIN tenant t ON o.tenant = t.id ` + where +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='origin') as res`
}

func selectQuery() string {

	selectStmt := `SELECT
cg.name as cachegroup,
o.cachegroup as cachegroup_id,
o.coordinate as coordinate_id,
c.name as coordinate,
d.xml_id as deliveryservice,
o.deliveryservice as deliveryservice_id,
o.fqdn,
o.id,
o.ip6_address,
o.ip_address,
o.is_primary,
o.last_updated,
o.name,
o.port,
p.name as profile,
o.profile as profile_id,
o.protocol as protocol,
t.name as tenant,
o.tenant as tenant_id

FROM origin o

JOIN deliveryservice d ON o.deliveryservice = d.id
LEFT JOIN cachegroup cg ON o.cachegroup = cg.id
LEFT JOIN coordinate c ON o.coordinate = c.id
LEFT JOIN profile p ON o.profile = p.id
LEFT JOIN tenant t ON o.tenant = t.id`

	return selectStmt
}

func checkTenancy(originTenantID, deliveryserviceID *int, tx *sqlx.Tx, user *auth.CurrentUser) (error, error, int) {
	if originTenantID == nil {
		return tc.NilTenantError, nil, http.StatusForbidden
	}
	authorized, err := tenant.IsResourceAuthorizedToUserTx(*originTenantID, user, tx.Tx)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	if !authorized {
		return tc.TenantUserNotAuthError, nil, http.StatusForbidden
	}

	var deliveryserviceTenantID int
	if err := tx.QueryRow(`SELECT tenant_id FROM deliveryservice where id = $1`, *deliveryserviceID).Scan(&deliveryserviceTenantID); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("checking tenancy: requested delivery service does not exist"), nil, http.StatusBadRequest
		}
		log.Errorf("could not get tenant_id from deliveryservice %d: %++v\n", *deliveryserviceID, err)
		return err, nil, http.StatusBadRequest
	}
	authorized, err = tenant.IsResourceAuthorizedToUserTx(deliveryserviceTenantID, user, tx.Tx)
	if err != nil {
		return err, nil, http.StatusBadRequest
	}
	if !authorized {
		return tc.TenantDSUserNotAuthError, nil, http.StatusForbidden
	}
	return nil, nil, http.StatusOK
}

//The TOOrigin implementation of the Updater interface
//all implementations of Updater should use transactions and return the proper errorType
//ParsePQUniqueConstraintError is used to determine if an origin with conflicting values exists
//if so, it will return an errorType of DataConflict and the type should be appended to the
//generic error message returned
func (origin *TOOrigin) Update(h http.Header) (error, error, int) {
	// TODO: enhance tenancy framework to handle this in isTenantAuthorized()
	userErr, sysErr, errCode := checkTenancy(origin.TenantID, origin.DeliveryServiceID, origin.ReqInfo.Tx, origin.ReqInfo.User)
	if userErr != nil || sysErr != nil {
		return userErr, sysErr, errCode
	}

	isPrimary := false
	ds := 0
	var existingLastUpdated *tc.TimeNoMod

	q := `SELECT is_primary, deliveryservice, last_updated FROM origin WHERE id = $1`
	if err := origin.ReqInfo.Tx.QueryRow(q, *origin.ID).Scan(&isPrimary, &ds, &existingLastUpdated); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("origin not found"), nil, http.StatusNotFound
		}
		return nil, errors.New("origin update: querying: " + err.Error()), http.StatusInternalServerError
	}

	if !api.IsUnmodified(h, existingLastUpdated.Time) {
		return errors.New("resource was modified"), nil, http.StatusPreconditionFailed
	}

	if isPrimary && *origin.DeliveryServiceID != ds {
		return errors.New("cannot update the delivery service of a primary origin"), nil, http.StatusBadRequest
	}

	log.Debugf("about to run exec query: %s with origin: %++v", updateQuery(), origin)
	resultRows, err := origin.ReqInfo.Tx.NamedQuery(updateQuery(), origin)
	if err != nil {
		return api.ParseDBError(err)
	}
	defer resultRows.Close()

	var lastUpdated tc.TimeNoMod
	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&lastUpdated); err != nil {
			return nil, errors.New("origin update: scanning: " + err.Error()), http.StatusInternalServerError
		}
	}

	if rowsAffected == 0 {
		return nil, errors.New("origin update: no rows returned"), http.StatusInternalServerError
	} else if rowsAffected > 1 {
		return nil, errors.New("origin update: multiple rows returned"), http.StatusInternalServerError
	}
	origin.LastUpdated = &lastUpdated
	return nil, nil, http.StatusOK
}

func updateQuery() string {
	query := `UPDATE
origin SET
cachegroup=:cachegroup_id,
coordinate=:coordinate_id,
deliveryservice=:deliveryservice_id,
fqdn=:fqdn,
ip6_address=:ip6_address,
ip_address=:ip_address,
name=:name,
port=:port,
profile=:profile_id,
protocol=:protocol,
tenant=:tenant_id
WHERE id=:id RETURNING last_updated`
	return query
}

//The TOOrigin implementation of the Inserter interface
//all implementations of Inserter should use transactions and return the proper errorType
//ParsePQUniqueConstraintError is used to determine if an origin with conflicting values exists
//if so, it will return an errorType of DataConflict and the type should be appended to the
//generic error message returned
//The insert sql returns the id and lastUpdated values of the newly inserted origin and have
//to be added to the struct
func (origin *TOOrigin) Create() (error, error, int) {
	// TODO: enhance tenancy framework to handle this in isTenantAuthorized()
	userErr, sysErr, errCode := checkTenancy(origin.TenantID, origin.DeliveryServiceID, origin.ReqInfo.Tx, origin.ReqInfo.User)
	if userErr != nil || sysErr != nil {
		return userErr, sysErr, errCode
	}

	resultRows, err := origin.ReqInfo.Tx.NamedQuery(insertQuery(), origin)
	if err != nil {
		return api.ParseDBError(err)
	}
	defer resultRows.Close()

	var id int
	var lastUpdated tc.TimeNoMod
	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&id, &lastUpdated); err != nil {
			return nil, errors.New("origin create: scanning: " + err.Error()), http.StatusInternalServerError
		}
	}
	if rowsAffected == 0 {
		return nil, errors.New("origin create: no rows returned"), http.StatusInternalServerError
	} else if rowsAffected > 1 {
		return nil, errors.New("origin create: multiple rows returned"), http.StatusInternalServerError
	}
	origin.SetKeys(map[string]interface{}{"id": id})
	origin.LastUpdated = &lastUpdated

	return nil, nil, http.StatusOK
}

func insertQuery() string {
	query := `INSERT INTO origin (
cachegroup,
coordinate,
deliveryservice,
fqdn,
ip6_address,
ip_address,
name,
port,
profile,
protocol,
tenant) VALUES (
:cachegroup_id,
:coordinate_id,
:deliveryservice_id,
:fqdn,
:ip6_address,
:ip_address,
:name,
:port,
:profile_id,
:protocol,
:tenant_id) RETURNING id,last_updated`
	return query
}

//The Origin implementation of the Deleter interface
//all implementations of Deleter should use transactions and return the proper errorType
func (origin *TOOrigin) Delete() (error, error, int) {
	isPrimary := false
	q := `SELECT is_primary FROM origin WHERE id = $1`
	if err := origin.ReqInfo.Tx.QueryRow(q, *origin.ID).Scan(&isPrimary); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("origin not found"), nil, http.StatusNotFound
		}
		return nil, errors.New("origin delete: is_primary scanning: " + err.Error()), http.StatusInternalServerError
	}
	if isPrimary {
		return errors.New("cannot delete a primary origin"), nil, http.StatusBadRequest
	}

	result, err := origin.ReqInfo.Tx.NamedExec(deleteQuery(), origin)
	if err != nil {
		return nil, errors.New("origin delete: query: " + err.Error()), http.StatusInternalServerError
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, errors.New("origin delete: getting rows affected: " + err.Error()), http.StatusInternalServerError
	}
	if rowsAffected != 1 {
		return nil, errors.New("origin delete: multiple rows affected"), http.StatusInternalServerError
	}

	return nil, nil, http.StatusOK
}

func deleteQuery() string {
	query := `DELETE FROM origin
WHERE id=:id`
	return query
}
