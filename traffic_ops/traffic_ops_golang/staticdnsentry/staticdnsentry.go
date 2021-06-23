package staticdnsentry

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
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
)

type TOStaticDNSEntry struct {
	api.InfoerImpl `json:"-"`
	tc.StaticDNSEntryNullable
}

func (v *TOStaticDNSEntry) GetLastUpdated() (*time.Time, bool, error) {
	return api.GetLastUpdated(v.Info().Tx, *v.ID, "staticdnsentry")
}

func (v *TOStaticDNSEntry) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOStaticDNSEntry) InsertQuery() string           { return insertQuery() }
func (v *TOStaticDNSEntry) NewReadObj() interface{}       { return &tc.StaticDNSEntryNullable{} }
func (v *TOStaticDNSEntry) SelectQuery() string           { return selectQuery() }
func (v *TOStaticDNSEntry) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"address":           dbhelpers.WhereColumnInfo{Column: "sde.address"},
		"cachegroup":        dbhelpers.WhereColumnInfo{Column: "cg.name"},
		"cachegroupId":      dbhelpers.WhereColumnInfo{Column: "cg.id"},
		"deliveryservice":   dbhelpers.WhereColumnInfo{Column: "ds.xml_id"},
		"deliveryserviceId": dbhelpers.WhereColumnInfo{Column: "sde.deliveryservice"},
		"host":              dbhelpers.WhereColumnInfo{Column: "sde.host"},
		"id":                dbhelpers.WhereColumnInfo{Column: "sde.id"},
		"ttl":               dbhelpers.WhereColumnInfo{Column: "sde.ttl"},
		"type":              dbhelpers.WhereColumnInfo{Column: "tp.name"},
		"typeId":            dbhelpers.WhereColumnInfo{Column: "tp.id"},
	}
}
func (v *TOStaticDNSEntry) UpdateQuery() string { return updateQuery() }
func (v *TOStaticDNSEntry) DeleteQuery() string { return deleteQuery() }

func (staticDNSEntry TOStaticDNSEntry) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (staticDNSEntry TOStaticDNSEntry) GetKeys() (map[string]interface{}, bool) {
	if staticDNSEntry.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *staticDNSEntry.ID}, true
}

func (staticDNSEntry TOStaticDNSEntry) GetAuditName() string {
	if staticDNSEntry.Host != nil {
		return *staticDNSEntry.Host
	}
	if staticDNSEntry.ID != nil {
		return strconv.Itoa(*staticDNSEntry.ID)
	}
	return "0"
}

func (staticDNSEntry TOStaticDNSEntry) GetType() string { return "staticDNSEntry" }

func (staticDNSEntry *TOStaticDNSEntry) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	staticDNSEntry.ID = &i
}

// Validate fulfills the api.Validator interface
func (staticDNSEntry TOStaticDNSEntry) Validate() error {
	typeStr, err := tc.ValidateTypeID(staticDNSEntry.ReqInfo.Tx.Tx, &staticDNSEntry.TypeID, "staticdnsentry")
	if err != nil {
		return err
	}

	var addressErr, ttlErr error
	switch typeStr {
	case "A_RECORD":
		addressErr = validation.Validate(staticDNSEntry.Address, validation.Required, is.IPv4)
	case "AAAA_RECORD":
		addressErr = validation.Validate(staticDNSEntry.Address, validation.Required, is.IPv6)
	case "CNAME_RECORD":
		addressErr = validation.Validate(staticDNSEntry.Address, validation.Required, is.DNSName)
		address := *staticDNSEntry.Address
		if addressErr == nil {
			lastChar := address[len(address)-1:]
			if lastChar != "." {
				addressErr = fmt.Errorf("for type: CNAME_RECORD must have a trailing period")
			}
		}
	default:
		addressErr = validation.Validate(staticDNSEntry.Address, validation.Required)
	}

	if staticDNSEntry.TTL != nil {
		if *staticDNSEntry.TTL == 0 {
			ttlErr = validation.Validate(staticDNSEntry.TTL, is.Digit)
		}
	} else {
		ttlErr = validation.Validate(staticDNSEntry.TTL, validation.Required)
	}

	errs := validation.Errors{
		"host":              validation.Validate(staticDNSEntry.Host, validation.Required, is.DNSName),
		"address":           addressErr,
		"deliveryserviceId": validation.Validate(staticDNSEntry.DeliveryServiceID, validation.Required),
		"ttl":               ttlErr,
		"typeId":            validation.Validate(staticDNSEntry.TypeID, validation.Required),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (en *TOStaticDNSEntry) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	api.DefaultSort(en.Info(), "host")
	return api.GenericRead(h, en, useIMS)
}
func (en *TOStaticDNSEntry) Create() (error, error, int) { return api.GenericCreate(en) }
func (en *TOStaticDNSEntry) Update(h http.Header) (error, error, int) {
	return api.GenericUpdate(h, en)
}
func (en *TOStaticDNSEntry) Delete() (error, error, int) { return api.GenericDelete(en) }
func (v *TOStaticDNSEntry) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(sde.last_updated) as t FROM staticdnsentry as sde
JOIN type as tp on sde.type = tp.id
LEFT JOIN cachegroup as cg ON sde.cachegroup = cg.id
JOIN deliveryservice as ds on sde.deliveryservice = ds.id ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='staticdnsentry') as res`
}

func insertQuery() string {
	query := `INSERT INTO staticdnsentry (
address,
deliveryservice,
cachegroup,
host,
type,
ttl) VALUES (
:address,
:deliveryservice_id,
:cachegroup_id,
:host,
:type_id,
:ttl) RETURNING id,last_updated`
	return query
}

func updateQuery() string {
	query := `UPDATE
staticdnsentry SET
id=:id,
address=:address,
deliveryservice=:deliveryservice_id,
cachegroup=:cachegroup_id,
host=:host,
type=:type_id,
ttl=:ttl
WHERE id=:id RETURNING last_updated`
	return query
}

func selectQuery() string {
	return `SELECT
ds.xml_id as dsname,
sde.host,
sde.id as id,
sde.deliveryservice as deliveryservice_id,
sde.ttl,
sde.address,
sde.last_updated,
tp.id as type_id,
tp.name as type,
cg.id as cachegroup_id,
cg.name as cachegroup
FROM staticdnsentry as sde
JOIN type as tp on sde.type = tp.id
LEFT JOIN cachegroup as cg ON sde.cachegroup = cg.id
JOIN deliveryservice as ds on sde.deliveryservice = ds.id
`
}

func deleteQuery() string {
	query := `DELETE FROM staticdnsentry
WHERE id=:id`
	return query
}
