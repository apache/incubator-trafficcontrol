package asn

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

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	validation "github.com/go-ozzo/ozzo-validation"
)

//we need a type alias to define functions on
type TOASNV11 struct {
	ReqInfo *api.APIInfo `json:"-"`
	tc.ASNNullable
}

func (v *TOASNV11) APIInfo() *api.APIInfo         { return v.ReqInfo }
func (v *TOASNV11) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOASNV11) InsertQuery() string           { return insertQuery() }
func (v *TOASNV11) NewReadObj() interface{}       { return &tc.ASNNullable{} }
func (v *TOASNV11) SelectQuery() string           { return selectQuery() }
func (v *TOASNV11) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"asn":            dbhelpers.WhereColumnInfo{"a.asn", nil},
		"cachegroup":     dbhelpers.WhereColumnInfo{"c.id", nil},
		"id":             dbhelpers.WhereColumnInfo{"a.id", api.IsInt},
		"cachegroupName": dbhelpers.WhereColumnInfo{"c.name", nil},
	}
}
func (v *TOASNV11) UpdateQuery() string { return updateQuery() }
func (v *TOASNV11) DeleteQuery() string { return deleteQuery() }

func GetTypeSingleton() api.CRUDFactory {
	return func(reqInfo *api.APIInfo) api.CRUDer {
		toReturn := TOASNV11{reqInfo, tc.ASNNullable{}}
		return &toReturn
	}
}

func (asn TOASNV11) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"id", api.GetIntKey}}
}

// func (asn TOASNV12) GetKeyFieldsInfo() []api.KeyFieldInfo { return TOASNV11(asn).GetKeyFieldsInfo() }

// Implementation of the Identifier, Validator interface functions

func (asn TOASNV11) GetKeys() (map[string]interface{}, bool) {
	if asn.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *asn.ID}, true
}

func (asn *TOASNV11) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	asn.ID = &i
}

func (asn TOASNV11) GetAuditName() string {
	if asn.ASN != nil {
		return strconv.Itoa(*asn.ASN)
	}
	if asn.ID != nil {
		return strconv.Itoa(*asn.ID)
	}
	return "unknown"
}

func (asn TOASNV11) GetType() string {
	return "asn"
}

func (asn TOASNV11) Validate() error {
	errs := validation.Errors{
		"asn":          validation.Validate(asn.ASN, validation.NotNil, validation.Min(0)),
		"cachegroupId": validation.Validate(asn.CachegroupID, validation.NotNil, validation.Min(0)),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (as *TOASNV11) Create() (error, error, int)              { return api.GenericCreate(as) }
func (as *TOASNV11) Read() ([]interface{}, error, error, int) { return api.GenericRead(as) }
func (as *TOASNV11) Update() (error, error, int)              { return api.GenericUpdate(as) }
func (as *TOASNV11) Delete() (error, error, int)              { return api.GenericDelete(as) }

// V11ReadAll implements the asns 1.1 route, which is different from the 1.1 route for a single ASN and from 1.2+ routes, in that it wraps the content in an additional "asns" object.
func V11ReadAll(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	asns, userErr, sysErr, errCode := api.GenericRead(&TOASNV11{ReqInfo: inf})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	api.WriteResp(w, r, tc.ASNsV11{asns})
}

func selectQuery() string {
	return `
SELECT
 a.id,
 a.asn,
 a.last_updated,
 a.cachegroup AS cachegroup_id,
 c.name AS cachegroup
FROM
  asn a
JOIN
  cachegroup c ON a.cachegroup = c.id
`
}

func insertQuery() string {
	return `
INSERT INTO
  asn (asn, cachegroup)
VALUES
  (:asn, :cachegroup_id)
RETURNING id, last_updated
`
}

func updateQuery() string {
	return `
UPDATE
  asn
SET
  asn        = :asn,
  cachegroup = :cachegroup_id
WHERE
  id = :id
RETURNING
  last_updated
`
}

func deleteQuery() string {
	return `DELETE FROM asn WHERE id=:id`
}
