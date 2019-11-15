
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
	_ "github.com/apache/trafficcontrol/traffic_ops/experimental/server/output_format" // needed for swagger
	"github.com/jmoiron/sqlx"
	"log"
	"time"
)

type Regexes struct {
	Id        int64        `db:"id" json:"id"`
	Pattern   string       `db:"pattern" json:"pattern"`
	CreatedAt time.Time    `db:"created_at" json:"createdAt"`
	Links     RegexesLinks `json:"_links" db:-`
}

type RegexesLinks struct {
	Self             string           `db:"self" json:"_self"`
	RegexesTypesLink RegexesTypesLink `json:"regexes_types" db:-`
}

// @Title getRegexesById
// @Description retrieves the regexes information for a certain id
// @Accept  application/json
// @Param   id              path    int     false        "The row id"
// @Success 200 {array}    Regexes
// @Resource /api/2.0
// @Router /api/2.0/regexes/{id} [get]
func getRegex(id int64, db *sqlx.DB) (interface{}, error) {
	ret := []Regexes{}
	arg := Regexes{}
	arg.Id = id
	queryStr := "select *, concat('" + API_PATH + "regexes/', id) as self"
	queryStr += ", concat('" + API_PATH + "regexes_types/', type) as regexes_types_name_ref"
	queryStr += " from regexes WHERE id=:id"
	nstmt, err := db.PrepareNamed(queryStr)
	err = nstmt.Select(&ret, arg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	nstmt.Close()
	return ret, nil
}

// @Title getRegexess
// @Description retrieves the regexes
// @Accept  application/json
// @Success 200 {array}    Regexes
// @Resource /api/2.0
// @Router /api/2.0/regexes [get]
func getRegexes(db *sqlx.DB) (interface{}, error) {
	ret := []Regexes{}
	queryStr := "select *, concat('" + API_PATH + "regexes/', id) as self"
	queryStr += ", concat('" + API_PATH + "regexes_types/', type) as regexes_types_name_ref"
	queryStr += " from regexes"
	err := db.Select(&ret, queryStr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return ret, nil
}

// @Title postRegexes
// @Description enter a new regexes
// @Accept  application/json
// @Param                 Body body     Regexes   true "Regexes object that should be added to the table"
// @Success 200 {object}    output_format.ApiWrapper
// @Resource /api/2.0
// @Router /api/2.0/regexes [post]
func postRegex(payload []byte, db *sqlx.DB) (interface{}, error) {
	var v Regexes
	err := json.Unmarshal(payload, &v)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	sqlString := "INSERT INTO regexes("
	sqlString += "pattern"
	sqlString += ",type"
	sqlString += ",created_at"
	sqlString += ") VALUES ("
	sqlString += ":pattern"
	sqlString += ",:type"
	sqlString += ",:created_at"
	sqlString += ")"
	result, err := db.NamedExec(sqlString, v)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return result, err
}

// @Title putRegexes
// @Description modify an existing regexesentry
// @Accept  application/json
// @Param   id              path    int     true        "The row id"
// @Param                 Body body     Regexes   true "Regexes object that should be added to the table"
// @Success 200 {object}    output_format.ApiWrapper
// @Resource /api/2.0
// @Router /api/2.0/regexes/{id}  [put]
func putRegex(id int64, payload []byte, db *sqlx.DB) (interface{}, error) {
	var arg Regexes
	err := json.Unmarshal(payload, &arg)
	arg.Id = id
	if err != nil {
		log.Println(err)
		return nil, err
	}
	sqlString := "UPDATE regexes SET "
	sqlString += "pattern = :pattern"
	sqlString += ",type = :type"
	sqlString += ",created_at = :created_at"
	sqlString += " WHERE id=:id"
	result, err := db.NamedExec(sqlString, arg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return result, err
}

// @Title delRegexesById
// @Description deletes regexes information for a certain id
// @Accept  application/json
// @Param   id              path    int     false        "The row id"
// @Success 200 {array}    Regexes
// @Resource /api/2.0
// @Router /api/2.0/regexes/{id} [delete]
func delRegex(id int64, db *sqlx.DB) (interface{}, error) {
	arg := Regexes{}
	arg.Id = id
	result, err := db.NamedExec("DELETE FROM regexes WHERE id=:id", arg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return result, err
}
