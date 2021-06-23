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
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault"
)

const (
	CDNDNSSECKeyType     = "dnssec"
	DNSSECStatusExisting = "existing"

	DNSSECGenerationCPURatio = 0.66
)

func CreateDNSSECKeys(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deleting CDN DNSSEC keys from Traffic Vault: Traffic Vault is not configured"))
		return
	}

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

	if err := generateStoreDNSSECKeys(inf.Tx.Tx, cdnName, cdnDomain, uint64(*req.TTL), uint64(*req.KSKExpirationDays), uint64(*req.ZSKExpirationDays), int64(*req.EffectiveDateUnix), inf.Vault, r.Context()); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("generating and storing DNSSEC CDN keys: "+err.Error()))
		return
	}
	// NOTE: using a separate transaction (with its own timeout) for the changelog because the main
	// transaction can time out if DNSSEC generation takes too long
	db, err := api.GetDB(r.Context())
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("generating CDN DNSSEC keys: getting DB from request context for changelog: "+err.Error()))
		return
	}
	logCtx, logCancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer logCancel()
	logTx, err := db.BeginTxx(logCtx, nil)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("generating CDN DNSSEC keys: could not begin transaction for changelog: "+err.Error()))
		return
	}
	defer func() {
		if err := logTx.Commit(); err != nil && err != sql.ErrTxDone {
			log.Errorln("generating CDN DNSSEC keys: committing transaction for changelog: " + err.Error())
		}
	}()
	api.CreateChangeLogRawTx(api.ApiChange, "CDN: "+cdnName+", ID: "+strconv.Itoa(cdnID)+", ACTION: Generated DNSSEC keys", inf.User, logTx.Tx)
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
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deleting CDN DNSSEC keys from Traffic Vault: Traffic Vault is not configured"))
		return
	}

	cdnName := inf.Params["name"]

	tvKeys, keysExist, err := inf.Vault.GetDNSSECKeys(cdnName, inf.Tx.Tx, r.Context())
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

	keys, err := deliveryservice.MakeDNSSECKeysFromTrafficVaultKeys(tvKeys, dsTTL)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("creating DNSSEC keys object from Traffic Vault keys: "+err.Error()))
		return
	}
	api.WriteResp(w, r, keys)
}

func GetDNSSECKeysV11(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	cdnName := inf.Params["name"]
	riakKeys, keysExist, err := inf.Vault.GetDNSSECKeys(cdnName, inf.Tx.Tx, r.Context())
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
	cdnName string,
	cdnDomain string,
	ttlSeconds uint64,
	kExpDays uint64,
	zExpDays uint64,
	effectiveDateUnix int64,
	tv trafficvault.TrafficVault,
	ctx context.Context,
) error {

	zExp := time.Duration(zExpDays) * time.Hour * 24
	kExp := time.Duration(kExpDays) * time.Hour * 24
	ttl := time.Duration(ttlSeconds) * time.Second

	oldKeys, oldKeysExist, err := tv.GetDNSSECKeys(cdnName, tx, ctx)
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

	newKeys := tc.DNSSECKeysTrafficVault{}
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

	jobList := make([]dnssecGenJob, 0, len(dses))
	for _, ds := range dses {
		if !ds.Type.IsHTTP() && !ds.Type.IsDNS() {
			continue // skip delivery services that aren't DNS or HTTP (e.g. ANY_MAP)
		}

		matchlist, ok := matchLists[ds.Name]
		if !ok {
			return errors.New("no regex match list found for delivery service '" + ds.Name)
		}

		exampleURLs := deliveryservice.MakeExampleURLs(ds.Protocol, ds.Type, ds.RoutingName, matchlist, cdnDomain)
		jobList = append(jobList, dnssecGenJob{
			XMLID:       ds.Name,
			ExampleURLs: exampleURLs,
			CDNKeys:     cdnKeys,
			KExp:        kExp,
			ZExp:        zExp,
			TTL:         ttl,
			OverrideTTL: true,
		})
	}
	numWorkers := int(math.Max(1, math.Floor(float64(runtime.NumCPU())*DNSSECGenerationCPURatio)))
	jobChan := make(chan dnssecGenJob, len(jobList))
	resultChan := make(chan dnssecGenResult, len(jobList))
	panickedChan := make(chan struct{}, numWorkers)
	wg := sync.WaitGroup{}
	wg.Add(numWorkers)
	for w := 0; w < numWorkers; w++ {
		go dnssecGenWorker(w, &wg, jobChan, resultChan, panickedChan)
	}
	for _, j := range jobList {
		jobChan <- j
	}
	close(jobChan)
	wg.Wait()
	select {
	case <-panickedChan:
		return errors.New("creating DNSSEC keys, at least one worker goroutine panicked")
	default:
		log.Infoln("no DNSSEC generation worker goroutines panicked")
	}
	for i := 0; i < len(jobList); i++ {
		res := <-resultChan
		if res.Error != nil {
			return fmt.Errorf("creating DNSSEC keys for delivery service %s: %s", res.XMLID, res.Error.Error())
		}
		newKeys[res.XMLID] = *res.Keys
	}

	if err := tv.PutDNSSECKeys(cdnName, newKeys, tx, ctx); err != nil {
		return errors.New("putting CDN DNSSEC keys in Traffic Vault: " + err.Error())
	}
	return nil
}

func dnssecGenWorker(id int, waitGroup *sync.WaitGroup, jobs <-chan dnssecGenJob, results chan<- dnssecGenResult, panicked chan<- struct{}) {
	log.Infof("DNSSEC gen worker %d starting", id)
	defer func() {
		if r := recover(); r != nil {
			panicked <- struct{}{}
			log.Errorf("DNSSEC gen worker %d recovered from panic: %v", id, r)
		}
		waitGroup.Done()
		log.Infof("DNSSEC gen worker %d exiting", id)
	}()
	for j := range jobs {
		log.Infof("DNSSEC gen worker %d creating keys for %s", id, j.XMLID)
		res := dnssecGenResult{XMLID: j.XMLID}
		dsKeys, err := deliveryservice.CreateDNSSECKeys(j.ExampleURLs, j.CDNKeys, j.KExp, j.ZExp, j.TTL, j.OverrideTTL)
		if err != nil {
			res.Error = err
		} else {
			res.Keys = &dsKeys
		}
		results <- res
	}
}

type dnssecGenJob struct {
	XMLID       string
	ExampleURLs []string
	CDNKeys     tc.DNSSECKeySetV11
	KExp        time.Duration
	ZExp        time.Duration
	TTL         time.Duration
	OverrideTTL bool
}

type dnssecGenResult struct {
	XMLID string
	Keys  *tc.DNSSECKeySetV11
	Error error
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
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		writeError(w, r, inf.Tx.Tx, errCode, userErr, sysErr, deprecated)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		writeError(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deleting CDN DNSSEC keys from Traffic Vault: Traffic Vault is not configured"), deprecated)
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
	if err := inf.Vault.DeleteDNSSECKeys(key, inf.Tx.Tx, r.Context()); err != nil {
		writeError(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deleting CDN DNSSEC keys: "+err.Error()), deprecated)
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
