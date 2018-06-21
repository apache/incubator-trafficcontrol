
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

type ExtensionsTypes struct {
	Name        string               `db:"name" json:"name"`
	Description string               `db:"description" json:"description"`
	CreatedAt   time.Time            `db:"created_at" json:"createdAt"`
	Links       ExtensionsTypesLinks `json:"_links" db:-`
}

type ExtensionsTypesLinks struct {
	Self string `db:"self" json:"_self"`
}

type ExtensionsTypesLink struct {
	ID  string `db:"extensions_type" json:"name"`
	Ref string `db:"extensions_types_name_ref" json:"_ref"`
}

// @Title getExtensionsTypesById
// @Description retrieves the extensions_types information for a certain id
// @Accept  application/json
// @Param   id              path    int     false        "The row id"
// @Success 200 {array}    ExtensionsTypes
// @Resource /api/2.0
// @Router /api/2.0/extensions_types/{id} [get]
func getExtensionsType(name string, db *sqlx.DB) (interface{}, error) {
	ret := []ExtensionsTypes{}
	arg := ExtensionsTypes{}
	arg.Name = name
	queryStr := "select *, concat('" + API_PATH + "extensions_types/', name) as self"
	queryStr += " from extensions_types WHERE name=:name"
	nstmt, err := db.PrepareNamed(queryStr)
	err = nstmt.Select(&ret, arg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	nstmt.Close()
	return ret, nil
}

// @Title getExtensionsTypess
// @Description retrieves the extensions_types
// @Accept  application/json
// @Success 200 {array}    ExtensionsTypes
// @Resource /api/2.0
// @Router /api/2.0/extensions_types [get]
func getExtensionsTypes(db *sqlx.DB) (interface{}, error) {
	ret := []ExtensionsTypes{}
	queryStr := "select *, concat('" + API_PATH + "extensions_types/', name) as self"
	queryStr += " from extensions_types"
	err := db.Select(&ret, queryStr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return ret, nil
}

// @Title postExtensionsTypes
// @Description enter a new extensions_types
// @Accept  application/json
// @Param                 Body body     ExtensionsTypes   true "ExtensionsTypes object that should be added to the table"
// @Success 200 {object}    output_format.ApiWrapper
// @Resource /api/2.0
// @Router /api/2.0/extensions_types [post]
func postExtensionsType(payload []byte, db *sqlx.DB) (interface{}, error) {
	var v ExtensionsTypes
	err := json.Unmarshal(payload, &v)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	sqlString := "INSERT INTO extensions_types("
	sqlString += "name"
	sqlString += ",description"
	sqlString += ",created_at"
	sqlString += ") VALUES ("
	sqlString += ":name"
	sqlString += ",:description"
	sqlString += ",:created_at"
	sqlString += ")"
	result, err := db.NamedExec(sqlString, v)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return result, err
}

// @Title putExtensionsTypes
// @Description modify an existing extensions_typesentry
// @Accept  application/json
// @Param   id              path    int     true        "The row id"
// @Param                 Body body     ExtensionsTypes   true "ExtensionsTypes object that should be added to the table"
// @Success 200 {object}    output_format.ApiWrapper
// @Resource /api/2.0
// @Router /api/2.0/extensions_types/{id}  [put]
func putExtensionsType(name string, payload []byte, db *sqlx.DB) (interface{}, error) {
	var arg ExtensionsTypes
	err := json.Unmarshal(payload, &arg)
	arg.Name = name
	if err != nil {
		log.Println(err)
		return nil, err
	}
	sqlString := "UPDATE extensions_types SET "
	sqlString += "name = :name"
	sqlString += ",description = :description"
	sqlString += ",created_at = :created_at"
	sqlString += " WHERE name=:name"
	result, err := db.NamedExec(sqlString, arg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return result, err
}

// @Title delExtensionsTypesById
// @Description deletes extensions_types information for a certain id
// @Accept  application/json
// @Param   id              path    int     false        "The row id"
// @Success 200 {array}    ExtensionsTypes
// @Resource /api/2.0
// @Router /api/2.0/extensions_types/{id} [delete]
func delExtensionsType(name string, db *sqlx.DB) (interface{}, error) {
	arg := ExtensionsTypes{}
	arg.Name = name
	result, err := db.NamedExec("DELETE FROM extensions_types WHERE name=:name", arg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return result, err
}
