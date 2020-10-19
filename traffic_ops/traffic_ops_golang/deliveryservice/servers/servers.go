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
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// TODeliveryServiceRequest provides a type alias to define functions on
type TODeliveryServiceServer struct {
	api.APIInfoImpl `json:"-"`
	tc.DeliveryServiceServer
	TenantIDs          pq.Int64Array `json:"-" db:"accessibleTenants"`
	DeliveryServiceIDs pq.Int64Array `json:"-" db:"dsids"`
	ServerIDs          pq.Int64Array `json:"-" db:"serverids"`
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
	useIMS := false
	cfg, err := api.GetConfig(r.Context())
	if err != nil {
		log.Warnf("Couldnt get the config %v", err)
	}
	if cfg != nil {
		useIMS = cfg.UseIMS
	}
	inf, errs := api.NewInfo(r, nil, []string{"limit", "page"})
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	dss := TODeliveryServiceServer{}
	dss.SetInfo(inf)
	results, err, maxTime := dss.readDSS(nil, inf.Tx, inf.User, inf.Params, inf.IntParams, nil, nil, useIMS)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	if maxTime != nil && api.SetLastModifiedHeader(r, useIMS) {
		// RFC1123
		date := maxTime.Format("Mon, 02 Jan 2006 15:04:05 MST")
		w.Header().Add(rfc.LastModified, date)
	}
	// statusnotmodified
	if err == nil && results == nil {
		w.WriteHeader(http.StatusNotModified)
	}
	api.WriteRespRaw(w, r, results)
}

// ReadDSSHandlerV14 list all of the Deliveryservice Servers in response to requests to api/1.4/deliveryserviceserver.
func ReadDSSHandlerV14(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, nil, []string{"limit", "page"})
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	dsIDs := []int64{}
	dsIDStrs := strings.Split(inf.Params["deliveryserviceids"], ",")
	for _, dsIDStr := range dsIDStrs {
		dsIDStr = strings.TrimSpace(dsIDStr)
		if dsIDStr == "" {
			continue
		}
		dsID, err := strconv.Atoi(dsIDStr)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, 400, errors.New("deliveryserviceids query parameter must be a comma-delimited list of integers, got '"+inf.Params["deliveryserviceids"]+"'"), nil)
			return
		}
		dsIDs = append(dsIDs, int64(dsID))
	}

	serverIDs := []int64{}
	serverIDStrs := strings.Split(inf.Params["serverids"], ",")
	for _, serverIDStr := range serverIDStrs {
		serverIDStr = strings.TrimSpace(serverIDStr)
		if serverIDStr == "" {
			continue
		}
		serverID, err := strconv.Atoi(serverIDStr)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, 400, errors.New("serverids query parameter must be a comma-delimited list of integers, got '"+inf.Params["serverids"]+"'"), nil)
			return
		}
		serverIDs = append(serverIDs, int64(serverID))
	}

	dss := TODeliveryServiceServer{}
	dss.SetInfo(inf)
	cfg, e := api.GetConfig(r.Context())
	useIMS := false
	if e == nil && cfg != nil {
		useIMS = cfg.UseIMS
	} else {
		log.Warnf("Couldn't get config %v", e)
	}

	results, err, maxTime := dss.readDSS(r.Header, inf.Tx, inf.User, inf.Params, inf.IntParams, dsIDs, serverIDs, useIMS)
	if maxTime != nil && api.SetLastModifiedHeader(r, useIMS) {
		// RFC1123
		date := maxTime.Format("Mon, 02 Jan 2006 15:04:05 MST")
		w.Header().Add(rfc.LastModified, date)
	}
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	// statusnotmodified
	if err == nil && results == nil {
		w.WriteHeader(http.StatusNotModified)
	}
	api.WriteRespRaw(w, r, results)
}

func (dss *TODeliveryServiceServer) readDSS(h http.Header, tx *sqlx.Tx, user *auth.CurrentUser, params map[string]string, intParams map[string]int, dsIDs []int64, serverIDs []int64, useIMS bool) (*tc.DeliveryServiceServerResponse, error, *time.Time) {
	var maxTime time.Time
	var runSecond bool
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

	tenantIDs, err := tenant.GetUserTenantIDListTx(tx.Tx, user.TenantID)
	if err != nil {
		return nil, errors.New("getting user tenant ID list: " + err.Error()), nil
	}
	for _, id := range tenantIDs {
		dss.TenantIDs = append(dss.TenantIDs, int64(id))
	}
	dss.ServerIDs = serverIDs
	dss.DeliveryServiceIDs = dsIDs
	query1, err := selectQuery(orderby, strconv.Itoa(limit), strconv.Itoa(offset), dsIDs, serverIDs, true)
	if err != nil {
		log.Warnf("Error getting the max last updated query %v", err)
	}
	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(tx, h, map[string]interface{}{}, query1)
		if !runSecond {
			log.Debugln("IMS HIT")
			return nil, nil, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}
	query, err := selectQuery(orderby, strconv.Itoa(limit), strconv.Itoa(offset), dsIDs, serverIDs, false)
	if err != nil {
		return nil, errors.New("creating query for DeliveryserviceServers: " + err.Error()), nil
	}
	log.Debugln("Query is ", query)

	rows, err := tx.NamedQuery(query, dss)
	if err != nil {
		return nil, errors.New("Error querying DeliveryserviceServers: " + err.Error()), nil
	}
	defer rows.Close()
	servers := []tc.DeliveryServiceServer{}
	for rows.Next() {
		s := tc.DeliveryServiceServer{}
		if err = rows.StructScan(&s); err != nil {
			return nil, errors.New("error parsing dss rows: " + err.Error()), nil
		}
		servers = append(servers, s)
	}
	return &tc.DeliveryServiceServerResponse{orderby, servers, page, limit}, nil, &maxTime
}

func selectQuery(orderBy string, limit string, offset string, dsIDs []int64, serverIDs []int64, getMaxQuery bool) (string, error) {
	selectStmt := `SELECT
	s.deliveryService,
	s.server,
	s.last_updated
	FROM deliveryservice_server s`

	if getMaxQuery {
		selectStmt = `SELECT max(t) from (
		SELECT max(s.last_updated) as t FROM deliveryservice_server s`
	}
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

	// TODO refactor to use dbhelpers.AddTenancyCheck
	selectStmt += `
JOIN deliveryservice d on s.deliveryservice = d.id
WHERE d.tenant_id = ANY(CAST(:accessibleTenants AS bigint[]))
`
	if len(dsIDs) > 0 {
		selectStmt += `
AND s.deliveryservice = ANY(:dsids)
`
	}
	if len(serverIDs) > 0 {
		selectStmt += `
AND s.server = ANY(:serverids)
`
	}

	if orderBy != "" {
		selectStmt += ` ORDER BY ` + orderBy
	}

	selectStmt += ` LIMIT ` + limit + ` OFFSET ` + offset + ` ROWS`
	if getMaxQuery {
		return selectStmt + `UNION ALL
		select max(last_updated) as t from last_deleted l where l.table_name='deliveryservice_server') as res`, nil
	}
	return selectStmt, nil
}

type DSServerIds struct {
	DsId    *int  `json:"dsId" db:"deliveryservice"`
	Servers []int `json:"servers"`
	Replace *bool `json:"replace"`
}

type TODSServerIds DSServerIds

func GetReplaceHandler(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, nil, []string{"limit", "page"})
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
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

	ds, ok, err := GetDSInfo(inf.Tx.Tx, *dsId)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryserviceserver getting XMLID: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("no delivery service with that ID exists"), nil)
		return
	}
	if errs = tenant.Check(inf.User, ds.Name, inf.Tx.Tx); errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	serverNamesCdnIdAndTypes, err := dbhelpers.GetServerHostNamesAndTypesFromIDs(inf.Tx.Tx, servers)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, err, nil)
		return
	}
	userErr := ValidateDSSAssignments(ds, serverNamesCdnIdAndTypes)
	if userErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, userErr, nil)
		return
	}

	errs = ValidateServerCapabilities(ds.ID, serverNamesCdnIdAndTypes, inf.Tx.Tx)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
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
			inf.HandleErrs(w, r, api.ParseDBError(err))
			return
		}
		respServers = append(respServers, server)
	}

	if err := deliveryservice.EnsureParams(inf.Tx.Tx, *dsId, ds.Name, ds.EdgeHeaderRewrite, ds.MidHeaderRewrite, ds.RegexRemap, ds.CacheURL, ds.SigningAlgorithm, ds.Type, ds.MaxOriginConnections); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice_server replace ensuring ds parameters: "+err.Error()))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+ds.Name+", ID: "+strconv.Itoa(*dsId)+", ACTION: Replace existing servers assigned to delivery service", inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "server assignements complete", tc.DSSMapResponse{*dsId, *payload.Replace, respServers})
}

type TODeliveryServiceServers tc.DeliveryServiceServers

// GetCreateHandler assigns an existing Server to and existing Deliveryservice in response to api/1.1/deliveryservices/{xml_id}/servers
func GetCreateHandler(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, []string{"xml_id"}, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	dsName := inf.Params["xml_id"]

	if errs = tenant.Check(inf.User, dsName, inf.Tx.Tx); errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}

	ds, ok, err := GetDSInfoByName(inf.Tx.Tx, dsName)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("ds servers create scanning: "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, errors.New("delivery service not found"))
		return
	}

	// get list of server Ids to insert
	payload := tc.DeliveryServiceServers{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON"), nil)
		return
	}
	payload.XmlId = dsName
	serverNames := payload.ServerNames

	serverNamesCdnIdAndTypes, err := dbhelpers.GetServerTypesCdnIdFromHostNames(inf.Tx.Tx, serverNames)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, err, nil)
		return
	}

	userErr := ValidateDSSAssignments(ds, serverNamesCdnIdAndTypes)
	if userErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, userErr, nil)
		return
	}

	errs = ValidateServerCapabilities(ds.ID, serverNamesCdnIdAndTypes, inf.Tx.Tx)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}

	res, err := inf.Tx.Tx.Exec(`INSERT INTO deliveryservice_server (deliveryservice, server) SELECT $1, id FROM server WHERE host_name = ANY($2::text[])`, ds.ID, pq.Array(serverNames))
	if err != nil {
		inf.HandleErrs(w, r, api.ParseDBError(err))
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

	if err := deliveryservice.EnsureParams(inf.Tx.Tx, ds.ID, ds.Name, ds.EdgeHeaderRewrite, ds.MidHeaderRewrite, ds.RegexRemap, ds.CacheURL, ds.SigningAlgorithm, ds.Type, ds.MaxOriginConnections); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice_server replace ensuring ds parameters: "+err.Error()))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+dsName+", ID: "+strconv.Itoa(ds.ID)+", ACTION: Assigned servers "+strings.Join(serverNames, ", ")+" to delivery service", inf.User, inf.Tx.Tx)
	api.WriteResp(w, r, tc.DeliveryServiceServers{payload.ServerNames, payload.XmlId})
}

// ValidateDSSAssignments returns an error if the given servers cannot be assigned to the given delivery service.
func ValidateDSSAssignments(ds DSInfo, servers []dbhelpers.ServerHostNameCDNIDAndType) error {
	if ds.Topology == nil {
		for _, s := range servers {
			if ds.CDNID != nil && s.CDNID != *ds.CDNID {
				return errors.New("server and delivery service CDNs do not match")
			}
		}
		return nil
	}
	for _, s := range servers {
		if s.Type != tc.OriginTypeName {
			return errors.New("only servers of type ORG may be assigned to topology-based delivery services")
		}
	}
	return nil
}

// ValidateServerCapabilities checks that the delivery service's requirements are met by each server to be assigned.
func ValidateServerCapabilities(dsID int, serverNamesAndTypes []dbhelpers.ServerHostNameCDNIDAndType, tx *sql.Tx) apierrors.Errors {
	nonOriginServerNames := []string{}
	for _, s := range serverNamesAndTypes {
		if strings.HasPrefix(s.Type, tc.EdgeTypePrefix) {
			nonOriginServerNames = append(nonOriginServerNames, s.HostName)
		}
	}

	var sCaps []string
	dsCaps, err := dbhelpers.GetDSRequiredCapabilitiesFromID(dsID, tx)

	errs := apierrors.New()
	if err != nil {
		errs.SystemError = err
		errs.Code = http.StatusInternalServerError
		return errs
	}

	for _, name := range nonOriginServerNames {
		sCaps, err = dbhelpers.GetServerCapabilitiesFromName(name, tx)
		if err != nil {
			errs.Code = http.StatusInternalServerError
			errs.SystemError = err
			return errs
		}
		for _, dsc := range dsCaps {
			if !util.ContainsStr(sCaps, dsc) {
				errs.UserError = fmt.Errorf("Caching server cannot be assigned to this delivery service without having the required delivery service capabilities: [%v] for server %s", dsCaps, name)
				errs.Code = http.StatusBadRequest
				return errs
			}
		}
	}

	return errs
}

func insertIdsQuery() string {
	query := `INSERT INTO deliveryservice_server (deliveryservice, server)
VALUES (:id, :server )`
	return query
}

// GetReadAssigned retrieves lists of servers  based in the filter identified in the request: api/1.1/deliveryservices/{id}/servers|unassigned_servers|eligible
func GetReadAssigned(w http.ResponseWriter, r *http.Request) {
	getRead(w, r, false, tc.Alerts{})
}

// GetReadUnassigned retrieves lists of servers  based in the filter identified in the request: api/1.1/deliveryservices/{id}/servers|unassigned_servers|eligible
func GetReadUnassigned(w http.ResponseWriter, r *http.Request) {
	alerts := api.CreateDeprecationAlerts(nil)
	getRead(w, r, true, alerts)
}

func getRead(w http.ResponseWriter, r *http.Request, unassigned bool, alerts tc.Alerts) {
	inf, errs := api.NewInfo(r, []string{"id"}, []string{"id"})
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	servers, err := read(inf.Tx, inf.IntParams["id"], inf.User, unassigned)
	if err != nil {
		alerts.AddNewAlert(tc.ErrorLevel, err.Error())
		api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
		return
	}

	if inf.Version.Major < 3 {
		v11ServerList := []tc.DSServerV11{}
		for _, srv := range servers {
			v11server := tc.DSServerV11{}
			v11server.DSServerBase = srv.DSServerBase

			interfaces := *srv.ServerInterfaces
			legacyInterface, err := tc.InterfaceInfoToLegacyInterfaces(interfaces)
			if err != nil {
				api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("converting to server detail v11: "+err.Error()))
				return
			}
			v11server.LegacyInterfaceDetails = legacyInterface

			v11ServerList = append(v11ServerList, v11server)
		}
		api.WriteAlertsObj(w, r, http.StatusOK, alerts, v11ServerList)
		return
	}

	api.WriteAlertsObj(w, r, http.StatusOK, alerts, servers)
}

func read(tx *sqlx.Tx, dsID int, user *auth.CurrentUser, unassigned bool) ([]tc.DSServer, error) {
	queryDataString :=
		`,
cg.name as cachegroup,
s.cachegroup as cachegroup_id,
s.cdn_id,
cdn.name as cdn_name,
s.domain_name,
s.guid,
s.host_name,
s.https_port,
s.ilo_ip_address,
s.ilo_ip_gateway,
s.ilo_ip_netmask,
s.ilo_password,
s.ilo_username,
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
`

	queryFormatString := `
SELECT
	s.id
	%v
FROM server s
JOIN cachegroup cg ON s.cachegroup = cg.id
JOIN cdn cdn ON s.cdn_id = cdn.id
JOIN phys_location pl ON s.phys_location = pl.id
JOIN profile p ON s.profile = p.id
JOIN status st ON s.status = st.id
JOIN type t ON s.type = t.id `
	if unassigned {
		queryFormatString += `WHERE s.id not in (select server from deliveryservice_server where deliveryservice = $1)`
	} else {
		queryFormatString += `WHERE s.id in (select server from deliveryservice_server where deliveryservice = $1)`
	}

	idRows, err := tx.Queryx(fmt.Sprintf(queryFormatString, ""), dsID)
	if err != nil {
		return nil, errors.New("error querying dss ids: " + err.Error())
	}
	var serverIDs []int
	for idRows.Next() {
		var serverID int
		err := idRows.Scan(&serverID)
		if err != nil {
			return nil, errors.New("error scanning dss id rows: " + err.Error())
		}

		serverIDs = append(serverIDs, serverID)
	}

	serversMap, err := dbhelpers.GetServersInterfaces(serverIDs, tx.Tx)
	if err != nil {
		return nil, errors.New("unable to get server interfaces: " + err.Error())
	}

	rows, err := tx.Queryx(fmt.Sprintf(queryFormatString, queryDataString), dsID)
	if err != nil {
		return nil, errors.New("error querying dss rows: " + err.Error())
	}
	defer rows.Close()

	servers := []tc.DSServer{}
	for rows.Next() {
		s := tc.DSServer{}
		err := rows.Scan(
			&s.ID,
			&s.Cachegroup,
			&s.CachegroupID,
			&s.CDNID,
			&s.CDNName,
			&s.DomainName,
			&s.GUID,
			&s.HostName,
			&s.HTTPSPort,
			&s.ILOIPAddress,
			&s.ILOIPGateway,
			&s.ILOIPNetmask,
			&s.ILOPassword,
			&s.ILOUsername,
			&s.LastUpdated,
			&s.MgmtIPAddress,
			&s.MgmtIPGateway,
			&s.MgmtIPNetmask,
			&s.OfflineReason,
			&s.PhysLocation,
			&s.PhysLocationID,
			&s.Profile,
			&s.ProfileDesc,
			&s.ProfileID,
			&s.Rack,
			&s.RouterHostName,
			&s.RouterPortName,
			&s.Status,
			&s.StatusID,
			&s.TCPPort,
			&s.Type,
			&s.TypeID,
			&s.UpdPending,
		)
		if err != nil {
			return nil, errors.New("error scanning dss rows: " + err.Error())
		}
		s.ServerInterfaces = &[]tc.ServerInterfaceInfo{}
		if interfacesMap, ok := serversMap[*s.ID]; ok {
			for _, interfaceInfo := range interfacesMap {
				*s.ServerInterfaces = append(*s.ServerInterfaces, interfaceInfo)
			}
		}

		if user.PrivLevel < auth.PrivLevelAdmin {
			s.ILOPassword = util.StrPtr("")
		}
		servers = append(servers, s)
	}
	return servers, nil
}

type TODSSDeliveryService struct {
	api.APIInfoImpl `json:"-"`
	tc.DeliveryServiceNullable
}

// Read shows all of the delivery services associated with the specified server.
func (dss *TODSSDeliveryService) Read(h http.Header, useIMS bool) ([]interface{}, apierrors.Errors, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	returnable := []interface{}{}
	params := dss.APIInfo().Params
	tx := dss.APIInfo().Tx.Tx
	user := dss.APIInfo().User
	errs := apierrors.New()

	// TODO: I think maybe this could utilize IntParams?
	if err := api.IsInt(params["id"]); err != nil {
		errs.UserError = err
		errs.Code = http.StatusBadRequest
		return nil, errs, nil
	}

	if _, ok := params["orderby"]; !ok {
		params["orderby"] = "xml_id"
	}

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"xml_id": dbhelpers.WhereColumnInfo{"ds.xml_id", nil},
		"xmlId":  dbhelpers.WhereColumnInfo{"ds.xml_id", nil},
	}
	where, orderBy, pagination, queryValues, dbErrs := dbhelpers.BuildWhereAndOrderByAndPagination(params, queryParamsToSQLCols)
	if len(dbErrs) > 0 {
		errs.SetSystemError("reading server dses: " + util.JoinErrsStr(dbErrs))
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	}

	if where != "" {
		where = where + " AND "
	} else {
		where = "WHERE "
	}
	where += "ds.id in (SELECT deliveryService FROM deliveryservice_server where server = :server)"

	tenantIDs, err := tenant.GetUserTenantIDListTx(tx, user.TenantID)
	if err != nil {
		log.Errorln("received error querying for user's tenants: " + err.Error())
		errs.SystemError = err
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	}
	where, queryValues = dbhelpers.AddTenancyCheck(where, queryValues, "ds.tenant_id", tenantIDs)
	query := deliveryservice.GetDSSelectQuery() + where + orderBy + pagination
	queryValues["server"] = dss.APIInfo().Params["id"]

	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(dss.APIInfo().Tx, h, queryValues, selectMaxLastUpdatedQuery(where))
		if !runSecond {
			log.Debugln("IMS HIT")
			return returnable, apierrors.Errors{Code: http.StatusNotModified}, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}
	log.Debugln("generated deliveryServices query: " + query)
	log.Debugf("executing with values: %++v\n", queryValues)

	dses, errs := deliveryservice.GetDeliveryServices(query, queryValues, dss.APIInfo().Tx)
	if errs.Occurred() {
		if errs.SystemError != nil {
			errs.SystemError = fmt.Errorf("reading server dses: %v ", errs.SystemError)
		}
		// TODO: this maybe shouldn't change the status code
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	}

	for _, ds := range dses {
		returnable = append(returnable, ds)
	}
	return returnable, errs, &maxTime
}

func selectMaxLastUpdatedQuery(where string) string {
	return `SELECT max(t) from (
		SELECT max(dss.last_updated) as t from deliveryservice_server dss JOIN deliveryservice ds ON ds.id = dss.deliveryservice ` + where +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='deliveryservice_server') as res`
}

type DSInfo struct {
	ID                   int
	Name                 string
	Type                 tc.DSType
	EdgeHeaderRewrite    *string
	MidHeaderRewrite     *string
	RegexRemap           *string
	SigningAlgorithm     *string
	CacheURL             *string
	MaxOriginConnections *int
	Topology             *string
	CDNID                *int
}

// GetDSInfo loads the DeliveryService fields needed by Delivery Service Servers from the database, from the ID. Returns the data, whether the delivery service was found, and any error.
func GetDSInfo(tx *sql.Tx, id int) (DSInfo, bool, error) {
	qry := `
SELECT
  ds.xml_id,
  tp.name as type,
  ds.edge_header_rewrite,
  ds.mid_header_rewrite,
  ds.regex_remap,
  ds.signing_algorithm,
  ds.cacheurl,
  ds.max_origin_connections,
  ds.topology,
  ds.cdn_id
FROM
  deliveryservice ds
  JOIN type tp ON ds.type = tp.id
WHERE
  ds.id = $1
`
	di := DSInfo{ID: id}
	if err := tx.QueryRow(qry, id).Scan(&di.Name, &di.Type, &di.EdgeHeaderRewrite, &di.MidHeaderRewrite, &di.RegexRemap, &di.SigningAlgorithm, &di.CacheURL, &di.MaxOriginConnections, &di.Topology, &di.CDNID); err != nil {
		if err == sql.ErrNoRows {
			return DSInfo{}, false, nil
		}
		return DSInfo{}, false, fmt.Errorf("querying delivery service server ds info '%v': %v", id, err)
	}
	di.Type = tc.DSTypeFromString(string(di.Type))
	return di, true, nil
}

// GetDSInfoByName loads the DeliveryService fields needed by Delivery Service Servers from the database, from the ID. Returns the data, whether the delivery service was found, and any error.
func GetDSInfoByName(tx *sql.Tx, dsName string) (DSInfo, bool, error) {
	qry := `
SELECT
  ds.id,
  tp.name as type,
  ds.edge_header_rewrite,
  ds.mid_header_rewrite,
  ds.regex_remap,
  ds.signing_algorithm,
  ds.cacheurl,
  ds.max_origin_connections,
  ds.topology,
  ds.cdn_id
FROM
  deliveryservice ds
  JOIN type tp ON ds.type = tp.id
WHERE
  ds.xml_id = $1
`
	di := DSInfo{Name: dsName}
	if err := tx.QueryRow(qry, dsName).Scan(&di.ID, &di.Type, &di.EdgeHeaderRewrite, &di.MidHeaderRewrite, &di.RegexRemap, &di.SigningAlgorithm, &di.CacheURL, &di.MaxOriginConnections, &di.Topology, &di.CDNID); err != nil {
		if err == sql.ErrNoRows {
			return DSInfo{}, false, nil
		}
		return DSInfo{}, false, fmt.Errorf("querying delivery service server ds info by name '%v': %v", dsName, err)
	}
	di.Type = tc.DSTypeFromString(string(di.Type))
	return di, true, nil
}
