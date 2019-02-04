package riaksvc

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

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"

	"github.com/basho/riak-go-client"
)

const DeliveryServiceSSLKeysBucket = "ssl"
const DNSSECKeysBucket = "dnssec"
const DSSSLKeyVersionLatest = "latest"
const DefaultDSSSLKeyVersion = DSSSLKeyVersionLatest
const URLSigKeysBucket = "url_sig_keys"

func MakeDSSSLKeyKey(dsName, version string) string {
	if version == "" {
		version = DefaultDSSSLKeyVersion
	}
	return dsName + "-" + version
}

func GetDeliveryServiceSSLKeysObj(xmlID string, version string, tx *sql.Tx, authOpts *riak.AuthOptions) (tc.DeliveryServiceSSLKeys, bool, error) {
	cluster, err := GetPooledCluster(tx, authOpts)
	if err != nil {
		return tc.DeliveryServiceSSLKeys{}, false, errors.New("getting riak pooled cluster: " + err.Error())
	}

	// get the deliveryservice ssl keys by xmlID and version
	ro, err := FetchObjectValues(MakeDSSSLKeyKey(xmlID, version), DeliveryServiceSSLKeysBucket, cluster)
	if err != nil {
		return tc.DeliveryServiceSSLKeys{}, false, err
	}
	if len(ro) == 0 {
		return tc.DeliveryServiceSSLKeys{}, false, nil // not found
	}

	key := tc.DeliveryServiceSSLKeys{}
	if err := json.Unmarshal(ro[0].Value, &key); err != nil {
		log.Errorf("failed at unmarshaling sslkey response: %s\n", err)
		return tc.DeliveryServiceSSLKeys{}, false, errors.New("unmarshalling Riak result: " + err.Error())
	}
	return key, true, nil
}

func PutDeliveryServiceSSLKeysObj(key tc.DeliveryServiceSSLKeys, tx *sql.Tx, authOpts *riak.AuthOptions) error {
	keyJSON, err := json.Marshal(&key)
	if err != nil {
		return errors.New("marshalling key: " + err.Error())
	}

	cluster, err := GetPooledCluster(tx, authOpts)
	if err != nil {
		return errors.New("getting riak pooled cluster: " + err.Error())
	}

	obj := &riak.Object{
		ContentType:     "application/json",
		Charset:         "utf-8",
		ContentEncoding: "utf-8",
		Key:             MakeDSSSLKeyKey(key.DeliveryService, key.Version.String()),
		Value:           []byte(keyJSON),
	}
	if err := SaveObject(obj, DeliveryServiceSSLKeysBucket, cluster); err != nil {
		return errors.New("saving Riak object: " + err.Error())
	}
	obj.Key = MakeDSSSLKeyKey(key.DeliveryService, DSSSLKeyVersionLatest)
	if err := SaveObject(obj, DeliveryServiceSSLKeysBucket, cluster); err != nil {
		return errors.New("saving Riak object: " + err.Error())
	}

	return nil
}

func Ping(tx *sql.Tx, authOpts *riak.AuthOptions) (tc.RiakPingResp, error) {
	servers, err := GetRiakServers(tx)
	if err != nil {
		return tc.RiakPingResp{}, errors.New("getting riak servers: " + err.Error())
	}
	for _, server := range servers {
		cluster, err := GetRiakStorageCluster([]ServerAddr{server}, authOpts)
		if err != nil {
			log.Errorf("RiakServersToCluster error for server %+v: %+v\n", server, err.Error())
			continue // try another server
		}
		if err = cluster.Start(); err != nil {
			log.Errorln("starting Riak cluster (for ping): " + err.Error())
			continue
		}
		if err := PingCluster(cluster); err != nil {
			if err := cluster.Stop(); err != nil {
				log.Errorln("stopping Riak cluster (after ping error): " + err.Error())
			}
			log.Errorf("Riak PingCluster error for server %+v: %+v\n", server, err.Error())
			continue
		}
		if err := cluster.Stop(); err != nil {
			log.Errorln("stopping Riak cluster (after ping success): " + err.Error())
		}
		return tc.RiakPingResp{Status: "OK", Server: server.FQDN + ":" + server.Port}, nil
	}
	return tc.RiakPingResp{}, errors.New("failed to ping any Riak server")
}

func GetDNSSECKeys(cdnName string, tx *sql.Tx, authOpts *riak.AuthOptions) (tc.DNSSECKeysRiak, bool, error) {
	cluster, err := GetPooledCluster(tx, authOpts)
	if err != nil {
		return tc.DNSSECKeysRiak{}, false, errors.New("getting riak pooled cluster: " + err.Error())
	}

	ro, err := FetchObjectValues(cdnName, DNSSECKeysBucket, cluster)
	if err != nil {
		return tc.DNSSECKeysRiak{}, false, err
	}
	if len(ro) == 0 {
		return tc.DNSSECKeysRiak{}, false, nil // not found
	}

	key := tc.DNSSECKeysRiak{}
	if err := json.Unmarshal(ro[0].Value, &key); err != nil {
		return tc.DNSSECKeysRiak{}, false, errors.New("unmarshalling Riak dnssec response: " + err.Error())
	}
	return key, true, nil
}

func PutDNSSECKeys(keys tc.DNSSECKeysRiak, cdnName string, tx *sql.Tx, authOpts *riak.AuthOptions) error {
	keyJSON, err := json.Marshal(&keys)
	if err != nil {
		return errors.New("marshalling keys: " + err.Error())
	}
	cluster, err := GetPooledCluster(tx, authOpts)
	if err != nil {
		return errors.New("getting riak pooled cluster: " + err.Error())
	}
	obj := &riak.Object{
		ContentType:     "application/json",
		Charset:         "utf-8",
		ContentEncoding: "utf-8",
		Key:             cdnName,
		Value:           []byte(keyJSON),
	}
	if err = SaveObject(obj, DNSSECKeysBucket, cluster); err != nil {
		return errors.New("saving Riak object: " + err.Error())
	}
	return nil
}

func GetBucketKey(tx *sql.Tx, authOpts *riak.AuthOptions, bucket string, key string) ([]byte, bool, error) {
	cluster, err := GetPooledCluster(tx, authOpts)
	if err != nil {
		return nil, false, errors.New("getting riak pooled cluster: " + err.Error())
	}

	// get the deliveryservice ssl keys by xmlID and version
	ro, err := FetchObjectValues(key, bucket, cluster)
	if err != nil {
		return nil, false, err
	}
	if len(ro) == 0 {
		return nil, false, nil // not found
	}
	return ro[0].Value, true, nil
}

func DeleteDSSSLKeys(tx *sql.Tx, authOpts *riak.AuthOptions, xmlID string, version string) error {
	cluster, err := GetPooledCluster(tx, authOpts)
	if err != nil {
		return errors.New("getting riak pooled cluster: " + err.Error())
	}

	if err := DeleteObject(MakeDSSSLKeyKey(xmlID, version), DeliveryServiceSSLKeysBucket, cluster); err != nil {
		return errors.New("deleting SSL keys: " + err.Error())
	}
	return nil
}

// GetURLSigConfigFileName returns the filename of the Apache Traffic Server URLSig config file
// TODO move to ats config directory/file
func GetURLSigConfigFileName(ds tc.DeliveryServiceName) string {
	return "url_sig_" + string(ds) + ".config"
}

func GetURLSigKeys(tx *sql.Tx, authOpts *riak.AuthOptions, ds tc.DeliveryServiceName) (tc.URLSigKeys, bool, error) {
	cluster, err := GetPooledCluster(tx, authOpts)
	if err != nil {
		return tc.URLSigKeys{}, false, errors.New("getting riak pooled cluster: " + err.Error())
	}

	key := GetURLSigConfigFileName(ds)
	ro, err := FetchObjectValues(key, URLSigKeysBucket, cluster)
	if err != nil {
		return tc.URLSigKeys{}, false, err
	}
	if len(ro) == 0 {
		return tc.URLSigKeys{}, false, nil // not found
	}

	val := tc.URLSigKeys{}
	if err := json.Unmarshal(ro[0].Value, &val); err != nil {
		return tc.URLSigKeys{}, false, errors.New("unmarshalling Riak response: " + err.Error())
	}
	return val, true, nil
}

func PutURLSigKeys(tx *sql.Tx, authOpts *riak.AuthOptions, ds tc.DeliveryServiceName, keys tc.URLSigKeys) error {
	keyJSON, err := json.Marshal(&keys)
	if err != nil {
		return errors.New("marshalling keys: " + err.Error())
	}

	cluster, err := GetPooledCluster(tx, authOpts)
	if err != nil {
		return errors.New("getting riak pooled cluster: " + err.Error())
	}

	obj := &riak.Object{
		ContentType:     "application/json",
		Charset:         "utf-8",
		ContentEncoding: "utf-8",
		Key:             GetURLSigConfigFileName(ds),
		Value:           []byte(keyJSON),
	}
	if err = SaveObject(obj, URLSigKeysBucket, cluster); err != nil {
		return errors.New("saving Riak object: " + err.Error())
	}
	return nil
}

const SSLKeysIndex = "sslkeys"
const CDNSSLKeysLimit = 1000 // TODO: emulates Perl; reevaluate?

func GetCDNSSLKeysObj(tx *sql.Tx, authOpts *riak.AuthOptions, cdnName string) ([]tc.CDNSSLKey, error) {
	cluster, err := GetPooledCluster(tx, authOpts)
	if err != nil {
		return nil, errors.New("getting riak pooled cluster: " + err.Error())
	}

	// get the deliveryservice ssl keys by xmlID and version
	query := `cdn:` + cdnName
	filterQuery := `_yz_rk:*latest`
	searchDocs, err := Search(cluster, SSLKeysIndex, query, filterQuery, CDNSSLKeysLimit)
	if err != nil {
		return nil, errors.New("riak search error: " + err.Error())
	}
	if len(searchDocs) == 0 {
		return []tc.CDNSSLKey{}, nil // no error, and leave keys empty
	}
	return SearchDocsToCDNSSLKeys(searchDocs), nil
}

// SearchDocsToCDNSSLKeys converts the SearchDoc array returned by Riak into a CDNSSLKey slice. If a SearchDoc doesn't contain expected fields, it creates the key with those fields defaulted to empty strings.
func SearchDocsToCDNSSLKeys(docs []*riak.SearchDoc) []tc.CDNSSLKey {
	keys := []tc.CDNSSLKey{}
	for _, doc := range docs {
		key := tc.CDNSSLKey{}
		if dss := doc.Fields["deliveryservice"]; len(dss) > 0 {
			key.DeliveryService = dss[0]
		}
		if hosts := doc.Fields["hostname"]; len(hosts) > 0 {
			key.HostName = hosts[0]
		}
		if crts := doc.Fields["certificate.crt"]; len(crts) > 0 {
			key.Certificate.Crt = crts[0]
		}
		if keys := doc.Fields["certificate.key"]; len(keys) > 0 {
			key.Certificate.Key = keys[0]
		}
		keys = append(keys, key)
	}
	return keys
}
