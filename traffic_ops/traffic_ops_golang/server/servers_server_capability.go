package server

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
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	ServerCapabilityQueryParam = "serverCapability"
	ServerQueryParam           = "serverId"
	ServerHostNameQueryParam   = "serverHostName"
)

type (
	TOServerServerCapability struct {
		api.InfoerImpl `json:"-"`
		tc.ServerServerCapability
	}

	DSTenant struct {
		TenantID int `db:"tenant_id"`
		ID       int `db:"id"`
	}
)

func (ssc *TOServerServerCapability) SetLastUpdated(t tc.TimeNoMod) { ssc.LastUpdated = &t }
func (ssc *TOServerServerCapability) NewReadObj() interface{} {
	return &tc.ServerServerCapability{}
}
func (ssc *TOServerServerCapability) SelectQuery() string { return scSelectQuery() }
func (ssc *TOServerServerCapability) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		ServerCapabilityQueryParam: dbhelpers.WhereColumnInfo{Column: "sc.server_capability"},
		ServerQueryParam:           dbhelpers.WhereColumnInfo{Column: "s.id", Checker: api.IsInt},
		ServerHostNameQueryParam:   dbhelpers.WhereColumnInfo{Column: "s.host_name"},
	}

}
func (ssc *TOServerServerCapability) DeleteQuery() string { return scDeleteQuery() }
func (ssc TOServerServerCapability) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{
		{Field: ServerQueryParam, Func: api.GetIntKey},
		{Field: ServerCapabilityQueryParam, Func: api.GetStringKey},
	}
}

// Need to satisfy Identifier interface but is a no-op as path does not have Update
func (ssc TOServerServerCapability) GetKeys() (map[string]interface{}, bool) {
	if ssc.ServerID == nil {
		return map[string]interface{}{ServerQueryParam: 0}, false
	}
	if ssc.ServerCapability == nil {
		return map[string]interface{}{ServerCapabilityQueryParam: 0}, false
	}
	return map[string]interface{}{
		ServerQueryParam:           *ssc.ServerID,
		ServerCapabilityQueryParam: *ssc.ServerCapability,
	}, true
}

func (ssc *TOServerServerCapability) SetKeys(keys map[string]interface{}) {
	sID, _ := keys[ServerQueryParam].(int)
	ssc.ServerID = &sID

	sc, _ := keys[ServerCapabilityQueryParam].(string)
	ssc.ServerCapability = &sc
}

func (ssc *TOServerServerCapability) GetAuditName() string {
	if ssc.ServerCapability != nil {
		return *ssc.ServerCapability
	}
	return "unknown"
}

func (ssc *TOServerServerCapability) GetType() string {
	return "server server_capability"
}

// Validate fulfills the api.Validator interface
func (ssc TOServerServerCapability) Validate() error {
	errs := validation.Errors{
		ServerQueryParam:           validation.Validate(ssc.ServerID, validation.Required),
		ServerCapabilityQueryParam: validation.Validate(ssc.ServerCapability, validation.Required),
	}

	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (ssc *TOServerServerCapability) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	api.DefaultSort(ssc.Info(), "serverHostName")
	return api.GenericRead(h, ssc, useIMS)
}
func (v *TOServerServerCapability) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(sc.last_updated) as t from server_server_capability sc
JOIN server s ON sc.server = s.id ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='server_server_capability') as res`
}

func (ssc *TOServerServerCapability) Delete() (error, error, int) {
	tenantIDs, err := tenant.GetUserTenantIDListTx(ssc.Info().Tx.Tx, ssc.Info().User.TenantID)
	if err != nil {
		return nil, fmt.Errorf("deleting servers_server_capability: %v", err), http.StatusInternalServerError
	}
	accessibleTenants := make(map[int]struct{}, len(tenantIDs))
	for _, id := range tenantIDs {
		accessibleTenants[id] = struct{}{}
	}
	userErr, sysErr, status := checkTopologyBasedDSRequiredCapabilities(ssc, accessibleTenants)
	if userErr != nil || sysErr != nil {
		return userErr, sysErr, status
	}

	userErr, sysErr, status = checkDSRequiredCapabilities(ssc, accessibleTenants)
	if userErr != nil || sysErr != nil {
		return userErr, sysErr, status
	}

	return api.GenericDelete(ssc)
}

func checkTopologyBasedDSRequiredCapabilities(ssc *TOServerServerCapability, accessibleTenants map[int]struct{}) (error, error, int) {
	dsRows, err := ssc.Info().Tx.Tx.Query(getTopologyBasedDSesReqCapQuery(), ssc.ServerID, ssc.ServerCapability)
	if err != nil {
		return nil, fmt.Errorf("querying topology-based DSes with the required capability %s: %v", *ssc.ServerCapability, err), http.StatusInternalServerError
	}
	defer log.Close(dsRows, "closing dsRows in checkTopologyBasedDSRequiredCapabilities")

	xmlidToTopology := make(map[string]string)
	xmlidToTenantID := make(map[string]int)
	xmlidToReqCaps := make(map[string][]string)
	for dsRows.Next() {
		xmlID := ""
		topology := ""
		tenantID := 0
		reqCaps := []string{}
		if err := dsRows.Scan(&xmlID, &topology, &tenantID, pq.Array(&reqCaps)); err != nil {
			return nil, fmt.Errorf("scanning dsRows in checkTopologyBasedDSRequiredCapabilities: %v", err), http.StatusInternalServerError
		}
		xmlidToTenantID[xmlID] = tenantID
		xmlidToTopology[xmlID] = topology
		xmlidToReqCaps[xmlID] = reqCaps
	}
	if len(xmlidToTopology) == 0 {
		return nil, nil, http.StatusOK
	}

	serverRows, err := ssc.Info().Tx.Tx.Query(getServerCapabilitiesOfCachegoupQuery(), ssc.ServerID, ssc.ServerCapability)
	if err != nil {
		return nil, fmt.Errorf("querying server capabilitites of server %d's cachegroup: %v", *ssc.ServerID, err), http.StatusInternalServerError
	}
	defer log.Close(serverRows, "closing serverRows in checkTopologyBasedDSRequiredCapabilities")

	serverIDToCapabilities := make(map[int]map[string]struct{})
	for serverRows.Next() {
		serverID := 0
		capabilities := []string{}
		if err := serverRows.Scan(&serverID, pq.Array(&capabilities)); err != nil {
			return nil, fmt.Errorf("scanning serverRows in checkTopologyBasedDSRequiredCapabilities: %v", err), http.StatusInternalServerError
		}
		serverIDToCapabilities[serverID] = make(map[string]struct{})
		for _, c := range capabilities {
			serverIDToCapabilities[serverID][c] = struct{}{}
		}
	}

	unsatisfiedDSes := []string{}
	for ds, dsReqCaps := range xmlidToReqCaps {
		dsIsSatisfied := false
		for _, serverCaps := range serverIDToCapabilities {
			serverHasCapabilities := true
			for _, dsReqCap := range dsReqCaps {
				if _, ok := serverCaps[dsReqCap]; !ok {
					serverHasCapabilities = false
					break
				}
			}
			if serverHasCapabilities {
				dsIsSatisfied = true
				break
			}
		}
		if !dsIsSatisfied {
			unsatisfiedDSes = append(unsatisfiedDSes, ds)
		}
	}
	if len(unsatisfiedDSes) == 0 {
		return nil, nil, http.StatusOK
	}

	dsStrings := make([]string, 0, len(unsatisfiedDSes))
	for _, ds := range unsatisfiedDSes {
		if _, ok := accessibleTenants[xmlidToTenantID[ds]]; ok {
			dsStrings = append(dsStrings, "(xml_id = "+ds+", topology = "+xmlidToTopology[ds]+")")
		}
	}
	return fmt.Errorf("this capability is required by delivery services, but there are no other servers in this server's cachegroup to satisfy them %s", strings.Join(dsStrings, ", ")), nil, http.StatusBadRequest
}

func checkDSRequiredCapabilities(ssc *TOServerServerCapability, accessibleTenants map[int]struct{}) (error, error, int) {
	// Ensure that the user is not removing a server capability from the server
	// that is required by the delivery services the server is assigned to (if applicable)
	dsIDs := []int64{}
	if err := ssc.Info().Tx.Tx.QueryRow(checkDSReqCapQuery(), ssc.ServerID, ssc.ServerCapability).Scan(pq.Array(&dsIDs)); err != nil {
		return nil, fmt.Errorf("checking removing server server capability would still suffice delivery service requried capabilites: %v", err), http.StatusInternalServerError
	}

	if len(dsIDs) > 0 {
		return ssc.buildDSReqCapError(dsIDs, accessibleTenants)
	}
	return nil, nil, http.StatusOK
}

func (ssc *TOServerServerCapability) buildDSReqCapError(dsIDs []int64, accessibleTenants map[int]struct{}) (error, error, int) {

	dsTenantIDs, err := getDSTenantIDsByIDs(ssc.Info().Tx, dsIDs)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	authDSIDs := []string{}

	for _, dsTenantID := range dsTenantIDs {
		if _, ok := accessibleTenants[dsTenantID.TenantID]; ok {
			if ok {
				authDSIDs = append(authDSIDs, strconv.Itoa(dsTenantID.ID))
			}
			continue
		}
	}

	dsStr := "delivery services"
	if len(authDSIDs) > 0 {
		dsStr = fmt.Sprintf("the delivery services %v", strings.Join(authDSIDs, ","))
	}
	return fmt.Errorf("cannot remove the capability %v from the server %v as the server is assigned to %v that require it", *ssc.ServerCapability, *ssc.ServerID, dsStr), nil, http.StatusBadRequest
}

func (ssc *TOServerServerCapability) Create() (error, error, int) {
	tx := ssc.Info().Tx

	// Check existence prior to checking type
	_, exists, err := dbhelpers.GetServerNameFromID(tx.Tx, *ssc.ServerID)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	if !exists {
		return fmt.Errorf("server %v does not exist", *ssc.ServerID), nil, http.StatusNotFound
	}

	// Ensure type is correct
	correctType := true
	if err := tx.Tx.QueryRow(scCheckServerTypeQuery(), ssc.ServerID).Scan(&correctType); err != nil {
		return nil, fmt.Errorf("checking server type: %v", err), http.StatusInternalServerError
	}
	if !correctType {
		return fmt.Errorf("server %v has an incorrect server type. Server capabilities can only be assigned to EDGE or MID servers", *ssc.ServerID), nil, http.StatusBadRequest
	}

	resultRows, err := tx.NamedQuery(scInsertQuery(), ssc)
	if err != nil {
		return api.ParseDBError(err)
	}
	defer resultRows.Close()

	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.StructScan(&ssc); err != nil {
			return nil, errors.New(ssc.GetType() + " create scanning: " + err.Error()), http.StatusInternalServerError
		}
	}
	if rowsAffected == 0 {
		return nil, errors.New(ssc.GetType() + " create: no " + ssc.GetType() + " was inserted, no rows was returned"), http.StatusInternalServerError
	} else if rowsAffected > 1 {
		return nil, errors.New("too many rows returned from " + ssc.GetType() + " insert"), http.StatusInternalServerError
	}

	return nil, nil, http.StatusOK
}

func scSelectQuery() string {
	return `SELECT
sc.server_capability,
sc.server,
sc.last_updated,
s.host_name as host_name
FROM server_server_capability sc
JOIN server s ON sc.server = s.id`
}

func scDeleteQuery() string {
	return `DELETE FROM server_server_capability
WHERE server = :server AND server_capability = :server_capability`
}

func scInsertQuery() string {
	return `INSERT INTO server_server_capability (
server_capability,
server) VALUES (
:server_capability,
:server) RETURNING server, server_capability, last_updated`
}

func scCheckServerTypeQuery() string {
	return `
SELECT EXISTS (
	SELECT s.id
	FROM server s
	JOIN type t ON s.type = t.id
	WHERE s.id = $1
	AND t.use_in_table = 'server'
	AND (t.name LIKE 'MID%' OR t.name LIKE 'EDGE%'))`
}

func checkDSReqCapQuery() string {
	return `
SELECT ARRAY(
	SELECT dsrc.deliveryservice_id
	FROM deliveryservices_required_capability as dsrc
	WHERE deliveryservice_id IN (
		SELECT deliveryservice
		FROM deliveryservice_server
		WHERE server = $1)
	AND dsrc.required_capability = $2)`
}

// get the topology-based DSes (with all their required capabilities) that a given
// server is assigned to, filtered by the given capability
func getTopologyBasedDSesReqCapQuery() string {
	return `
SELECT
  ds.xml_id,
  ds.topology,
  ds.tenant_id,
  ARRAY_AGG(dsrc.required_capability) AS req_caps
FROM server s
JOIN cachegroup c ON s.cachegroup = c.id
JOIN topology_cachegroup tc ON c.name = tc.cachegroup
JOIN deliveryservice ds ON ds.topology = tc.topology
JOIN deliveryservices_required_capability dsrc ON dsrc.deliveryservice_id = ds.id
WHERE s.id = $1
GROUP BY ds.xml_id, ds.tenant_id, ds.topology
HAVING $2 = ANY(ARRAY_AGG(dsrc.required_capability))
`
}

// get all the capabilities of the servers in a given server's cachegroup
// that have a given capability
func getServerCapabilitiesOfCachegoupQuery() string {
	return `
SELECT s.id, ARRAY_AGG(ssc.server_capability) AS capabilities
FROM server s
JOIN cachegroup c ON c.id = s.cachegroup AND c.id = (SELECT cachegroup FROM server WHERE server.id = $1)
JOIN server_server_capability ssc ON ssc.server = s.id
WHERE
  s.cdn_id = (SELECT cdn_id FROM server WHERE server.id = $1)
  AND s.id != $1
GROUP BY s.id
HAVING $2 = ANY(ARRAY_AGG(ssc.server_capability));
`
}

func getDSTenantIDsByIDs(tx *sqlx.Tx, dsIDs []int64) ([]DSTenant, error) {
	dsTenantIDs := []DSTenant{}

	query, args, err := sqlx.In("SELECT id, tenant_id FROM deliveryservice where id IN (?);", dsIDs)
	if err != nil {
		return nil, fmt.Errorf("building query for getting delivery services' tenants: %v", err)
	}
	query = tx.Rebind(query)
	resultRows, err := tx.Queryx(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying tenant IDs for delivery service IDs: %v", err)
	}
	defer log.Close(resultRows, "closing resultRows in getDSTenantIDsByIDs")

	for resultRows.Next() {
		dsTenantID := DSTenant{}
		if err := resultRows.StructScan(&dsTenantID); err != nil {
			return nil, errors.New("scanning delivery service tenant ID: " + err.Error())
		}
		dsTenantIDs = append(dsTenantIDs, dsTenantID)
	}

	return dsTenantIDs, nil
}
