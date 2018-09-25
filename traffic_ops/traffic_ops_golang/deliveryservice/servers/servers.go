package servers

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
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// TODeliveryServiceRequest provides a type alias to define functions on
type TODeliveryServiceServer struct {
	ReqInfo *api.APIInfo `json:"-"`
	tc.DeliveryServiceServer
}

func GetRefType(inf *api.APIInfo) *TODeliveryServiceServer {
	s := TODeliveryServiceServer{ReqInfo: inf}
	return &s
}

func (dss TODeliveryServiceServer) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"deliveryservice", api.GetIntKey}, {"server", api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (dss TODeliveryServiceServer) GetKeys() (map[string]interface{}, bool) {
	if dss.DeliveryService == nil {
		return map[string]interface{}{"deliveryservice": 0}, false
	}
	if dss.Server == nil {
		return map[string]interface{}{"server": 0}, false
	}
	keys := make(map[string]interface{})
	ds_id := *dss.DeliveryService
	server_id := *dss.Server

	keys["deliveryservice"] = ds_id
	keys["server"] = server_id
	return keys, true
}

func (dss *TODeliveryServiceServer) GetAuditName() string {
	if dss.DeliveryService != nil {
		return strconv.Itoa(*dss.DeliveryService) + "-" + strconv.Itoa(*dss.Server)
	}
	return "unknown"
}

func (dss *TODeliveryServiceServer) GetType() string {
	return "deliveryserviceServers"
}

func (dss *TODeliveryServiceServer) SetKeys(keys map[string]interface{}) {
	ds_id, _ := keys["deliveryservice"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	dss.DeliveryService = &ds_id

	server_id, _ := keys["server"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	dss.Server = &server_id
}

// Validate fulfills the api.Validator interface
func (dss *TODeliveryServiceServer) Validate(tx *sql.Tx) error {

	errs := validation.Errors{
		"deliveryservice": validation.Validate(dss.DeliveryService, validation.Required),
		"server":          validation.Validate(dss.Server, validation.Required),
	}

	return util.JoinErrs(tovalidate.ToErrors(errs))
}

// ReadDSSHandler list all of the Deliveryservice Servers in response to requests to api/1.1/deliveryserviceserver$
func ReadDSSHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"limit", "page"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	results, err := GetRefType(inf).readDSS(inf.Tx, inf.User, inf.Params, inf.IntParams)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	api.WriteRespRaw(w, r, results)
}

func (dss *TODeliveryServiceServer) readDSS(tx *sqlx.Tx, user *auth.CurrentUser, params map[string]string, intParams map[string]int) (*tc.DeliveryServiceServerResponse, error) {
	orderby := params["orderby"]
	limit := 20
	offset := 0
	page := 0
	err := error(nil)
	if plimit, ok := intParams["limit"]; ok {
		limit = plimit
	}
	if ppage, ok := intParams["page"]; ok {
		page = ppage
		offset = page
		if offset > 0 {
			offset -= 1
		}
		offset *= limit
	}
	if orderby == "" {
		orderby = "deliveryService"
	}

	query, err := selectQuery(orderby, strconv.Itoa(limit), strconv.Itoa(offset))
	if err != nil {
		return nil, errors.New("creating query for DeliveryserviceServers: " + err.Error())
	}
	log.Debugln("Query is ", query)

	rows, err := tx.NamedQuery(query, dss)
	if err != nil {
		return nil, errors.New("Error querying DeliveryserviceServers: " + err.Error())
	}
	defer rows.Close()
	servers := []tc.DeliveryServiceServer{}
	for rows.Next() {
		s := tc.DeliveryServiceServer{}
		if err = rows.StructScan(&s); err != nil {
			return nil, errors.New("error parsing dss rows: " + err.Error())
		}
		servers = append(servers, s)
	}
	return &tc.DeliveryServiceServerResponse{orderby, servers, page, limit}, nil
}

func selectQuery(orderBy string, limit string, offset string) (string, error) {
	selectStmt := `SELECT
	s.deliveryService,
	s.server,
	s.last_updated
	FROM deliveryservice_server s`

	allowedOrderByCols := map[string]string{
		"":                "",
		"deliveryservice": "s.deliveryService",
		"server":          "s.server",
		"lastupdated":     "s.last_updated",
		"deliveryService": "s.deliveryService",
		"lastUpdated":     "s.last_updated",
		"last_updated":    "s.last_updated",
	}
	orderBy, ok := allowedOrderByCols[orderBy]
	if !ok {
		return "", errors.New("orderBy '" + orderBy + "' not permitted")
	}

	if orderBy != "" {
		selectStmt += ` ORDER BY ` + orderBy
	}

	selectStmt += ` LIMIT ` + limit + ` OFFSET ` + offset + ` ROWS`
	return selectStmt, nil
}

func deleteQuery() string {
	query := `DELETE FROM deliveryservice_server
	WHERE deliveryservice=:deliveryservice and server=:server`
	return query
}

type DSServerIds struct {
	DsId    *int  `json:"dsId" db:"deliveryservice"`
	Servers []int `json:"servers"`
	Replace *bool `json:"replace"`
}

type TODSServerIds DSServerIds

func createServersForDsIdRef() *TODSServerIds {
	var dsserversRef = TODSServerIds(DSServerIds{})
	return &dsserversRef
}

func GetReplaceHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"limit", "page"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	payload := DSServerIds{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON"), nil)
		return
	}

	servers := payload.Servers
	dsId := payload.DsId
	if servers == nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("servers must exist in post"), nil)
		return
	}
	if dsId == nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("dsid must exist in post"), nil)
		return
	}
	if payload.Replace == nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("replace must exist in post"), nil)
		return
	}

	xmlID, ok, err := deliveryservice.GetXMLID(inf.Tx.Tx, *dsId)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryserviceserver getting XMLID: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("no delivery service with that ID exists"), nil)
		return
	}
	if userErr, sysErr, errCode := tenant.Check(inf.User, xmlID, inf.Tx.Tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	if *payload.Replace {
		// delete existing
		_, err := inf.Tx.Tx.Exec("DELETE FROM deliveryservice_server WHERE deliveryservice = $1", *dsId)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("unable to remove the existing servers assigned to the delivery service: "+err.Error()))
			return
		}
	}

	respServers := []int{}
	for _, server := range servers {
		dtos := map[string]interface{}{"id": dsId, "server": server}
		if _, err := inf.Tx.NamedExec(insertIdsQuery(), dtos); err != nil {
			usrErr, sysErr, code := api.ParseDBError(err)
			api.HandleErr(w, r, inf.Tx.Tx, code, usrErr, sysErr)
			return
		}
		respServers = append(respServers, server)
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "server assignements complete", tc.DSSMapResponse{*dsId, *payload.Replace, respServers})
}

type TODeliveryServiceServers tc.DeliveryServiceServers

func createServersRef() *TODeliveryServiceServers {
	serversRef := TODeliveryServiceServers(tc.DeliveryServiceServers{})
	return &serversRef
}

// GetCreateHandler assigns an existing Server to and existing Deliveryservice in response to api/1.1/deliveryservices/{xml_id}/servers
func GetCreateHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"xml_id"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if userErr, sysErr, errCode := tenant.Check(inf.User, inf.Params["xml_id"], inf.Tx.Tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	dsID := 0
	if err := inf.Tx.Tx.QueryRow(selectDeliveryService(), inf.Params["xml_id"]).Scan(&dsID); err != nil {
		if err == sql.ErrNoRows {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, errors.New("delivery service not found"))
			return
		}
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("ds servers create scanning: "+err.Error()))
		return
	}

	// get list of server Ids to insert
	payload := tc.DeliveryServiceServers{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON"), nil)
		return
	}
	payload.XmlId = inf.Params["xml_id"]
	serverNames := payload.ServerNames

	res, err := inf.Tx.Tx.Exec(`INSERT INTO deliveryservice_server (deliveryservice, server) SELECT $1, id FROM server WHERE host_name = ANY($2::text[])`, dsID, pq.Array(serverNames))
	if err != nil {

		usrErr, sysErr, code := api.ParseDBError(err)
		api.HandleErr(w, r, inf.Tx.Tx, code, usrErr, sysErr)
		return
	}

	if rowsAffected, err := res.RowsAffected(); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("ds servers inserting for create delivery service servers: getting rows affected: "+err.Error()))
		return
	} else if int(rowsAffected) != len(serverNames) {
		// this happens when the names they gave don't exist
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("servers not found"), nil)
		return
	}
	api.WriteResp(w, r, tc.DeliveryServiceServers{payload.ServerNames, payload.XmlId})
}

func selectDeliveryService() string {
	query := `SELECT id FROM deliveryservice WHERE xml_id = $1`
	return query
}

func insertIdsQuery() string {
	query := `INSERT INTO deliveryservice_server (deliveryservice, server)
VALUES (:id, :server )`
	return query
}

func selectServerIds() string {
	query := `SELECT id FROM server WHERE host_name in (?)`
	return query
}

// GetReadAssigned retrieves lists of servers  based in the filter identified in the request: api/1.1/deliveryservices/{id}/servers|unassigned_servers|eligible
func GetReadAssigned(w http.ResponseWriter, r *http.Request) {
	getRead(w, r, false)
}

// GetReadUnassigned retrieves lists of servers  based in the filter identified in the request: api/1.1/deliveryservices/{id}/servers|unassigned_servers|eligible
func GetReadUnassigned(w http.ResponseWriter, r *http.Request) {
	getRead(w, r, true)
}

func getRead(w http.ResponseWriter, r *http.Request, unassigned bool) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	servers, err := read(inf.Tx, inf.IntParams["id"], inf.User, unassigned)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	api.WriteResp(w, r, servers)
}

func read(tx *sqlx.Tx, dsID int, user *auth.CurrentUser, unassigned bool) ([]tc.DSServer, error) {
	where := `WHERE s.id in (select server from deliveryservice_server where deliveryservice = $1)`
	if unassigned {
		where = `WHERE s.id not in (select server from deliveryservice_server where deliveryservice = $1)`
	}
	query := dssSelectQuery() + where
	log.Debugln("Query is ", query)
	rows, err := tx.Queryx(query, dsID)
	if err != nil {
		return nil, errors.New("error querying dss rows: " + err.Error())
	}
	defer rows.Close()

	servers := []tc.DSServer{}
	for rows.Next() {
		s := tc.DSServer{}
		if err = rows.StructScan(&s); err != nil {
			return nil, errors.New("error scanning dss rows: " + err.Error())
		}
		if !user.HasCapability(tc.ServerSecureCapability) {
			s.ILOPassword = util.StrPtr("")
		}
		servers = append(servers, s)
	}
	return servers, nil
}

func dssSelectQuery() string {

	const JumboFrameBPS = 9000

	// COALESCE is needed to default values that are nil in the database
	// because Go does not allow that to marshal into the struct
	selectStmt := `SELECT
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
	s.router_host_name,
	s.router_port_name,
	st.name as status,
	s.status as status_id,
	s.tcp_port,
	t.name as server_type,
	s.type as server_type_id,
	s.upd_pending as upd_pending
	FROM server s
	JOIN cachegroup cg ON s.cachegroup = cg.id
	JOIN cdn cdn ON s.cdn_id = cdn.id
	JOIN phys_location pl ON s.phys_location = pl.id
	JOIN profile p ON s.profile = p.id
	JOIN status st ON s.status = st.id
	JOIN type t ON s.type = t.id `

	return selectStmt
}

type TODSSDeliveryService struct {
	ReqInfo *api.APIInfo `json:"-"`
	tc.DSSDeliveryService
}

func (dss *TODSSDeliveryService) APIInfo() *api.APIInfo {
	return dss.ReqInfo
}

func TypeSingleton(reqInfo *api.APIInfo) api.Reader {
	return &TODSSDeliveryService{reqInfo, tc.DSSDeliveryService{}}
}

// Read shows all of the delivery services associated with the specified server.
func (dss *TODSSDeliveryService) Read() ([]interface{}, error, error, int) {
	orderby := dss.APIInfo().Params["orderby"]
	if orderby == "" {
		orderby = "deliveryService"
	}

	query := SDSSelectQuery()
	log.Debugln("Query is ", query)

	rows, err := dss.APIInfo().Tx.Queryx(query, dss.APIInfo().Params["id"])
	if err != nil {
		log.Errorf("Error querying DeliveryserviceServers: %v", err)
		return nil, nil, errors.New("dss querying: " + err.Error()), http.StatusInternalServerError
	}
	defer rows.Close()

	services := []interface{}{}
	for rows.Next() {
		var s tc.DSSDeliveryService
		if err = rows.StructScan(&s); err != nil {
			return nil, nil, errors.New("dss scanning: " + err.Error()), http.StatusInternalServerError
		}
		services = append(services, s)
	}

	return services, nil, nil, http.StatusOK
}

func SDSSelectQuery() string {

	selectStmt := `SELECT
 		active,
		ccr_dns_ttl,
		cdn_id,
		cacheurl,
		check_path,
		dns_bypass_cname,
		dns_bypass_ip,
		dns_bypass_ip6,
		dns_bypass_ttl,
		dscp,
		display_name,
		edge_header_rewrite,
		geo_limit,
		geo_limit_countries,
		geolimit_redirect_url,
		geo_provider,
		global_max_mbps,
		global_max_tps,
		http_bypass_fqdn,
		id,
		ipv6_routing_enabled,
		info_url,
		initial_dispersion,
		last_updated,
		logs_enabled,
		long_desc,
		long_desc_1,
		long_desc_2,
		max_dns_answers,
		mid_header_rewrite,
		miss_lat,
		miss_long,
		multi_site_origin,
		multi_site_origin_algorithm,
		(SELECT o.protocol::text || '://' || o.fqdn || rtrim(concat(':', o.port::text), ':')
		FROM origin o
		WHERE o.deliveryservice = d.id
		AND o.is_primary) as org_server_fqdn,
		origin_shield,
		profile,
		protocol,
		qstring_ignore,
		range_request_handling,
		regex_remap,
		regional_geo_blocking,
		remap_text,
		routing_name,
		ssl_key_version,
		signing_algorithm,
		tr_request_headers,
		tr_response_headers,
		tenant_id,
		type,
		xml_id
	FROM deliveryservice d
		WHERE id in (SELECT deliveryService FROM deliveryservice_server where server = $1)`
	return selectStmt
}

func updateQuery() string {
	query := `UPDATE
	profile_parameter SET
	profile=:profile_id,
	parameter=:parameter_id
	WHERE profile=:profile_id AND 
      parameter = :parameter_id 
      RETURNING last_updated`
	return query
}
