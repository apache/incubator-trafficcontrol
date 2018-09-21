package cdnfederation

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
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/asaskevich/govalidator"
	"github.com/go-ozzo/ozzo-validation"
)

// we need a type alias to define functions on
type TOCDNFederation struct {
	ReqInfo *api.APIInfo `json:"-"`
	tc.CDNFederation
	TenantID *int `json:"-" db:"tenant_id"`
}

func (v *TOCDNFederation) APIInfo() *api.APIInfo         { return v.ReqInfo }
func (v *TOCDNFederation) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOCDNFederation) InsertQuery() string           { return insertQuery() }
func (v *TOCDNFederation) NewReadObj() interface{}       { return &TOCDNFederation{} }
func (v *TOCDNFederation) SelectQuery() string {
	if v.ID != nil {
		return selectByID()
	}
	return selectByCDNName()
}
func (v *TOCDNFederation) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	cols := map[string]dbhelpers.WhereColumnInfo{
		"id": dbhelpers.WhereColumnInfo{Column: "federation.id", Checker: api.IsInt},
	}
	if v.ID == nil {
		cols["name"] = dbhelpers.WhereColumnInfo{Column: "cdn.name", Checker: nil}
	}
	return cols
}
func (v *TOCDNFederation) DeleteQuery() string { return deleteQuery() }
func (v *TOCDNFederation) UpdateQuery() string { return updateQuery() }

// Used for all CRUD routes
func GetTypeSingleton() api.CRUDFactory {
	return func(reqInfo *api.APIInfo) api.CRUDer {
		return &TOCDNFederation{reqInfo, tc.CDNFederation{}, nil}
	}
}

// Fufills `Identifier' interface
func (fed TOCDNFederation) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

// Fufills `Identifier' interface
func (fed TOCDNFederation) GetKeys() (map[string]interface{}, bool) {
	if fed.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *fed.ID}, true
}

// Fufills `Identifier' interface
func (fed TOCDNFederation) GetAuditName() string {
	if fed.CName != nil {
		return *fed.CName
	}
	if fed.ID != nil {
		return strconv.Itoa(*fed.ID)
	}
	return "unknown"
}

// Fufills `Identifier' interface
func (fed TOCDNFederation) GetType() string {
	return "cdnfederation"
}

// Fufills `Create' interface
func (fed *TOCDNFederation) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) // non-panicking type assertion
	fed.ID = &i
}

// Fulfills `Validate' interface
func (fed *TOCDNFederation) Validate() error {

	isDNSName := validation.NewStringRule(govalidator.IsDNSName, "must be a valid hostname")
	endsWithDot := validation.NewStringRule(
		func(str string) bool {
			return strings.HasSuffix(str, ".")
		}, "must end with a period")

	// cname regex: (^\S*\.$), ttl regex: (^\d+$)
	validateErrs := validation.Errors{
		"cname": validation.Validate(fed.CName, validation.Required, endsWithDot, isDNSName),
		"ttl":   validation.Validate(fed.TTL, validation.Required, validation.Min(0)),
	}
	return util.JoinErrs(tovalidate.ToErrors(validateErrs))
}

// fedAPIInfo.Params["name"] is not used on creation, rather the cdn name
// is connected when the federations/:id/deliveryservice links a federation
// Note: cdns and deliveryservies have a 1-1 relationship
func (fed *TOCDNFederation) Create() (error, error, int) {
	// Deliveryservice IDs should not be included on create.
	if fed.DeliveryServiceIDs != nil {
		fed.DsId = nil
		fed.XmlId = nil
		fed.DeliveryServiceIDs = nil
	}
	return api.GenericCreate(fed)
}

// returning true indicates the data related to the given tenantID should be visible
// `tenantIDs` is presumed to be unsorted, and a nil tenantID is viewable by everyone
func checkTenancy(tenantID *int, tenantIDs []int) bool {
	if tenantID == nil {
		return true
	}
	for _, id := range tenantIDs {
		if id == *tenantID {
			return true
		}
	}
	return false
}

func (fed *TOCDNFederation) Read() ([]interface{}, error, error, int) {
	if idstr, ok := fed.APIInfo().Params["id"]; ok {
		id, err := strconv.Atoi(idstr)
		if err != nil {
			return nil, errors.New("id must be an integer"), nil, http.StatusBadRequest
		}
		fed.ID = util.IntPtr(id)
	}

	tenantIDs, err := tenant.GetUserTenantIDListTx(fed.APIInfo().Tx.Tx, fed.APIInfo().User.TenantID)
	if err != nil {
		return nil, nil, errors.New("getting tenant list for user: " + err.Error()), http.StatusInternalServerError
	}

	federations, userErr, sysErr, errCode := api.GenericRead(fed)
	if userErr != nil || sysErr != nil {
		return nil, userErr, sysErr, errCode
	}

	filteredFederations := []interface{}{}
	for _, ifederation := range federations {
		federation := ifederation.(*TOCDNFederation)
		if !checkTenancy(federation.TenantID, tenantIDs) {
			return nil, errors.New("user not authorized for requested federation"), nil, http.StatusForbidden
		}
		filteredFederations = append(filteredFederations, federation.CDNFederation)
	}

	if len(filteredFederations) == 0 {
		if fed.ID != nil {
			return nil, errors.New("not found"), nil, http.StatusNotFound
		}
		if ok, err := dbhelpers.CDNExists(fed.APIInfo().Params["name"], fed.APIInfo().Tx); err != nil {
			return nil, nil, errors.New("verifying CDN exists: " + err.Error()), http.StatusInternalServerError
		} else if !ok {
			return nil, errors.New("cdn not found"), nil, http.StatusNotFound
		}
	}
	return filteredFederations, nil, nil, http.StatusOK
}

func (fed *TOCDNFederation) Update() (error, error, int) {
	userErr, sysErr, errCode := fed.isTenantAuthorized()
	if userErr != nil || sysErr != nil {
		return userErr, sysErr, errCode
	}
	// Deliveryservice IDs should not be included on update.
	if fed.DeliveryServiceIDs != nil {
		fed.DsId = nil
		fed.XmlId = nil
		fed.DeliveryServiceIDs = nil
	}
	return api.GenericUpdate(fed)
}

// Delete implements the Deleter interface for TOCDNFederation.
// In the perl version, :name is ignored. It is not even verified whether or not
// :name is a real cdn that exists. This mimicks the perl behavior.
func (fed *TOCDNFederation) Delete() (error, error, int) {
	userErr, sysErr, errCode := fed.isTenantAuthorized()
	if userErr != nil || sysErr != nil {
		return userErr, sysErr, errCode
	}
	return api.GenericDelete(fed)
}

func (fed TOCDNFederation) isTenantAuthorized() (error, error, int) {
	tenantID, err := getTenantIDFromFedID(*fed.ID, fed.APIInfo().Tx.Tx)
	if err != nil {
		// If nobody has claimed a tenant, that federation is publicly visible.
		// This logically follows /federations/:id/deliveryservices
		if err == sql.ErrNoRows {
			return nil, nil, http.StatusOK
		}
		return nil, errors.New("getting tenant id from federation: " + err.Error()), http.StatusInternalServerError
	}

	// TODO: After IsResourceAuthorizedToUserTx is updated to no longer have `use_tenancy`,
	// that will probably be better to use. For now, use the list. Issue #2602
	list, err := tenant.GetUserTenantIDListTx(fed.APIInfo().Tx.Tx, fed.APIInfo().User.TenantID)
	if err != nil {
		return nil, errors.New("getting federation tenant list: " + err.Error()), http.StatusInternalServerError
	}
	for _, id := range list {
		if id == tenantID {
			return nil, nil, http.StatusOK
		}
	}
	return errors.New("unauthorized for tenant"), nil, http.StatusForbidden
}

func getTenantIDFromFedID(id int, tx *sql.Tx) (int, error) {
	tenantID := 0
	query := `
	SELECT ds.tenant_id FROM federation AS f
	JOIN federation_deliveryservice AS fd ON f.id = fd.federation
	JOIN deliveryservice AS ds ON ds.id = fd.deliveryservice
	WHERE f.id = $1`
	err := tx.QueryRow(query, id).Scan(&tenantID)
	return tenantID, err
}

func selectByID() string {
	return `
	SELECT
	ds.tenant_id,
	federation.id AS id,
	federation.cname,
	federation.ttl,
	federation.description,
	federation.last_updated,
	ds.id AS ds_id,
	ds.xml_id
	FROM federation
	LEFT JOIN federation_deliveryservice AS fd ON federation.id = fd.federation
	LEFT JOIN deliveryservice AS ds ON ds.id = fd.deliveryservice`
	// WHERE federation.id = :id (determined by dbhelper)
}

func selectByCDNName() string {
	return `
	SELECT
	ds.tenant_id,
	federation.id AS id,
	federation.cname,
	federation.ttl,
	federation.description,
	federation.last_updated,
	ds.id AS ds_id,
	ds.xml_id
	FROM federation
	JOIN federation_deliveryservice AS fd ON federation.id = fd.federation
	JOIN deliveryservice AS ds ON ds.id = fd.deliveryservice
	JOIN cdn ON cdn.id = cdn_id`
	// WHERE cdn.name = :cdn_name (determined by dbhelper)
}

func updateQuery() string {
	return `
UPDATE federation SET
	cname = :cname,
	ttl = :ttl,
	description = :description
WHERE
  id=:id
RETURNING last_updated`
}

func insertQuery() string {
	return `
	INSERT INTO federation (
	cname,
 	ttl,
 	description
  ) VALUES (
 	:cname,
	:ttl,
	:description
	) RETURNING id, last_updated`
}

func deleteQuery() string {
	return `DELETE FROM federation WHERE id = :id`
}
