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
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"

	"github.com/basho/riak-go-client"
)

const deliveryServiceSSLKeysBucket = "ssl"
const dnssecKeysBucket = "dnssec"
const dsSSLKeyVersionLatest = "latest"
const defaultDSSSLKeyVersion = dsSSLKeyVersionLatest
const urlSigKeysBucket = "url_sig_keys"

// cdnURIKeysBucket is the namespace or bucket used for CDN URI signing keys.
const cdnURIKeysBucket = "cdn_uri_sig_keys"

func makeDSSSLKeyKey(dsName, version string) string {
	if version == "" {
		version = defaultDSSSLKeyVersion
	}
	return dsName + "-" + version
}

func getDeliveryServiceSSLKeysObjV15(xmlID string, version string, tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint) (tc.DeliveryServiceSSLKeysV15, bool, error) {
	key := tc.DeliveryServiceSSLKeysV15{}
	found := false
	err := withCluster(tx, authOpts, riakPort, func(cluster StorageCluster) error {
		// get the deliveryservice ssl keys by xmlID and version
		ro, err := fetchObjectValues(makeDSSSLKeyKey(xmlID, version), deliveryServiceSSLKeysBucket, cluster)
		if err != nil {
			return err
		}
		if len(ro) == 0 {
			return nil // not found
		}
		if err := json.Unmarshal(ro[0].Value, &key); err != nil {
			log.Errorf("failed at unmarshaling sslkey response: %s\n", err)
			return errors.New("unmarshalling Riak result: " + err.Error())
		}
		found = true
		return nil
	})
	if err != nil {
		return key, false, err
	}
	return key, found, nil
}

func putDeliveryServiceSSLKeysObj(key tc.DeliveryServiceSSLKeys, tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint) error {
	keyJSON, err := json.Marshal(&key)
	if err != nil {
		return errors.New("marshalling key: " + err.Error())
	}
	err = withCluster(tx, authOpts, riakPort, func(cluster StorageCluster) error {
		obj := &riak.Object{
			ContentType:     rfc.ApplicationJSON,
			Charset:         "utf-8",
			ContentEncoding: "utf-8",
			Key:             makeDSSSLKeyKey(key.DeliveryService, key.Version.String()),
			Value:           []byte(keyJSON),
		}
		if err := saveObject(obj, deliveryServiceSSLKeysBucket, cluster); err != nil {
			return errors.New("saving Riak object: " + err.Error())
		}
		obj.Key = makeDSSSLKeyKey(key.DeliveryService, dsSSLKeyVersionLatest)
		if err := saveObject(obj, deliveryServiceSSLKeysBucket, cluster); err != nil {
			return errors.New("saving Riak object: " + err.Error())
		}
		return nil
	})
	return err
}

func ping(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint) (tc.TrafficVaultPing, error) {
	servers, err := getRiakServers(tx, riakPort)
	if err != nil {
		return tc.TrafficVaultPing{}, errors.New("getting riak servers: " + err.Error())
	}
	for _, server := range servers {
		cluster, err := getRiakStorageCluster([]ServerAddr{server}, authOpts)
		if err != nil {
			log.Errorf("RiakServersToCluster error for server %+v: %+v\n", server, err.Error())
			continue // try another server
		}
		if err = cluster.Start(); err != nil {
			log.Errorln("starting Riak cluster (for ping): " + err.Error())
			continue
		}
		if err := pingCluster(cluster); err != nil {
			if err := cluster.Stop(); err != nil {
				log.Errorln("stopping Riak cluster (after ping error): " + err.Error())
			}
			log.Errorf("Riak pingCluster error for server %+v: %+v\n", server, err.Error())
			continue
		}
		if err := cluster.Stop(); err != nil {
			log.Errorln("stopping Riak cluster (after ping success): " + err.Error())
		}
		return tc.TrafficVaultPing{Status: "OK", Server: server.FQDN + ":" + server.Port}, nil
	}
	return tc.TrafficVaultPing{}, errors.New("failed to ping any Riak server")
}

func getDNSSECKeys(cdnName string, tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint) (tc.DNSSECKeysRiak, bool, error) {
	key := tc.DNSSECKeysRiak{}
	found := false
	err := withCluster(tx, authOpts, riakPort, func(cluster StorageCluster) error {
		ro, err := fetchObjectValues(cdnName, dnssecKeysBucket, cluster)
		if err != nil {
			return err
		}
		if len(ro) == 0 {
			return nil // not found
		}
		if err := json.Unmarshal(ro[0].Value, &key); err != nil {
			return errors.New("unmarshalling Riak dnssec response: " + err.Error())
		}
		found = true
		return nil
	})
	if err != nil {
		return key, false, err
	}
	return key, found, nil
}

func putDNSSECKeys(keys tc.DNSSECKeysRiak, cdnName string, tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint) error {
	keyJSON, err := json.Marshal(&keys)
	if err != nil {
		return errors.New("marshalling keys: " + err.Error())
	}

	err = withCluster(tx, authOpts, riakPort, func(cluster StorageCluster) error {
		obj := &riak.Object{
			ContentType:     "application/json",
			Charset:         "utf-8",
			ContentEncoding: "utf-8",
			Key:             cdnName,
			Value:           []byte(keyJSON),
		}
		if err = saveObject(obj, dnssecKeysBucket, cluster); err != nil {
			return errors.New("saving Riak object: " + err.Error())
		}
		return nil
	})
	return err
}

func deleteDNSSECKeys(cdnName string, tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint) error {
	cluster, err := getPooledCluster(tx, authOpts, riakPort)
	if err != nil {
		return errors.New("getting riak cluster: " + err.Error())
	}
	if err := deleteObject(cdnName, dnssecKeysBucket, cluster); err != nil {
		return errors.New("deleting riak object: " + err.Error())
	}
	return nil
}

func getBucketKey(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, bucket string, key string) ([]byte, bool, error) {
	val := []byte{}
	found := false
	err := withCluster(tx, authOpts, riakPort, func(cluster StorageCluster) error {
		// get the deliveryservice ssl keys by xmlID and version
		ro, err := fetchObjectValues(key, bucket, cluster)
		if err != nil {
			return err
		}
		if len(ro) == 0 {
			return nil // not found
		}
		val = ro[0].Value
		found = true
		return nil
	})
	if err != nil {
		return val, false, err
	}
	return val, found, nil
}

func deleteDSSSLKeys(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, xmlID string, version string) error {
	err := withCluster(tx, authOpts, riakPort, func(cluster StorageCluster) error {
		if err := deleteObject(makeDSSSLKeyKey(xmlID, version), deliveryServiceSSLKeysBucket, cluster); err != nil {
			return errors.New("deleting SSL keys: " + err.Error())
		}
		return nil
	})
	return err
}

// deleteDeliveryServicesSSLKey deletes a Delivery Service SSL key.
// This should almost never be used directly, prefer deleteDSSSLKeys instead.
// This should only be used to delete keys, which may not conform to the makeDSSSLKeyKey format. For example when deleting all keys on a delivery service, and some may have been created manually outside Traffic Ops, or are otherwise malformed.
func deleteDeliveryServicesSSLKey(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, key string) error {
	err := withCluster(tx, authOpts, riakPort, func(cluster StorageCluster) error {
		if err := deleteObject(key, deliveryServiceSSLKeysBucket, cluster); err != nil {
			return errors.New("deleting SSL keys: " + err.Error())
		}
		return nil
	})
	return err
}

func getURISigningKeys(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, xmlID string) ([]byte, bool, error) {
	cluster, err := getPooledCluster(tx, authOpts, riakPort)
	if err != nil {
		return nil, false, errors.New("getting pooled Riak cluster: " + err.Error())
	}
	ro, err := fetchObjectValues(xmlID, cdnURIKeysBucket, cluster)
	if err != nil {
		return nil, false, errors.New("fetching riak objects: " + err.Error())
	}
	if len(ro) == 0 {
		return []byte{}, false, nil
	}
	if ro[0].Value == nil {
		return ro[0].Value, false, nil
	}
	return ro[0].Value, true, nil
}

func deleteURISigningKeys(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, xmlID string) error {
	cluster, err := getPooledCluster(tx, authOpts, riakPort)
	if err != nil {
		return errors.New("getting pooled Riak cluster: " + err.Error())
	}
	if err := deleteObject(xmlID, cdnURIKeysBucket, cluster); err != nil {
		return errors.New("deleting object: " + err.Error())
	}
	return nil
}

func putURISigningKeys(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, xmlID string, keysJson []byte) error {
	cluster, err := getPooledCluster(tx, authOpts, riakPort)
	if err != nil {
		return errors.New("getting pooled Riak cluster: " + err.Error())
	}
	obj := &riak.Object{
		ContentType:     "text/json",
		Charset:         "utf-8",
		ContentEncoding: "utf-8",
		Key:             xmlID,
		Value:           keysJson,
	}
	if err = saveObject(obj, cdnURIKeysBucket, cluster); err != nil {
		return errors.New("saving riak object: " + err.Error())
	}
	return nil
}

// getURLSigConfigFileName returns the filename of the Apache Traffic Server URLSig config file
// TODO move to ats config directory/file
func getURLSigConfigFileName(ds tc.DeliveryServiceName) string {
	return "url_sig_" + string(ds) + ".config"
}

func getURLSigKeys(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, ds tc.DeliveryServiceName) (tc.URLSigKeys, bool, error) {
	val := tc.URLSigKeys{}
	found := false
	key := getURLSigConfigFileName(ds)
	err := withCluster(tx, authOpts, riakPort, func(cluster StorageCluster) error {
		ro, err := fetchObjectValues(key, urlSigKeysBucket, cluster)
		if err != nil {
			return err
		}
		if len(ro) == 0 {
			return nil // not found
		}
		if err := json.Unmarshal(ro[0].Value, &val); err != nil {
			return errors.New("unmarshalling Riak response: " + err.Error())
		}
		found = true
		return nil
	})
	if err != nil {
		return val, false, err
	}
	return val, found, nil
}

func putURLSigKeys(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, ds tc.DeliveryServiceName, keys tc.URLSigKeys) error {
	keyJSON, err := json.Marshal(&keys)
	if err != nil {
		return errors.New("marshalling keys: " + err.Error())
	}
	err = withCluster(tx, authOpts, riakPort, func(cluster StorageCluster) error {
		obj := &riak.Object{
			ContentType:     "application/json",
			Charset:         "utf-8",
			ContentEncoding: "utf-8",
			Key:             getURLSigConfigFileName(ds),
			Value:           []byte(keyJSON),
		}
		if err = saveObject(obj, urlSigKeysBucket, cluster); err != nil {
			return errors.New("saving Riak object: " + err.Error())
		}
		return nil
	})
	return err
}

func deleteURLSigningKeys(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, ds tc.DeliveryServiceName) error {
	cluster, err := getPooledCluster(tx, authOpts, riakPort)
	if err != nil {
		return errors.New("getting pooled Riak cluster: " + err.Error())
	}
	key := getURLSigConfigFileName(ds)
	if err := deleteObject(key, urlSigKeysBucket, cluster); err != nil {
		return errors.New("deleting object: " + err.Error())
	}
	return nil
}

const sslKeysIndex = "sslkeys"
const cdnSSLKeysLimit = 1000 // TODO: emulates Perl; reevaluate?

func getCDNSSLKeysObj(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, cdnName string) ([]tc.CDNSSLKey, error) {
	keys := []tc.CDNSSLKey{}
	err := withCluster(tx, authOpts, riakPort, func(cluster StorageCluster) error {
		// get the deliveryservice ssl keys by xmlID and version
		query := `cdn:` + cdnName
		filterQuery := `_yz_rk:*latest`
		fields := []string{"deliveryservice", "hostname", "certificate.crt", "certificate.key"}
		searchDocs, err := search(cluster, sslKeysIndex, query, filterQuery, cdnSSLKeysLimit, fields)
		if err != nil {
			return errors.New("riak search error: " + err.Error())
		}
		if len(searchDocs) == 0 {
			return nil // no error, and leave keys empty
		}
		keys = searchDocsToCDNSSLKeys(searchDocs)
		return nil
	})
	if err != nil {
		return nil, errors.New("with cluster error: " + err.Error())
	}
	return keys, nil
}

// searchDocsToCDNSSLKeys converts the SearchDoc array returned by Riak into a CDNSSLKey slice. If a SearchDoc doesn't contain expected fields, it creates the key with those fields defaulted to empty strings.
func searchDocsToCDNSSLKeys(docs []*riak.SearchDoc) []tc.CDNSSLKey {
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

// deleteOldDeliveryServiceSSLKeys deletes all the SSL keys in Riak for delivery services in the given CDN that are not in the given existingXMLIDs.
func deleteOldDeliveryServiceSSLKeys(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, cdn tc.CDNName, existingXMLIDs map[string]struct{}) error {
	dsVersions := map[string][]string{}
	err := withCluster(tx, authOpts, riakPort, func(cluster StorageCluster) error {
		// get the deliveryservice ssl keys by xmlID and version
		query := `cdn:` + string(cdn)
		filterQuery := ""
		fields := []string{"_yz_rk", "deliveryservice"} // '_yz_rk' is the magic Riak field that populates the key. Without this, doc.Key would be empty.
		searchDocs, err := search(cluster, sslKeysIndex, query, filterQuery, cdnSSLKeysLimit, fields)
		if err != nil {
			return errors.New("riak search error: " + err.Error())
		}
		if len(searchDocs) == 0 {
			return nil // no error, and leave keys empty
		}

		for _, doc := range searchDocs {
			dses := doc.Fields["deliveryservice"]
			if len(dses) == 0 {
				log.Errorln("Riak had a CDN '" + string(cdn) + "' key with no delivery service '" + doc.Key + "' - ignoring!")
				continue
			}
			if len(dses) > 1 {
				log.Errorf("Riak had a CDN '"+string(cdn)+"' key with multiple delivery services '"+doc.Key+"' deliveryservices '%+v' - ignoring all but the first!\n", dses)
			}
			ds := dses[0]

			dsVersions[ds] = append(dsVersions[ds], doc.Key)
		}
		return nil
	})
	if err != nil {
		return errors.New("with cluster error: " + err.Error())
	}

	successes := []string{}
	failures := []string{}
	for ds, riakKeys := range dsVersions {
		if _, ok := existingXMLIDs[ds]; ok {
			continue
		}
		for _, riakKey := range riakKeys {
			err := deleteDeliveryServicesSSLKey(tx, authOpts, riakPort, riakKey)
			if err != nil {
				log.Errorln("deleting Traffic Vault SSL keys for Delivery Service '" + ds + "' key '" + riakKey + "': " + err.Error())
				failures = append(failures, ds)
			} else {
				log.Infoln("Deleted Traffic Vault SSL keys for delivery service which has been deleted in the database '" + string(ds) + "' key '" + riakKey + "'")
				successes = append(successes, ds)
			}
		}
	}
	if len(failures) > 0 {
		return errors.New("successfully deleted Traffic Vault SSL keys for deleted dses [" + strings.Join(successes, ", ") + "], but failed to delete Traffic Vault SSL keys for [" + strings.Join(failures, ", ") + "]; see the error log for details")
	}
	return nil
}
