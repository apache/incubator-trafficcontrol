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
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"

	"github.com/basho/riak-go-client"
)

const (
	RiakPort = 8087
	TimeOut  = time.Second * 5

	DefaultHealthCheckInterval         = time.Second * 5
	DefaultMaxCommandExecutionAttempts = 5
)

var (
	clusterServers []ServerAddr
	sharedCluster  *riak.Cluster
	clusterMutex   sync.Mutex

	healthCheckInterval time.Duration
)

type AuthOptions riak.AuthOptions

// StorageCluster ...
type StorageCluster interface {
	Start() error
	Stop() error
	Execute(riak.Command) error
}

// RiakStorageCluster ...
type RiakStorageCluster struct {
	Cluster *riak.Cluster
}

// Stop ...
func (ri RiakStorageCluster) Stop() error {
	return ri.Cluster.Stop()
}

// Start ...
func (ri RiakStorageCluster) Start() error {
	return ri.Cluster.Start()
}

// Execute ...
func (ri RiakStorageCluster) Execute(command riak.Command) error {
	return ri.Cluster.Execute(command)
}

func GetRiakConfig(riakConfigFile string) (bool, *riak.AuthOptions, error) {
	riakConfString, err := ioutil.ReadFile(riakConfigFile)
	if err != nil {
		return false, nil, fmt.Errorf("reading riak conf '%v': %v", riakConfigFile, err)
	}

	riakConfBytes := []byte(riakConfString)

	rconf := &riak.AuthOptions{}
	rconf.TlsConfig = &tls.Config{}
	err = json.Unmarshal(riakConfBytes, &rconf)
	if err != nil {
		return false, nil, fmt.Errorf("Unmarshaling riak conf '%v': %v", riakConfigFile, err)
	}

	type config struct {
		Hci string `json:"HealthCheckInterval"`
	}

	var checkconfig config
	err = json.Unmarshal(riakConfBytes, &checkconfig)
	if err == nil {
		hci, _ := time.ParseDuration(checkconfig.Hci)
		if 0 < hci {
			healthCheckInterval = hci
		}
	} else {
		log.Infoln("Error unmarshalling riak config options: " + err.Error())
	}

	if healthCheckInterval <= 0 {
		healthCheckInterval = DefaultHealthCheckInterval
		log.Infoln("HeathCheckInterval override")
	}

	log.Infoln("Riak health check interval set to:", healthCheckInterval)

	return true, rconf, nil
}

// deletes an object from riak storage
func DeleteObject(key string, bucket string, cluster StorageCluster) error {
	if cluster == nil {
		return errors.New("ERROR: No valid cluster on which to execute a command")
	}

	// build store command and execute.
	cmd, err := riak.NewDeleteValueCommandBuilder().
		WithBucket(bucket).
		WithKey(key).
		WithTimeout(TimeOut).
		Build()
	if err != nil {
		return err
	}

	return cluster.Execute(cmd)
}

// PingCluster pings the given Riak cluster, and returns nil on success, or any error
func PingCluster(cluster StorageCluster) error {
	if cluster == nil {
		return errors.New("ERROR: No valid cluster on which to execute a command")
	}
	pingCommandBuilder := riak.PingCommandBuilder{}
	iCmd, err := pingCommandBuilder.Build()
	if err != nil {
		return errors.New("building riak ping command: " + err.Error())
	}
	if err := cluster.Execute(iCmd); err != nil {
		return errors.New("executing riak ping command: " + err.Error())
	}
	cmd, ok := iCmd.(*riak.PingCommand)
	if !ok {
		return fmt.Errorf("unexpected riak command type: %T", iCmd)
	}
	if err := cmd.Error(); err != nil {
		return errors.New("riak ping command returned error: " + err.Error())
	}
	if !cmd.Success() {
		return errors.New("riak ping command returned failure, but no error")
	}
	return nil
}

// fetch an object from riak storage
func FetchObjectValues(key string, bucket string, cluster StorageCluster) ([]*riak.Object, error) {
	if cluster == nil {
		return nil, errors.New("ERROR: No valid cluster on which to execute a command")
	}
	// build the fetch command
	cmd, err := riak.NewFetchValueCommandBuilder().
		WithBucket(bucket).
		WithKey(key).
		WithTimeout(TimeOut).
		Build()
	if err != nil {
		return nil, err
	}

	if err = cluster.Execute(cmd); err != nil {
		return nil, err
	}
	fvc := cmd.(*riak.FetchValueCommand)

	// no object found with given key
	if fvc.Response == nil || fvc.Response.IsNotFound {
		return nil, nil
	}
	return fvc.Response.Values, nil
}

// saves an object to riak storage
func SaveObject(obj *riak.Object, bucket string, cluster StorageCluster) error {
	if cluster == nil {
		return errors.New("ERROR: No valid cluster on which to execute a command")
	}
	if obj == nil {
		return errors.New("ERROR: cannot save a nil object")
	}
	// build store command and execute.
	cmd, err := riak.NewStoreValueCommandBuilder().
		WithBucket(bucket).
		WithContent(obj).
		WithTimeout(TimeOut).
		Build()
	if err != nil {
		return err
	}

	return cluster.Execute(cmd)
}

type ServerAddr struct {
	FQDN string
	Port string
}

func GetRiakServers(tx *sql.Tx) ([]ServerAddr, error) {
	rows, err := tx.Query(`
SELECT CONCAT(s.host_name, '.', s.domain_name) FROM server s
JOIN type t ON s.type = t.id
JOIN status st ON s.status = st.id
WHERE t.name = 'RIAK' AND st.name = 'ONLINE'
`)
	if err != nil {
		return nil, errors.New("querying riak servers: " + err.Error())
	}
	defer rows.Close()
	servers := []ServerAddr{}
	portStr := strconv.Itoa(RiakPort)
	for rows.Next() {
		s := ServerAddr{Port: portStr}
		if err := rows.Scan(&s.FQDN); err != nil {
			return nil, errors.New("scanning riak servers: " + err.Error())
		}
		servers = append(servers, s)
	}

	return servers, nil
}

func GetRiakCluster(servers []ServerAddr, authOptions *riak.AuthOptions) (*riak.Cluster, error) {
	if authOptions == nil {
		return nil, errors.New("ERROR: no riak auth information from riak.conf, cannot authenticate to any riak servers")
	}
	nodes := []*riak.Node{}
	for _, srv := range servers {
		nodeOpts := &riak.NodeOptions{
			RemoteAddress:       srv.FQDN + ":" + srv.Port,
			AuthOptions:         authOptions,
			HealthCheckInterval: healthCheckInterval,
		}
		nodeOpts.AuthOptions.TlsConfig.ServerName = srv.FQDN
		node, err := riak.NewNode(nodeOpts)
		if err != nil {
			return nil, errors.New("creating riak node: " + err.Error())
		}
		nodes = append(nodes, node)
	}
	if len(nodes) == 0 {
		return nil, errors.New("ERROR: no available riak servers")
	}
	opts := &riak.ClusterOptions{
		Nodes:             nodes,
		ExecutionAttempts: DefaultMaxCommandExecutionAttempts,
	}
	cluster, err := riak.NewCluster(opts)
	if err != nil {
		return nil, errors.New("creating riak cluster: " + err.Error())
	}
	return cluster, err
}

func GetRiakStorageCluster(servers []ServerAddr, authOptions *riak.AuthOptions) (StorageCluster, error) {
	cluster, err := GetRiakCluster(servers, authOptions)
	if err != nil {
		return nil, err
	}
	return RiakStorageCluster{Cluster: cluster}, nil
}

func GetPooledCluster(tx *sql.Tx, authOptions *riak.AuthOptions) (StorageCluster, error) {

	clusterMutex.Lock()
	defer clusterMutex.Unlock()

	tryLoad := false

	// should we try to reload the cluster?
	newservers, err := GetRiakServers(tx)

	if err == nil {
		if 0 < len(newservers) {
			sort.Slice(newservers, func(ii, jj int) bool {
				return newservers[ii].FQDN < newservers[jj].FQDN ||
					(newservers[ii].FQDN == newservers[jj].FQDN && newservers[ii].Port < newservers[jj].Port)
			})
			if !reflect.DeepEqual(newservers, clusterServers) {
				tryLoad = true
				log.Infoln("Attempting to load a new set of riak servers")
				log.Infoln("new riak servers")
				for _, srv := range newservers {
					log.Infoln(" ", srv.FQDN+":"+srv.Port)
				}
			}
		}
	} else {
		log.Errorln("getting riak servers: " + err.Error())
	}

	if tryLoad {
		newcluster, err := GetRiakCluster(newservers, authOptions)
		if err == nil {
			if err := newcluster.Start(); err == nil {
				log.Infoln("New cluster started")

				if sharedCluster != nil {
					runtime.SetFinalizer(sharedCluster, sharedCluster.Stop())
				}

				sharedCluster = newcluster
				clusterServers = newservers
			} else {
				log.Errorln("starting riak cluster, reverting to previous: " + err.Error())
			}
		} else {
			log.Errorln("creating riak cluster, reverting to previous: " + err.Error())
		}
	}

	cluster := sharedCluster

	if cluster == nil {
		log.Errorln("GetPooledCluster failed, returning nil cluster")
		return nil, errors.New("GetPooledCluster unable to return cluster")
	}

	return RiakStorageCluster{Cluster: cluster}, nil
}

// Search searches Riak for the given query. Returns nil and a nil error if no object was found.
func Search(cluster StorageCluster, index string, query string, filterQuery string, numRows int) ([]*riak.SearchDoc, error) {
	iCmd, err := riak.NewSearchCommandBuilder().
		WithIndexName(index).
		WithQuery(query).
		WithFilterQuery(filterQuery).
		WithNumRows(uint32(numRows)).
		Build()
	if err != nil {
		return nil, errors.New("building Riak command: " + err.Error())
	}
	if err = cluster.Execute(iCmd); err != nil {
		return nil, errors.New("executing Riak command index '" + index + "' query '" + query + "': " + err.Error())
	}
	cmd, ok := iCmd.(*riak.SearchCommand)
	if !ok {
		return nil, fmt.Errorf("Riak command unexpected type %T", iCmd)
	}
	if cmd.Response == nil || cmd.Response.NumFound == 0 {
		return nil, nil
	}
	return cmd.Response.Docs, nil
}
