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

const (
	userQueryParam     = "userID"
	userRoleQueryParam = "role"
	fedQueryParam      = "id"
	fedUserType        = "federation users"
)

// TOUsers data structure to use on read/delete of federation users
type TOUsers struct {
	api.InfoerImpl `json:"-"`
	Federation     *int `json:"-" db:"federation"`
	tc.FederationUser
}

func (v *TOUsers) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(fedu.last_updated) as t FROM federation_tmuser fedu
RIGHT JOIN tm_user u ON fedu.tm_user = u.id
JOIN role r ON u.role = r.id ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='federation_tmuser') as res`
}

func (v *TOUsers) NewReadObj() interface{} { return &tc.FederationUser{} }
func (v *TOUsers) DeleteQuery() string {
	return `
DELETE FROM federation_tmuser
WHERE federation = :federation AND tm_user = :id
`
}

func (v *TOUsers) SelectQuery() string {
	return `
SELECT
u.id,
u.username,
u.full_name,
u.company,
u.email,
r.name as role_name
FROM federation_tmuser fedu
RIGHT JOIN tm_user u ON fedu.tm_user = u.id
JOIN role r ON u.role = r.id
`
}

func (v *TOUsers) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		userQueryParam:     {Column: "u.id", Checker: api.IsInt},
		userRoleQueryParam: {Column: "r.name"},
		fedQueryParam:      {Column: "fedu.federation", Checker: api.IsInt},
	}
}

func (v TOUsers) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{
		{Field: userQueryParam, Func: api.GetIntKey},
		{Field: fedQueryParam, Func: api.GetIntKey},
	}
}

func (v TOUsers) GetKeys() (map[string]interface{}, bool) {
	if v.ID == nil {
		return map[string]interface{}{userQueryParam: 0}, false
	}
	if v.Federation == nil {
		return map[string]interface{}{fedQueryParam: 0}, false
	}
	return map[string]interface{}{
		userQueryParam: *v.ID,
		fedQueryParam:  *v.Federation,
	}, true
}

func (v *TOUsers) SetKeys(keys map[string]interface{}) {
	usr, _ := keys[userQueryParam].(int)
	v.ID = &usr

	fed, _ := keys[fedQueryParam].(int)
	v.Federation = &fed
}

func (v *TOUsers) GetAuditName() string {
	if v.ID != nil {
		return strconv.Itoa(*v.ID)
	}
	return "unknown"
}

func (v *TOUsers) GetType() string {
	return fedUserType
}

func (v *TOUsers) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	fedIDStr := v.Info().Params["id"]
	fedID, err := strconv.Atoi(fedIDStr)
	if err != nil {
		return nil, errors.New("federation id must be an integer"), nil, http.StatusBadRequest, nil
	}
	_, exists, err := getFedNameByID(v.Info().Tx.Tx, fedID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting federation cname from ID %v: %v", fedID, err), http.StatusInternalServerError, nil
	} else if !exists {
		return nil, fmt.Errorf("federation %v not found", fedID), nil, http.StatusNotFound, nil
	}
	return api.GenericRead(h, v, useIMS)
}

func (v *TOUsers) Delete() (error, error, int) { return api.GenericDelete(v) }

func PostUsers(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	fedID := inf.IntParams["id"]
	fedName, exists, err := getFedNameByID(inf.Tx.Tx, fedID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("getting federation cname from ID %v: %v", fedID, err))
		return
	} else if !exists {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, fmt.Errorf("federation %v not found", fedID), nil)
		return
	}

	post := tc.FederationUserPost{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &post); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("parse error: "+err.Error()), nil)
		return
	}

	if post.Replace != nil && *post.Replace {
		if err := deleteFedUsers(inf.Tx.Tx, fedID); err != nil {
			userErr, sysErr, errCode := api.ParseDBError(err)
			api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
	}

	if len(post.IDs) > 0 {
		if err := insertFedUsers(inf.Tx.Tx, fedID, post.IDs); err != nil {
			userErr, sysErr, errCode := api.ParseDBError(err)
			api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
			return
		}
	}
	api.CreateChangeLogRawTx(api.ApiChange, fmt.Sprintf("FEDERATION: %v, ID: %v, ACTION: Assign Users to federation", fedName, fedID), inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, fmt.Sprintf("%v user(s) were assigned to the %v federation", strconv.Itoa(len(post.IDs)), fedName), post)
}

func deleteFedUsers(tx *sql.Tx, fedID int) error {
	qry := `DELETE FROM federation_tmuser WHERE federation = $1`
	_, err := tx.Exec(qry, fedID)
	return err
}

func insertFedUsers(tx *sql.Tx, fedID int, userIDs []int) error {
	qry := `
INSERT INTO federation_tmuser (federation, tm_user)
VALUES ($1, unnest($2::integer[]))
`
	_, err := tx.Exec(qry, fedID, pq.Array(userIDs))
	return err
}
