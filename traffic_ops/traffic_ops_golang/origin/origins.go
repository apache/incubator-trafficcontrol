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

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
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
	api.APIInfoImpl `json:"-"`
	tc.Origin
}

func (origin *TOOrigin) SetID(i int) {
	origin.ID = &i
}

func (origin TOOrigin) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"id", api.GetIntKey}}
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

func (origin *TOOrigin) Read(h http.Header, useIMS bool) ([]interface{}, apierrors.Errors, *time.Time) {
	returnable := []interface{}{}
	origins, errs, maxTime := getOrigins(h, origin.ReqInfo.Params, origin.ReqInfo.Tx, origin.ReqInfo.User, useIMS)
	if errs.Occurred() {
		return nil, errs, nil
	}

	for _, origin := range origins {
		returnable = append(returnable, origin)
	}

	return returnable, errs, maxTime
}

func getOrigins(h http.Header, params map[string]string, tx *sqlx.Tx, user *auth.CurrentUser, useIMS bool) ([]tc.Origin, apierrors.Errors, *time.Time) {
	var rows *sqlx.Rows
	var err error
	var maxTime time.Time
	var runSecond bool

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"cachegroup":      dbhelpers.WhereColumnInfo{"o.cachegroup", api.IsInt},
		"coordinate":      dbhelpers.WhereColumnInfo{"o.coordinate", api.IsInt},
		"deliveryservice": dbhelpers.WhereColumnInfo{"o.deliveryservice", api.IsInt},
		"id":              dbhelpers.WhereColumnInfo{"o.id", api.IsInt},
		"name":            dbhelpers.WhereColumnInfo{"o.name", nil},
		"primary":         dbhelpers.WhereColumnInfo{"o.is_primary", api.IsBool},
		"profileId":       dbhelpers.WhereColumnInfo{"o.profile", api.IsInt},
		"tenant":          dbhelpers.WhereColumnInfo{"o.tenant", api.IsInt},
	}

	errs := apierrors.New()
	where, orderBy, pagination, queryValues, dbErrs := dbhelpers.BuildWhereAndOrderByAndPagination(params, queryParamsToSQLCols)
	if len(dbErrs) > 0 {
		errs.UserError = util.JoinErrs(dbErrs)
		errs.Code = http.StatusBadRequest
		return nil, errs, nil
	}
	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(tx, h, queryValues, selectMaxLastUpdatedQuery(where))
		if !runSecond {
			log.Debugln("IMS HIT")
			return []tc.Origin{}, apierrors.Errors{Code: http.StatusNotModified}, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	tenantIDs, err := tenant.GetUserTenantIDListTx(tx.Tx, user.TenantID)
	if err != nil {
		log.Errorln("received error querying for user's tenants: " + err.Error())
		errs.SystemError = tc.DBError
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	}
	where, queryValues = dbhelpers.AddTenancyCheck(where, queryValues, "o.tenant", tenantIDs)

	query := selectQuery() + where + orderBy + pagination
	log.Debugln("Query is ", query)

	rows, err = tx.NamedQuery(query, queryValues)
	if err != nil {
		errs.SystemError = fmt.Errorf("querying: %v", err)
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	}
	defer rows.Close()

	origins := []tc.Origin{}

	for rows.Next() {
		var s tc.Origin
		if err = rows.StructScan(&s); err != nil {
			errs.SystemError = fmt.Errorf("getting origins: %v", err)
			errs.Code = http.StatusInternalServerError
			return nil, errs, nil
		}
		origins = append(origins, s)
	}
	return origins, errs, &maxTime
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

func checkTenancy(originTenantID, deliveryserviceID *int, tx *sqlx.Tx, user *auth.CurrentUser) apierrors.Errors {
	if originTenantID == nil {
		return apierrors.Errors{
			Code:      http.StatusForbidden,
			UserError: tc.NilTenantError,
		}
	}
	authorized, err := tenant.IsResourceAuthorizedToUserTx(*originTenantID, user, tx.Tx)
	if err != nil {
		return apierrors.Errors{
			Code:        http.StatusInternalServerError,
			SystemError: err,
		}
	}
	if !authorized {
		return apierrors.Errors{
			Code:      http.StatusForbidden,
			UserError: tc.TenantUserNotAuthError,
		}
	}

	var deliveryserviceTenantID int
	if err := tx.QueryRow(`SELECT tenant_id FROM deliveryservice where id = $1`, *deliveryserviceID).Scan(&deliveryserviceTenantID); err != nil {
		errs := apierrors.Errors{
			Code: http.StatusBadRequest,
		}
		if err == sql.ErrNoRows {
			errs.SetUserError("checking tenancy: requested delivery service does not exist")
			return errs
		}
		log.Errorf("could not get tenant_id from deliveryservice %d: %++v\n", *deliveryserviceID, err)
		// TODO: don't expose DB errors to the user
		errs.UserError = err
		return errs
	}
	authorized, err = tenant.IsResourceAuthorizedToUserTx(deliveryserviceTenantID, user, tx.Tx)
	if err != nil {
		return apierrors.Errors{
			Code:      http.StatusBadRequest,
			UserError: err,
		}
	}
	if !authorized {
		return apierrors.Errors{
			Code:      http.StatusForbidden,
			UserError: tc.TenantDSUserNotAuthError,
		}
	}
	return apierrors.New()
}

//The TOOrigin implementation of the Updater interface
//all implementations of Updater should use transactions and return the proper errorType
//ParsePQUniqueConstraintError is used to determine if an origin with conflicting values exists
//if so, it will return an errorType of DataConflict and the type should be appended to the
//generic error message returned
func (origin *TOOrigin) Update() apierrors.Errors {
	// TODO: enhance tenancy framework to handle this in isTenantAuthorized()
	errs := checkTenancy(origin.TenantID, origin.DeliveryServiceID, origin.ReqInfo.Tx, origin.ReqInfo.User)
	if errs.Occurred() {
		return errs
	}

	isPrimary := false
	ds := 0
	q := `SELECT is_primary, deliveryservice FROM origin WHERE id = $1`
	if err := origin.ReqInfo.Tx.QueryRow(q, *origin.ID).Scan(&isPrimary, &ds); err != nil {
		if err == sql.ErrNoRows {
			errs.SetUserError("origin not found")
			errs.Code = http.StatusNotFound
		} else {
			errs.SetSystemError("origin update: querying: " + err.Error())
			errs.Code = http.StatusInternalServerError
		}
		return errs
	}
	if isPrimary && *origin.DeliveryServiceID != ds {
		errs.SetUserError("cannot update the delivery service of a primary origin")
		errs.Code = http.StatusBadRequest
		return errs
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
			errs.SetSystemError("origin update: scanning: " + err.Error())
			errs.Code = http.StatusInternalServerError
			return errs
		}
	}

	if rowsAffected == 0 {
		errs.SetSystemError("origin update: no rows returned")
		errs.Code = http.StatusInternalServerError
	} else if rowsAffected > 1 {
		errs.SetSystemError("origin update: multiple rows returned")
		errs.Code = http.StatusInternalServerError
	}
	origin.LastUpdated = &lastUpdated
	return errs
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
func (origin *TOOrigin) Create() apierrors.Errors {
	// TODO: enhance tenancy framework to handle this in isTenantAuthorized()
	errs := checkTenancy(origin.TenantID, origin.DeliveryServiceID, origin.ReqInfo.Tx, origin.ReqInfo.User)
	if errs.Occurred() {
		return errs
	}

	resultRows, err := origin.ReqInfo.Tx.NamedQuery(insertQuery(), origin)
	if err != nil {
		return api.ParseDBError(err)
	}
	defer resultRows.Close()

	var id int
	var lastUpdated tc.TimeNoMod
	errs = apierrors.Errors{
		Code: http.StatusInternalServerError,
	}
	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&id, &lastUpdated); err != nil {
			errs.SystemError = errors.New("origin create: scanning: " + err.Error())
			return errs
		}
	}

	if rowsAffected == 0 {
		errs.SetSystemError("origin create: no rows returned")
		return errs
	} else if rowsAffected > 1 {
		errs.SetSystemError("origin create: multiple rows returned")
		return errs
	}
	origin.SetKeys(map[string]interface{}{"id": id})
	origin.LastUpdated = &lastUpdated

	return apierrors.New()
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
func (origin *TOOrigin) Delete() apierrors.Errors {
	errs := apierrors.New()
	isPrimary := false
	q := `SELECT is_primary FROM origin WHERE id = $1`
	if err := origin.ReqInfo.Tx.QueryRow(q, *origin.ID).Scan(&isPrimary); err != nil {
		if err == sql.ErrNoRows {
			errs.SetUserError("origin not found")
			errs.Code = http.StatusNotFound
		} else {
			errs.SetSystemError("origin delete: is_primary scanning: " + err.Error())
			errs.Code = http.StatusInternalServerError
		}
		return errs
	}
	if isPrimary {
		errs.SetUserError("cannot delete a primary origin")
		errs.Code = http.StatusBadRequest
		return errs
	}

	result, err := origin.ReqInfo.Tx.NamedExec(deleteQuery(), origin)
	if err != nil {
		errs.SetSystemError("origin delete: query: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return errs
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		errs.SetSystemError("origin delete: getting rows affected: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return errs
	}
	if rowsAffected != 1 {
		errs.SetSystemError("origin delete: multiple rows affected")
		errs.Code = http.StatusInternalServerError
	}

	return errs
}

func deleteQuery() string {
	query := `DELETE FROM origin
WHERE id=:id`
	return query
}
