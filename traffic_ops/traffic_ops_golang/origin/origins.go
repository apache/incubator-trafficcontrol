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
	"strconv"

	"github.com/apache/incubator-trafficcontrol/lib/go-log"
	"github.com/apache/incubator-trafficcontrol/lib/go-tc"
	"github.com/apache/incubator-trafficcontrol/lib/go-tc/v13"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/tovalidate"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

//we need a type alias to define functions on
type TOOrigin v13.OriginNullable

//the refType is passed into the handlers where a copy of its type is used to decode the json.
var refType = TOOrigin{}

func GetRefType() *TOOrigin {
	return &refType
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

func (origin *TOOrigin) Validate(db *sqlx.DB) []error {

	noSpaces := validation.NewStringRule(tovalidate.NoSpaces, "cannot contain spaces")
	validProtocol := validation.NewStringRule(tovalidate.IsOneOfStringICase("http", "https"), "must be http or https")
	portErr := "must be a valid integer between 1 and 65535"

	validateErrs := validation.Errors{
		"cachegroupId":      validation.Validate(origin.CachegroupID, validation.Min(1)),
		"coordinateId":      validation.Validate(origin.CoordinateID, validation.Min(1)),
		"deliveryServiceId": validation.Validate(origin.DeliveryServiceID, validation.Min(1)),
		"fqdn":              validation.Validate(origin.FQDN, validation.Required, is.DNSName),
		"ip6Address":        validation.Validate(origin.IP6Address, validation.NilOrNotEmpty, is.IPv6),
		"ipAddress":         validation.Validate(origin.IPAddress, validation.NilOrNotEmpty, is.IPv4),
		"name":              validation.Validate(origin.Name, validation.Required, noSpaces),
		"port":              validation.Validate(origin.Port, validation.NilOrNotEmpty.Error(portErr), validation.Min(1).Error(portErr), validation.Max(65535).Error(portErr)),
		"profileId":         validation.Validate(origin.ProfileID, validation.Min(1)),
		"protocol":          validation.Validate(origin.Protocol, validation.Required, validProtocol),
		"tenantId":          validation.Validate(origin.TenantID, validation.Min(1)),
	}
	return tovalidate.ToErrors(validateErrs)
}

// GetTenantID returns a pointer to the Origin's tenant ID from the DB and any error encountered
func (origin *TOOrigin) GetTenantID(db *sqlx.DB) (*int, error) {
	if origin.ID != nil {
		var tenantID *int
		if err := db.QueryRow(`SELECT tenant FROM origin where id = $1`, *origin.ID).Scan(&tenantID); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, fmt.Errorf("querying tenant ID for origin ID '%v': %v", *origin.ID, err)
		}
		return tenantID, nil
	}
	return nil, nil
}

func (origin *TOOrigin) IsTenantAuthorized(user auth.CurrentUser, db *sqlx.DB) (bool, error) {
	currentTenantID, err := origin.GetTenantID(db)
	if err != nil {
		return false, err
	}

	if currentTenantID != nil && tenant.IsTenancyEnabled(db) {
		return tenant.IsResourceAuthorizedToUser(*currentTenantID, user, db)
	}

	return true, nil
}

// filterAuthorized will filter a slice of OriginNullables based upon tenant. It assumes that tenancy is enabled
func filterAuthorized(origins []v13.OriginNullable, user auth.CurrentUser, db *sqlx.DB) ([]v13.OriginNullable, error) {
	newOrigins := []v13.OriginNullable{}
	for _, origin := range origins {
		if origin.TenantID == nil {
			if origin.ID == nil {
				return nil, errors.New("isResourceAuthorized for origin with nil ID: no tenant ID")
			} else {
				return nil, fmt.Errorf("isResourceAuthorized for origin %d: no tenant ID", *origin.ID)
			}
		}
		// TODO add/use a helper func to make a single SQL call, for performance
		ok, err := tenant.IsResourceAuthorizedToUser(*origin.TenantID, user, db)
		if err != nil {
			if origin.ID == nil {
				return nil, errors.New("isResourceAuthorized for origin with nil ID: " + err.Error())
			} else {
				return nil, fmt.Errorf("isResourceAuthorized for origin %d: "+err.Error(), *origin.ID)
			}
		}
		if !ok {
			continue
		}
		newOrigins = append(newOrigins, origin)
	}
	return newOrigins, nil
}

func (origin *TOOrigin) Read(db *sqlx.DB, params map[string]string, user auth.CurrentUser) ([]interface{}, []error, tc.ApiErrorType) {
	returnable := []interface{}{}

	privLevel := user.PrivLevel

	origins, errs, errType := getOrigins(params, db, privLevel)
	if len(errs) > 0 {
		return nil, errs, errType
	}

	var err error
	if tenant.IsTenancyEnabled(db) {
		origins, err = filterAuthorized(origins, user, db)
		if err != nil {
			log.Errorln("Checking tenancy: " + err.Error())
			return nil, []error{errors.New("Error checking tenancy.")}, tc.SystemError
		}
	}

	for _, origin := range origins {
		returnable = append(returnable, origin)
	}

	return returnable, nil, tc.NoError
}

func getOrigins(params map[string]string, db *sqlx.DB, privLevel int) ([]v13.OriginNullable, []error, tc.ApiErrorType) {
	var rows *sqlx.Rows
	var err error

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"cachegroup":      dbhelpers.WhereColumnInfo{"o.cachegroup", api.IsInt},
		"coordinate":      dbhelpers.WhereColumnInfo{"o.coordinate", api.IsInt},
		"deliveryservice": dbhelpers.WhereColumnInfo{"o.deliveryservice", api.IsInt},
		"id":              dbhelpers.WhereColumnInfo{"o.id", api.IsInt},
		"name":            dbhelpers.WhereColumnInfo{"o.name", nil},
		"profileId":       dbhelpers.WhereColumnInfo{"o.profile", api.IsInt},
		"tenant":          dbhelpers.WhereColumnInfo{"o.tenant", api.IsInt},
	}

	where, orderBy, queryValues, errs := dbhelpers.BuildWhereAndOrderBy(params, queryParamsToSQLCols)
	if len(errs) > 0 {
		return nil, errs, tc.DataConflictError
	}

	query := selectQuery() + where + orderBy
	log.Debugln("Query is ", query)

	rows, err = db.NamedQuery(query, queryValues)
	if err != nil {
		return nil, []error{fmt.Errorf("querying: %v", err)}, tc.SystemError
	}
	defer rows.Close()

	origins := []v13.OriginNullable{}

	for rows.Next() {
		var s v13.OriginNullable
		if err = rows.StructScan(&s); err != nil {
			return nil, []error{fmt.Errorf("getting origins: %v", err)}, tc.SystemError
		}
		origins = append(origins, s)
	}
	return origins, nil, tc.NoError
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
o.last_updated,
o.name,
o.port,
p.name as profile,
o.profile as profile_id,
o.protocol as protocol,
t.name as tenant,
o.tenant as tenant_id

FROM origin o

LEFT JOIN cachegroup cg ON o.cachegroup = cg.id
LEFT JOIN deliveryservice d ON o.deliveryservice = d.id
LEFT JOIN coordinate c ON o.coordinate = c.id
LEFT JOIN profile p ON o.profile = p.id
LEFT JOIN tenant t ON o.tenant = t.id`

	return selectStmt
}

func checkTenancy(originTenantID, deliveryserviceID *int, db *sqlx.DB, user auth.CurrentUser) (error, tc.ApiErrorType) {
	if tenant.IsTenancyEnabled(db) {
		if originTenantID == nil {
			return tc.NilTenantError, tc.ForbiddenError
		}
		authorized, err := tenant.IsResourceAuthorizedToUser(*originTenantID, user, db)
		if err != nil {
			return err, tc.SystemError
		}
		if !authorized {
			return tc.TenantUserNotAuthError, tc.ForbiddenError
		}

		if deliveryserviceID != nil {
			var deliveryserviceTenantID *int
			if err := db.QueryRow(`SELECT tenant_id FROM deliveryservice where id = $1`, *deliveryserviceID).Scan(&deliveryserviceTenantID); err != nil {
				if err == sql.ErrNoRows {
					return errors.New("checking tenancy: requested delivery service does not exist"), tc.DataConflictError
				}
				log.Errorf("could not get tenant_id from deliveryservice %d: %++v\n", *deliveryserviceID, err)
				return err, tc.SystemError
			}
			if deliveryserviceTenantID != nil {
				authorized, err := tenant.IsResourceAuthorizedToUser(*deliveryserviceTenantID, user, db)
				if err != nil {
					return err, tc.SystemError
				}
				if !authorized {
					return tc.TenantDSUserNotAuthError, tc.ForbiddenError
				}
			}
		}
	}
	return nil, tc.NoError
}

//The TOOrigin implementation of the Updater interface
//all implementations of Updater should use transactions and return the proper errorType
//ParsePQUniqueConstraintError is used to determine if an origin with conflicting values exists
//if so, it will return an errorType of DataConflict and the type should be appended to the
//generic error message returned
func (origin *TOOrigin) Update(db *sqlx.DB, user auth.CurrentUser) (error, tc.ApiErrorType) {
	// TODO: enhance tenancy framework to handle this in isTenantAuthorized()
	err, errType := checkTenancy(origin.TenantID, origin.DeliveryServiceID, db, user)
	if err != nil {
		return err, errType
	}

	rollbackTransaction := true
	tx, err := db.Beginx()
	defer func() {
		if tx == nil || !rollbackTransaction {
			return
		}
		err := tx.Rollback()
		if err != nil {
			log.Errorln(errors.New("rolling back transaction: " + err.Error()))
		}
	}()

	if err != nil {
		log.Error.Printf("could not begin transaction: %v", err)
		return tc.DBError, tc.SystemError
	}

	log.Debugf("about to run exec query: %s with origin: %++v", updateQuery(), origin)
	resultRows, err := tx.NamedQuery(updateQuery(), origin)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			err, eType := dbhelpers.ParsePQUniqueConstraintError(pqErr)
			if eType == tc.DataConflictError {
				return errors.New("an origin with " + err.Error()), eType
			}
			return err, eType
		} else {
			log.Errorf("received error: %++v from update execution", err)
			return tc.DBError, tc.SystemError
		}
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

	if rowsAffected == 0 {
		err = errors.New("no origin was updated, no id was returned")
		log.Errorln(err)
		return tc.DBError, tc.SystemError
	} else if rowsAffected > 1 {
		err = errors.New("too many ids returned from origin update")
		log.Errorln(err)
		return tc.DBError, tc.SystemError
	}
	origin.LastUpdated = &lastUpdated
	err = tx.Commit()
	if err != nil {
		log.Errorln("Could not commit transaction: ", err)
		return tc.DBError, tc.SystemError
	}
	rollbackTransaction = false
	return nil, tc.NoError
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
func (origin *TOOrigin) Create(db *sqlx.DB, user auth.CurrentUser) (error, tc.ApiErrorType) {
	// TODO: enhance tenancy framework to handle this in isTenantAuthorized()
	err, errType := checkTenancy(origin.TenantID, origin.DeliveryServiceID, db, user)
	if err != nil {
		return err, errType
	}

	rollbackTransaction := true
	tx, err := db.Beginx()
	defer func() {
		if tx == nil || !rollbackTransaction {
			return
		}
		err := tx.Rollback()
		if err != nil {
			log.Errorln(errors.New("rolling back transaction: " + err.Error()))
		}
	}()

	if err != nil {
		log.Error.Printf("could not begin transaction: %v", err)
		return tc.DBError, tc.SystemError
	}

	resultRows, err := tx.NamedQuery(insertQuery(), origin)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			err, eType := dbhelpers.ParsePQUniqueConstraintError(pqErr)
			if eType == tc.DataConflictError {
				return errors.New("an origin with " + err.Error()), eType
			}
			return err, eType
		} else {
			log.Errorf("received non pq error: %++v from create execution", err)
			return tc.DBError, tc.SystemError
		}
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
		err = errors.New("no origin was inserted, no id was returned")
		log.Errorln(err)
		return tc.DBError, tc.SystemError
	} else if rowsAffected > 1 {
		err = errors.New("too many ids returned from origin insert")
		log.Errorln(err)
		return tc.DBError, tc.SystemError
	}
	origin.SetKeys(map[string]interface{}{"id": id})
	origin.LastUpdated = &lastUpdated
	err = tx.Commit()
	if err != nil {
		log.Errorln("Could not commit transaction: ", err)
		return tc.DBError, tc.SystemError
	}
	rollbackTransaction = false
	return nil, tc.NoError
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
func (origin *TOOrigin) Delete(db *sqlx.DB, user auth.CurrentUser) (error, tc.ApiErrorType) {
	rollbackTransaction := true
	tx, err := db.Beginx()
	defer func() {
		if tx == nil || !rollbackTransaction {
			return
		}
		err := tx.Rollback()
		if err != nil {
			log.Errorln(errors.New("rolling back transaction: " + err.Error()))
		}
	}()

	if err != nil {
		log.Error.Printf("could not begin transaction: %v", err)
		return tc.DBError, tc.SystemError
	}
	log.Debugf("about to run exec query: %s with origin: %++v", deleteQuery(), origin)
	result, err := tx.NamedExec(deleteQuery(), origin)
	if err != nil {
		log.Errorf("received error: %++v from delete execution", err)
		return tc.DBError, tc.SystemError
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tc.DBError, tc.SystemError
	}
	if rowsAffected != 1 {
		if rowsAffected < 1 {
			return errors.New("no origin with that id found"), tc.DataMissingError
		} else {
			return fmt.Errorf("this delete affected too many rows: %d", rowsAffected), tc.SystemError
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Errorln("Could not commit transaction: ", err)
		return tc.DBError, tc.SystemError
	}
	rollbackTransaction = false
	return nil, tc.NoError
}

func deleteQuery() string {
	query := `DELETE FROM origin
WHERE id=:id`
	return query
}
