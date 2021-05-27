package deliveryservice

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
	"regexp"
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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

	"github.com/asaskevich/govalidator"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type tierType int

const (
	midTier tierType = iota
	edgeTier
)

//we need a type alias to define functions on
type TODeliveryService struct {
	api.APIInfoImpl
	tc.DeliveryServiceV4
}

// TODeliveryServiceOldDetails is the struct to store the old details while updating a DS.
type TODeliveryServiceOldDetails struct {
	OldOrgServerFqdn *string
	OldCdnName       string
	OldCdnId         int
	OldRoutingName   string
	OldSSLKeyVersion *int
}

func (ds TODeliveryService) MarshalJSON() ([]byte, error) {
	return json.Marshal(ds.DeliveryServiceV4)
}

func (ds *TODeliveryService) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &ds.DeliveryServiceV4)
}

func (ds *TODeliveryService) APIInfo() *api.APIInfo { return ds.ReqInfo }

func (ds *TODeliveryService) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	ds.ID = &i
}

func (ds TODeliveryService) GetKeys() (map[string]interface{}, bool) {
	if ds.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *ds.ID}, true
}

func (ds TODeliveryService) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

func (ds *TODeliveryService) GetAuditName() string {
	return ds.XMLID
}

func (ds *TODeliveryService) GetType() string {
	return "ds"
}

// IsTenantAuthorized checks that the user is authorized for both the delivery service's existing tenant, and the new tenant they're changing it to (if different).
func (ds *TODeliveryService) IsTenantAuthorized(user *auth.CurrentUser) (bool, error) {
	return isTenantAuthorized(ds.ReqInfo, &ds.DeliveryServiceV4)
}

func CreateV12(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ds := tc.DeliveryServiceNullableV12{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}

	res, status, userErr, sysErr := createV12(w, r, inf, ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice creation was successful.", []tc.DeliveryServiceNullableV12{*res})
}
func CreateV13(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ds := tc.DeliveryServiceNullableV13{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}

	res, status, userErr, sysErr := createV13(w, r, inf, ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice creation was successful.", []tc.DeliveryServiceNullableV13{*res})
}
func CreateV14(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ds := tc.DeliveryServiceNullableV14{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}

	res, status, userErr, sysErr := createV14(w, r, inf, ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice creation was successful.", []tc.DeliveryServiceNullableV14{*res})
}

// 	TODO allow users to post names (type, cdn, etc) and get the IDs from the names. This isn't trivial to do in a single query, without dynamically building the entire insert query, and ideally inserting would be one query. But it'd be much more convenient for users. Alternatively, remove IDs from the database entirely and use real candidate keys.
func CreateV15(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ds := tc.DeliveryServiceNullableV15{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}

	res, status, userErr, sysErr := createV15(w, r, inf, ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice creation was successful.", []tc.DeliveryServiceNullableV15{*res})
}
func CreateV30(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ds := tc.DeliveryServiceV30{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}

	res, status, userErr, sysErr := createV30(w, r, inf, ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice creation was successful.", []tc.DeliveryServiceV30{*res})
}
func CreateV31(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ds := tc.DeliveryServiceV31{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}

	res, status, userErr, sysErr := createV31(w, r, inf, ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice creation was successful.", []tc.DeliveryServiceV31{*res})
}
func CreateV40(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ds := tc.DeliveryServiceV40{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}

	res, status, userErr, sysErr := createV40(w, r, inf, ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice creation was successful.", []tc.DeliveryServiceV40{*res})
}

func createV12(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, reqDS tc.DeliveryServiceNullableV12) (*tc.DeliveryServiceNullableV12, int, error, error) {
	dsV13 := tc.DeliveryServiceNullableV13{DeliveryServiceNullableV12: reqDS}
	res, status, userErr, sysErr := createV13(w, r, inf, dsV13)
	if res != nil {
		return &res.DeliveryServiceNullableV12, status, userErr, sysErr
	}
	return nil, status, userErr, sysErr
}
func createV13(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, reqDS tc.DeliveryServiceNullableV13) (*tc.DeliveryServiceNullableV13, int, error, error) {
	dsV14 := tc.DeliveryServiceNullableV14{DeliveryServiceNullableV13: reqDS}
	res, status, userErr, sysErr := createV14(w, r, inf, dsV14)
	if res != nil {
		return &res.DeliveryServiceNullableV13, status, userErr, sysErr
	}
	return nil, status, userErr, sysErr
}
func createV14(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, reqDS tc.DeliveryServiceNullableV14) (*tc.DeliveryServiceNullableV14, int, error, error) {
	dsV15 := tc.DeliveryServiceNullableV15{DeliveryServiceNullableV14: reqDS}
	res, status, userErr, sysErr := createV15(w, r, inf, dsV15)
	if res != nil {
		return &res.DeliveryServiceNullableV14, status, userErr, sysErr
	}
	return nil, status, userErr, sysErr
}
func createV15(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, reqDS tc.DeliveryServiceNullableV15) (*tc.DeliveryServiceNullableV15, int, error, error) {
	dsV30 := tc.DeliveryServiceV30{DeliveryServiceNullableV15: reqDS}
	res, status, userErr, sysErr := createV30(w, r, inf, dsV30)
	if res != nil {
		return &res.DeliveryServiceNullableV15, status, userErr, sysErr
	}
	return nil, status, userErr, sysErr
}
func createV30(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, dsV30 tc.DeliveryServiceV30) (*tc.DeliveryServiceV30, int, error, error) {
	ds := tc.DeliveryServiceV31{DeliveryServiceV30: dsV30}
	res, status, userErr, sysErr := createV31(w, r, inf, ds)
	if res != nil {
		return &res.DeliveryServiceV30, status, userErr, sysErr
	}
	return nil, status, userErr, sysErr
}
func createV31(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, dsV31 tc.DeliveryServiceV31) (*tc.DeliveryServiceV31, int, error, error) {
	tx := inf.Tx.Tx
	dsNullable := tc.DeliveryServiceNullableV30(dsV31)
	ds := dsNullable.UpgradeToV4()
	res, status, userErr, sysErr := createV40(w, r, inf, tc.DeliveryServiceV40(ds))
	if res == nil {
		return nil, status, userErr, sysErr
	}

	ds = tc.DeliveryServiceV4(*res)
	if dsV31.CacheURL != nil {
		_, err := tx.Exec("UPDATE deliveryservice SET cacheurl = $1 WHERE ID = $2",
			&dsV31.CacheURL,
			&ds.ID)
		if err != nil {
			usrErr, sysErr, code := api.ParseDBError(err)
			return nil, code, usrErr, sysErr
		}
	}

	if err := EnsureCacheURLParams(tx, *ds.ID, ds.XMLID, dsV31.CacheURL); err != nil {
		return nil, http.StatusInternalServerError, nil, err
	}

	oldRes := tc.DeliveryServiceV31(ds.DowngradeToV3())
	return &oldRes, status, userErr, sysErr
}

// create creates the given ds in the database, and returns the DS with its id and other fields created on insert set. On error, the HTTP status code, user error, and system error are returned. The status code SHOULD NOT be used, if both errors are nil.
func createV40(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, dsV40 tc.DeliveryServiceV40) (*tc.DeliveryServiceV40, int, error, error) {
	user := inf.User
	tx := inf.Tx.Tx
	ds := tc.DeliveryServiceV4(dsV40)
	if err := Validate(tx, &ds); err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid request: " + err.Error()), nil
	}

	if authorized, err := isTenantAuthorized(inf, &ds); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("checking tenant: " + err.Error())
	} else if !authorized {
		return nil, http.StatusForbidden, errors.New("not authorized on this tenant"), nil
	}

	// TODO change DeepCachingType to implement sql.Valuer and sql.Scanner, so sqlx struct scan can be used.
	deepCachingType := ds.DeepCachingType.String() // necessary, because DeepCachingType's default needs to insert the string, not "", and Query doesn't call .String().

	if errCode, userErr, sysErr := dbhelpers.CheckTopology(inf.Tx, ds); userErr != nil || sysErr != nil {
		return nil, errCode, userErr, sysErr
	}
	resultRows, err := tx.Query(insertQuery(),
		&ds.Active,
		&ds.AnonymousBlockingEnabled,
		&ds.CCRDNSTTL,
		&ds.CDNID,
		&ds.CheckPath,
		&ds.ConsistentHashRegex,
		&deepCachingType,
		&ds.DisplayName,
		&ds.DNSBypassCNAME,
		&ds.DNSBypassIP,
		&ds.DNSBypassIP6,
		&ds.DNSBypassTTL,
		&ds.DSCP,
		&ds.EdgeHeaderRewrite,
		&ds.GeoLimitRedirectURL,
		&ds.GeoLimit,
		&ds.GeoLimitCountries,
		&ds.GeoProvider,
		&ds.GlobalMaxMBPS,
		&ds.GlobalMaxTPS,
		&ds.FQPacingRate,
		&ds.HTTPBypassFQDN,
		&ds.InfoURL,
		&ds.InitialDispersion,
		&ds.IPV6RoutingEnabled,
		&ds.LogsEnabled,
		&ds.LongDesc,
		&ds.LongDesc1,
		&ds.LongDesc2,
		&ds.MaxDNSAnswers,
		&ds.MaxOriginConnections,
		&ds.MidHeaderRewrite,
		&ds.MissLat,
		&ds.MissLong,
		&ds.MultiSiteOrigin,
		&ds.OriginShield,
		&ds.ProfileID,
		&ds.Protocol,
		&ds.QStringIgnore,
		&ds.RangeRequestHandling,
		&ds.RegexRemap,
		&ds.RegionalGeoBlocking,
		&ds.RemapText,
		&ds.RoutingName,
		&ds.SigningAlgorithm,
		&ds.SSLKeyVersion,
		&ds.TenantID,
		&ds.Topology,
		&ds.TRRequestHeaders,
		&ds.TRResponseHeaders,
		&ds.TypeID,
		&ds.XMLID,
		&ds.EcsEnabled,
		&ds.RangeSliceBlockSize,
		&ds.FirstHeaderRewrite,
		&ds.InnerHeaderRewrite,
		&ds.LastHeaderRewrite,
		&ds.ServiceCategory,
		&ds.MaxRequestHeaderBytes,
	)

	if err != nil {
		usrErr, sysErr, code := api.ParseDBError(err)
		return nil, code, usrErr, sysErr
	}
	defer resultRows.Close()

	id := 0
	var lastUpdated time.Time
	if !resultRows.Next() {
		return nil, http.StatusInternalServerError, nil, errors.New("no deliveryservice request inserted, no id was returned")
	}
	if err := resultRows.Scan(&id, &lastUpdated); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("could not scan id from insert: " + err.Error())
	}
	if resultRows.Next() {
		return nil, http.StatusInternalServerError, nil, errors.New("too many ids returned from deliveryservice request insert")
	}
	ds.ID = &id

	if ds.ID == nil {
		return nil, http.StatusInternalServerError, nil, errors.New("missing id after insert")
	}
	dsType, err := getTypeFromID(ds.TypeID, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("getting delivery service type: " + err.Error())
	}
	ds.Type = &dsType

	if err := createDefaultRegex(tx, *ds.ID, ds.XMLID); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("creating default regex: " + err.Error())
	}

	if _, err := createConsistentHashQueryParams(tx, *ds.ID, ds.ConsistentHashQueryParams); err != nil {
		usrErr, sysErr, code := api.ParseDBError(err)
		return nil, code, usrErr, sysErr
	}

	matchlists, err := GetDeliveryServicesMatchLists([]string{ds.XMLID}, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("creating DS: reading matchlists: " + err.Error())
	}
	if matchlist, ok := matchlists[ds.XMLID]; !ok {
		return nil, http.StatusInternalServerError, nil, errors.New("creating DS: reading matchlists: not found")
	} else {
		ds.MatchList = matchlist
	}

	cdnName, cdnDomain, dnssecEnabled, err := getCDNNameDomainDNSSecEnabled(*ds.ID, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("creating DS: getting CDN info: " + err.Error())
	}

	ds.ExampleURLs = MakeExampleURLs(ds.Protocol, *ds.Type, ds.RoutingName, ds.MatchList, cdnDomain)

	if err := EnsureParams(tx, *ds.ID, ds.XMLID, ds.EdgeHeaderRewrite, ds.MidHeaderRewrite, ds.RegexRemap, ds.SigningAlgorithm, dsType, ds.MaxOriginConnections); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("ensuring ds parameters:: " + err.Error())
	}

	if dnssecEnabled && ds.Type.UsesDNSSECKeys() {
		if !inf.Config.TrafficVaultEnabled {
			return nil, http.StatusInternalServerError, nil, errors.New("cannot create DNSSEC keys for delivery service: Traffic Vault is not configured")
		}
		if userErr, sysErr, statusCode := PutDNSSecKeys(tx, ds.XMLID, cdnName, ds.ExampleURLs, inf.Vault, r.Context()); userErr != nil || sysErr != nil {
			return nil, statusCode, userErr, sysErr
		}
	}

	if err := createPrimaryOrigin(tx, user, ds); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("creating delivery service: " + err.Error())
	}

	ds.LastUpdated = lastUpdated
	if err := api.CreateChangeLogRawErr(api.ApiChange, "DS: "+ds.XMLID+", ID: "+strconv.Itoa(*ds.ID)+", ACTION: Created delivery service", user, tx); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("error writing to audit log: " + err.Error())
	}

	dsV40 = tc.DeliveryServiceV40(ds)
	return &dsV40, http.StatusOK, nil, nil
}

func createDefaultRegex(tx *sql.Tx, dsID int, xmlID string) error {
	regexStr := `.*\.` + xmlID + `\..*`
	regexID := 0
	if err := tx.QueryRow(`INSERT INTO regex (type, pattern) VALUES ((select id from type where name = 'HOST_REGEXP'), $1::text) RETURNING id`, regexStr).Scan(&regexID); err != nil {
		return errors.New("insert regex: " + err.Error())
	}
	if _, err := tx.Exec(`INSERT INTO deliveryservice_regex (deliveryservice, regex, set_number) VALUES ($1::bigint, $2::bigint, 0)`, dsID, regexID); err != nil {
		return errors.New("executing parameter query to insert location: " + err.Error())
	}
	return nil
}
func createConsistentHashQueryParams(tx *sql.Tx, dsID int, consistentHashQueryParams []string) (int, error) {
	if len(consistentHashQueryParams) == 0 {
		return 0, nil
	}
	c := 0
	q := `INSERT INTO deliveryservice_consistent_hash_query_param (name, deliveryservice_id) VALUES ($1, $2)`
	for _, k := range consistentHashQueryParams {
		if _, err := tx.Exec(q, k, dsID); err != nil {
			return c, err
		}
		c++
	}

	return c, nil
}
func (ds *TODeliveryService) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	version := ds.APIInfo().Version
	if version == nil {
		return nil, nil, errors.New("TODeliveryService.Read called with nil API version"), http.StatusInternalServerError, nil
	}
	if version.Major == 1 && version.Minor < 1 {
		return nil, nil, fmt.Errorf("TODeliveryService.Read called with invalid API version: %d.%d", version.Major, version.Minor), http.StatusInternalServerError, nil
	}

	returnable := []interface{}{}
	dses, userErr, sysErr, errCode, maxTime := readGetDeliveryServices(h, ds.APIInfo(), useIMS)

	if sysErr != nil {
		sysErr = errors.New("reading dses: " + sysErr.Error())
		errCode = http.StatusInternalServerError
	}
	if userErr != nil || sysErr != nil {
		return nil, userErr, sysErr, errCode, nil
	}

	for _, ds := range dses {
		switch {
		// NOTE: it's required to handle minor version cases in a descending >= manner
		case version.Major > 3 && version.Minor >= 0:
			returnable = append(returnable, ds)
		case version.Major > 2 && version.Minor >= 1:
			returnable = append(returnable, ds.DowngradeToV3())
		case version.Major > 2:
			returnable = append(returnable, ds.DowngradeToV3().DeliveryServiceV30)
		case version.Major > 1 || version.Minor >= 5:
			returnable = append(returnable, ds.DowngradeToV3().DeliveryServiceNullableV15)
		case version.Minor >= 4:
			returnable = append(returnable, ds.DowngradeToV3().DeliveryServiceNullableV14)
		case version.Minor >= 3:
			returnable = append(returnable, ds.DowngradeToV3().DeliveryServiceNullableV13)
		case version.Minor >= 1:
			returnable = append(returnable, ds.DowngradeToV3().DeliveryServiceNullableV12)
		default:
			return nil, nil, fmt.Errorf("TODeliveryService.Read called with invalid API version: %d.%d", version.Major, version.Minor), http.StatusInternalServerError, nil
		}
	}
	return returnable, nil, nil, errCode, maxTime
}

func UpdateV12(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	id := inf.IntParams["id"]

	ds := tc.DeliveryServiceNullableV12{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	ds.ID = &id

	res, status, userErr, sysErr := updateV12(w, r, inf, &ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice update was successful.", []tc.DeliveryServiceNullableV12{*res})
}
func UpdateV13(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	id := inf.IntParams["id"]

	ds := tc.DeliveryServiceNullableV13{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	ds.ID = &id

	res, status, userErr, sysErr := updateV13(w, r, inf, &ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice update was successful.", []tc.DeliveryServiceNullableV13{*res})
}
func UpdateV14(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	id := inf.IntParams["id"]

	ds := tc.DeliveryServiceNullableV14{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	ds.ID = &id

	res, status, userErr, sysErr := updateV14(w, r, inf, &ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice update was successful.", []tc.DeliveryServiceNullableV14{*res})
}
func UpdateV15(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	id := inf.IntParams["id"]

	ds := tc.DeliveryServiceNullableV15{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	ds.ID = &id

	res, status, userErr, sysErr := updateV15(w, r, inf, &ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice update was successful.", []tc.DeliveryServiceNullableV15{*res})
}
func UpdateV30(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	id := inf.IntParams["id"]

	ds := tc.DeliveryServiceV30{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	ds.ID = &id

	res, status, userErr, sysErr := updateV30(w, r, inf, &ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice update was successful.", []tc.DeliveryServiceV30{*res})
}
func UpdateV31(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	id := inf.IntParams["id"]

	ds := tc.DeliveryServiceV31{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	ds.ID = &id
	res, status, userErr, sysErr := updateV31(w, r, inf, &ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice update was successful.", []tc.DeliveryServiceV31{*res})
}
func UpdateV40(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	id := inf.IntParams["id"]

	ds := tc.DeliveryServiceV40{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	ds.ID = &id
	res, status, userErr, sysErr := updateV40(w, r, inf, &ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice update was successful.", []tc.DeliveryServiceV40{*res})
}

func updateV12(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, reqDS *tc.DeliveryServiceNullableV12) (*tc.DeliveryServiceNullableV12, int, error, error) {
	dsV13 := tc.DeliveryServiceNullableV13{DeliveryServiceNullableV12: *reqDS}
	// query the DB for existing 1.3 fields in order to "upgrade" this 1.2 request into a 1.3 request
	query := `
SELECT
  ds.deep_caching_type,
  ds.fq_pacing_rate,
  ds.signing_algorithm,
  ds.tr_response_headers,
  ds.tr_request_headers
FROM
  deliveryservice ds
WHERE
  ds.id = $1`
	if err := inf.Tx.Tx.QueryRow(query, *reqDS.ID).Scan(
		&dsV13.DeepCachingType,
		&dsV13.FQPacingRate,
		&dsV13.SigningAlgorithm,
		&dsV13.TRResponseHeaders,
		&dsV13.TRRequestHeaders,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, fmt.Errorf("delivery service ID %d not found", *dsV13.ID), nil
		}
		return nil, http.StatusInternalServerError, nil, fmt.Errorf("querying delivery service ID %d: %s", *dsV13.ID, err.Error())
	}
	if dsV13.DeepCachingType != nil {
		*dsV13.DeepCachingType = tc.DeepCachingTypeFromString(string(*dsV13.DeepCachingType))
	}

	res, status, userErr, sysErr := updateV13(w, r, inf, &dsV13)
	if res != nil {
		return &res.DeliveryServiceNullableV12, status, userErr, sysErr
	}
	return nil, status, userErr, sysErr
}
func updateV13(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, reqDS *tc.DeliveryServiceNullableV13) (*tc.DeliveryServiceNullableV13, int, error, error) {
	dsV14 := tc.DeliveryServiceNullableV14{DeliveryServiceNullableV13: *reqDS}
	// query the DB for existing 1.4 fields in order to "upgrade" this 1.3 request into a 1.4 request
	query := `
SELECT
  ds.consistent_hash_regex,
  ds.max_origin_connections,
  (SELECT ARRAY_AGG(name ORDER BY name)
    FROM deliveryservice_consistent_hash_query_param
    WHERE deliveryservice_id = ds.id) AS query_keys
FROM
  deliveryservice ds
WHERE
  ds.id = $1`
	if err := inf.Tx.Tx.QueryRow(query, *reqDS.ID).Scan(
		&dsV14.ConsistentHashRegex,
		&dsV14.MaxOriginConnections,
		pq.Array(&dsV14.ConsistentHashQueryParams),
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, fmt.Errorf("delivery service ID %d not found", *dsV14.ID), nil
		}
		return nil, http.StatusInternalServerError, nil, fmt.Errorf("querying delivery service ID %d: %s", *dsV14.ID, err.Error())
	}
	res, status, userErr, sysErr := updateV14(w, r, inf, &dsV14)
	if res != nil {
		return &res.DeliveryServiceNullableV13, status, userErr, sysErr
	}
	return nil, status, userErr, sysErr
}
func updateV14(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, reqDS *tc.DeliveryServiceNullableV14) (*tc.DeliveryServiceNullableV14, int, error, error) {
	dsV15 := tc.DeliveryServiceNullableV15{DeliveryServiceNullableV14: *reqDS}
	// query the DB for existing 1.5 fields in order to "upgrade" this 1.4 request into a 1.5 request
	query := `
SELECT
  ds.ecs_enabled,
  ds.range_slice_block_size
FROM
  deliveryservice ds
WHERE
  ds.id = $1`
	if err := inf.Tx.Tx.QueryRow(query, *reqDS.ID).Scan(
		&dsV15.EcsEnabled,
		&dsV15.RangeSliceBlockSize,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, fmt.Errorf("delivery service ID %d not found", *dsV15.ID), nil
		}
		return nil, http.StatusInternalServerError, nil, fmt.Errorf("querying delivery service ID %d: %s", *dsV15.ID, err.Error())
	}
	res, status, userErr, sysErr := updateV15(w, r, inf, &dsV15)
	if res != nil {
		return &res.DeliveryServiceNullableV14, status, userErr, sysErr
	}
	return nil, status, userErr, sysErr
}
func updateV15(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, reqDS *tc.DeliveryServiceNullableV15) (*tc.DeliveryServiceNullableV15, int, error, error) {
	dsV30 := tc.DeliveryServiceV30{DeliveryServiceNullableV15: *reqDS}
	// query the DB for existing 3.0 fields in order to "upgrade" this 1.5 request into a 3.0 request
	query := `
SELECT
  ds.topology,
  ds.first_header_rewrite,
  ds.inner_header_rewrite,
  ds.last_header_rewrite,
  ds.service_category
FROM
  deliveryservice ds
WHERE
  ds.id = $1`
	if err := inf.Tx.Tx.QueryRow(query, *reqDS.ID).Scan(
		&dsV30.Topology,
		&dsV30.FirstHeaderRewrite,
		&dsV30.InnerHeaderRewrite,
		&dsV30.LastHeaderRewrite,
		&dsV30.ServiceCategory,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, fmt.Errorf("delivery service ID %d not found", *dsV30.ID), nil
		}
		return nil, http.StatusInternalServerError, nil, fmt.Errorf("querying delivery service ID %d: %s", *dsV30.ID, err.Error())
	}
	res, status, userErr, sysErr := updateV30(w, r, inf, &dsV30)
	if res != nil {
		return &res.DeliveryServiceNullableV15, status, userErr, sysErr
	}
	return nil, status, userErr, sysErr
}
func updateV30(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, dsV30 *tc.DeliveryServiceV30) (*tc.DeliveryServiceV30, int, error, error) {
	dsV31 := tc.DeliveryServiceV31{DeliveryServiceV30: *dsV30}
	// query the DB for existing 3.1 fields in order to "upgrade" this 3.0 request into a 3.1 request
	query := `
SELECT
  ds.max_request_header_bytes
FROM
  deliveryservice ds
WHERE
  ds.id = $1`
	if err := inf.Tx.Tx.QueryRow(query, *dsV30.ID).Scan(
		&dsV31.MaxRequestHeaderBytes,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, fmt.Errorf("delivery service ID %d not found", *dsV31.ID), nil
		}
		return nil, http.StatusInternalServerError, nil, fmt.Errorf("querying delivery service ID %d: %s", *dsV31.ID, err.Error())
	}
	res, status, userErr, sysErr := updateV31(w, r, inf, &dsV31)
	if res != nil {
		return &res.DeliveryServiceV30, status, userErr, sysErr
	}
	return nil, status, userErr, sysErr
}
func updateV31(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, dsV31 *tc.DeliveryServiceV31) (*tc.DeliveryServiceV31, int, error, error) {
	dsNull := tc.DeliveryServiceNullableV30(*dsV31)
	ds := dsNull.UpgradeToV4()
	dsV40 := tc.DeliveryServiceV40(ds)
	tx := inf.Tx.Tx
	res, status, usrErr, sysErr := updateV40(w, r, inf, &dsV40)
	if res == nil {
		return nil, status, usrErr, sysErr
	}
	ds = tc.DeliveryServiceV4(*res)
	if dsV31.CacheURL != nil {
		_, err := tx.Exec("UPDATE deliveryservice SET cacheurl = $1 WHERE ID = $2",
			&dsV31.CacheURL,
			&ds.ID)
		if err != nil {
			usrErr, sysErr, code := api.ParseDBError(err)
			return nil, code, usrErr, sysErr
		}
	}

	if err := EnsureCacheURLParams(tx, *ds.ID, ds.XMLID, dsV31.CacheURL); err != nil {
		return nil, http.StatusInternalServerError, nil, err
	}

	oldRes := tc.DeliveryServiceV31(ds.DowngradeToV3())
	return &oldRes, http.StatusOK, nil, nil
}
func updateV40(w http.ResponseWriter, r *http.Request, inf *api.APIInfo, dsV40 *tc.DeliveryServiceV40) (*tc.DeliveryServiceV40, int, error, error) {
	tx := inf.Tx.Tx
	user := inf.User
	ds := tc.DeliveryServiceV4(*dsV40)
	if err := Validate(tx, &ds); err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid request: " + err.Error()), nil
	}

	if authorized, err := isTenantAuthorized(inf, &ds); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("checking tenant: " + err.Error())
	} else if !authorized {
		return nil, http.StatusForbidden, errors.New("not authorized on this tenant"), nil
	}

	if ds.ID == nil {
		return nil, http.StatusBadRequest, errors.New("missing id"), nil
	}

	dsType, ok, err := getDSType(tx, ds.XMLID)
	if !ok {
		return nil, http.StatusNotFound, errors.New("delivery service '" + ds.XMLID + "' not found"), nil
	}
	if err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("getting delivery service type during update: " + err.Error())
	}

	errCode := http.StatusOK
	var userErr error
	var sysErr error
	var oldDetails TODeliveryServiceOldDetails
	var sslKeysExist, cdnRoutingDetailDiff bool
	if dsType.HasSSLKeys() {
		oldDetails, userErr, sysErr, errCode = getOldDetails(*ds.ID, tx)
		if userErr != nil || sysErr != nil {
			return nil, errCode, userErr, sysErr
		}
		if sslKeysExist, err = getSSLVersion(ds.XMLID, tx); err != nil {
			return nil, http.StatusInternalServerError, nil, fmt.Errorf("querying delivery service with sslKeyVersion failed: %s", err)
		}
		if sslKeysExist {
			if oldDetails.OldCdnId != ds.CDNID {
				cdnRoutingDetailDiff = true
			}
			if ds.CDNName != nil && oldDetails.OldCdnName != *ds.CDNName {
				cdnRoutingDetailDiff = true
			}
			if oldDetails.OldRoutingName != ds.RoutingName {
				cdnRoutingDetailDiff = true
			}
			if cdnRoutingDetailDiff {
				return nil, http.StatusBadRequest, errors.New("delivery service has ssl keys that cannot be automatically changed, therefore CDN and routing name are immutable"), nil
			}
			ds.SSLKeyVersion = oldDetails.OldSSLKeyVersion
		}
	}

	// TODO change DeepCachingType to implement sql.Valuer and sql.Scanner, so sqlx struct scan can be used.
	deepCachingType := ds.DeepCachingType.String() // necessary, because DeepCachingType's default needs to insert the string, not "", and Query doesn't call .String().

	userErr, sysErr, errCode = api.CheckIfUnModified(r.Header, inf.Tx, *ds.ID, "deliveryservice")
	if userErr != nil || sysErr != nil {
		return nil, errCode, userErr, sysErr
	}

	if errCode, userErr, sysErr = dbhelpers.CheckTopology(inf.Tx, ds); userErr != nil || sysErr != nil {
		return nil, errCode, userErr, sysErr
	}

	if ds.Topology != nil {
		requiredCapabilities, err := dbhelpers.GetDSRequiredCapabilitiesFromID(*ds.ID, tx)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, errors.New("getting existing DS required capabilities: " + err.Error())
		}
		if len(requiredCapabilities) > 0 {
			if userErr, sysErr, status := EnsureTopologyBasedRequiredCapabilities(tx, *ds.ID, *ds.Topology, requiredCapabilities); userErr != nil || sysErr != nil {
				return nil, status, userErr, sysErr
			}
		}

		userErr, sysErr, status := dbhelpers.CheckOriginServerInDSCG(tx, *ds.ID, *ds.Topology)
		if userErr != nil || sysErr != nil {
			return nil, status, userErr, sysErr
		}
	}

	resultRows, err := tx.Query(updateDSQuery(),
		&ds.Active,
		&ds.CCRDNSTTL,
		&ds.CDNID,
		&ds.CheckPath,
		&deepCachingType,
		&ds.DisplayName,
		&ds.DNSBypassCNAME,
		&ds.DNSBypassIP,
		&ds.DNSBypassIP6,
		&ds.DNSBypassTTL,
		&ds.DSCP,
		&ds.EdgeHeaderRewrite,
		&ds.GeoLimitRedirectURL,
		&ds.GeoLimit,
		&ds.GeoLimitCountries,
		&ds.GeoProvider,
		&ds.GlobalMaxMBPS,
		&ds.GlobalMaxTPS,
		&ds.FQPacingRate,
		&ds.HTTPBypassFQDN,
		&ds.InfoURL,
		&ds.InitialDispersion,
		&ds.IPV6RoutingEnabled,
		&ds.LogsEnabled,
		&ds.LongDesc,
		&ds.LongDesc1,
		&ds.LongDesc2,
		&ds.MaxDNSAnswers,
		&ds.MidHeaderRewrite,
		&ds.MissLat,
		&ds.MissLong,
		&ds.MultiSiteOrigin,
		&ds.OriginShield,
		&ds.ProfileID,
		&ds.Protocol,
		&ds.QStringIgnore,
		&ds.RangeRequestHandling,
		&ds.RegexRemap,
		&ds.RegionalGeoBlocking,
		&ds.RemapText,
		&ds.RoutingName,
		&ds.SigningAlgorithm,
		&ds.SSLKeyVersion,
		&ds.TenantID,
		&ds.TRRequestHeaders,
		&ds.TRResponseHeaders,
		&ds.TypeID,
		&ds.XMLID,
		&ds.AnonymousBlockingEnabled,
		&ds.ConsistentHashRegex,
		&ds.MaxOriginConnections,
		&ds.EcsEnabled,
		&ds.RangeSliceBlockSize,
		&ds.Topology,
		&ds.FirstHeaderRewrite,
		&ds.InnerHeaderRewrite,
		&ds.LastHeaderRewrite,
		&ds.ServiceCategory,
		&ds.MaxRequestHeaderBytes,
		&ds.ID)

	if err != nil {
		usrErr, sysErr, code := api.ParseDBError(err)
		return nil, code, usrErr, sysErr
	}
	defer resultRows.Close()
	if !resultRows.Next() {
		return nil, http.StatusNotFound, errors.New("no delivery service found with this id"), nil
	}
	var lastUpdated time.Time
	if err := resultRows.Scan(&lastUpdated); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("scan updating delivery service: " + err.Error())
	}
	if resultRows.Next() {
		return nil, http.StatusInternalServerError, nil, errors.New("updating delivery service " + ds.XMLID + ": " + "this update affected too many rows: > 1")
	}

	if ds.ID == nil {
		return nil, http.StatusInternalServerError, nil, errors.New("missing id after update")
	}
	newDSType, err := getTypeFromID(ds.TypeID, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("getting delivery service type after update: " + err.Error())
	}
	ds.Type = &newDSType

	cdnDomain, err := getCDNDomain(*ds.ID, tx) // need to get the domain again, in case it changed.
	if err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("getting CDN domain: (" + cdnDomain + ") after update: " + err.Error())
	}

	matchLists, err := GetDeliveryServicesMatchLists([]string{ds.XMLID}, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("getting matchlists after update: " + err.Error())
	}
	if ml, ok := matchLists[ds.XMLID]; !ok {
		return nil, http.StatusInternalServerError, nil, errors.New("no matchlists after update")
	} else {
		ds.MatchList = ml
	}

	if err := EnsureParams(tx, *ds.ID, ds.XMLID, ds.EdgeHeaderRewrite, ds.MidHeaderRewrite, ds.RegexRemap, ds.SigningAlgorithm, newDSType, ds.MaxOriginConnections); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("ensuring ds parameters:: " + err.Error())
	}

	if oldDetails.OldOrgServerFqdn != nil && ds.OrgServerFQDN != nil && *oldDetails.OldOrgServerFqdn != *ds.OrgServerFQDN {
		if err := updatePrimaryOrigin(tx, user, ds); err != nil {
			return nil, http.StatusInternalServerError, nil, errors.New("updating delivery service: " + err.Error())
		}
	}

	ds.LastUpdated = lastUpdated

	// the update may change or delete the query params -- delete existing and re-add if any provided
	q := `DELETE FROM deliveryservice_consistent_hash_query_param WHERE deliveryservice_id = $1`
	if res, err := tx.Exec(q, *ds.ID); err != nil {
		return nil, http.StatusInternalServerError, nil, fmt.Errorf("deleting consistent hash query params for ds %s: %w", ds.XMLID, err)
	} else if c, _ := res.RowsAffected(); c > 0 {
		api.CreateChangeLogRawTx(api.ApiChange, "DS: "+ds.XMLID+", ID: "+strconv.Itoa(*ds.ID)+", ACTION: Deleted "+strconv.FormatInt(c, 10)+" consistent hash query params", user, tx)
	}

	if _, err = createConsistentHashQueryParams(tx, *ds.ID, ds.ConsistentHashQueryParams); err != nil {
		usrErr, sysErr, code := api.ParseDBError(err)
		return nil, code, usrErr, sysErr
	}

	if err := api.CreateChangeLogRawErr(api.ApiChange, "Updated ds: "+ds.XMLID+" id: "+strconv.Itoa(*ds.ID), user, tx); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("writing change log entry: " + err.Error())
	}
	dsV40 = (*tc.DeliveryServiceV40)(&ds)
	return dsV40, http.StatusOK, nil, nil
}

//Delete is the DeliveryService implementation of the Deleter interface.
func (ds *TODeliveryService) Delete() (error, error, int) {
	if ds.ID == nil {
		return errors.New("missing id"), nil, http.StatusBadRequest
	}

	xmlID, ok, err := GetXMLID(ds.ReqInfo.Tx.Tx, *ds.ID)
	if err != nil {
		return nil, errors.New("ds delete: getting xmlid: " + err.Error()), http.StatusInternalServerError
	} else if !ok {
		return errors.New("delivery service not found"), nil, http.StatusNotFound
	}
	ds.XMLID = xmlID

	// Note ds regexes MUST be deleted before the ds, because there's a ON DELETE CASCADE on deliveryservice_regex (but not on regex).
	// Likewise, it MUST happen in a transaction with the later DS delete, so they aren't deleted if the DS delete fails.
	if _, err := ds.ReqInfo.Tx.Tx.Exec(`DELETE FROM regex WHERE id IN (SELECT regex FROM deliveryservice_regex WHERE deliveryservice=$1)`, *ds.ID); err != nil {
		return nil, errors.New("TODeliveryService.Delete deleting regexes for delivery service: " + err.Error()), http.StatusInternalServerError
	}

	if _, err := ds.ReqInfo.Tx.Tx.Exec(`DELETE FROM deliveryservice_regex WHERE deliveryservice=$1`, *ds.ID); err != nil {
		return nil, errors.New("TODeliveryService.Delete deleting delivery service regexes: " + err.Error()), http.StatusInternalServerError
	}

	userErr, sysErr, errCode := api.GenericDelete(ds)
	if userErr != nil || sysErr != nil {
		return userErr, sysErr, errCode
	}

	paramConfigFilePrefixes := []string{"hdr_rw_", "hdr_rw_mid_", "regex_remap_", "cacheurl_"}
	configFiles := []string{}
	for _, prefix := range paramConfigFilePrefixes {
		configFiles = append(configFiles, prefix+ds.XMLID+".config")
	}

	if _, err := ds.ReqInfo.Tx.Tx.Exec(`DELETE FROM parameter WHERE name = 'location' AND config_file = ANY($1)`, pq.Array(configFiles)); err != nil {
		return nil, errors.New("TODeliveryService.Delete deleting delivery service parameteres: " + err.Error()), http.StatusInternalServerError
	}

	return nil, nil, http.StatusOK
}

func (v *TODeliveryService) DeleteQuery() string {
	return `DELETE FROM deliveryservice WHERE id = :id`
}

func isActiveState(s string) error {
	switch tc.DeliveryServiceActiveState(s) {
	case tc.DS_ACTIVE:
		fallthrough
	case tc.DS_INACTIVE:
		fallthrough
	case tc.DS_PRIMED:
		return nil
	}
	return fmt.Errorf("'%s' is not a known Delivery Service Active State", s)
}

func readGetDeliveryServices(h http.Header, inf *api.APIInfo, useIMS bool) ([]tc.DeliveryServiceV4, error, error, int, *time.Time) {
	if inf == nil || inf.Tx == nil || inf.Tx.Tx == nil {
		return nil, nil, errors.New("cannot get Delivery Services with nil APIInfo/database transaction"), http.StatusInternalServerError, nil
	}
	var maxTime time.Time
	var runSecond bool
	if strings.HasSuffix(inf.Params["id"], ".json") {
		inf.Params["id"] = inf.Params["id"][:len(inf.Params["id"])-len(".json")]
	}
	if _, ok := inf.Params["orderby"]; !ok {
		inf.Params["orderby"] = "xml_id"
	}

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"id":               {Column: "ds.id", Checker: api.IsInt},
		"cdn":              {Column: "ds.cdn_id", Checker: api.IsInt},
		"xml_id":           {Column: "ds.xml_id"},
		"xmlId":            {Column: "ds.xml_id"},
		"profile":          {Column: "ds.profile", Checker: api.IsInt},
		"type":             {Column: "ds.type", Checker: api.IsInt},
		"logsEnabled":      {Column: "ds.logs_enabled", Checker: api.IsBool},
		"tenant":           {Column: "ds.tenant_id", Checker: api.IsInt},
		"signingAlgorithm": {Column: "ds.signing_algorithm"},
		"topology":         {Column: "ds.topology"},
		"serviceCategory":  {Column: "ds.service_category"},
	}

	if inf.Version.Major >= 4 {
		queryParamsToSQLCols["active"] = dbhelpers.WhereColumnInfo{
			Checker: isActiveState,
			Column:  "ds.active",
		}
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(inf.Params, queryParamsToSQLCols)
	if len(errs) > 0 {
		return nil, util.JoinErrs(errs), nil, http.StatusBadRequest, nil
	}
	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(inf.Tx, h, queryValues, selectMaxLastUpdatedQuery(where))
		if !runSecond {
			log.Debugln("IMS HIT")
			return []tc.DeliveryServiceV4{}, nil, nil, http.StatusNotModified, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}
	tenantIDs, err := tenant.GetUserTenantIDListTx(inf.Tx.Tx, inf.User.TenantID)

	if err != nil {
		log.Errorln("received error querying for user's tenants: " + err.Error())
		return nil, nil, tc.DBError, http.StatusInternalServerError, &maxTime
	}

	where, queryValues = dbhelpers.AddTenancyCheck(where, queryValues, "ds.tenant_id", tenantIDs)

	if accessibleTo, ok := inf.Params["accessibleTo"]; ok {
		if err := api.IsInt(accessibleTo); err != nil {
			log.Errorln("unknown parameter value: " + err.Error())
			return nil, errors.New("accessibleTo must be an integer"), nil, http.StatusBadRequest, &maxTime
		}
		accessibleTo, _ := strconv.Atoi(accessibleTo)
		accessibleTenants, err := tenant.GetUserTenantIDListTx(inf.Tx.Tx, accessibleTo)
		if err != nil {
			log.Errorln("unable to get tenants: " + err.Error())
			return nil, nil, tc.DBError, http.StatusInternalServerError, &maxTime
		}
		where += " AND ds.tenant_id = ANY(CAST(:accessibleTo AS bigint[])) "
		queryValues["accessibleTo"] = pq.Array(accessibleTenants)
	}

	if inf.Version.Major < 4 {
		if isActive, ok := inf.Params["active"]; ok {
			if err := api.IsBool(isActive); err != nil {
				return nil, errors.New("active must be a boolean"), nil, http.StatusBadRequest, &maxTime
			}
			isActive, _ := strconv.ParseBool(isActive)
			where += " AND ds.active"
			if isActive {
				where += " = "
			} else {
				where += " <> "
			}
			where += "'ACTIVE'"
		}
	}

	query := SelectDeliveryServicesQuery + where + orderBy + pagination
	log.Debugln("generated deliveryServices query: " + query)
	log.Debugf("executing with values: %++v\n", queryValues)

	r, e1, e2, code := GetDeliveryServices(query, queryValues, inf.Tx)
	return r, e1, e2, code, &maxTime
}

func requiredIfMatchesTypeName(patterns []string, typeName string) func(interface{}) error {
	return func(value interface{}) error {
		switch v := value.(type) {
		case *int:
			if v != nil {
				return nil
			}
		case *bool:
			if v != nil {
				return nil
			}
		case *string:
			if v != nil {
				return nil
			}
		case *float64:
			if v != nil {
				return nil
			}
		default:
			return fmt.Errorf("validation failure: unknown type %T", value)
		}
		pattern := strings.Join(patterns, "|")
		err := error(nil)
		match := false
		if typeName != "" {
			match, err = regexp.MatchString(pattern, typeName)
			if match {
				return fmt.Errorf("is required if type is '%s'", typeName)
			}
		}
		return err
	}
}

func Validate(tx *sql.Tx, ds *tc.DeliveryServiceV4) error {
	sanitize(ds)
	neverOrAlways := validation.NewStringRule(tovalidate.IsOneOfStringICase("NEVER", "ALWAYS"),
		"must be one of 'NEVER' or 'ALWAYS'")
	isDNSName := validation.NewStringRule(govalidator.IsDNSName, "must be a valid hostname")
	noPeriods := validation.NewStringRule(tovalidate.NoPeriods, "cannot contain periods")
	noSpaces := validation.NewStringRule(tovalidate.NoSpaces, "cannot contain spaces")
	noLineBreaks := validation.NewStringRule(tovalidate.NoLineBreaks, "cannot contain line breaks")
	errs := tovalidate.ToErrors(validation.Errors{
		"active":              validation.Validate(ds.Active, validation.NotNil),
		"cdnId":               validation.Validate(ds.CDNID, validation.Required),
		"deepCachingType":     validation.Validate(ds.DeepCachingType, neverOrAlways),
		"displayName":         validation.Validate(ds.DisplayName, validation.Required, validation.Length(1, 48)),
		"dscp":                validation.Validate(ds.DSCP, validation.NotNil, validation.Min(0)),
		"geoLimit":            validation.Validate(ds.GeoLimit, validation.NotNil),
		"geoProvider":         validation.Validate(ds.GeoProvider, validation.NotNil),
		"logsEnabled":         validation.Validate(ds.LogsEnabled, validation.NotNil),
		"regionalGeoBlocking": validation.Validate(ds.RegionalGeoBlocking, validation.NotNil),
		"remapText":           validation.Validate(ds.RemapText, noLineBreaks),
		"routingName":         validation.Validate(ds.RoutingName, isDNSName, noPeriods, validation.Length(1, 48)),
		"typeId":              validation.Validate(ds.TypeID, validation.Required, validation.Min(1)),
		"xmlId":               validation.Validate(ds.XMLID, validation.Required, noSpaces, noPeriods, validation.Length(1, 48)),
	})
	if err := validateTopologyFields(ds); err != nil {
		errs = append(errs, err)
	}
	if err := validateTypeFields(tx, ds); err != nil {
		errs = append(errs, errors.New("type fields: "+err.Error()))
	}
	if len(errs) == 0 {
		return nil
	}
	return util.JoinErrs(errs)
}

func validateTopologyFields(ds *tc.DeliveryServiceV4) error {
	if ds.Topology != nil && (ds.EdgeHeaderRewrite != nil || ds.MidHeaderRewrite != nil) {
		return errors.New("cannot set edgeHeaderRewrite or midHeaderRewrite while a Topology is assigned. Use firstHeaderRewrite, innerHeaderRewrite, and/or lastHeaderRewrite instead")
	}
	if ds.Topology == nil && (ds.FirstHeaderRewrite != nil || ds.InnerHeaderRewrite != nil || ds.LastHeaderRewrite != nil) {
		return errors.New("cannot set firstHeaderRewrite, innerHeaderRewrite, or lastHeaderRewrite unless this delivery service is assigned to a Topology. Use edgeHeaderRewrite and/or midHeaderRewrite instead")
	}
	return nil
}

func parseOrgServerFQDN(orgServerFQDN string) (*string, *string, *string, error) {
	originRegex := regexp.MustCompile(`^(https?)://([^:]+)(:(\d+))?$`)
	matches := originRegex.FindStringSubmatch(orgServerFQDN)
	if len(matches) == 0 {
		return nil, nil, nil, fmt.Errorf("unable to parse invalid orgServerFqdn: '%s'", orgServerFQDN)
	}

	protocol := strings.ToLower(matches[1])
	FQDN := matches[2]

	if len(protocol) == 0 || len(FQDN) == 0 {
		return nil, nil, nil, fmt.Errorf("empty Origin protocol or FQDN parsed from '%s'", orgServerFQDN)
	}

	var port *string
	if len(matches[4]) != 0 {
		port = &matches[4]
	}
	return &protocol, &FQDN, port, nil
}

func validateOrgServerFQDN(orgServerFQDN string) bool {
	_, fqdn, port, err := parseOrgServerFQDN(orgServerFQDN)
	if err != nil || !govalidator.IsHost(*fqdn) || (port != nil && !govalidator.IsPort(*port)) {
		return false
	}
	return true
}

func validateTypeFields(tx *sql.Tx, ds *tc.DeliveryServiceV4) error {
	// Validate the TypeName related fields below
	err := error(nil)
	DNSRegexType := "^DNS.*$"
	HTTPRegexType := "^HTTP.*$"
	SteeringRegexType := "^STEERING.*$"
	latitudeErr := "Must be a floating point number within the range +-90"
	longitudeErr := "Must be a floating point number within the range +-180"

	typeName, err := tc.ValidateTypeID(tx, &ds.TypeID, "deliveryservice")
	if err != nil {
		return err
	}

	errs := validation.Errors{
		"consistentHashQueryParams": validation.Validate(ds,
			validation.By(func(dsi interface{}) error {
				ds := dsi.(*tc.DeliveryServiceV4)
				if len(ds.ConsistentHashQueryParams) == 0 || tc.DSType(typeName).IsHTTP() {
					return nil
				}
				return fmt.Errorf("consistentHashQueryParams not allowed for '%s' deliveryservice type", typeName)
			})),
		"initialDispersion": validation.Validate(ds.InitialDispersion,
			validation.By(requiredIfMatchesTypeName([]string{HTTPRegexType}, typeName)),
			validation.By(tovalidate.IsGreaterThanZero)),
		"ipv6RoutingEnabled": validation.Validate(&ds.IPV6RoutingEnabled,
			validation.By(requiredIfMatchesTypeName([]string{SteeringRegexType, DNSRegexType, HTTPRegexType}, typeName))),
		"missLat": validation.Validate(ds.MissLat,
			validation.By(requiredIfMatchesTypeName([]string{DNSRegexType, HTTPRegexType}, typeName)),
			validation.Min(-90.0).Error(latitudeErr),
			validation.Max(90.0).Error(latitudeErr)),
		"missLong": validation.Validate(ds.MissLong,
			validation.By(requiredIfMatchesTypeName([]string{DNSRegexType, HTTPRegexType}, typeName)),
			validation.Min(-180.0).Error(longitudeErr),
			validation.Max(180.0).Error(longitudeErr)),
		"multiSiteOrigin": validation.Validate(ds.MultiSiteOrigin,
			validation.By(requiredIfMatchesTypeName([]string{DNSRegexType, HTTPRegexType}, typeName))),
		"orgServerFqdn": validation.Validate(ds.OrgServerFQDN,
			validation.By(requiredIfMatchesTypeName([]string{DNSRegexType, HTTPRegexType}, typeName)),
			validation.NewStringRule(validateOrgServerFQDN, "must start with http:// or https:// and be followed by a valid hostname with an optional port (no trailing slash)")),
		"rangeSliceBlockSize": validation.Validate(ds,
			validation.By(func(dsi interface{}) error {
				ds := dsi.(*tc.DeliveryServiceV4)
				if ds.RangeRequestHandling != nil {
					if *ds.RangeRequestHandling == 3 {
						return validation.Validate(ds.RangeSliceBlockSize, validation.Required,
							// Per Slice Plugin implementation
							validation.Min(tc.MinRangeSliceBlockSize), // 256KiB
							validation.Max(tc.MaxRangeSliceBlockSize), // 32MiB
						)
					}
					if ds.RangeSliceBlockSize != nil {
						return errors.New("rangeSliceBlockSize can only be set if the rangeRequestHandling is set to 3 (Use the Slice Plugin)")
					}
				}
				return nil
			})),
		"protocol": validation.Validate(ds.Protocol,
			validation.By(requiredIfMatchesTypeName([]string{SteeringRegexType, DNSRegexType, HTTPRegexType}, typeName))),
		"qstringIgnore": validation.Validate(ds.QStringIgnore,
			validation.By(requiredIfMatchesTypeName([]string{DNSRegexType, HTTPRegexType}, typeName))),
		"rangeRequestHandling": validation.Validate(ds.RangeRequestHandling,
			validation.By(requiredIfMatchesTypeName([]string{DNSRegexType, HTTPRegexType}, typeName))),
		"topology": validation.Validate(ds,
			validation.By(func(dsi interface{}) error {
				ds := dsi.(*tc.DeliveryServiceV4)
				if ds.Topology != nil && tc.DSType(typeName).IsSteering() {
					return fmt.Errorf("steering deliveryservice types cannot be assigned to a topology")
				}
				return nil
			})),
		"maxRequestHeaderBytes": validation.Validate(ds,
			validation.By(func(dsi interface{}) error {
				ds := dsi.(*tc.DeliveryServiceV4)
				if ds.MaxRequestHeaderBytes == nil {
					return errors.New("maxRequestHeaderBytes empty, must be a valid positive value")
				}
				if *ds.MaxRequestHeaderBytes < 0 || *ds.MaxRequestHeaderBytes > 2147483647 {
					return errors.New("maxRequestHeaderBytes must be a valid non negative value between 0 and 2147483647")
				}
				return nil
			})),
	}
	toErrs := tovalidate.ToErrors(errs)
	if len(toErrs) > 0 {
		return errors.New(util.JoinErrsStr(toErrs))
	}
	return nil
}

func selectMaxLastUpdatedQuery(where string) string {
	return `SELECT max(t) from (
		SELECT max(ds.last_updated) as t from deliveryservice as ds
	JOIN type ON ds.type = type.id
	JOIN cdn ON ds.cdn_id = cdn.id
	LEFT JOIN profile ON ds.profile = profile.id
	LEFT JOIN tenant ON ds.tenant_id = tenant.id ` + where +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='deliveryservice') as res`
}

func getOldDetails(id int, tx *sql.Tx) (TODeliveryServiceOldDetails, error, error, int) {
	q := `
SELECT ds.routing_name, ds.ssl_key_version, cdn.name, cdn.id,
(SELECT o.protocol::text || '://' || o.fqdn || rtrim(concat(':', o.port::text), ':')
FROM origin o
WHERE o.deliveryservice = ds.id
AND o.is_primary) as org_server_fqdn
FROM  deliveryservice as ds
JOIN cdn ON ds.cdn_id = cdn.id
WHERE ds.id=$1
`
	var oldDetails TODeliveryServiceOldDetails
	if err := tx.QueryRow(q, id).Scan(&oldDetails.OldRoutingName, &oldDetails.OldSSLKeyVersion, &oldDetails.OldCdnName, &oldDetails.OldCdnId, &oldDetails.OldOrgServerFqdn); err != nil {
		if err == sql.ErrNoRows {
			return oldDetails, fmt.Errorf("querying delivery service %v host name: no such delivery service exists", id), nil, http.StatusNotFound
		}
		return oldDetails, nil, fmt.Errorf("querying delivery service %v host name: "+err.Error(), id), http.StatusInternalServerError
	}
	return oldDetails, nil, nil, http.StatusOK
}

func getTypeFromID(id int, tx *sql.Tx) (tc.DSType, error) {
	// TODO combine with getOldDetails, to only make one query?
	name := ""
	if err := tx.QueryRow(`SELECT name FROM type WHERE id = $1`, id).Scan(&name); err != nil {
		return "", fmt.Errorf("querying type ID %v: "+err.Error()+"\n", id)
	}
	return tc.DSTypeFromString(name), nil
}

func updatePrimaryOrigin(tx *sql.Tx, user *auth.CurrentUser, ds tc.DeliveryServiceV4) error {
	count := 0
	q := `SELECT count(*) FROM origin WHERE deliveryservice = $1 AND is_primary`
	if err := tx.QueryRow(q, *ds.ID).Scan(&count); err != nil {
		return fmt.Errorf("querying existing primary origin for ds %s: %w", ds.XMLID, err)
	}

	if ds.OrgServerFQDN == nil || *ds.OrgServerFQDN == "" {
		if count == 1 {
			// the update is removing the existing orgServerFQDN, so the existing row needs to be deleted
			q = `DELETE FROM origin WHERE deliveryservice = $1 AND is_primary`
			if _, err := tx.Exec(q, *ds.ID); err != nil {
				return fmt.Errorf("deleting primary origin for ds %s: %w", ds.XMLID, err)
			}
			api.CreateChangeLogRawTx(api.ApiChange, "DS: "+ds.XMLID+", ID: "+strconv.Itoa(*ds.ID)+", ACTION: Deleted primary origin", user, tx)
		}
		return nil
	}

	if count == 0 {
		// orgServerFQDN is going from null to not null, so the primary origin needs to be created
		return createPrimaryOrigin(tx, user, ds)
	}

	protocol, fqdn, port, err := parseOrgServerFQDN(*ds.OrgServerFQDN)
	if err != nil {
		return fmt.Errorf("updating primary origin: %v", err)
	}

	name := ""
	q = `UPDATE origin SET protocol = $1, fqdn = $2, port = $3 WHERE is_primary AND deliveryservice = $4 RETURNING name`
	if err := tx.QueryRow(q, protocol, fqdn, port, *ds.ID).Scan(&name); err != nil {
		return fmt.Errorf("update primary origin for ds %s from '%s': %w", ds.XMLID, *ds.OrgServerFQDN, err)
	}

	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+ds.XMLID+", ID: "+strconv.Itoa(*ds.ID)+", ACTION: Updated primary origin: "+name, user, tx)

	return nil
}

func createPrimaryOrigin(tx *sql.Tx, user *auth.CurrentUser, ds tc.DeliveryServiceV4) error {
	if ds.OrgServerFQDN == nil {
		return nil
	}

	protocol, fqdn, port, err := parseOrgServerFQDN(*ds.OrgServerFQDN)
	if err != nil {
		return fmt.Errorf("creating primary origin: %v", err)
	}

	originID := 0
	q := `INSERT INTO origin (name, fqdn, protocol, is_primary, port, deliveryservice, tenant) VALUES ($1, $2, $3, TRUE, $4, $5, $6) RETURNING id`
	if err := tx.QueryRow(q, ds.XMLID, fqdn, protocol, port, ds.ID, ds.TenantID).Scan(&originID); err != nil {
		return fmt.Errorf("insert origin from '%s': %s", *ds.OrgServerFQDN, err.Error())
	}

	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+ds.XMLID+", ID: "+strconv.Itoa(*ds.ID)+", ACTION: Created primary origin id: "+strconv.Itoa(originID), user, tx)

	return nil
}

func getDSType(tx *sql.Tx, xmlid string) (tc.DSType, bool, error) {
	name := ""
	if err := tx.QueryRow(`SELECT name FROM type WHERE id = (select type from deliveryservice where xml_id = $1)`, xmlid).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("querying deliveryservice type name: " + err.Error())
	}
	return tc.DSTypeFromString(name), true, nil
}

func GetDeliveryServices(query string, queryValues map[string]interface{}, tx *sqlx.Tx) ([]tc.DeliveryServiceV4, error, error, int) {
	rows, err := tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, nil, fmt.Errorf("querying: %v", err), http.StatusInternalServerError
	}
	defer rows.Close()

	dses := []tc.DeliveryServiceV4{}
	dsCDNDomains := map[string]string{}

	// ensure json generated from this slice won't come out as `null` if empty
	dsQueryParams := []string{}

	for rows.Next() {
		ds := tc.DeliveryServiceV4{}
		cdnDomain := ""
		err := rows.Scan(&ds.Active,
			&ds.AnonymousBlockingEnabled,
			&ds.CCRDNSTTL,
			&ds.CDNID,
			&ds.CDNName,
			&ds.CheckPath,
			&ds.ConsistentHashRegex,
			&ds.DeepCachingType,
			&ds.DisplayName,
			&ds.DNSBypassCNAME,
			&ds.DNSBypassIP,
			&ds.DNSBypassIP6,
			&ds.DNSBypassTTL,
			&ds.DSCP,
			&ds.EcsEnabled,
			&ds.EdgeHeaderRewrite,
			&ds.FirstHeaderRewrite,
			&ds.GeoLimitRedirectURL,
			&ds.GeoLimit,
			&ds.GeoLimitCountries,
			&ds.GeoProvider,
			&ds.GlobalMaxMBPS,
			&ds.GlobalMaxTPS,
			&ds.FQPacingRate,
			&ds.HTTPBypassFQDN,
			&ds.ID,
			&ds.InfoURL,
			&ds.InitialDispersion,
			&ds.InnerHeaderRewrite,
			&ds.IPV6RoutingEnabled,
			&ds.LastHeaderRewrite,
			&ds.LastUpdated,
			&ds.LogsEnabled,
			&ds.LongDesc,
			&ds.LongDesc1,
			&ds.LongDesc2,
			&ds.MaxDNSAnswers,
			&ds.MaxOriginConnections,
			&ds.MaxRequestHeaderBytes,
			&ds.MidHeaderRewrite,
			&ds.MissLat,
			&ds.MissLong,
			&ds.MultiSiteOrigin,
			&ds.OrgServerFQDN,
			&ds.OriginShield,
			&ds.ProfileID,
			&ds.ProfileName,
			&ds.ProfileDesc,
			&ds.Protocol,
			&ds.QStringIgnore,
			pq.Array(&dsQueryParams),
			&ds.RangeRequestHandling,
			&ds.RegexRemap,
			&ds.RegionalGeoBlocking,
			&ds.RemapText,
			&ds.RoutingName,
			&ds.ServiceCategory,
			&ds.SigningAlgorithm,
			&ds.RangeSliceBlockSize,
			&ds.SSLKeyVersion,
			&ds.TenantID,
			&ds.Tenant,
			&ds.Topology,
			&ds.TRRequestHeaders,
			&ds.TRResponseHeaders,
			&ds.Type,
			&ds.TypeID,
			&ds.XMLID,
			&cdnDomain)

		if err != nil {
			return nil, nil, fmt.Errorf("getting delivery services: %v", err), http.StatusInternalServerError
		}

		ds.ConsistentHashQueryParams = []string{}
		if len(dsQueryParams) >= 0 {
			// ensure unique and in consistent order
			m := make(map[string]struct{}, len(dsQueryParams))
			for _, k := range dsQueryParams {
				if _, exists := m[k]; exists {
					continue
				}
				m[k] = struct{}{}
				ds.ConsistentHashQueryParams = append(ds.ConsistentHashQueryParams, k)
			}
		}

		dsCDNDomains[ds.XMLID] = cdnDomain
		// TODO: why?
		ds.DeepCachingType = tc.DeepCachingTypeFromString(string(ds.DeepCachingType))

		ds.Signed = ds.SigningAlgorithm != nil && *ds.SigningAlgorithm == tc.SigningAlgorithmURLSig

		dses = append(dses, ds)
	}

	dsNames := make([]string, len(dses), len(dses))
	for i, ds := range dses {
		dsNames[i] = ds.XMLID
	}

	matchLists, err := GetDeliveryServicesMatchLists(dsNames, tx.Tx)
	if err != nil {
		return nil, nil, errors.New("getting delivery service matchlists: " + err.Error()), http.StatusInternalServerError
	}
	for i, ds := range dses {
		matchList, ok := matchLists[ds.XMLID]
		if !ok {
			continue
		}
		ds.MatchList = matchList
		ds.ExampleURLs = MakeExampleURLs(ds.Protocol, *ds.Type, ds.RoutingName, ds.MatchList, dsCDNDomains[ds.XMLID])
		dses[i] = ds
	}

	return dses, nil, nil, http.StatusOK
}

func getCDNDomain(dsID int, tx *sql.Tx) (string, error) {
	q := `SELECT cdn.domain_name from cdn where cdn.id = (SELECT ds.cdn_id from deliveryservice as ds where ds.id = $1)`
	cdnDomain := ""
	if err := tx.QueryRow(q, dsID).Scan(&cdnDomain); err != nil {
		return "", fmt.Errorf("getting CDN domain for delivery service '%v': "+err.Error(), dsID)
	}
	return cdnDomain, nil
}

func getCDNNameDomainDNSSecEnabled(dsID int, tx *sql.Tx) (string, string, bool, error) {
	q := `SELECT cdn.name, cdn.domain_name, cdn.dnssec_enabled from cdn where cdn.id = (SELECT ds.cdn_id from deliveryservice as ds where ds.id = $1)`
	cdnName := ""
	cdnDomain := ""
	dnssecEnabled := false
	if err := tx.QueryRow(q, dsID).Scan(&cdnName, &cdnDomain, &dnssecEnabled); err != nil {
		return "", "", false, fmt.Errorf("getting dnssec_enabled for delivery service '%v': "+err.Error(), dsID)
	}
	return cdnName, cdnDomain, dnssecEnabled, nil
}

// makeExampleURLs creates the example URLs for a delivery service. The dsProtocol may be nil, if the delivery service type doesn't have a protocol (e.g. ANY_MAP).
func MakeExampleURLs(protocol *int, dsType tc.DSType, routingName string, matchList []tc.DeliveryServiceMatch, cdnDomain string) []string {
	examples := []string{}
	scheme := ""
	scheme2 := ""
	if protocol != nil {
		switch *protocol {
		case 0:
			scheme = "http"
		case 1:
			scheme = "https"
		case 2:
			fallthrough
		case 3:
			scheme = "http"
			scheme2 = "https"
		default:
			scheme = "http"
		}
	} else {
		scheme = "http"
	}
	dsIsDNS := dsType.IsDNS()
	regexReplacer := strings.NewReplacer(`\`, ``, `.*`, ``, `.`, ``)
	for _, match := range matchList {
		if dsIsDNS || match.Type == tc.DSMatchTypeHostRegex {
			host := regexReplacer.Replace(match.Pattern)
			if match.SetNumber == 0 {
				examples = append(examples, scheme+`://`+routingName+`.`+host+`.`+cdnDomain)
				if scheme2 != "" {
					examples = append(examples, scheme2+`://`+routingName+`.`+host+`.`+cdnDomain)
				}
				continue
			}
			examples = append(examples, scheme+`://`+match.Pattern)
			if scheme2 != "" {
				examples = append(examples, scheme2+`://`+match.Pattern)
			}
		} else if match.Type == tc.DSMatchTypePathRegex {
			examples = append(examples, match.Pattern)
		}
	}
	return examples
}

func GetDeliveryServicesMatchLists(dses []string, tx *sql.Tx) (map[string][]tc.DeliveryServiceMatch, error) {
	// TODO move somewhere generic
	q := `
SELECT ds.xml_id as ds_name, t.name as type, r.pattern, COALESCE(dsr.set_number, 0)
FROM regex as r
JOIN deliveryservice_regex as dsr ON dsr.regex = r.id
JOIN deliveryservice as ds on ds.id = dsr.deliveryservice
JOIN type as t ON r.type = t.id
WHERE ds.xml_id = ANY($1)
ORDER BY dsr.set_number
`
	rows, err := tx.Query(q, pq.Array(dses))
	if err != nil {
		return nil, errors.New("getting delivery service regexes: " + err.Error())
	}
	defer rows.Close()

	matches := map[string][]tc.DeliveryServiceMatch{}
	for rows.Next() {
		m := tc.DeliveryServiceMatch{}
		dsName := ""
		matchTypeStr := ""
		if err := rows.Scan(&dsName, &matchTypeStr, &m.Pattern, &m.SetNumber); err != nil {
			return nil, errors.New("scanning delivery service regexes: " + err.Error())
		}
		matchType := tc.DSMatchTypeFromString(matchTypeStr)
		if matchType == tc.DSMatchTypeInvalid {
			return nil, errors.New("getting delivery service regexes: got invalid delivery service match type '" + matchTypeStr + "'")
		}
		m.Type = matchType
		matches[dsName] = append(matches[dsName], m)
	}
	return matches, nil
}

// EnsureParams ensures the given delivery service's necessary parameters exist on profiles of servers assigned to the delivery service.
// Note the edgeHeaderRewrite, midHeaderRewrite, regexRemap may be nil, if the delivery service does not have those values.
func EnsureParams(tx *sql.Tx, dsID int, xmlID string, edgeHeaderRewrite *string, midHeaderRewrite *string, regexRemap *string, signingAlgorithm *string, dsType tc.DSType, maxOriginConns *int) error {
	if err := ensureHeaderRewriteParams(tx, dsID, xmlID, edgeHeaderRewrite, edgeTier, dsType, maxOriginConns); err != nil {
		return errors.New("creating edge header rewrite parameters: " + err.Error())
	}
	if err := ensureHeaderRewriteParams(tx, dsID, xmlID, midHeaderRewrite, midTier, dsType, maxOriginConns); err != nil {
		return errors.New("creating mid header rewrite parameters: " + err.Error())
	}
	if err := ensureRegexRemapParams(tx, dsID, xmlID, regexRemap); err != nil {
		return errors.New("creating mid regex remap parameters: " + err.Error())
	}
	if err := ensureURLSigParams(tx, dsID, xmlID, signingAlgorithm); err != nil {
		return errors.New("creating urlsig parameters: " + err.Error())
	}
	return nil
}

// EnsureCacheURLParams ensures the given delivery service's cachrurl parameters exist on profiles of servers assigned to the delivery service.
func EnsureCacheURLParams(tx *sql.Tx, dsID int, xmlID string, cacheURL *string) error {
	configFile := "cacheurl_" + xmlID + ".config"
	if cacheURL == nil || *cacheURL == "" {
		return deleteLocationParam(tx, configFile)
	}
	locationParamID, err := ensureLocation(tx, configFile)
	if err != nil {
		return err
	}
	return createDSLocationProfileParams(tx, locationParamID, dsID)
}

func ensureHeaderRewriteParams(tx *sql.Tx, dsID int, xmlID string, hdrRW *string, tier tierType, dsType tc.DSType, maxOriginConns *int) error {
	configFile := "hdr_rw_" + xmlID + ".config"
	if tier == midTier {
		configFile = "hdr_rw_mid_" + xmlID + ".config"
	}

	if tier == midTier && dsType.IsLive() && !dsType.IsNational() {
		// live local DSes don't get header rewrite rules on the mid so cleanup any location params related to mids
		return deleteLocationParam(tx, configFile)
	}

	hasMaxOriginConns := *maxOriginConns > 0 && ((tier == midTier) == dsType.UsesMidCache())
	if (hdrRW == nil || *hdrRW == "") && !hasMaxOriginConns {
		return deleteLocationParam(tx, configFile)
	}
	locationParamID, err := ensureLocation(tx, configFile)
	if err != nil {
		return err
	}
	if tier != midTier {
		return createDSLocationProfileParams(tx, locationParamID, dsID)
	}
	profileParameterQuery := `
INSERT INTO profile_parameter (profile, parameter)
SELECT DISTINCT(profile), $1::bigint FROM server
WHERE server.type IN (SELECT id from type where type.name like 'MID%' and type.use_in_table = 'server')
AND server.cdn_id = (select cdn_id from deliveryservice where id = $2)
ON CONFLICT DO NOTHING
`
	if _, err := tx.Exec(profileParameterQuery, locationParamID, dsID); err != nil {
		return fmt.Errorf("parameter query to insert profile_parameters query '"+profileParameterQuery+"' location parameter ID '%v' delivery service ID '%v': %v", locationParamID, dsID, err)
	}
	return nil
}

func ensureURLSigParams(tx *sql.Tx, dsID int, xmlID string, signingAlgorithm *string) error {
	configFile := "url_sig_" + xmlID + ".config"
	if signingAlgorithm == nil || *signingAlgorithm != tc.SigningAlgorithmURLSig {
		return deleteLocationParam(tx, configFile)
	}
	locationParamID, err := ensureLocation(tx, configFile)
	if err != nil {
		return err
	}
	return createDSLocationProfileParams(tx, locationParamID, dsID)
}

func ensureRegexRemapParams(tx *sql.Tx, dsID int, xmlID string, regexRemap *string) error {
	configFile := "regex_remap_" + xmlID + ".config"
	if regexRemap == nil || *regexRemap == "" {
		return deleteLocationParam(tx, configFile)
	}
	locationParamID, err := ensureLocation(tx, configFile)
	if err != nil {
		return err
	}
	return createDSLocationProfileParams(tx, locationParamID, dsID)
}

// createDSLocationProfileParams adds the given parameter to all profiles assigned to servers which are assigned to the given delivery service.
func createDSLocationProfileParams(tx *sql.Tx, locationParamID int, deliveryServiceID int) error {
	profileParameterQuery := `
INSERT INTO profile_parameter (profile, parameter)
SELECT DISTINCT(profile), $1::bigint FROM server
WHERE server.id IN (SELECT server from deliveryservice_server where deliveryservice = $2)
ON CONFLICT DO NOTHING
`
	if _, err := tx.Exec(profileParameterQuery, locationParamID, deliveryServiceID); err != nil {
		return errors.New("inserting profile_parameters: " + err.Error())
	}
	return nil
}

// ensureLocation ensures a location parameter exists for the given config file. If not, it creates one, with the same value as the 'remap.config' file parameter. Returns the ID of the location parameter.
func ensureLocation(tx *sql.Tx, configFile string) (int, error) {
	atsConfigLocation := ""
	if err := tx.QueryRow(`SELECT value FROM parameter WHERE name = 'location' AND config_file = 'remap.config'`).Scan(&atsConfigLocation); err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("executing parameter query for ATS config location: parameter missing (do you have a name=location config_file=remap.config parameter?")
		}
		return 0, errors.New("executing parameter query for ATS config location: " + err.Error())
	}
	atsConfigLocation = strings.TrimRight(atsConfigLocation, `/`)

	locationParamID := 0
	existingLocationErr := tx.QueryRow(`SELECT id FROM parameter WHERE name = 'location' AND config_file = $1`, configFile).Scan(&locationParamID)
	if existingLocationErr != nil && existingLocationErr != sql.ErrNoRows {
		return 0, errors.New("executing parameter query for existing location: " + existingLocationErr.Error())
	}

	if existingLocationErr == sql.ErrNoRows {
		resultRows, err := tx.Query(`INSERT INTO parameter (config_file, name, value) VALUES ($1, 'location', $2) RETURNING id`, configFile, atsConfigLocation)
		if err != nil {
			return 0, errors.New("executing parameter query to insert location: " + err.Error())
		}
		defer resultRows.Close()
		if !resultRows.Next() {
			return 0, errors.New("parameter query to insert location didn't return id")
		}
		if err := resultRows.Scan(&locationParamID); err != nil {
			return 0, errors.New("parameter query to insert location returned id scan: " + err.Error())
		}
		if resultRows.Next() {
			return 0, errors.New("parameter query to insert location returned too many rows (>1)")
		}
	}
	return locationParamID, nil
}

func deleteLocationParam(tx *sql.Tx, configFile string) error {
	id := 0
	err := tx.QueryRow(`DELETE FROM parameter WHERE name = 'location' AND config_file = $1 RETURNING id`, configFile).Scan(&id)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		log.Errorln("deleting name=location config_file=" + configFile + " parameter: " + err.Error())
		return errors.New("executing parameter delete: " + err.Error())
	}
	if _, err := tx.Exec(`DELETE FROM profile_parameter WHERE parameter = $1`, id); err != nil {
		log.Errorf("deleting parameter name=location config_file=%v id=%v profile_parameter: %v", configFile, id, err)
		return errors.New("executing parameter profile_parameter delete: " + err.Error())
	}
	return nil
}

// getTenantID returns the tenant Id of the given delivery service. Note it may return a nil id and nil error, if the tenant ID in the database is nil.
func getTenantID(tx *sql.Tx, ds *tc.DeliveryServiceV4) (*int, error) {
	if ds.ID != nil {
		existingID, _, err := getDSTenantIDByID(tx, *ds.ID) // ignore exists return - if the DS is new, we only need to check the user input tenant
		return existingID, err
	}
	existingID, _, err := getDSTenantIDByName(tx, tc.DeliveryServiceName(ds.XMLID)) // ignore exists return - if the DS is new, we only need to check the user input tenant
	return existingID, err
}

func isTenantAuthorized(inf *api.APIInfo, ds *tc.DeliveryServiceV4) (bool, error) {
	tx := inf.Tx.Tx
	user := inf.User

	existingID, err := getTenantID(inf.Tx.Tx, ds)
	if err != nil {
		return false, errors.New("getting tenant ID: " + err.Error())
	}
	if existingID != nil {
		ds.TenantID = *existingID
		userAuthorizedForExistingDSTenant, err := tenant.IsResourceAuthorizedToUserTx(*existingID, user, tx)
		if err != nil {
			return false, errors.New("checking authorization for existing DS ID: " + err.Error())
		}
		if !userAuthorizedForExistingDSTenant {
			return false, nil
		}
	}
	if ds.TenantID != 0 {
		userAuthorizedForNewDSTenant, err := tenant.IsResourceAuthorizedToUserTx(ds.TenantID, user, tx)
		if err != nil {
			return false, errors.New("checking authorization for new DS ID: " + err.Error())
		}
		if !userAuthorizedForNewDSTenant {
			return false, nil
		}
	}
	return true, nil
}

// getDSTenantIDByID returns the tenant ID, whether the delivery service exists, and any error.
func getDSTenantIDByID(tx *sql.Tx, id int) (*int, bool, error) {
	tenantID := (*int)(nil)
	if err := tx.QueryRow(`SELECT tenant_id FROM deliveryservice where id = $1`, id).Scan(&tenantID); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("querying tenant ID for delivery service ID '%v': %v", id, err)
	}
	return tenantID, true, nil
}

// getDSTenantIDByName returns the tenant ID, whether the delivery service exists, and any error.
func getDSTenantIDByName(tx *sql.Tx, ds tc.DeliveryServiceName) (*int, bool, error) {
	tenantID := (*int)(nil)
	if err := tx.QueryRow(`SELECT tenant_id FROM deliveryservice where xml_id = $1`, ds).Scan(&tenantID); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("querying tenant ID for delivery service name '%v': %v", ds, err)
	}
	return tenantID, true, nil
}

// GetXMLID loads the DeliveryService's xml_id from the database, from the ID. Returns whether the delivery service was found, and any error.
func GetXMLID(tx *sql.Tx, id int) (string, bool, error) {
	xmlID := ""
	if err := tx.QueryRow(`SELECT xml_id FROM deliveryservice where id = $1`, id).Scan(&xmlID); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("querying xml_id for delivery service ID '%v': %v", id, err)
	}
	return xmlID, true, nil
}

// getSSLVersion reports a boolean value, confirming whether DS has a SSL version or not
func getSSLVersion(xmlId string, tx *sql.Tx) (bool, error) {
	var exists bool
	row := tx.QueryRow(`SELECT EXISTS(SELECT * FROM deliveryservice WHERE xml_id = $1 AND ssl_key_version>=1)`, xmlId)
	err := row.Scan(&exists)
	return exists, err
}

func setNilIfEmpty(ptrs ...**string) {
	for _, s := range ptrs {
		if *s != nil && strings.TrimSpace(**s) == "" {
			*s = nil
		}
	}
}

func sanitize(ds *tc.DeliveryServiceV4) {
	if ds.GeoLimitCountries != nil {
		*ds.GeoLimitCountries = strings.ToUpper(strings.Replace(*ds.GeoLimitCountries, " ", "", -1))
	}
	if ds.ProfileID != nil && *ds.ProfileID == -1 {
		ds.ProfileID = nil
	}
	setNilIfEmpty(
		&ds.EdgeHeaderRewrite,
		&ds.MidHeaderRewrite,
		&ds.FirstHeaderRewrite,
		&ds.InnerHeaderRewrite,
		&ds.LastHeaderRewrite,
	)
	if ds.RoutingName == "" {
		ds.RoutingName = tc.DefaultRoutingName
	}
	signedAlgorithm := tc.SigningAlgorithmURLSig
	if ds.Signed && (ds.SigningAlgorithm == nil || *ds.SigningAlgorithm == "") {
		ds.SigningAlgorithm = &signedAlgorithm
	}
	if !ds.Signed && ds.SigningAlgorithm != nil && *ds.SigningAlgorithm == signedAlgorithm {
		ds.Signed = true
	}
	if ds.MaxOriginConnections == nil || *ds.MaxOriginConnections < 0 {
		ds.MaxOriginConnections = util.IntPtr(0)
	}
	ds.DeepCachingType = tc.DeepCachingTypeFromString(string(ds.DeepCachingType))
	if ds.MaxRequestHeaderBytes == nil {
		ds.MaxRequestHeaderBytes = util.IntPtr(tc.DefaultMaxRequestHeaderBytes)
	}
}

// SelectDeliveryServicesQuery is a PostgreSQL query used to fetch Delivery
// Services from the Traffic Ops Database.
const SelectDeliveryServicesQuery = `
SELECT
ds.active,
	ds.anonymous_blocking_enabled,
	ds.ccr_dns_ttl,
	ds.cdn_id,
	cdn.name AS cdnName,
	ds.check_path,
	ds.consistent_hash_regex,
	CAST(ds.deep_caching_type AS text) AS deep_caching_type,
	ds.display_name,
	ds.dns_bypass_cname,
	ds.dns_bypass_ip,
	ds.dns_bypass_ip6,
	ds.dns_bypass_ttl,
	ds.dscp,
	ds.ecs_enabled,
	ds.edge_header_rewrite,
	ds.first_header_rewrite,
	ds.geolimit_redirect_url,
	ds.geo_limit,
	ds.geo_limit_countries,
	ds.geo_provider,
	ds.global_max_mbps,
	ds.global_max_tps,
	ds.fq_pacing_rate,
	ds.http_bypass_fqdn,
	ds.id,
	ds.info_url,
	ds.initial_dispersion,
	ds.inner_header_rewrite,
	ds.ipv6_routing_enabled,
	ds.last_header_rewrite,
	ds.last_updated,
	ds.logs_enabled,
	ds.long_desc,
	ds.long_desc_1,
	ds.long_desc_2,
	ds.max_dns_answers,
	ds.max_origin_connections,
	ds.max_request_header_bytes,
	ds.mid_header_rewrite,
	COALESCE(ds.miss_lat, 0.0),
	COALESCE(ds.miss_long, 0.0),
	ds.multi_site_origin,
	(SELECT o.protocol::::text || ':://' || o.fqdn || rtrim(concat('::', o.port::::text), '::')
		FROM origin o
		WHERE o.deliveryservice = ds.id
		AND o.is_primary) AS org_server_fqdn,
	ds.origin_shield,
	ds.profile AS profileID,
	profile.name AS profile_name,
	profile.description  AS profile_description,
	ds.protocol,
	ds.qstring_ignore,
	(SELECT ARRAY_AGG(name ORDER BY name)
		FROM deliveryservice_consistent_hash_query_param
				WHERE deliveryservice_id = ds.id) AS query_keys,
	ds.range_request_handling,
	ds.regex_remap,
	ds.regional_geo_blocking,
	ds.remap_text,
	ds.routing_name,
	ds.service_category,
	ds.signing_algorithm,
	ds.range_slice_block_size,
	ds.ssl_key_version,
	ds.tenant_id,
	tenant.name,
	ds.topology,
	ds.tr_request_headers,
	ds.tr_response_headers,
	type.name,
	ds.type AS type_id,
	ds.xml_id,
	cdn.domain_name AS cdn_domain
FROM deliveryservice AS ds
JOIN type ON ds.type = type.id
JOIN cdn ON ds.cdn_id = cdn.id
LEFT JOIN profile ON ds.profile = profile.id
LEFT JOIN tenant ON ds.tenant_id = tenant.id
`

func updateDSQuery() string {
	return `
UPDATE
deliveryservice SET
active=$1,
ccr_dns_ttl=$2,
cdn_id=$3,
check_path=$4,
deep_caching_type=$5,
display_name=$6,
dns_bypass_cname=$7,
dns_bypass_ip=$8,
dns_bypass_ip6=$9,
dns_bypass_ttl=$10,
dscp=$11,
edge_header_rewrite=$12,
geolimit_redirect_url=$13,
geo_limit=$14,
geo_limit_countries=$15,
geo_provider=$16,
global_max_mbps=$17,
global_max_tps=$18,
fq_pacing_rate=$19,
http_bypass_fqdn=$20,
info_url=$21,
initial_dispersion=$22,
ipv6_routing_enabled=$23,
logs_enabled=$24,
long_desc=$25,
long_desc_1=$26,
long_desc_2=$27,
max_dns_answers=$28,
mid_header_rewrite=$29,
miss_lat=$30,
miss_long=$31,
multi_site_origin=$32,
origin_shield=$33,
profile=$34,
protocol=$35,
qstring_ignore=$36,
range_request_handling=$37,
regex_remap=$38,
regional_geo_blocking=$39,
remap_text=$40,
routing_name=$41,
signing_algorithm=$42,
ssl_key_version=$43,
tenant_id=$44,
tr_request_headers=$45,
tr_response_headers=$46,
type=$47,
xml_id=$48,
anonymous_blocking_enabled=$49,
consistent_hash_regex=$50,
max_origin_connections=$51,
ecs_enabled=$52,
range_slice_block_size=$53,
topology=$54,
first_header_rewrite=$55,
inner_header_rewrite=$56,
last_header_rewrite=$57,
service_category=$58,
max_request_header_bytes=$59
WHERE id=$60
RETURNING last_updated
`
}

func insertQuery() string {
	return `
INSERT INTO deliveryservice (
active,
anonymous_blocking_enabled,
ccr_dns_ttl,
cdn_id,
check_path,
consistent_hash_regex,
deep_caching_type,
display_name,
dns_bypass_cname,
dns_bypass_ip,
dns_bypass_ip6,
dns_bypass_ttl,
dscp,
edge_header_rewrite,
geolimit_redirect_url,
geo_limit,
geo_limit_countries,
geo_provider,
global_max_mbps,
global_max_tps,
fq_pacing_rate,
http_bypass_fqdn,
info_url,
initial_dispersion,
ipv6_routing_enabled,
logs_enabled,
long_desc,
long_desc_1,
long_desc_2,
max_dns_answers,
max_origin_connections,
mid_header_rewrite,
miss_lat,
miss_long,
multi_site_origin,
origin_shield,
profile,
protocol,
qstring_ignore,
range_request_handling,
regex_remap,
regional_geo_blocking,
remap_text,
routing_name,
signing_algorithm,
ssl_key_version,
tenant_id,
topology,
tr_request_headers,
tr_response_headers,
type,
xml_id,
ecs_enabled,
range_slice_block_size,
first_header_rewrite,
inner_header_rewrite,
last_header_rewrite,
service_category,
max_request_header_bytes
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52,$53,$54,$55,$56,$57,$58,$59)
RETURNING id, last_updated
`
}
