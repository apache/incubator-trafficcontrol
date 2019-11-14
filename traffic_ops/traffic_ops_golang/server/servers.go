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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/jmoiron/sqlx"
)

//we need a type alias to define functions on
type TOServer struct {
	api.APIInfoImpl `json:"-"`
	LastUpdatedAny  time.Time `json:"-" db:"last_updated_any"`
	tc.ServerNullable
}

// LastModified implements api.Modifieder.
// Note this does NOT return tc.ServerNullable.LastModified, which is the last_updated of the server table only. Rather, this returns a time which is the last modified of any queried table.
func (v TOServer) LastModified() time.Time { return v.LastUpdatedAny }

func (v *TOServer) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOServer) InsertQuery() string           { return insertQuery() }
func (v *TOServer) UpdateQuery() string           { return updateQuery() }
func (v *TOServer) DeleteQuery() string           { return deleteQuery() }

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

type DBServerNullable struct {
	tc.ServerNullable
	LastUpdatedAny *time.Time `json:"-" db:"last_updated_any"`
}

func (server *TOServer) Read() ([]interface{}, error, error, int) {
	returnable := []interface{}{}

	servers, lastUpdatedAny, userErr, sysErr, errCode := getServers(server.ReqInfo.Params, server.ReqInfo.Tx, server.ReqInfo.LastModified, server.ReqInfo.User)
	if userErr != nil || sysErr != nil {
		return nil, userErr, sysErr, errCode
	}
	if errCode == http.StatusNotModified {
		return nil, nil, nil, http.StatusNotModified
	}

	server.LastUpdatedAny = lastUpdatedAny

	for _, server := range servers {
		returnable = append(returnable, server)
	}

	return returnable, nil, nil, http.StatusOK
}

// getServers returns the list of servers, the last updated time of any table on any server, any user error, any system error, and the error code to use in the event of an error.
// If both errors are nil, the caller is expected to ignore the error code and return whatever code it finds appropriate (usually http.StatusOK). If both errors are nil, the code is not guaranteed to be an appropriate code to return to clients.
// Note as a special case, the servers and both errors may be nil, with the code http.StatusNotModified. In which case, it is expected to return a 304 Not Modified with no body to the client.
func getServers(params map[string]string, tx *sqlx.Tx, lastModified *time.Time, user *auth.CurrentUser) ([]tc.ServerNullable, time.Time, error, error, int) {
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
			return nil, time.Time{}, errors.New("dsId must be an integer"), nil, http.StatusBadRequest
		}
		userErr, sysErr, _ := tenant.CheckID(tx.Tx, user, dsID)
		if userErr != nil || sysErr != nil {
			return nil, time.Time{}, errors.New("Forbidden"), nil, http.StatusForbidden
		}
		// only if dsId is part of params: add join on deliveryservice_server table
		queryAddition = `
LEFT JOIN deliveryservice_server dss ON dss.server = s.id
`
		// depending on ds type, also need to add mids
		dsType, err := deliveryservice.GetDeliveryServiceType(dsID, tx.Tx)
		if err != nil {
			return nil, time.Time{}, err, nil, http.StatusBadRequest
		}
		usesMids = dsType.UsesMidCache()
		log.Debugf("Servers for ds %d; uses mids? %v\n", dsID, usesMids)
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(params, queryParamsToSQLCols)
	if len(errs) > 0 {
		return nil, time.Time{}, util.JoinErrs(errs), nil, http.StatusBadRequest
	}

	if lastModified != nil {
		serversModified := serverModified(tx, *lastModified, queryAddition+where+pagination, queryValues)
		if usesMids {
			// have to do a second modified check, because this endpoint does 2 queries if dsId exists and usesMids.
			edgeIDQuery := `select id from server ` + queryAddition + where + pagination
			midWhere := getMidServersWhereServersQuery(edgeIDQuery)
			// If the edgeIDQuery contained a join of deliveryservice_server, it got added to the where,
			// but there's no top-level JOIN, and serverModified will think there is when it sees the table name.
			// Therefore, we have to add it, so the table exists when serverModified adds 'dss.last_modified'.
			if strings.Contains(midWhere, "deliveryservice_server") {
				midWhere = ` LEFT JOIN deliveryservice_server dss ON dss.server = s.id ` + midWhere
			}
			midsModified := serverModified(tx, *lastModified, midWhere, queryValues)
			if midsModified {
				serversModified = true
			}
		}
		if !serversModified {
			return nil, time.Time{}, nil, nil, http.StatusNotModified
		}
	}

	query := selectQuery(queryAddition + where + orderBy + pagination)

	log.Debugln("Query is ", query)

	rows, err := tx.NamedQuery(query, queryValues)
	if err != nil {
		userErr, sysErr, errCode := api.ParseDBError(err)
		return nil, time.Time{}, userErr, sysErr, errCode
	}
	defer rows.Close()

	servers := []tc.ServerNullable{}

	HiddenField := "********"

	lastUpdatedAny := time.Time{}

	for rows.Next() {
		s := DBServerNullable{}
		if err = rows.StructScan(&s); err != nil {
			userErr, sysErr, errCode := api.ParseDBError(err)
			return nil, time.Time{}, userErr, sysErr, errCode
		}
		if user.PrivLevel < auth.PrivLevelOperations {
			s.ILOPassword = &HiddenField
			s.XMPPPasswd = &HiddenField
		}
		if s.LastUpdatedAny.After(lastUpdatedAny) {
			lastUpdatedAny = *s.LastUpdatedAny
		}
		servers = append(servers, s.ServerNullable)
	}

	// if ds requested uses mid-tier caches, add those to the list as well
	if usesMids {
		mids, midLastUpdatedAny, userErr, sysErr, errCode := getMidServers(servers, tx)
		if userErr != nil || sysErr != nil {
			return nil, time.Time{}, userErr, sysErr, errCode
		}

		if midLastUpdatedAny.After(lastUpdatedAny) {
			lastUpdatedAny = midLastUpdatedAny
		}

		for _, server := range mids {
			servers = append(servers, server)
		}
	}

	return servers, lastUpdatedAny, nil, nil, http.StatusOK
}

// getMidServers gets mids used by the servers in this ds
// Original comment from the Perl code:
// if the delivery service employs mids, we're gonna pull mid servers too by pulling the cachegroups of the edges and finding those cachegroups parent cachegroup...
// then we see which servers have cachegroup in parent cachegroup list...that's how we find mids for the ds :)
func getMidServers(servers []tc.ServerNullable, tx *sqlx.Tx) ([]tc.ServerNullable, time.Time, error, error, int) {
	if len(servers) == 0 {
		return nil, time.Time{}, nil, nil, http.StatusOK
	}
	where := getMidServersWhere(servers)
	qry := selectQuery(where)
	rows, err := tx.Queryx(qry)
	if err != nil {
		userErr, sysErr, errCode := api.ParseDBError(err)
		return nil, time.Time{}, userErr, sysErr, errCode
	}

	lastUpdatedAny := time.Time{}
	var mids []tc.ServerNullable
	for rows.Next() {
		s := DBServerNullable{}
		if err := rows.StructScan(&s); err != nil {
			userErr, sysErr, errCode := api.ParseDBError(err)
			return nil, time.Time{}, userErr, sysErr, errCode
		}
		if s.LastUpdatedAny.After(lastUpdatedAny) {
			lastUpdatedAny = *s.LastUpdatedAny
		}
		mids = append(mids, s.ServerNullable)
	}
	return mids, lastUpdatedAny, nil, nil, http.StatusOK
}

func getMidServersWhere(edgeServers []tc.ServerNullable) string {
	ids := []string{}
	for _, s := range edgeServers {
		ids = append(ids, strconv.Itoa(*s.ID))
	}
	edgeIDs := strings.Join(ids, ",")
	// TODO: include secondary parent?

	return getMidServersWhereServersQuery(edgeIDs)
}

func getMidServersWhereServersQuery(serverIDQuery string) string {
	return `
WHERE s.id IN (
	SELECT mid.id FROM server mid
	JOIN cachegroup cg ON cg.id IN (
		SELECT cg.parent_cachegroup_id
		FROM server s
		JOIN cachegroup cg ON cg.id = s.cachegroup
		WHERE s.id IN (` + serverIDQuery + `))
	JOIN type t ON mid.type = (SELECT id FROM type WHERE name = 'MID'))
`
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

func selectQuery(where string) string {
	//
	// WARNING: if you add a table to this query, you MUST make sure makeServerModifiedQry is correct, as well as last_updated_any!
	//
	// For tables with IDs directly in the server table, make sure the table is added to makeServerModifiedQry.
	//
	// If the ID is _not_ in server (for example, a one-to-many relationship like deliveryservice_server),
	// you must instead create triggers on update, insert and delete to update the last_updated of the associated server.
	// See traffic_ops/app/db/migrations/20190810000000_add-ds-update-last-modified-triggers.sql
	//
	// Also, be sure to add your table to the last_updated_any subquery.
	//
	maybeDSSLastUpdated := ``
	if strings.Contains(where, "deliveryservice_server") {
		maybeDSSLastUpdated += `dss.last_updated,`
	}
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
GREATEST(
  s.last_updated,
  cg.last_updated,
  cdn.last_updated,
  pl.last_updated,
  p.last_updated,
  st.last_updated,
  ` + maybeDSSLastUpdated + `
  t.last_updated
) AS last_updated_any,
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
JOIN type t ON s.type = t.id` + where
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
