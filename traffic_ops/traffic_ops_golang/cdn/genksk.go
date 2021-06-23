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
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
)

const DefaultKSKTTLSeconds = 60
const DefaultKSKEffectiveMultiplier = 2
const DefaultKSKExpiration = 365 * 24 * time.Hour
const DefaultZSKExpiration = 30 * 24 * time.Hour

func GenerateKSK(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("generating CDN KSK: Traffic Vault is not configured"))
		return
	}

	cdnName := tc.CDNName(inf.Params["name"])
	req := tc.CDNGenerateKSKReq{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &req); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("parsing request: "+err.Error()), nil)
		return
	}

	cdnDomain, cdnExists, err := dbhelpers.GetCDNDomainFromName(inf.Tx.Tx, cdnName)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting CDN domain: "+err.Error()))
		return
	}
	if !cdnExists {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("cdn '"+string(cdnName)+"' not found"), nil)
		return
	}

	cdnID, ok, err := getCDNIDFromName(inf.Tx.Tx, cdnName)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting cdn ID from name '"+string(cdnName)+"': "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
		return
	}

	ttl, multiplier, err := getKSKParams(inf.Tx.Tx, cdnName)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting CDN KSK parameters: "+err.Error()))
		return
	}
	if ttl == nil {
		kskTTL := uint64(DefaultKSKTTLSeconds)
		ttl = &kskTTL
	}
	if multiplier == nil {
		mult := uint64(DefaultKSKEffectiveMultiplier)
		multiplier = &mult
	}

	dnssecKeys, ok, err := inf.Vault.GetDNSSECKeys(string(cdnName), inf.Tx.Tx, r.Context())
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting CDN DNSSEC keys: "+err.Error()))
		return
	}
	if !ok {
		log.Warnln("Generating CDN '" + string(cdnName) + "' KSK: no keys found in Traffic Vault, generating and inserting new key anyway")
	}

	isKSK := true
	cdnDNSDomain := cdnDomain + "."
	newKey, err := regenExpiredKeys(isKSK, cdnDNSDomain, dnssecKeys[string(cdnName)], *req.EffectiveDate, true, true)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("regenerating CDN DNSSEC keys: "+err.Error()))
		return
	}
	dnssecKeys[string(cdnName)] = newKey

	if err := inf.Vault.PutDNSSECKeys(string(cdnName), dnssecKeys, inf.Tx.Tx, r.Context()); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("putting CDN DNSSEC keys: "+err.Error()))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "CDN: "+string(cdnName)+", ID: "+strconv.Itoa(cdnID)+", ACTION: Generated KSK DNSSEC keys", inf.User, inf.Tx.Tx)
	api.WriteResp(w, r, "Successfully generated ksk dnssec keys for "+string(cdnName))
}

// getKSKParams returns the CDN's profile's tld.ttls.DNSKEY and DNSKEY.effective.multiplier parameters. If either parameter doesn't exist, nil is returned.
func getKSKParams(tx *sql.Tx, cdn tc.CDNName) (*uint64, *uint64, error) {
	qry := `
WITH cdn_profile_id AS (
  SELECT
    p.id
  FROM
    profile p
    JOIN cdn c ON c.id = p.cdn
  WHERE
    c.name = $1
    AND (p.name like 'CCR%' OR p.name like 'TR%')
  FETCH FIRST 1 ROWS ONLY
)
SELECT
  pa.name,
  pa.value
FROM
  parameter pa
  JOIN profile_parameter pp ON pp.parameter = pa.id
  JOIN profile pr ON pr.id = pp.profile
  JOIN cdn_profile_id pi on pi.id = pr.id
WHERE
  (pa.name = 'tld.ttls.DNSKEY' OR pa.name = 'DNSKEY.effective.multiplier')
`
	rows, err := tx.Query(qry, cdn)
	if err != nil {
		return nil, nil, errors.New("getting cdn ksk parameters: " + err.Error())
	}
	defer rows.Close()
	ttl := (*uint64)(nil)
	mult := (*uint64)(nil)

	for rows.Next() {
		name := ""
		valStr := ""
		if err := rows.Scan(&name, &valStr); err != nil {
			return nil, nil, errors.New("scanning cdn ksk parameters: " + err.Error())
		}
		val, err := strconv.ParseUint(valStr, 10, 64)
		if err != nil {
			log.Warnln("getting CDN KSK parameters: parameter '" + name + "' value '" + valStr + "' is not a number, skipping")
			continue
		}
		if name == "tld.ttls.DNSKEY" {
			ttl = &val
		} else {
			mult = &val
		}
	}
	return ttl, mult, nil
}

const DefaultDNSSECKeyTTL = 60 * time.Second

// regenExpiredKeys regenerates expired keys. The key is the map key into the keys object, which may be a CDN name or a delivery service name.
// The name is the name of the key, either the CDN name or the Delivery Service name. If existingKeys contains any keys marked "new", the name argument is not used, but the name of the previously-new key is used instead. These should match, and a warning is logged if they differ.
func regenExpiredKeys(typeKSK bool, name string, existingKeys tc.DNSSECKeySetV11, effectiveDate time.Time, tld bool, resetExp bool) (tc.DNSSECKeySetV11, error) {
	existingKey := ([]tc.DNSSECKeyV11)(nil)
	if typeKSK {
		existingKey = existingKeys.KSK
	} else {
		existingKey = existingKeys.ZSK
	}

	oldKey := tc.DNSSECKeyV11{}
	oldKeyFound := false
	for _, key := range existingKey {
		if key.Status == tc.DNSSECKeyStatusNew {
			oldKey = key
			oldKeyFound = true
			break
		}
	}

	newInception := time.Now()
	defaultExpiration := DefaultZSKExpiration
	if typeKSK {
		defaultExpiration = DefaultKSKExpiration
	}
	newExpiration := newInception.Add(defaultExpiration)

	ttl := DefaultDNSSECKeyTTL
	if oldKeyFound {
		expiration := oldKey.ExpirationDateUnix
		inception := oldKey.InceptionDateUnix
		const secPerDay = 86400
		expirationDays := (expiration - inception) / secPerDay

		if name != oldKey.Name {
			log.Warnln("regenExpiredKeys given name '" + name + "' which doesn't match existingKey name '" + oldKey.Name + "' - using existing key name in refresh! Ignoring expected passed name!")
		}

		name = oldKey.Name
		ttl = time.Duration(oldKey.TTLSeconds) * time.Second
		newExpiration = time.Now().Add(time.Duration(expirationDays) * time.Hour * 24)
	}

	keyType := tc.DNSSECKSKType
	if !typeKSK {
		keyType = tc.DNSSECZSKType
	}
	newKey, err := deliveryservice.GetDNSSECKeysV11(keyType, name, ttl, newInception, newExpiration, tc.DNSSECKeyStatusNew, effectiveDate, tld)
	if err != nil {
		return tc.DNSSECKeySetV11{}, errors.New("getting and generating DNSSEC keys: " + err.Error())
	}

	newKeys := []tc.DNSSECKeyV11{newKey}
	if oldKeyFound {
		oldKey.Status = tc.DNSSECKeyStatusExpired
		if resetExp {
			oldKey.ExpirationDateUnix = effectiveDate.Unix()
		}

		newKeys = append(newKeys, oldKey)
	}

	regenKeys := tc.DNSSECKeySetV11{}
	if typeKSK {
		regenKeys = tc.DNSSECKeySetV11{ZSK: existingKeys.ZSK, KSK: newKeys}
	} else {
		regenKeys = tc.DNSSECKeySetV11{ZSK: newKeys, KSK: existingKeys.KSK}
	}
	return regenKeys, nil
}
