package role

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
	"net/http"
	"strconv"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type TORole struct {
	api.APIInfoImpl `json:"-"`
	tc.Role
	LastUpdated    *tc.TimeNoMod   `json:"-"`
	PQCapabilities *pq.StringArray `json:"-" db:"capabilities"`
}

func (v *TORole) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(last_updated) as t from ` + tableName + ` r ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='` + tableName + `') as res`
}

func (v *TORole) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TORole) InsertQuery() string           { return insertQuery() }
func (v *TORole) NewReadObj() interface{}       { return &TORole{} }
func (v *TORole) SelectQuery() string           { return selectQuery() }
func (v *TORole) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"name":      dbhelpers.WhereColumnInfo{"name", nil},
		"id":        dbhelpers.WhereColumnInfo{"id", api.IsInt},
		"privLevel": dbhelpers.WhereColumnInfo{"priv_level", api.IsInt}}
}
func (v *TORole) UpdateQuery() string { return updateQuery() }
func (v *TORole) DeleteQuery() string { return deleteQuery() }

func (role TORole) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"id", api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (role TORole) GetKeys() (map[string]interface{}, bool) {
	if role.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *role.ID}, true
}

func (role TORole) GetAuditName() string {
	if role.Name != nil {
		return *role.Name
	}
	if role.ID != nil {
		return strconv.Itoa(*role.ID)
	}
	return "0"
}

func (role TORole) GetType() string {
	return "role"
}

func (role *TORole) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	role.ID = &i
}

// Validate fulfills the api.Validator interface
func (role TORole) Validate() error {
	errs := validation.Errors{
		"name":        validation.Validate(role.Name, validation.Required),
		"description": validation.Validate(role.Description, validation.Required),
		"privLevel":   validation.Validate(role.PrivLevel, validation.Required)}

	errsToReturn := tovalidate.ToErrors(errs)
	checkCaps := `SELECT cap FROM UNNEST($1::text[]) AS cap WHERE NOT cap =  ANY(ARRAY(SELECT c.name FROM capability AS c WHERE c.name = ANY($1)))`
	var badCaps []string
	if role.ReqInfo.Tx != nil {
		err := role.ReqInfo.Tx.Select(&badCaps, checkCaps, pq.Array(role.Capabilities))
		if err != nil {
			log.Errorf("got error from selecting bad capabilities: %v", err)
			return tc.DBError
		}
		if len(badCaps) > 0 {
			errsToReturn = append(errsToReturn, fmt.Errorf("can not add non-existent capabilities: %v", badCaps))
		}
	}
	return util.JoinErrs(errsToReturn)
}

func (role *TORole) Create() apierrors.Errors {
	if *role.PrivLevel > role.ReqInfo.User.PrivLevel {
		return apierrors.Errors{
			Code:      http.StatusBadRequest,
			UserError: errors.New("can not create a role with a higher priv level than your own"),
		}
	}

	errs := api.GenericCreate(role)
	if errs.Occurred() {
		return errs
	}

	//after we have role ID we can associate the capabilities:
	if role.Capabilities != nil && len(*role.Capabilities) > 0 {
		errs = role.createRoleCapabilityAssociations(role.ReqInfo.Tx)
		if errs.Occurred() {
			return errs
		}
	}
	return apierrors.New()
}

func (role *TORole) createRoleCapabilityAssociations(tx *sqlx.Tx) apierrors.Errors {
	result, err := tx.Exec(associateCapabilities(), role.ID, pq.Array(role.Capabilities))
	if err != nil {
		return apierrors.Errors{
			Code:        http.StatusInternalServerError,
			SystemError: fmt.Errorf("creating role capabilities: %v", err),
		}
	}

	if rows, err := result.RowsAffected(); err != nil {
		log.Errorf("could not check result after inserting role_capability relations: %v", err)
	} else if expected := len(*role.Capabilities); int(rows) != expected {
		log.Errorf("wrong number of role_capability rows created: %d expected: %d", rows, expected)
	}
	return apierrors.New()
}

func (role *TORole) deleteRoleCapabilityAssociations(tx *sqlx.Tx) apierrors.Errors {
	result, err := tx.Exec(deleteAssociatedCapabilities(), role.ID)
	if err != nil {
		return apierrors.Errors{
			SystemError: errors.New("deleting role capabilities: " + err.Error()),
			Code:        http.StatusInternalServerError,
		}
	}

	if _, err = result.RowsAffected(); err != nil {
		log.Errorf("could not check result after inserting role_capability relations: %v", err)
	}
	// TODO verify expected row count shouldn't be checked?
	return apierrors.New()
}

func (role *TORole) Read(h http.Header, useIMS bool) ([]interface{}, apierrors.Errors, *time.Time) {
	version := role.APIInfo().Version
	api.DefaultSort(role.APIInfo(), "name")
	vals, errs, maxTime := api.GenericRead(h, role, useIMS)
	// TODO: Maybe it doesn't matter, but I think this should first check if an
	// error has occurred. Or actually maybe this could just be an '||' check?
	if errs.Code == http.StatusNotModified {
		return []interface{}{}, errs, maxTime
	}
	if errs.Occurred() {
		return nil, errs, maxTime
	}

	returnable := []interface{}{}
	for _, val := range vals {
		rl := val.(*TORole)
		switch {
		case version.Major > 1 || version.Minor >= 3:
			caps := ([]string)(*rl.PQCapabilities)
			rl.Capabilities = &caps
			returnable = append(returnable, rl)
		case version.Minor >= 1:
			returnable = append(returnable, rl.RoleV11)
		}
	}
	return returnable, errs, maxTime
}

func (role *TORole) Update() apierrors.Errors {
	errs := apierrors.New()
	if *role.PrivLevel > role.ReqInfo.User.PrivLevel {
		errs.SetUserError("can not create a role with a higher priv level than your own")
		errs.Code = http.StatusForbidden
		return errs
	}
	errs = api.GenericUpdate(role)
	if errs.Occurred() {
		return errs
	}

	// TODO cascade delete, to automatically do this in SQL?
	if role.Capabilities != nil && *role.Capabilities != nil {
		errs = role.deleteRoleCapabilityAssociations(role.ReqInfo.Tx)
		if errs.Occurred() {
			return errs
		}
		errs = role.createRoleCapabilityAssociations(role.ReqInfo.Tx)
		return errs
	}
	return errs
}

func (role *TORole) Delete() apierrors.Errors {
	assignedUsers := 0
	if err := role.ReqInfo.Tx.Get(&assignedUsers, "SELECT COUNT(id) FROM tm_user WHERE role=$1", role.ID); err != nil {
		return apierrors.Errors{
			SystemError: errors.New("role delete counting assigned users: " + err.Error()),
			Code:        http.StatusInternalServerError,
		}
	} else if assignedUsers != 0 {
		return apierrors.Errors{
			SystemError: fmt.Errorf("can not delete a role with %d assigned users", assignedUsers),
			Code:        http.StatusBadRequest,
		}
	}

	errs := api.GenericDelete(role)
	if errs.Occurred() {
		return errs
	}
	return role.deleteRoleCapabilityAssociations(role.ReqInfo.Tx)
}

func selectQuery() string {
	return `SELECT
id,
name,
description,
priv_level,
ARRAY(SELECT rc.cap_name FROM role_capability AS rc WHERE rc.role_id=id) AS capabilities
FROM role`
}

func updateQuery() string {
	return `UPDATE
role SET
name=:name,
description=:description
WHERE id=:id RETURNING last_updated`
}

func deleteAssociatedCapabilities() string {
	return `DELETE FROM role_capability
WHERE role_id=$1`
}

func associateCapabilities() string {
	return `INSERT INTO role_capability (
role_id,
cap_name) WITH
	q1 AS ( SELECT * FROM (VALUES ($1::bigint)) AS role_id ),
	q2 AS (SELECT UNNEST($2::text[]))
	SELECT * FROM q1,q2`
}

func insertQuery() string {
	return `INSERT INTO role (
name,
description,
priv_level
) VALUES (
:name,
:description,
:priv_level
)
RETURNING id, last_updated`
}

func deleteQuery() string {
	return `DELETE FROM role WHERE id = :id`
}
