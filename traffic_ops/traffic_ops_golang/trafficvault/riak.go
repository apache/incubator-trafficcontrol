package trafficvault

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

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault/backends/riaksvc"
)

const RiakBackendName = "riak"

type Riak struct {
	cfg riaksvc.Config
}

func (r *Riak) GetDeliveryServiceSSLKeys(xmlID string, version string, tx *sql.Tx) (tc.DeliveryServiceSSLKeysV15, bool, error) {
	return riaksvc.GetDeliveryServiceSSLKeysObjV15(xmlID, version, tx, &r.cfg.AuthOptions, &r.cfg.Port)
}

func (r *Riak) PutDeliveryServiceSSLKeys(key tc.DeliveryServiceSSLKeys, tx *sql.Tx) error {
	return riaksvc.PutDeliveryServiceSSLKeysObj(key, tx, &r.cfg.AuthOptions, &r.cfg.Port)
}

func (r *Riak) DeleteDeliveryServiceSSLKeys(xmlID string, version string, tx *sql.Tx) error {
	return riaksvc.DeleteDSSSLKeys(tx, &r.cfg.AuthOptions, &r.cfg.Port, xmlID, version)
}

func (r *Riak) DeleteOldDeliveryServiceSSLKeys(existingXMLIDs map[string]struct{}, cdnName string, tx *sql.Tx) error {
	return riaksvc.DeleteOldDeliveryServiceSSLKeys(tx, &r.cfg.AuthOptions, &r.cfg.Port, tc.CDNName(cdnName), existingXMLIDs)
}

func (r *Riak) GetCDNSSLKeys(cdnName string, tx *sql.Tx) ([]tc.CDNSSLKey, error) {
	return riaksvc.GetCDNSSLKeysObj(tx, &r.cfg.AuthOptions, &r.cfg.Port, cdnName)
}

func (r *Riak) GetDNSSECKeys(cdnName string, tx *sql.Tx) (tc.DNSSECKeysTrafficVault, bool, error) {
	keys, exists, err := riaksvc.GetDNSSECKeys(cdnName, tx, &r.cfg.AuthOptions, &r.cfg.Port)
	return tc.DNSSECKeysTrafficVault(keys), exists, err
}

func (r *Riak) PutDNSSECKeys(cdnName string, keys tc.DNSSECKeysTrafficVault, tx *sql.Tx) error {
	return riaksvc.PutDNSSECKeys(tc.DNSSECKeysRiak(keys), cdnName, tx, &r.cfg.AuthOptions, &r.cfg.Port)
}

func (r *Riak) DeleteDNSSECKeys(cdnName string, tx *sql.Tx) error {
	return riaksvc.DeleteDNSSECKeys(cdnName, tx, &r.cfg.AuthOptions, &r.cfg.Port)
}

func (r *Riak) GetURLSigKeys(xmlID string, tx *sql.Tx) (tc.URLSigKeys, bool, error) {
	return riaksvc.GetURLSigKeys(tx, &r.cfg.AuthOptions, &r.cfg.Port, tc.DeliveryServiceName(xmlID))
}

func (r *Riak) PutURLSigKeys(xmlID string, keys tc.URLSigKeys, tx *sql.Tx) error {
	return riaksvc.PutURLSigKeys(tx, &r.cfg.AuthOptions, &r.cfg.Port, tc.DeliveryServiceName(xmlID), keys)
}

func (r *Riak) GetURISigningKeys(xmlID string, tx *sql.Tx) ([]byte, bool, error) {
	return riaksvc.GetURISigningKeys(tx, &r.cfg.AuthOptions, &r.cfg.Port, xmlID)
}

func (r *Riak) PutURISigningKeys(xmlID string, keysJson []byte, tx *sql.Tx) error {
	return riaksvc.PutURISigningKeys(tx, &r.cfg.AuthOptions, &r.cfg.Port, xmlID, keysJson)
}

func (r *Riak) DeleteURISigningKeys(xmlID string, tx *sql.Tx) error {
	return riaksvc.DeleteURISigningKeys(tx, &r.cfg.AuthOptions, &r.cfg.Port, xmlID)
}

func (r *Riak) Ping(tx *sql.Tx) (tc.TrafficVaultPingResponse, error) {
	resp, err := riaksvc.Ping(tx, &r.cfg.AuthOptions, &r.cfg.Port)
	return tc.TrafficVaultPingResponse(resp), err
}

func (r *Riak) GetBucketKey(bucket string, key string, tx *sql.Tx) ([]byte, bool, error) {
	return riaksvc.GetBucketKey(tx, &r.cfg.AuthOptions, &r.cfg.Port, bucket, key)
}

func init() {
	addBackend(RiakBackendName, riakConfigLoad)
}

func riakConfigLoad(b json.RawMessage) (TrafficVault, error) {
	_, riakCfg, err := riaksvc.UnmarshalRiakConfig(b)
	if err != nil {
		return nil, err
	}
	// TODO: validate the config
	return &Riak{cfg: riakCfg}, nil
}

// TODO: add unit test with:
// goodRiakConfig = `
// 	   {
// 	       "user": "riakuser",
// 	       "password": "password",
// 	       "port": 8087,
// 	       "MaxTLSVersion": "1.1",
// 	       "tlsConfig": {
// 	           "insecureSkipVerify": true
// 	       }
// 	   }
// 	   	`
