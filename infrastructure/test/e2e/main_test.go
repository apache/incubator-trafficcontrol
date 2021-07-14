package e2e

/*
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	toclient "github.com/apache/trafficcontrol/traffic_ops/client"
)

var Cfg *Config
var TOClient *toclient.Session

var TO struct {
	DeliveryServices       map[tc.DeliveryServiceName]tc.DeliveryService // TODO change to use CRConfig?
	Servers                map[tc.CacheName]tc.Server
	ServerDeliveryServices map[tc.CacheName][]tc.DeliveryServiceName
	DeliveryServiceServers map[tc.DeliveryServiceName][]tc.CacheName
}

const DefaultConfigPath = "e2e.cfg.json"

const Version = "0.1"
const UserAgent = "traffic-control-e2e-test/" + Version
const TOTimeout = 5 * time.Second

func TestMain(m *testing.M) {
	cfgFileName := flag.String("cfg", "", "The config file path")
	flag.Parse()

	if *cfgFileName == "" {
		fmt.Println("Error: config file flag 'cfg' is required")
		os.Exit(1)
	}
	cfg, err := LoadConfig(*cfgFileName)
	if err != nil {
		fmt.Println("Error: Loading config file: " + err.Error())
		os.Exit(1)
	}
	Cfg = cfg

	if err := log.InitCfg(cfg); err != nil {
		fmt.Println("Error: failed to create log writers: " + err.Error())
		os.Exit(1)
	}

	log.Infof("Loaded config file: %+v\n", cfg.TOURI)

	useTOClientCache := false
	TOClient, _, err = toclient.LoginWithAgent(cfg.TOURI, cfg.TOUser, cfg.TOPass, cfg.TOInsecure, UserAgent, useTOClientCache, TOTimeout)
	if err != nil {
		fmt.Println("Error: logging in to Traffic Ops: " + err.Error())
		os.Exit(1)
	}

	dsArr, err := TOClient.DeliveryServices()
	if err != nil {
		fmt.Println("Error: getting delivery services: " + err.Error())
		os.Exit(1)
	}
	TO.DeliveryServices = dsToMap(dsArr)

	serversArr, err := TOClient.Servers()
	if err != nil {
		fmt.Println("Error: getting delivery services: " + err.Error())
		os.Exit(1)
	}
	TO.Servers = serverToMap(serversArr)

	dssArr, _, err := TOClient.GetDeliveryServiceServer("0", "999999")
	if err != nil {
		fmt.Println("Error: getting delivery service servers: " + err.Error())
		os.Exit(1)
	}
	TO.ServerDeliveryServices, TO.DeliveryServiceServers = dssToMap(dssArr, TO.Servers, TO.DeliveryServices)

	exitCode := m.Run()

	// TODO teardown?

	os.Exit(exitCode)
}

func dsToMap(arr []tc.DeliveryService) map[tc.DeliveryServiceName]tc.DeliveryService {
	m := map[tc.DeliveryServiceName]tc.DeliveryService{}
	for _, ds := range arr {
		m[tc.DeliveryServiceName(ds.XMLID)] = ds
	}
	return m
}

func serverToMap(arr []tc.Server) map[tc.CacheName]tc.Server {
	m := map[tc.CacheName]tc.Server{}
	for _, server := range arr {
		m[tc.CacheName(server.HostName)] = server
	}
	return m
}

func dsMapToIDMap(dsMap map[tc.DeliveryServiceName]tc.DeliveryService) map[int]tc.DeliveryService {
	dsIDMap := map[int]tc.DeliveryService{}
	for _, ds := range dsMap {
		dsIDMap[ds.ID] = ds
	}
	return dsIDMap
}

func serverMapToIDMap(serverMap map[tc.CacheName]tc.Server) map[int]tc.Server {
	serverIDMap := map[int]tc.Server{}
	for _, sv := range serverMap {
		serverIDMap[sv.ID] = sv
	}
	return serverIDMap
}

func dssToMap(dssArr []tc.DeliveryServiceServer, servers map[tc.CacheName]tc.Server, dses map[tc.DeliveryServiceName]tc.DeliveryService) (map[tc.CacheName][]tc.DeliveryServiceName, map[tc.DeliveryServiceName][]tc.CacheName) {
	dsIDMap := dsMapToIDMap(dses)
	serverIDMap := serverMapToIDMap(servers)

	serverDSes := map[tc.CacheName][]tc.DeliveryServiceName{}
	dsServers := map[tc.DeliveryServiceName][]tc.CacheName{}

	for _, dss := range dssArr {
		if dss.Server == nil || dss.DeliveryService == nil {
			continue // TODO warn? error?
		}
		if _, ok := dsIDMap[*dss.DeliveryService]; !ok {
			continue // TODO warn? error?
		}
		if _, ok := serverIDMap[*dss.Server]; !ok {
			continue // TODO warn? error?
		}
		cacheName := tc.CacheName(serverIDMap[*dss.Server].HostName)
		dsName := tc.DeliveryServiceName(dsIDMap[*dss.DeliveryService].XMLID)
		serverDSes[cacheName] = append(serverDSes[cacheName], dsName)
		dsServers[dsName] = append(dsServers[dsName], cacheName)
	}
	return serverDSes, dsServers
}
