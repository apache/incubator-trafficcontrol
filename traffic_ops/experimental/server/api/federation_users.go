
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file was initially generated by gen_to_start.go (add link), as a start
// of the Traffic Ops golang data model

package api

import (
	"encoding/json"
	_ "github.com/Comcast/traffic_control/traffic_ops/experimental/server/output_format" // needed for swagger
	"github.com/jmoiron/sqlx"
	"log"
	"time"
)

type FederationUsers struct {
	FederationId int64                `db:"federation_id" json:"federationId"`
	Username     string               `db:"username" json:"username"`
	Role         string               `db:"role" json:"role"`
	CreatedAt    time.Time            `db:"created_at" json:"createdAt"`
	Links        FederationUsersLinks `json:"_links" db:-`
}

type FederationUsersLinks struct {
	Self string `db:"self" json:"_self"`
}

// @Title getFederationUsersById
// @Description retrieves the federation_users information for a certain id
// @Accept  application/json
// @Param   id              path    int     false        "The row id"
// @Success 200 {array}    FederationUsers
// @Resource /api/2.0
// @Router /api/2.0/federation_users/{id} [get]
func getFederationUser(federationId int64, username string, db *sqlx.DB) (interface{}, error) {
	ret := []FederationUsers{}
	arg := FederationUsers{}
	arg.FederationId = federationId
	arg.Username = username
	queryStr := "select *, concat('" + API_PATH + "federation_users', '/federation_id/', federation_id, '/username/', username) as self"
	queryStr += " from federation_users WHERE federation_id=:federation_id AND username=:username"
	nstmt, err := db.PrepareNamed(queryStr)
	err = nstmt.Select(&ret, arg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	nstmt.Close()
	return ret, nil
}

// @Title getFederationUserss
// @Description retrieves the federation_users
// @Accept  application/json
// @Success 200 {array}    FederationUsers
// @Resource /api/2.0
// @Router /api/2.0/federation_users [get]
func getFederationUsers(db *sqlx.DB) (interface{}, error) {
	ret := []FederationUsers{}
	queryStr := "select *, concat('" + API_PATH + "federation_users', '/federation_id/', federation_id, '/username/', username) as self"
	queryStr += " from federation_users"
	err := db.Select(&ret, queryStr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return ret, nil
}

// @Title postFederationUsers
// @Description enter a new federation_users
// @Accept  application/json
// @Param                 Body body     FederationUsers   true "FederationUsers object that should be added to the table"
// @Success 200 {object}    output_format.ApiWrapper
// @Resource /api/2.0
// @Router /api/2.0/federation_users [post]
func postFederationUser(payload []byte, db *sqlx.DB) (interface{}, error) {
	var v FederationUsers
	err := json.Unmarshal(payload, &v)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	sqlString := "INSERT INTO federation_users("
	sqlString += "federation_id"
	sqlString += ",username"
	sqlString += ",role"
	sqlString += ",created_at"
	sqlString += ") VALUES ("
	sqlString += ":federation_id"
	sqlString += ",:username"
	sqlString += ",:role"
	sqlString += ",:created_at"
	sqlString += ")"
	result, err := db.NamedExec(sqlString, v)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return result, err
}

// @Title putFederationUsers
// @Description modify an existing federation_usersentry
// @Accept  application/json
// @Param   id              path    int     true        "The row id"
// @Param                 Body body     FederationUsers   true "FederationUsers object that should be added to the table"
// @Success 200 {object}    output_format.ApiWrapper
// @Resource /api/2.0
// @Router /api/2.0/federation_users/{id}  [put]
func putFederationUser(federationId int64, username string, payload []byte, db *sqlx.DB) (interface{}, error) {
	var arg FederationUsers
	err := json.Unmarshal(payload, &arg)
	arg.FederationId = federationId
	arg.Username = username
	if err != nil {
		log.Println(err)
		return nil, err
	}
	sqlString := "UPDATE federation_users SET "
	sqlString += "federation_id = :federation_id"
	sqlString += ",username = :username"
	sqlString += ",role = :role"
	sqlString += ",created_at = :created_at"
	sqlString += " WHERE federation_id=:federation_id AND username=:username"
	result, err := db.NamedExec(sqlString, arg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return result, err
}

// @Title delFederationUsersById
// @Description deletes federation_users information for a certain id
// @Accept  application/json
// @Param   id              path    int     false        "The row id"
// @Success 200 {array}    FederationUsers
// @Resource /api/2.0
// @Router /api/2.0/federation_users/{id} [delete]
func delFederationUser(federationId int64, username string, db *sqlx.DB) (interface{}, error) {
	arg := FederationUsers{}
	arg.FederationId = federationId
	arg.Username = username
	result, err := db.NamedExec("DELETE FROM federation_users WHERE federation_id=:federation_id AND username=:username", arg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return result, err
}
