package servercapability

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
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"net/http"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
)

type TOServerCapability struct {
	api.APIInfoImpl `json:"-"`
	tc.ServerCapability
}

func (v *TOServerCapability) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOServerCapability) NewReadObj() interface{}       { return &tc.ServerCapability{} }

func (v *TOServerCapability) InsertQuery() string {
	return `
INSERT INTO server_capability (
  name
)
VALUES (
  :name
)
RETURNING last_updated
`
}

func (v *TOServerCapability) SelectQuery() string {
	return `
SELECT
  name,
  last_updated
FROM
  server_capability sc
`
}

func (v *TOServerCapability) DeleteQuery() string {
	return `
DELETE FROM server_capability WHERE name=:name
`
}

func (v *TOServerCapability) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"name": {"sc.name", nil},
	}
}

func (v TOServerCapability) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"name", api.GetStringKey}}
}

// Implementation of the Identifier, Validator interface functions
func (v TOServerCapability) GetKeys() (map[string]interface{}, bool) {
	return map[string]interface{}{"name": v.Name}, true
}

func (v *TOServerCapability) SetKeys(keys map[string]interface{}) {
	v.Name, _ = keys["name"].(string)
}

func (v *TOServerCapability) GetAuditName() string {
	return v.Name
}

func (v *TOServerCapability) GetType() string {
	return "server capability"
}

func (v *TOServerCapability) Validate() error {
	rule := validation.NewStringRule(tovalidate.IsAlphanumericUnderscoreDash, "must consist of only alphanumeric, dash, or underscore characters")
	errs := validation.Errors{
		"name": validation.Validate(v.Name, validation.Required, rule),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (v *TOServerCapability) Read(h map[string][]string) ([]interface{}, error, error, int) {
	ims := h["If-Modified-Since"]
	var modifiedSince time.Time
	var res []interface{}

	if ims == nil || len(ims) == 0 {
		return api.GenericRead(v)
	}
	if t, err := time.Parse(time.RFC1123, ims[0]); err != nil {
		return nil, err, nil, http.StatusBadRequest
	} else {
		modifiedSince = t
	}
	results, e1, e2, code := api.GenericRead(v)
	if e1 != nil || e2 != nil || len(results) == 0 {
		return results, e1, e2, code
	}
	for _, r := range results {
		obj := r.(*tc.ServerCapability)
		if !obj.LastUpdated.Before(modifiedSince) {
			return results, e1, e2, code
		}
	}
	return res, e1, e2, http.StatusNotModified
}
func (v *TOServerCapability) Create() (error, error, int) { return api.GenericCreateNameBasedID(v) }
func (v *TOServerCapability) Delete() (error, error, int) { return api.GenericDelete(v) }
