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
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/jmoiron/sqlx"
)

//we need a type alias to define functions on
type TOServer struct {
	ReqInfo *api.APIInfo `json:"-"`
	tc.ServerNullable
}

func (v *TOServer) APIInfo() *api.APIInfo         { return v.ReqInfo }
func (v *TOServer) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOServer) InsertQuery() string           { return insertQuery() }
func (v *TOServer) UpdateQuery() string           { return updateQuery() }
func (v *TOServer) DeleteQuery() string           { return deleteQuery() }

func GetTypeSingleton() api.CRUDFactory {
	return func(reqInfo *api.APIInfo) api.CRUDer {
		toReturn := TOServer{reqInfo, tc.ServerNullable{}}
		return &toReturn
	}
}

func (server TOServer) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"id", api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (server TOServer) GetKeys() (map[string]interface{}, bool) {
	if server.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *server.ID}, true
}

func (server *TOServer) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	server.ID = &i
}

func (server *TOServer) GetAuditName() string {
	if server.DomainName != nil {
		return *server.DomainName
	}
	if server.ID != nil {
		return strconv.Itoa(*server.ID)
	}
	return "unknown"
}

func (server *TOServer) GetType() string {
	return "server"
}

func (server *TOServer) Sanitize() {
	if server.IP6Address != nil && *server.IP6Address == "" {
		server.IP6Address = nil
	}
}

func (server *TOServer) Validate() error {
	server.Sanitize()
	noSpaces := validation.NewStringRule(tovalidate.NoSpaces, "cannot contain spaces")

	validateErrs := validation.Errors{
		"cachegroupId":   validation.Validate(server.CachegroupID, validation.NotNil),
		"cdnId":          validation.Validate(server.CDNID, validation.NotNil),
		"domainName":     validation.Validate(server.DomainName, validation.NotNil, noSpaces),
		"hostName":       validation.Validate(server.HostName, validation.NotNil, noSpaces),
		"interfaceMtu":   validation.Validate(server.InterfaceMtu, validation.NotNil),
		"interfaceName":  validation.Validate(server.InterfaceName, validation.NotNil),
		"ipAddress":      validation.Validate(server.IPAddress, validation.NotNil, is.IPv4),
		"ipNetmask":      validation.Validate(server.IPNetmask, validation.NotNil),
		"ipGateway":      validation.Validate(server.IPGateway, validation.NotNil),
		"ip6Address":     validation.Validate(server.IP6Address, validation.By(tovalidate.IsValidIPv6CIDROrAddress)),
		"physLocationId": validation.Validate(server.PhysLocationID, validation.NotNil),
		"profileId":      validation.Validate(server.ProfileID, validation.NotNil),
		"statusId":       validation.Validate(server.StatusID, validation.NotNil),
		"typeId":         validation.Validate(server.TypeID, validation.NotNil),
		"updPending":     validation.Validate(server.UpdPending, validation.NotNil),
		"httpsPort":      validation.Validate(server.HTTPSPort, validation.By(tovalidate.IsValidPortNumber)),
		"tcpPort":        validation.Validate(server.TCPPort, validation.By(tovalidate.IsValidPortNumber)),
	}
	errs := tovalidate.ToErrors(validateErrs)
	if len(errs) > 0 {
		return util.JoinErrs(errs)
	}

	if _, err := tc.ValidateTypeID(server.ReqInfo.Tx.Tx, server.TypeID, "server"); err != nil {
		return err
	}

	rows, err := server.ReqInfo.Tx.Tx.Query("select cdn from profile where id=$1", server.ProfileID)
	if err != nil {
		log.Error.Printf("could not execute select cdnID from profile: %s\n", err)
		errs = append(errs, tc.DBError)
		return util.JoinErrs(errs)
	}
	defer rows.Close()
	var cdnID int
	for rows.Next() {
		if err := rows.Scan(&cdnID); err != nil {
			log.Error.Printf("could not scan cdnID from profile: %s\n", err)
			errs = append(errs, errors.New("associated profile must have a cdn associated"))
			return util.JoinErrs(errs)
		}
	}
	log.Infof("got cdn id: %d from profile and cdn id: %d from server", cdnID, *server.CDNID)
	if cdnID != *server.CDNID {
		errs = append(errs, errors.New(fmt.Sprintf("CDN id '%d' for profile '%d' does not match Server CDN '%d'", cdnID, *server.ProfileID, *server.CDNID)))
	}
	return util.JoinErrs(errs)
}

// ChangeLogMessage implements the api.ChangeLogger interface for a custom log message
func (server TOServer) ChangeLogMessage(action string) (string, error) {

	var status string
	if server.Status != nil {
		status = *server.Status
	}

	var hostName string
	if server.HostName != nil {
		hostName = *server.HostName
	}

	var domainName string
	if server.DomainName != nil {
		domainName = *server.DomainName
	}

	var serverID string
	if server.ID != nil {
		serverID = strconv.Itoa(*server.ID)
	}

	message := action + ` ` + status + ` server: { "hostName":"` + hostName + `", "domainName":"` + domainName + `", id:` + serverID + ` }`

	return message, nil
}

func (server *TOServer) Read() ([]interface{}, error, error, int) {
	returnable := []interface{}{}

	servers, errs, errType := getServers(server.ReqInfo.Params, server.ReqInfo.Tx, server.ReqInfo.User)
	if len(errs) > 0 {
		for _, err := range errs {
			if err.Error() == `id cannot parse to integer` {
				return nil, errors.New("Resource not found."), nil, http.StatusInternalServerError //matches perl response
			}
		}
		userErr, sysErr, errCode := api.TypeErrsToAPIErr(errs, errType)
		return nil, userErr, sysErr, errCode
	}

	for _, server := range servers {
		returnable = append(returnable, server)
	}

	return returnable, nil, nil, http.StatusOK
}

// getDeliveryServiceType returns the type of the deliveryservice
func getDeliveryServiceType(dsID int, tx *sql.Tx) (tc.DSType, error) {
	var dsType tc.DSType
	if err := tx.QueryRow(`SELECT t.name FROM deliveryservice as ds JOIN type t ON ds.type = t.id WHERE ds.id=$1`, dsID).Scan(&dsType); err != nil {
		if err == sql.ErrNoRows {
			return tc.DSTypeInvalid, errors.New("a deliveryservice with id '" + strconv.Itoa(dsID) + "' was not found")
		}
		return tc.DSTypeInvalid, errors.New("querying type from delivery service: " + err.Error())
	}
	return dsType, nil
}

func getServers(params map[string]string, tx *sqlx.Tx, user *auth.CurrentUser) ([]tc.ServerNullable, []error, tc.ApiErrorType) {
	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"cachegroup":       dbhelpers.WhereColumnInfo{"s.cachegroup", api.IsInt},
		"parentCachegroup": dbhelpers.WhereColumnInfo{"cg.parent_cachegroup_id", api.IsInt},
		"cdn":              dbhelpers.WhereColumnInfo{"s.cdn_id", api.IsInt},
		"id":               dbhelpers.WhereColumnInfo{"s.id", api.IsInt},
		"hostName":         dbhelpers.WhereColumnInfo{"s.host_name", nil},
		"physLocation":     dbhelpers.WhereColumnInfo{"s.phys_location", api.IsInt},
		"profileId":        dbhelpers.WhereColumnInfo{"s.profile", api.IsInt},
		"status":           dbhelpers.WhereColumnInfo{"st.name", nil},
		"type":             dbhelpers.WhereColumnInfo{"t.name", nil},
		"dsId":             dbhelpers.WhereColumnInfo{"dss.deliveryservice", nil},
	}

	usesMids := false
	queryAddition := ""
	if dsIDStr, ok := params[`dsId`]; ok {
		// don't allow query on ds outside user's tenant
		dsID, err := strconv.Atoi(dsIDStr)
		if err != nil {
			return nil, []error{errors.New("dsId must be an integer")}, tc.DataMissingError
		}
		userErr, sysErr, _ := tenant.CheckID(tx.Tx, user, dsID)
		if userErr != nil || sysErr != nil {
			return nil, []error{errors.New("Forbidden")}, tc.ForbiddenError
		}
		// only if dsId is part of params: add join on deliveryservice_server table
		queryAddition = `
FULL OUTER JOIN deliveryservice_server dss ON dss.server = s.id
`
		// depending on ds type, also need to add mids
		dsType, err := getDeliveryServiceType(dsID, tx.Tx)
		if err != nil {
			return nil, []error{err}, tc.DataConflictError
		}
		usesMids = dsType.UsesMidCache()
		log.Debugf("Servers for ds %d; uses mids? %v\n", dsID, usesMids)
	}

	where, orderBy, queryValues, errs := dbhelpers.BuildWhereAndOrderBy(params, queryParamsToSQLCols)
	if len(errs) > 0 {
		return nil, errs, tc.DataConflictError
	}

	query := selectQuery() + queryAddition + where + orderBy
	log.Debugln("Query is ", query)

	rows, err := tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, []error{fmt.Errorf("querying: %v", err)}, tc.SystemError
	}
	defer rows.Close()

	servers := []tc.ServerNullable{}

	HiddenField := "********"

	for rows.Next() {
		var s tc.ServerNullable
		if err = rows.StructScan(&s); err != nil {
			return nil, []error{fmt.Errorf("getting servers: %v", err)}, tc.SystemError
		}

		if !user.HasCapability(tc.ServerSecureCapability) {
			s.ILOPassword = &HiddenField
			s.XMPPPasswd = &HiddenField
		}
		servers = append(servers, s)
	}

	// if ds requested uses mid-tier caches, add those to the list as well
	if usesMids {
		mids, errs, errType := getMidServers(servers, tx)

		log.Debugf("getting mids: %v, %v\n", errs, errType)
		if len(errs) > 0 {
			for _, err := range errs {
				if err.Error() == `id cannot parse to integer` {
					return nil, []error{errors.New("Resource not found.")}, tc.DataMissingError //matches perl response
				}
			}
			return nil, errs, errType
		}
		for _, server := range mids {
			servers = append(servers, server)
		}
	}

	return servers, nil, tc.NoError
}

// getMidServers gets mids used by the servers in this ds
// Original comment from the Perl code:
// if the delivery service employs mids, we're gonna pull mid servers too by pulling the cachegroups of the edges and finding those cachegroups parent cachegroup...
// then we see which servers have cachegroup in parent cachegroup list...that's how we find mids for the ds :)
func getMidServers(servers []tc.ServerNullable, tx *sqlx.Tx) ([]tc.ServerNullable, []error, tc.ApiErrorType) {
	if len(servers) == 0 {
		return nil, nil, tc.NoError
	}
	var ids []string
	for _, s := range servers {
		ids = append(ids, strconv.Itoa(*s.ID))
	}

	edgeIDs := strings.Join(ids, ",")
	// TODO: include secondary parent?
	q := selectQuery() + `
WHERE s.id IN (
	SELECT mid.id FROM server mid
	JOIN cachegroup cg ON cg.id IN (
		SELECT cg.parent_cachegroup_id
		FROM server s
		JOIN cachegroup cg ON cg.id = s.cachegroup
		WHERE s.id IN (` + edgeIDs + `))
	JOIN type t ON mid.type = (SELECT id FROM type WHERE name = 'MID'))
`
	rows, err := tx.Queryx(q)
	if err != nil {
		return nil, []error{err}, tc.DataConflictError
	}
	defer rows.Close()

	var mids []tc.ServerNullable
	for rows.Next() {
		var s tc.ServerNullable
		if err := rows.StructScan(&s); err != nil {
			log.Error.Printf("could not scan mid servers: %s\n", err)
			return nil, []error{err}, tc.SystemError
		}
		mids = append(mids, s)
	}
	return mids, nil, tc.NoError
}

func (sv *TOServer) Update() (error, error, int) {
	if sv.IP6Address != nil && len(strings.TrimSpace(*sv.IP6Address)) == 0 {
		sv.IP6Address = nil
	}
	return api.GenericUpdate(sv)
}

func (sv *TOServer) Create() (error, error, int) {
	// TODO put in Validate()
	if sv.IP6Address != nil && len(strings.TrimSpace(*sv.IP6Address)) == 0 {
		sv.IP6Address = nil
	}
	if sv.XMPPID == nil || *sv.XMPPID == "" {
		hostName := *sv.HostName
		sv.XMPPID = &hostName
	}
	return api.GenericCreate(sv)
}

func (sv *TOServer) Delete() (error, error, int) { return api.GenericDelete(sv) }

func selectQuery() string {
	const JumboFrameBPS = 9000
	return `SELECT
cg.name as cachegroup,
s.cachegroup as cachegroup_id,
s.cdn_id,
cdn.name as cdn_name,
s.domain_name,
s.guid,
s.host_name,
s.https_port,
s.id,
s.ilo_ip_address,
s.ilo_ip_gateway,
s.ilo_ip_netmask,
s.ilo_password,
s.ilo_username,
COALESCE(s.interface_mtu, ` + strconv.Itoa(JumboFrameBPS) + `) as interface_mtu,
s.interface_name,
s.ip6_address,
s.ip6_gateway,
s.ip_address,
s.ip_gateway,
s.ip_netmask,
s.last_updated,
s.mgmt_ip_address,
s.mgmt_ip_gateway,
s.mgmt_ip_netmask,
s.offline_reason,
pl.name as phys_location,
s.phys_location as phys_location_id,
p.name as profile,
p.description as profile_desc,
s.profile as profile_id,
s.rack,
s.reval_pending,
s.router_host_name,
s.router_port_name,
st.name as status,
s.status as status_id,
s.tcp_port,
t.name as server_type,
s.type as server_type_id,
s.upd_pending as upd_pending,
s.xmpp_id,
s.xmpp_passwd
FROM
  server s
JOIN cachegroup cg ON s.cachegroup = cg.id
JOIN cdn cdn ON s.cdn_id = cdn.id
JOIN phys_location pl ON s.phys_location = pl.id
JOIN profile p ON s.profile = p.id
JOIN status st ON s.status = st.id
JOIN type t ON s.type = t.id`
}

func insertQuery() string {
	query := `INSERT INTO server (
cachegroup,
cdn_id,
domain_name,
host_name,
https_port,
ilo_ip_address,
ilo_ip_netmask,
ilo_ip_gateway,
ilo_username,
ilo_password,
interface_mtu,
interface_name,
ip6_address,
ip6_gateway,
ip_address,
ip_netmask,
ip_gateway,
mgmt_ip_address,
mgmt_ip_netmask,
mgmt_ip_gateway,
offline_reason,
phys_location,
profile,
rack,
router_host_name,
router_port_name,
status,
tcp_port,
type,
upd_pending,
xmpp_id,
xmpp_passwd
) VALUES (
:cachegroup_id,
:cdn_id,
:domain_name,
:host_name,
:https_port,
:ilo_ip_address,
:ilo_ip_netmask,
:ilo_ip_gateway,
:ilo_username,
:ilo_password,
:interface_mtu,
:interface_name,
:ip6_address,
:ip6_gateway,
:ip_address,
:ip_netmask,
:ip_gateway,
:mgmt_ip_address,
:mgmt_ip_netmask,
:mgmt_ip_gateway,
:offline_reason,
:phys_location_id,
:profile_id,
:rack,
:router_host_name,
:router_port_name,
:status_id,
:tcp_port,
:server_type_id,
:upd_pending,
:xmpp_id,
:xmpp_passwd
) RETURNING id,last_updated`
	return query
}

func updateQuery() string {
	query := `UPDATE
server SET
cachegroup=:cachegroup_id,
cdn_id=:cdn_id,
domain_name=:domain_name,
host_name=:host_name,
https_port=:https_port,
ilo_ip_address=:ilo_ip_address,
ilo_ip_netmask=:ilo_ip_netmask,
ilo_ip_gateway=:ilo_ip_gateway,
ilo_username=:ilo_username,
ilo_password=:ilo_password,
interface_mtu=:interface_mtu,
interface_name=:interface_name,
ip6_address=:ip6_address,
ip6_gateway=:ip6_gateway,
ip_address=:ip_address,
ip_netmask=:ip_netmask,
ip_gateway=:ip_gateway,
mgmt_ip_address=:mgmt_ip_address,
mgmt_ip_netmask=:mgmt_ip_netmask,
mgmt_ip_gateway=:mgmt_ip_gateway,
offline_reason=:offline_reason,
phys_location=:phys_location_id,
profile=:profile_id,
rack=:rack,
router_host_name=:router_host_name,
router_port_name=:router_port_name,
status=:status_id,
tcp_port=:tcp_port,
type=:server_type_id,
upd_pending=:upd_pending,
xmpp_id=:xmpp_id,
xmpp_passwd=:xmpp_passwd
WHERE id=:id RETURNING last_updated`
	return query
}

func deleteQuery() string {
	return `DELETE FROM server WHERE id = :id`
}
