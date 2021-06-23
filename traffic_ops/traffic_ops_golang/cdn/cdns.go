package cdn

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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/asaskevich/govalidator"
	validation "github.com/go-ozzo/ozzo-validation"
)

//we need a type alias to define functions on
type TOCDN struct {
	api.InfoerImpl `json:"-"`
	tc.CDNNullable
}

func (v *TOCDN) GetLastUpdated() (*time.Time, bool, error) {
	return api.GetLastUpdated(v.Info().Tx, *v.ID, "cdn")
}

func (v *TOCDN) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(last_updated) as t from ` + tableName + ` c ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='` + tableName + `') as res`
}

func (v *TOCDN) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOCDN) InsertQuery() string           { return insertQuery() }
func (v *TOCDN) NewReadObj() interface{}       { return &tc.CDNNullable{} }
func (v *TOCDN) SelectQuery() string           { return selectQuery() }
func (v *TOCDN) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"domainName":    dbhelpers.WhereColumnInfo{Column: "domain_name"},
		"dnssecEnabled": dbhelpers.WhereColumnInfo{Column: "dnssec_enabled"},
		"id":            dbhelpers.WhereColumnInfo{Column: "id", Checker: api.IsInt},
		"name":          dbhelpers.WhereColumnInfo{Column: "name"},
	}
}
func (v *TOCDN) UpdateQuery() string { return updateQuery() }
func (v *TOCDN) DeleteQuery() string { return deleteQuery() }

func (cdn TOCDN) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (cdn TOCDN) GetKeys() (map[string]interface{}, bool) {
	if cdn.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *cdn.ID}, true
}

func (cdn TOCDN) GetAuditName() string {
	if cdn.Name != nil {
		return *cdn.Name
	}
	if cdn.ID != nil {
		return strconv.Itoa(*cdn.ID)
	}
	return "0"
}

func (cdn TOCDN) GetType() string {
	return "cdn"
}

func (cdn *TOCDN) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	cdn.ID = &i
}

func isValidCDNchar(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	if r == '.' || r == '-' {
		return true
	}
	return false
}

// IsValidCDNName returns true if the name contains only characters valid for a CDN name
func IsValidCDNName(str string) bool {
	i := strings.IndexFunc(str, func(r rune) bool { return !isValidCDNchar(r) })
	return i == -1
}

// Validate fulfills the api.Validator interface
func (cdn TOCDN) Validate() error {
	validName := validation.NewStringRule(IsValidCDNName, "invalid characters found - Use alphanumeric . or - .")
	validDomainName := validation.NewStringRule(govalidator.IsDNSName, "not a valid domain name")
	errs := validation.Errors{
		"name":       validation.Validate(cdn.Name, validation.Required, validName),
		"domainName": validation.Validate(cdn.DomainName, validation.Required, validDomainName),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (cdn *TOCDN) Create() (error, error, int) {
	*cdn.DomainName = strings.ToLower(*cdn.DomainName)
	return api.GenericCreate(cdn)
}

func (cdn *TOCDN) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	api.DefaultSort(cdn.Info(), "name")
	return api.GenericRead(h, cdn, useIMS)
}

func (cdn *TOCDN) Update(h http.Header) (error, error, int) {
	*cdn.DomainName = strings.ToLower(*cdn.DomainName)
	return api.GenericUpdate(h, cdn)
}

func (cdn *TOCDN) Delete() (error, error, int) { return api.GenericDelete(cdn) }

func selectQuery() string {
	query := `SELECT
dnssec_enabled,
domain_name,
id,
last_updated,
name

FROM cdn c`
	return query
}

func updateQuery() string {
	query := `UPDATE
cdn SET
dnssec_enabled=:dnssec_enabled,
domain_name=:domain_name,
name=:name
WHERE id=:id RETURNING last_updated`
	return query
}

func insertQuery() string {
	query := `INSERT INTO cdn (
dnssec_enabled,
domain_name,
name) VALUES (
:dnssec_enabled,
:domain_name,
:name) RETURNING id,last_updated`
	return query
}

func deleteQuery() string {
	query := `DELETE FROM cdn
WHERE id=:id`
	return query
}
