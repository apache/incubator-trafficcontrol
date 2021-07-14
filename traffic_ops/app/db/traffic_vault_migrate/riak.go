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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/basho/riak-go-client"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

const (
	BUCKET_SSL     = "ssl"
	BUCKET_DNSSEC  = "dnssec"
	BUCKET_URL_SIG = "url_sig_keys"
	BUCKET_URI_SIG = "cdn_uri_sig_keys"

	INDEX_SSL = "sslkeys"

	SCHEMA_RIAK_KEY    = "_yz_rk"
	SCHEMA_RIAK_BUCKET = "_yz_rb"
)

var (
	SCHEMA_SSL_FIELDS = [...]string{SCHEMA_RIAK_KEY, SCHEMA_RIAK_BUCKET}
)

// RiakConfig  represents the configuration options available to the Riak backend.
type RiakConfig struct {
	Host          string `json:"host"`
	Port          string `json:"port"`
	User          string `json:"user"`
	Password      string `json:"password"`
	Insecure      bool   `json:"insecure"`
	TLSVersionRaw string `json:"tlsVersion"`
	// Timeout is the number of seconds each command should use.
	Timeout int `json:"timeout"`

	TLSVersion uint16 `json:"-"`
}

// RiakBackend is the Riak implementation of TVBackend.
type RiakBackend struct {
	sslKeys        riakSSLKeyTable
	dnssecKeys     riakDNSSecKeyTable
	uriSigningKeys riakURISignKeyTable
	urlSigKeys     riakURLSigKeyTable
	cfg            RiakConfig
	cluster        *riak.Cluster
}

// String returns a high level overview of the backend and its keys.
func (rb *RiakBackend) String() string {
	data := fmt.Sprintf("Riak server %s@%s:%s\n", rb.cfg.User, rb.cfg.Host, rb.cfg.Port)
	data += fmt.Sprintf("\tSSL Keys: %d\n", len(rb.sslKeys.Records))
	data += fmt.Sprintf("\tDNSSec Keys: %d\n", len(rb.dnssecKeys.Records))
	data += fmt.Sprintf("\tURI Signing Keys: %d\n", len(rb.uriSigningKeys.Records))
	data += fmt.Sprintf("\tURL Sig Keys: %d\n", len(rb.urlSigKeys.Records))
	return data
}

// Name returns the name for this backend.
func (rb *RiakBackend) Name() string {
	return "Riak"
}

// ReadConfigFile takes in a filename and will read it into the backends config.
func (rb *RiakBackend) ReadConfigFile(configFile string) error {
	err := UnmarshalConfig(configFile, &rb.cfg)
	if err != nil {
		return err
	}

	switch rb.cfg.TLSVersionRaw {
	case "10":
		rb.cfg.TLSVersion = tls.VersionTLS10
	case "11":
		rb.cfg.TLSVersion = tls.VersionTLS11
	case "12":
		rb.cfg.TLSVersion = tls.VersionTLS12
	case "13":
		rb.cfg.TLSVersion = tls.VersionTLS13
	default:
		return fmt.Errorf("unknown tls version " + rb.cfg.TLSVersionRaw)
	}
	return nil
}

// Insert takes the current keys and inserts them into the backend DB.
func (rb *RiakBackend) Insert() error {
	if err := rb.sslKeys.insertKeys(rb.cluster, rb.cfg.Timeout); err != nil {
		return err
	}
	if err := rb.dnssecKeys.insertKeys(rb.cluster, rb.cfg.Timeout); err != nil {
		return err
	}
	if err := rb.urlSigKeys.insertKeys(rb.cluster, rb.cfg.Timeout); err != nil {
		return err
	}
	if err := rb.uriSigningKeys.insertKeys(rb.cluster, rb.cfg.Timeout); err != nil {
		return err
	}
	return nil
}

// ValidateKey validates that the keys are valid (in most cases, certain fields are not null).
func (rb *RiakBackend) ValidateKey() []string {
	allErrs := []string{}
	if errs := rb.sslKeys.validate(); errs != nil {
		allErrs = append(allErrs, errs...)
	}
	if errs := rb.dnssecKeys.validate(); errs != nil {
		allErrs = append(allErrs, errs...)
	}
	if errs := rb.uriSigningKeys.validate(); errs != nil {
		allErrs = append(allErrs, errs...)
	}
	if errs := rb.urlSigKeys.validate(); errs != nil {
		allErrs = append(allErrs, errs...)
	}

	return allErrs
}

// SetSSLKeys takes in keys and converts & encrypts the data into the backends internal format.
func (rb *RiakBackend) SetSSLKeys(keys []SSLKey) error {
	rb.sslKeys.fromGeneric(keys)
	return nil
}

// SetDNSSecKeys takes in keys and converts & encrypts the data into the backends internal format.
func (rb *RiakBackend) SetDNSSecKeys(keys []DNSSecKey) error {
	rb.dnssecKeys.fromGeneric(keys)
	return nil
}

// SetURISignKeys takes in keys and converts & encrypts the data into the backends internal format.
func (rb *RiakBackend) SetURISignKeys(keys []URISignKey) error {
	rb.uriSigningKeys.fromGeneric(keys)
	return nil
}

// SetURLSigKeys takes in keys and converts & encrypts the data into the backends internal format.
func (rb *RiakBackend) SetURLSigKeys(keys []URLSigKey) error {
	rb.urlSigKeys.fromGeneric(keys)
	return nil
}

// Start initiates the connection to the backend DB.
func (rb *RiakBackend) Start() error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: rb.cfg.Insecure,
		MaxVersion:         rb.cfg.TLSVersion,
	}
	auth := &riak.AuthOptions{
		User:      rb.cfg.User,
		Password:  rb.cfg.Password,
		TlsConfig: tlsConfig,
	}

	cluster, err := getRiakCluster(rb.cfg, auth)
	if err != nil {
		return err
	}
	if err := cluster.Start(); err != nil {
		return fmt.Errorf("unable to start riak cluster: %w", err)
	}

	rb.cluster = cluster
	rb.sslKeys = riakSSLKeyTable{}
	rb.dnssecKeys = riakDNSSecKeyTable{}
	rb.urlSigKeys = riakURLSigKeyTable{}
	rb.uriSigningKeys = riakURISignKeyTable{}
	return nil
}

// Close terminates the connection to the backend DB.
func (rb *RiakBackend) Close() error {
	if err := rb.cluster.Stop(); err != nil {
		return err
	}
	return nil
}

// Ping checks the connection to the backend DB.
func (rb *RiakBackend) Ping() error {
	return ping(rb.cluster)
}

// GetSSLKeys converts the backends internal key representation into the common representation (SSLKey).
func (rb *RiakBackend) GetSSLKeys() ([]SSLKey, error) {
	return rb.sslKeys.toGeneric(), nil
}

// GetDNSSecKeys converts the backends internal key representation into the common representation (DNSSecKey).
func (rb *RiakBackend) GetDNSSecKeys() ([]DNSSecKey, error) {
	return rb.dnssecKeys.toGeneric(), nil
}

// GetURISignKeys converts the pg internal key representation into the common representation (URISignKey).
func (rb *RiakBackend) GetURISignKeys() ([]URISignKey, error) {
	return rb.uriSigningKeys.toGeneric(), nil
}

// GetURLSigKeys converts the backends internal key representation into the common representation (URLSigKey).
func (rb *RiakBackend) GetURLSigKeys() ([]URLSigKey, error) {
	return rb.urlSigKeys.toGeneric(), nil
}

// Fetch gets all of the keys from the backend DB.
func (rb *RiakBackend) Fetch() error {
	if err := rb.sslKeys.gatherKeys(rb.cluster, rb.cfg.Timeout); err != nil {
		return err
	}
	if err := rb.dnssecKeys.gatherKeys(rb.cluster, rb.cfg.Timeout); err != nil {
		return err
	}
	if err := rb.urlSigKeys.gatherKeys(rb.cluster, rb.cfg.Timeout); err != nil {
		return err
	}
	if err := rb.uriSigningKeys.gatherKeys(rb.cluster, rb.cfg.Timeout); err != nil {
		return err
	}

	return nil
}

type riakSSLKeyRecord struct {
	tc.DeliveryServiceSSLKeys
	Version string
}
type riakSSLKeyTable struct {
	Records []riakSSLKeyRecord
}

func (tbl *riakSSLKeyTable) gatherKeys(cluster *riak.Cluster, timeout int) error {
	searchDocs, err := search(cluster, INDEX_SSL, "cdn:*", "", 1000, SCHEMA_SSL_FIELDS[:])
	if err != nil {
		return fmt.Errorf("RiakSSLKey gatherKeys: %w", err)
	}

	tbl.Records = make([]riakSSLKeyRecord, len(searchDocs))
	for i, doc := range searchDocs {
		objs, err := getObject(cluster, doc.Bucket, doc.Key, timeout)
		if err != nil {
			return err
		}
		if len(objs) < 1 {
			return fmt.Errorf("RiakSSLKey gatherKeys unable to find any objects with key %s and bucket %s, but search results were returned", doc.Key, doc.Bucket)
		}
		if len(objs) > 1 {
			return fmt.Errorf("RiakSSLKey gatherKeys key '%s' more than 1 ssl key record found %d\n", doc.Key, len(objs))
		}
		var obj tc.DeliveryServiceSSLKeys
		if err = json.Unmarshal(objs[0].Value, &obj); err != nil {
			return fmt.Errorf("RiakSSLKey gatherKeys key '%s' unable to unmarshal object into tc.DeliveryServiceSSLKeys: %w", doc.Key, err)
		}
		tbl.Records[i] = riakSSLKeyRecord{
			DeliveryServiceSSLKeys: obj,
			Version:                strings.Split(objs[0].Key, "-")[1],
		}
	}
	return nil
}
func (tbl *riakSSLKeyTable) toGeneric() []SSLKey {
	keys := make([]SSLKey, len(tbl.Records))

	for i, record := range tbl.Records {
		keys[i] = SSLKey{
			DeliveryServiceSSLKeys: record.DeliveryServiceSSLKeys,
			Version:                record.Version,
		}
	}

	return keys
}
func (tbl *riakSSLKeyTable) fromGeneric(keys []SSLKey) {
	tbl.Records = make([]riakSSLKeyRecord, len(keys))

	for i, record := range keys {
		tbl.Records[i] = riakSSLKeyRecord{
			DeliveryServiceSSLKeys: record.DeliveryServiceSSLKeys,
			Version:                record.Version,
		}
	}
}
func (tbl *riakSSLKeyTable) insertKeys(cluster *riak.Cluster, timeout int) error {
	for _, record := range tbl.Records {
		objBytes, err := json.Marshal(record.DeliveryServiceSSLKeys)
		if err != nil {
			return fmt.Errorf("RiakSSLKey insertKeys '%s' failed to marshal keys: %w", record.Key, err)
		}
		if err = setObject(cluster, makeRiakObject(objBytes, record.DeliveryService+"-"+record.Version), BUCKET_SSL, timeout); err != nil {
			return fmt.Errorf("RiakSSLKey insertKeys '%s': %w", record.Key, err)
		}
	}
	return nil
}
func (tbl *riakSSLKeyTable) validate() []string {
	errs := []string{}
	for _, record := range tbl.Records {
		if record.DeliveryService == "" {
			errs = append(errs, fmt.Sprintf("SSL Key '%s': Delivery Service is blank!", record.Key))
		}
		if record.CDN == "" {
			errs = append(errs, fmt.Sprintf("SSL Key '%s': CDN is blank!", record.Key))
		}
		if record.Version == "" {
			errs = append(errs, fmt.Sprintf("SSL Key '%s': Version is blank!", record.Key))
		}
	}
	return errs
}

type riakDNSSecKeyRecord struct {
	CDN string
	Key tc.DNSSECKeysRiak
}
type riakDNSSecKeyTable struct {
	Records []riakDNSSecKeyRecord
}

func (tbl *riakDNSSecKeyTable) gatherKeys(cluster *riak.Cluster, timeout int) error {
	tbl.Records = []riakDNSSecKeyRecord{}
	objs, err := getObjects(cluster, BUCKET_DNSSEC, timeout)
	if err != nil {
		return fmt.Errorf("RiakDNSSecKey gatherKeys: %w", err)
	}
	for _, obj := range objs {
		key := tc.DNSSECKeysRiak{}
		if err := json.Unmarshal(obj.Value, &key); err != nil {
			return fmt.Errorf("RiakDNSSecKey gatherKeys '%s' unable to unmarshal object to tc.DNSSECKeysRiak: %w", obj.Key, err)
		}
		tbl.Records = append(tbl.Records, riakDNSSecKeyRecord{
			CDN: obj.Key,
			Key: key,
		})
	}
	return nil
}
func (tbl *riakDNSSecKeyTable) toGeneric() []DNSSecKey {
	keys := make([]DNSSecKey, len(tbl.Records))

	for i, record := range tbl.Records {
		keys[i] = DNSSecKey{
			CDN:                    record.CDN,
			DNSSECKeysTrafficVault: tc.DNSSECKeysTrafficVault(record.Key),
		}
	}

	return keys
}
func (tbl *riakDNSSecKeyTable) fromGeneric(keys []DNSSecKey) {
	tbl.Records = make([]riakDNSSecKeyRecord, len(keys))

	for i, record := range keys {
		tbl.Records[i] = riakDNSSecKeyRecord{
			CDN: record.CDN,
			Key: tc.DNSSECKeysRiak(record.DNSSECKeysTrafficVault),
		}
	}
}
func (tbl *riakDNSSecKeyTable) insertKeys(cluster *riak.Cluster, timeout int) error {
	for _, record := range tbl.Records {
		objBytes, err := json.Marshal(record.Key)
		if err != nil {
			return fmt.Errorf("RiakDNSSecKey insertKeys '%s' error marshalling keys: %w", record.CDN, err)
		}

		if err = setObject(cluster, makeRiakObject(objBytes, record.CDN), BUCKET_DNSSEC, timeout); err != nil {
			return fmt.Errorf("RiakDNSSecKey insertKeys '%s': %w", record.CDN, err)
		}
	}
	return nil
}
func (tbl *riakDNSSecKeyTable) validate() []string {
	errs := []string{}
	for i, record := range tbl.Records {
		if record.CDN == "" {
			errs = append(errs, fmt.Sprintf("DNSSec Key #%d: CDN is blank!", i))
		}
	}
	return errs
}

type riakURLSigKeyRecord struct {
	Key             tc.URLSigKeys
	DeliveryService string
}
type riakURLSigKeyTable struct {
	Records []riakURLSigKeyRecord
}

func (tbl *riakURLSigKeyTable) gatherKeys(cluster *riak.Cluster, timeout int) error {
	tbl.Records = []riakURLSigKeyRecord{}
	objs, err := getObjects(cluster, BUCKET_URL_SIG, timeout)
	if err != nil {
		return fmt.Errorf("RiakURLSigKey gatherKeys: %w", err)
	}
	for _, obj := range objs {
		key := tc.URLSigKeys{}
		if err := json.Unmarshal(obj.Value, &key); err != nil {
			return fmt.Errorf("RiakURLSigKey gatherKeys '%s' unable to unamrshal object into tc.URLSigKeys: %w", obj.Key, err)
		}
		tbl.Records = append(tbl.Records, riakURLSigKeyRecord{
			DeliveryService: obj.Key,
			Key:             key,
		})
	}
	return nil
}
func (tbl *riakURLSigKeyTable) toGeneric() []URLSigKey {
	keys := make([]URLSigKey, len(tbl.Records))

	for i, record := range tbl.Records {
		keys[i] = URLSigKey{
			URLSigKeys:      record.Key,
			DeliveryService: record.DeliveryService,
		}
	}

	return keys
}
func (tbl *riakURLSigKeyTable) fromGeneric(keys []URLSigKey) {
	tbl.Records = make([]riakURLSigKeyRecord, len(keys))
	for i, key := range keys {
		tbl.Records[i] = riakURLSigKeyRecord{
			Key:             key.URLSigKeys,
			DeliveryService: key.DeliveryService,
		}
	}
}
func (tbl *riakURLSigKeyTable) insertKeys(cluster *riak.Cluster, timeout int) error {
	for _, record := range tbl.Records {
		objBytes, err := json.Marshal(record.Key)
		if err != nil {
			return fmt.Errorf("RiakURLSigKey insertKeys '%s' unable to marshal keys: %w", record.DeliveryService, err)
		}

		if err = setObject(cluster, makeRiakObject(objBytes, "url_sig_"+record.DeliveryService+".config"), BUCKET_URL_SIG, timeout); err != nil {
			return fmt.Errorf("RiakURLSigKey insertKeys '%s': %w", record.DeliveryService, err)
		}
	}
	return nil
}
func (tbl *riakURLSigKeyTable) validate() []string {
	errs := []string{}
	for i, record := range tbl.Records {
		if record.DeliveryService == "" {
			errs = append(errs, fmt.Sprintf("URL Key #%d: Delivery Service is blank!", i))
		}
	}
	return errs
}

type riakURISignKeyRecord struct {
	Key             map[string]tc.URISignerKeyset
	DeliveryService string
}
type riakURISignKeyTable struct {
	Records []riakURISignKeyRecord
}

func (tbl *riakURISignKeyTable) gatherKeys(cluster *riak.Cluster, timeout int) error {
	tbl.Records = []riakURISignKeyRecord{}
	objs, err := getObjects(cluster, BUCKET_URI_SIG, timeout)
	if err != nil {
		return fmt.Errorf("RiakURISignKey gatherKeys: %w", err)
	}
	for _, obj := range objs {
		key := map[string]tc.URISignerKeyset{}
		if err := json.Unmarshal(obj.Value, &key); err != nil {
			return fmt.Errorf("RiakURISignKey gatherKeys '%s' unable to unmarshal object into map[string]tc.URISignerKeySet: %w", obj.Key, err)
		}

		tbl.Records = append(tbl.Records, riakURISignKeyRecord{
			DeliveryService: obj.Key,
			Key:             key,
		})
	}
	return nil
}
func (tbl *riakURISignKeyTable) toGeneric() []URISignKey {
	keys := make([]URISignKey, len(tbl.Records))

	for i, record := range tbl.Records {
		keys[i] = URISignKey{
			DeliveryService: record.DeliveryService,
			Keys:            record.Key,
		}
	}

	return keys
}
func (tbl *riakURISignKeyTable) fromGeneric(keys []URISignKey) {
	tbl.Records = make([]riakURISignKeyRecord, len(keys))

	for i, record := range keys {
		tbl.Records[i] = riakURISignKeyRecord{
			Key:             record.Keys,
			DeliveryService: record.DeliveryService,
		}
	}
}
func (tbl *riakURISignKeyTable) insertKeys(cluster *riak.Cluster, timeout int) error {
	for _, record := range tbl.Records {
		objBytes, err := json.Marshal(record.Key)
		if err != nil {
			return fmt.Errorf("RiakURISignKey insertKeys '%s': unable to marshal key: %w", record.DeliveryService, err)
		}

		if err = setObject(cluster, makeRiakObject(objBytes, record.DeliveryService), BUCKET_URI_SIG, timeout); err != nil {
			return fmt.Errorf("RiakURISignKey insertKeys '%s': %w", record.DeliveryService, err)
		}
	}
	return nil
}
func (tbl *riakURISignKeyTable) validate() []string {
	errs := []string{}
	for i, record := range tbl.Records {
		if record.DeliveryService == "" {
			errs = append(errs, fmt.Sprintf("URI Signing Key #%d: Delivery Service is blank!", i))
		}
	}
	return errs
}

// Riak functions
func makeRiakObject(data []byte, key string) *riak.Object {
	return &riak.Object{
		ContentType:     rfc.ApplicationJSON,
		Charset:         "utf-8",
		ContentEncoding: "utf-8",
		Key:             key,
		Value:           data,
	}
}
func getObjects(cluster *riak.Cluster, bucket string, timeout int) ([]*riak.Object, error) {
	objs := []*riak.Object{}
	keys, err := listKeys(cluster, bucket, timeout)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		objects, err := getObject(cluster, bucket, key, timeout)
		if err != nil {
			return nil, err
		}
		if len(objects) > 1 {
			return nil, fmt.Errorf("Unexpected number of objects %d\n", len(objects))
		}

		objs = append(objs, objects[0])
	}

	return objs, nil
}
func getObject(cluster *riak.Cluster, bucket string, key string, timeout int) ([]*riak.Object, error) {
	cmd, err := riak.NewFetchValueCommandBuilder().
		WithBucket(bucket).
		WithKey(key).
		WithTimeout(time.Second * time.Duration(timeout)).
		Build()
	if err != nil {
		return nil, fmt.Errorf("error building riak fetch value command: %w", err)
	}

	if err := cluster.Execute(cmd); err != nil {
		return nil, fmt.Errorf("error executing riak fetch value command: %w", err)
	}

	fvc := cmd.(*riak.FetchValueCommand)
	rsp := fvc.Response

	if rsp.IsNotFound {
		return nil, fmt.Errorf("errors executing riak fetch value command; key not found: %s:%s", bucket, key)
	}

	return rsp.Values, nil
}
func setObject(cluster *riak.Cluster, obj *riak.Object, bucket string, timeout int) error {
	cmd, err := riak.NewStoreValueCommandBuilder().
		WithBucket(bucket).
		WithContent(obj).
		WithTimeout(time.Second * time.Duration(timeout)).
		Build()
	if err != nil {
		return fmt.Errorf("error building riak store value command: %w", err)
	}

	return cluster.Execute(cmd)
}
func search(cluster *riak.Cluster, index string, query string, filterQuery string, numRows uint32, fields []string) ([]*riak.SearchDoc, error) {
	var searchDocs []*riak.SearchDoc
	start := uint32(0)
	for i := uint32(0); ; i += 1 {
		riakCmd := riak.NewSearchCommandBuilder().
			WithQuery(query).
			WithNumRows(numRows).
			WithStart(start)
		if len(index) > 0 {
			riakCmd = riakCmd.WithIndexName(index)
		}
		if len(filterQuery) > 0 {
			riakCmd = riakCmd.WithFilterQuery(filterQuery)
		}
		if len(fields) > 0 {
			riakCmd = riakCmd.WithReturnFields(fields...)
		}
		iCmd, err := riakCmd.Build()

		if err != nil {
			return nil, fmt.Errorf("building Riak search command: %w", err)
		}
		if err = cluster.Execute(iCmd); err != nil {
			return nil, fmt.Errorf("executing Riak search command index '%s' query '%s': %w", index, query, err)
		}
		cmd, ok := iCmd.(*riak.SearchCommand)
		if !ok {
			return nil, fmt.Errorf("riak search command unexpected type %T", iCmd)
		}
		if cmd.Response == nil {
			return nil, errors.New("riak received nil response")
		}
		if start == 0 {
			if cmd.Response.NumFound == 0 {
				return nil, nil
			}
			if cmd.Response.NumFound <= numRows {
				return cmd.Response.Docs, nil
			} else if numRows < cmd.Response.NumFound {
				searchDocs = make([]*riak.SearchDoc, cmd.Response.NumFound)
			}
			if cmd.Response.NumFound > numRows*10 {
				fmt.Printf("WARNING: found %d rows, press enter to continue", cmd.Response.NumFound)
				_, _ = fmt.Scanln()
			}
		}

		// If the total number of docs is not evenly divisible by 1000.
		if uint32(len(cmd.Response.Docs)) < numRows {
			numRows = uint32(len(cmd.Response.Docs))
		}

		for responseIndex := uint32(0); responseIndex < numRows; responseIndex += 1 {
			returnIndex := responseIndex + start
			searchDocs[returnIndex] = cmd.Response.Docs[responseIndex]
		}
		if cmd.Response.NumFound == numRows+start {
			return searchDocs, nil
		}
		start += numRows
	}
}
func listKeys(cluster *riak.Cluster, bucket string, timeout int) ([]string, error) {
	cmd, err := riak.NewListKeysCommandBuilder().
		WithBucket(bucket).
		WithTimeout(time.Second * time.Duration(timeout)).
		WithAllowListing().
		Build()
	if err != nil {
		return nil, errors.New("building riak list keys command failed: " + err.Error())
	}

	if err := cluster.Execute(cmd); err != nil {
		return nil, errors.New("error executing riak list keys command: " + err.Error())
	}

	lkc := cmd.(*riak.ListKeysCommand)
	rsp := lkc.Response
	return rsp.Keys, nil
}
func ping(cluster *riak.Cluster) error {
	ping := riak.PingCommandBuilder{}
	cmd, err := ping.Build()
	if err != nil {
		return errors.New("failed to build riak ping command: " + err.Error())
	}

	if err = cluster.Execute(cmd); err != nil {
		return err
	}
	response, ok := cmd.(*riak.PingCommand)
	if !ok {
		return fmt.Errorf("unexpected riak command type for ping: %v", cmd)
	}

	if response.Error() != nil {
		return errors.New("riak ping command response error: " + response.Error().Error())
	}

	if !response.Success() {
		return errors.New("riak ping command returned unsuccessfully")
	}

	return nil
}
func getRiakCluster(srv RiakConfig, authOptions *riak.AuthOptions) (*riak.Cluster, error) {
	if authOptions == nil {
		return nil, errors.New("no riak auth information from riak.conf, cannot authenticate to any riak servers")
	}
	nodes := []*riak.Node{}
	nodeOpts := &riak.NodeOptions{
		RemoteAddress:       srv.Host + ":" + srv.Port,
		AuthOptions:         authOptions,
		HealthCheckInterval: time.Second * 5,
	}
	if nodeOpts.AuthOptions.TlsConfig != nil {
		nodeOpts.AuthOptions.TlsConfig.ServerName = srv.Host
	}
	node, err := riak.NewNode(nodeOpts)
	if err != nil {
		return nil, errors.New("creating riak node: " + err.Error())
	}
	nodes = append(nodes, node)
	if len(nodes) == 0 {
		return nil, errors.New("no available riak servers")
	}
	opts := &riak.ClusterOptions{
		Nodes:             nodes,
		ExecutionAttempts: 2,
	}
	cluster, err := riak.NewCluster(opts)
	if err != nil {
		return nil, errors.New("creating riak cluster: " + err.Error())
	}
	return cluster, err
}
