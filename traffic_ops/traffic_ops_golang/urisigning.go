package main

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
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/riaksvc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	"github.com/basho/riak-go-client"
	"github.com/lestrrat/go-jwx/jwk"
)

// CDNURIKeysBucket is the namespace or bucket used for CDN URI signing keys.
const CDNURIKeysBucket = "cdn_uri_sig_keys"

// URISignerKeyset is the container for the CDN URI signing keys
type URISignerKeyset struct {
	RenewalKid *string               `json:"renewal_kid"`
	Keys       []jwk.EssentialHeader `json:"keys"`
}

// endpoint handler for fetching uri signing keys from riak
func getURIsignkeysHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if inf.Config.RiakEnabled == false {
		api.HandleErr(w, r, inf.Tx, http.StatusServiceUnavailable, errors.New("The RIAK service is unavailable"), errors.New("getting Riak SSL keys by host name: riak is not configured"))
		return
	}

	xmlID := inf.Params["xmlID"]

	if userErr, sysErr, errCode := tenant.Check(inf.User, xmlID, inf.Tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx, errCode, userErr, sysErr)
		return
	}

	cluster, err := riaksvc.StartCluster(inf.Tx, inf.Config.RiakAuthOptions)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("starting riak cluster: "+err.Error()))
		return
	}
	defer riaksvc.StopCluster(cluster)

	ro, err := riaksvc.FetchObjectValues(xmlID, CDNURIKeysBucket, cluster)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("fetching riak objects: "+err.Error()))
		return
	}
	if len(ro) == 0 {
		api.WriteRespRaw(w, r, URISignerKeyset{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(ro[0].Value)
}

// removeDeliveryServiceURIKeysHandler is the HTTP DELETE handler used to remove urisigning keys assigned to a delivery service.
func removeDeliveryServiceURIKeysHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if inf.Config.RiakEnabled == false {
		api.HandleErr(w, r, inf.Tx, http.StatusServiceUnavailable, errors.New("The RIAK service is unavailable"), errors.New("getting Riak SSL keys by host name: riak is not configured"))
		return
	}

	xmlID := inf.Params["xmlID"]
	if userErr, sysErr, errCode := tenant.Check(inf.User, xmlID, inf.Tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx, errCode, userErr, sysErr)
		return
	}

	cluster, err := riaksvc.StartCluster(inf.Tx, inf.Config.RiakAuthOptions)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("starting riak cluster: "+err.Error()))
		return
	}
	defer riaksvc.StopCluster(cluster)

	ro, err := riaksvc.FetchObjectValues(xmlID, CDNURIKeysBucket, cluster)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("fetching riak objects: "+err.Error()))
		return
	}

	if len(ro) == 0 || ro[0].Value == nil {
		api.WriteRespAlert(w, r, tc.InfoLevel, "not deleted, no object found to delete")
		return
	}
	if err := riaksvc.DeleteObject(xmlID, CDNURIKeysBucket, cluster); err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("deleting riak object: "+err.Error()))
		return
	}
	api.WriteRespAlert(w, r, tc.SuccessLevel, "object deleted")
	return
}

// saveDeliveryServiceURIKeysHandler is the HTTP POST or PUT handler used to store urisigning keys to a delivery service.
func saveDeliveryServiceURIKeysHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if inf.Config.RiakEnabled == false {
		api.HandleErr(w, r, inf.Tx, http.StatusServiceUnavailable, errors.New("The RIAK service is unavailable"), errors.New("getting Riak SSL keys by host name: riak is not configured"))
		return
	}

	xmlID := inf.Params["xmlID"]

	if userErr, sysErr, errCode := tenant.Check(inf.User, xmlID, inf.Tx); userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx, errCode, userErr, sysErr)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, errors.New("failed to read body"), errors.New("failed to read body: "+err.Error()))
		return
	}
	keySet := map[string]URISignerKeyset{}
	if err := json.Unmarshal(data, &keySet); err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusBadRequest, errors.New("malformed JSON"), nil)
	}
	if err := validateURIKeyset(keySet); err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusBadRequest, errors.New("invalid keyset: "+err.Error()), nil)
		return
	}

	cluster, err := riaksvc.StartCluster(inf.Tx, inf.Config.RiakAuthOptions)
	if err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("starting riak cluster: "+err.Error()))
		return
	}
	defer riaksvc.StopCluster(cluster)

	obj := &riak.Object{
		ContentType:     "text/json",
		Charset:         "utf-8",
		ContentEncoding: "utf-8",
		Key:             xmlID,
		Value:           []byte(data),
	}

	if err = riaksvc.SaveObject(obj, CDNURIKeysBucket, cluster); err != nil {
		api.HandleErr(w, r, inf.Tx, http.StatusInternalServerError, nil, errors.New("saving riak object: "+err.Error()))
		return
	}
	w.Header().Set("Content-Type", tc.ApplicationJson)
	w.Write(data)
}

// validateURIKeyset validates URISigingKeyset json.
func validateURIKeyset(msg map[string]URISignerKeyset) error {
	var renewalKidFound int
	var renewalKidMatched = false

	for key, value := range msg {
		issuer := key
		renewalKid := value.RenewalKid
		if issuer == "" {
			return errors.New("JSON Keyset has no issuer")
		}

		if renewalKid != nil {
			renewalKidFound++
		}

		for _, skey := range value.Keys {
			if skey.Algorithm == "" {
				return errors.New("A Key has no algorithm, alg, specified")
			}
			if skey.KeyID == "" {
				return errors.New("A Key has no key id, kid, specified")
			}
			if renewalKid != nil && strings.Compare(*renewalKid, skey.KeyID) == 0 {
				renewalKidMatched = true
			}
		}
	}

	// should only have one renewal_kid
	switch renewalKidFound {
	case 0:
		return errors.New("No renewal_kid was found in any keyset")
	case 1: // okay, this is what we want
		break
	default:
		return errors.New("More than one renewal_kid was found in the keysets")
	}

	// the renewal_kid should match the kid of one key
	if !renewalKidMatched {
		return errors.New("No key was found with a kid that matches the renewal kid")
	}

	return nil
}
