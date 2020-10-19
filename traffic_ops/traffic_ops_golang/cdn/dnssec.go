package cdn

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
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/riaksvc"
)

const CDNDNSSECKeyType = "dnssec"
const DNSSECStatusExisting = "existing"

func CreateDNSSECKeys(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, nil, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	req := tc.CDNDNSSECGenerateReq{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &req); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("parsing request: "+err.Error()), nil)
		return
	}
	if req.EffectiveDateUnix == nil {
		now := tc.CDNDNSSECGenerateReqDate(time.Now().Unix())
		req.EffectiveDateUnix = &now
	}
	cdnName := *req.Key

	cdnID, ok, err := getCDNIDFromName(inf.Tx.Tx, tc.CDNName(cdnName))
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting cdn ID from name '"+cdnName+"': "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
		return
	}

	cdnDomain, cdnExists, err := dbhelpers.GetCDNDomainFromName(inf.Tx.Tx, tc.CDNName(cdnName))
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("create DNSSEC keys: getting CDN domain: "+err.Error()))
		return
	} else if !cdnExists {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("cdn '"+cdnName+"' not found"), nil)
		return
	}

	if err := generateStoreDNSSECKeys(inf.Tx.Tx, inf.Config, cdnName, cdnDomain, uint64(*req.TTL), uint64(*req.KSKExpirationDays), uint64(*req.ZSKExpirationDays), int64(*req.EffectiveDateUnix)); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("generating and storing DNSSEC CDN keys: "+err.Error()))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "CDN: "+string(cdnName)+", ID: "+strconv.Itoa(cdnID)+", ACTION: Generated DNSSEC keys", inf.User, inf.Tx.Tx)
	api.WriteResp(w, r, "Successfully created dnssec keys for "+cdnName)
}

// DefaultDSTTL is the default DS Record TTL to use, if no CDN Snapshot exists, or if no tld.ttls.DS parameter exists.
// This MUST be the same value as Traffic Router's default. Currently:
// traffic_router/core/src/main/java/com/comcast/cdn/traffic_control/traffic_router/core/dns/SignatureManager.java:476
// `final Long dsTtl = ZoneUtils.getLong(config.get("ttls"), "DS", 60);`.
// If Traffic Router and Traffic Ops differ, and a user is using the default, errors may occur!
// Users are advised to set the tld.ttls.DS CRConfig.json Parameter, so the default is not used!
// Traffic Ops functions SHOULD warn whenever this default is used.
const DefaultDSTTL = 60 * time.Second

func GetDNSSECKeys(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, []string{"name"}, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	cdnName := inf.Params["name"]

	riakKeys, keysExist, err := riaksvc.GetDNSSECKeys(cdnName, inf.Tx.Tx, inf.Config.RiakAuthOptions, inf.Config.RiakPort)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting DNSSEC CDN keys: "+err.Error()))
		return
	}
	if !keysExist {
		// TODO emulates Perl; change to error, 404?
		api.WriteRespAlertObj(w, r, tc.SuccessLevel, " - Dnssec keys for "+cdnName+" could not be found. ", struct{}{}) // emulates Perl
		return
	}

	dsTTL, err := GetDSRecordTTL(inf.Tx.Tx, cdnName)
	if err != nil {
		log.Errorln("Getting DNSSEC Keys: getting DS Record TTL from CRConfig Snapshot: " + err.Error())
		log.Errorf("Getting DNSSEC Keys: getting DS Record TTL failed, using default %v. It is STRONGLY ADVISED to fix the error, and ensure a CRConfig Snapshot exists for the CDN, and a tld.ttls.DS CRConfig.json Parameter exists on a Router Profile on the CDN. Default DS Records may cause unexpected behavior or errors!\n", DefaultDSTTL)
		dsTTL = DefaultDSTTL
	}

	keys, err := deliveryservice.MakeDNSSECKeysFromRiakKeys(riakKeys, dsTTL)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("creating DNSSEC keys object from Riak keys: "+err.Error()))
		return
	}
	api.WriteResp(w, r, keys)
}

func GetDNSSECKeysV11(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, []string{"name"}, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	cdnName := inf.Params["name"]
	riakKeys, keysExist, err := riaksvc.GetDNSSECKeys(cdnName, inf.Tx.Tx, inf.Config.RiakAuthOptions, inf.Config.RiakPort)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting DNSSEC CDN keys: "+err.Error()))
		return
	}
	if !keysExist {
		// TODO emulates Perl; change to error, 404?
		api.WriteRespAlertObj(w, r, tc.SuccessLevel, " - Dnssec keys for "+cdnName+" could not be found. ", struct{}{}) // emulates Perl
		return
	}
	api.WriteResp(w, r, riakKeys)
}

func GetDSRecordTTL(tx *sql.Tx, cdn string) (time.Duration, error) {
	ttlSeconds := 0
	if err := tx.QueryRow(`SELECT JSON_EXTRACT_PATH_TEXT(crconfig, 'config', 'ttls', 'DS') FROM snapshot WHERE cdn = $1`, cdn).Scan(&ttlSeconds); err != nil {
		return 0, errors.New("getting cdn '" + cdn + "' DS Record TTL from CRConfig: " + err.Error())
	}
	return time.Duration(ttlSeconds) * time.Second, nil
}

func generateStoreDNSSECKeys(
	tx *sql.Tx,
	cfg *config.Config,
	cdnName string,
	cdnDomain string,
	ttlSeconds uint64,
	kExpDays uint64,
	zExpDays uint64,
	effectiveDateUnix int64,
) error {

	zExp := time.Duration(zExpDays) * time.Hour * 24
	kExp := time.Duration(kExpDays) * time.Hour * 24
	ttl := time.Duration(ttlSeconds) * time.Second

	oldKeys, oldKeysExist, err := riaksvc.GetDNSSECKeys(cdnName, tx, cfg.RiakAuthOptions, cfg.RiakPort)
	if err != nil {
		return errors.New("getting old dnssec keys: " + err.Error())
	}

	dses, err := GetCDNDeliveryServices(tx, cdnName)
	if err != nil {
		return errors.New("getting cdn delivery services: " + err.Error())
	}

	cdnDNSDomain := cdnDomain
	if !strings.HasSuffix(cdnDNSDomain, ".") {
		cdnDNSDomain = cdnDNSDomain + "."
	}
	cdnDNSDomain = strings.ToLower(cdnDNSDomain)

	inception := time.Now()
	newCDNZSK, err := deliveryservice.GetDNSSECKeysV11(tc.DNSSECZSKType, cdnDNSDomain, ttl, inception, inception.Add(zExp), tc.DNSSECKeyStatusNew, time.Unix(effectiveDateUnix, 0), false)
	if err != nil {
		return errors.New("creating zsk for cdn: " + err.Error())
	}

	newCDNKSK, err := deliveryservice.GetDNSSECKeysV11(tc.DNSSECKSKType, cdnDNSDomain, ttl, inception, inception.Add(kExp), tc.DNSSECKeyStatusNew, time.Unix(effectiveDateUnix, 0), true)
	if err != nil {
		return errors.New("creating ksk for cdn: " + err.Error())
	}

	newCDNZSKs := []tc.DNSSECKeyV11{newCDNZSK}
	newCDNKSKs := []tc.DNSSECKeyV11{newCDNKSK}

	if oldKeysExist {
		oldKeyCDN, oldKeyCDNExists := oldKeys[cdnName]
		if oldKeyCDNExists && len(oldKeyCDN.KSK) > 0 {
			ksk := oldKeyCDN.KSK[0]
			ksk.Status = DNSSECStatusExisting
			ksk.TTLSeconds = uint64(ttl / time.Second)
			ksk.ExpirationDateUnix = effectiveDateUnix
			newCDNKSKs = append(newCDNKSKs, ksk)
		}
		if oldKeyCDNExists && len(oldKeyCDN.ZSK) > 0 {
			zsk := oldKeyCDN.ZSK[0]
			zsk.Status = DNSSECStatusExisting
			zsk.TTLSeconds = uint64(ttl / time.Second)
			zsk.ExpirationDateUnix = effectiveDateUnix
			newCDNZSKs = append(newCDNZSKs, zsk)
		}
	}

	newKeys := tc.DNSSECKeysV11{}
	newKeys[cdnName] = tc.DNSSECKeySetV11{ZSK: newCDNZSKs, KSK: newCDNKSKs}

	cdnKeys := newKeys[cdnName]

	dsNames := []string{}
	for _, ds := range dses {
		dsNames = append(dsNames, ds.Name)
	}
	matchLists, err := deliveryservice.GetDeliveryServicesMatchLists(dsNames, tx)
	if err != nil {
		return errors.New("getting delivery service matchlists: " + err.Error())
	}
	for _, ds := range dses {
		if !ds.Type.IsHTTP() && !ds.Type.IsDNS() {
			continue // skip delivery services that aren't DNS or HTTP (e.g. ANY_MAP)
		}

		matchlist, ok := matchLists[ds.Name]
		if !ok {
			return errors.New("no regex match list found for delivery service '" + ds.Name)
		}

		exampleURLs := deliveryservice.MakeExampleURLs(ds.Protocol, ds.Type, ds.RoutingName, matchlist, cdnDomain)
		log.Infoln("Creating keys for " + ds.Name)
		overrideTTL := true
		dsKeys, err := deliveryservice.CreateDNSSECKeys(tx, cfg, ds.Name, exampleURLs, cdnKeys, kExp, zExp, ttl, overrideTTL)
		if err != nil {
			return errors.New("creating delivery service DNSSEC keys: " + err.Error())
		}
		newKeys[ds.Name] = dsKeys
	}
	if err := riaksvc.PutDNSSECKeys(tc.DNSSECKeysRiak(newKeys), cdnName, tx, cfg.RiakAuthOptions, cfg.RiakPort); err != nil {
		return errors.New("putting Riak DNSSEC CDN keys: " + err.Error())
	}
	return nil
}

const API_DNSSECKEYS = "DELETE /cdns/name/:name/dnsseckeys"

type CDNDS struct {
	Name        string
	Protocol    *int
	Type        tc.DSType
	RoutingName string
}

// getCDNDeliveryServices returns basic data for the delivery services on the given CDN, as well as the CDN name, or any error.
func GetCDNDeliveryServices(tx *sql.Tx, cdn string) ([]CDNDS, error) {
	q := `
SELECT ds.xml_id, ds.protocol, t.name as type, ds.routing_name
FROM deliveryservice as ds
JOIN cdn ON ds.cdn_id = cdn.id
JOIN type as t ON ds.type = t.id
WHERE cdn.name = $1
`
	rows, err := tx.Query(q, cdn)
	if err != nil {
		return nil, errors.New("getting cdn delivery services: " + err.Error())
	}
	defer rows.Close()
	dses := []CDNDS{}
	for rows.Next() {
		ds := CDNDS{}
		dsTypeStr := ""
		if err := rows.Scan(&ds.Name, &ds.Protocol, &dsTypeStr, &ds.RoutingName); err != nil {
			return nil, errors.New("scanning cdn delivery services: " + err.Error())
		}
		dsType := tc.DSTypeFromString(dsTypeStr)
		if dsType == tc.DSTypeInvalid {
			return nil, errors.New("got invalid delivery service type '" + dsTypeStr + "'")
		}
		ds.Type = dsType
		dses = append(dses, ds)
	}
	return dses, nil
}

func DeleteDNSSECKeys(w http.ResponseWriter, r *http.Request) {
	deleteDNSSECKeys(w, r, false)
}

func DeleteDNSSECKeysDeprecated(w http.ResponseWriter, r *http.Request) {
	deleteDNSSECKeys(w, r, true)
}

func writeError(w http.ResponseWriter, r *http.Request, tx *sql.Tx, statusCode int, userErr error, sysErr error, deprecated bool) {
	if deprecated {
		api.HandleDeprecatedErr(w, r, tx, statusCode, userErr, sysErr, util.StrPtr(API_DNSSECKEYS))
	} else {
		api.HandleErr(w, r, tx, statusCode, userErr, sysErr)
	}
}

func deleteDNSSECKeys(w http.ResponseWriter, r *http.Request, deprecated bool) {
	inf, errs := api.NewInfo(r, []string{"name"}, nil)
	if errs.Occurred() {
		// TODO: I think maybe this should just be using the optional
		// deprecation error handler.
		writeError(w, r, inf.Tx.Tx, errs.Code, errs.UserError, errs.SystemError, deprecated)
		return
	}
	defer inf.Close()

	cluster, err := riaksvc.GetPooledCluster(inf.Tx.Tx, inf.Config.RiakAuthOptions, inf.Config.RiakPort)
	if err != nil {
		writeError(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting riak cluster: "+err.Error()), deprecated)
		return
	}

	key := inf.Params["name"]
	cdnID, ok, err := getCDNIDFromName(inf.Tx.Tx, tc.CDNName(key))
	if err != nil {
		writeError(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting cdn id: "+err.Error()), deprecated)
		return
	} else if !ok {
		writeError(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil, deprecated)
		return
	}

	if err := riaksvc.DeleteObject(key, CDNDNSSECKeyType, cluster); err != nil {
		writeError(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deleting cdn dnssec keys: "+err.Error()), deprecated)
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "CDN: "+key+", ID: "+strconv.Itoa(cdnID)+", ACTION: Deleted DNSSEC keys", inf.User, inf.Tx.Tx)
	successMsg := "Successfully deleted " + CDNDNSSECKeyType + " for " + key
	if deprecated {
		api.WriteAlertsObj(w, r, http.StatusOK, api.CreateDeprecationAlerts(util.StrPtr(API_DNSSECKEYS)), successMsg)
	} else {
		api.WriteResp(w, r, successMsg)

	}
}

// getCDNIDFromName returns the CDN's ID if a CDN with the given name exists
func getCDNIDFromName(tx *sql.Tx, name tc.CDNName) (int, bool, error) {
	id := 0
	if err := tx.QueryRow(`SELECT id FROM cdn WHERE name = $1`, name).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, false, nil
		}
		return id, false, errors.New("querying CDN ID: " + err.Error())
	}
	return id, true, nil
}
