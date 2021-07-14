package crconfig

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
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type StringArray pq.StringArray

func randBool() *bool {
	b := rand.Int()%2 == 0
	return &b
}
func randStr() *string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890-_"
	num := 100
	s := ""
	for i := 0; i < num; i++ {
		s += string(chars[rand.Intn(len(chars))])
	}
	return &s
}
func randStrArray() []string {
	num := 100
	sArray := make([]string, num)
	for i := 0; i < num; i++ {
		sArray[i] = *randStr()
	}
	return sArray
}
func randInt() *int {
	i := rand.Int()
	return &i
}
func randInt64() *int64 {
	i := int64(rand.Int63())
	return &i
}
func randFloat64() *float64 {
	f := rand.Float64()
	return &f
}

func randomIPv4() *string {
	first := rand.Int31n(256)
	second := rand.Int31n(256)
	third := rand.Int31n(256)
	fourth := rand.Int31n(256)
	str := fmt.Sprintf("%d.%d.%d.%d", first, second, third, fourth)
	return &str
}

func randomIPv6() *string {
	ip := net.IP([]byte{
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
		uint8(rand.Int31n(256)),
	}).String()
	return &ip
}

func randServer(ipService bool, ip6Service bool) tc.CRConfigTrafficOpsServer {
	status := tc.CRConfigServerStatus(*randStr())
	cachegroup := randStr()
	ip := new(string)
	ip6 := new(string)
	inf := new(string)

	if ipService {
		ip = randomIPv4()
		inf = randStr()
	}
	if ip6Service {
		ip6 = randomIPv6()
		inf = randStr()
	}

	return tc.CRConfigTrafficOpsServer{
		CacheGroup:      cachegroup,
		Capabilities:    randStrArray(),
		Fqdn:            randStr(),
		HashCount:       randInt(),
		HashId:          randStr(),
		HttpsPort:       randInt(),
		InterfaceName:   inf,
		Ip:              ip,
		Ip6:             ip6,
		LocationId:      cachegroup,
		Port:            randInt(),
		Profile:         randStr(),
		ServerStatus:    &status,
		ServerType:      randStr(),
		RoutingDisabled: *randInt64(),
	}
}

func ExpectedGetServerParams() map[string]ServerParams {
	return map[string]ServerParams{
		"cache0": ServerParams{
			APIPort:          randStr(),
			SecureAPIPort:    randStr(),
			Weight:           randFloat64(),
			WeightMultiplier: randFloat64(),
		},
		"cache1": ServerParams{
			APIPort:          randStr(),
			Weight:           randFloat64(),
			WeightMultiplier: randFloat64(),
		},
	}
}

func MockGetServerParams(mock sqlmock.Sqlmock, expected map[string]ServerParams, cdn string) {
	rows := sqlmock.NewRows([]string{"host_name", "name", "value"})
	rows = rows.AddRow("cache0", "api.port", *expected["cache0"].APIPort)
	rows = rows.AddRow("cache0", "secure.api.port", *expected["cache0"].SecureAPIPort)
	rows = rows.AddRow("cache0", "weight", *expected["cache0"].Weight)
	rows = rows.AddRow("cache0", "weightMultiplier", *expected["cache0"].WeightMultiplier)
	rows = rows.AddRow("cache1", "api.port", *expected["cache1"].APIPort)
	rows = rows.AddRow("cache1", "weight", *expected["cache1"].Weight)
	rows = rows.AddRow("cache1", "weightMultiplier", *expected["cache1"].WeightMultiplier)
	mock.ExpectQuery("SELECT").WithArgs(cdn).WillReturnRows(rows)
}

func TestGetServerParams(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	cdn := "mycdn"

	mock.ExpectBegin()
	expected := ExpectedGetServerParams()
	MockGetServerParams(mock, expected, cdn)
	mock.ExpectCommit()

	dbCtx, cancelTx := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancelTx()
	tx, err := db.BeginTx(dbCtx, nil)
	if err != nil {
		t.Fatalf("creating transaction: %v", err)
	}
	defer tx.Commit()

	actual, err := getServerParams(cdn, tx)
	if err != nil {
		t.Fatalf("getServerParams expected: nil error, actual: %v", err)
	}

	if len(actual) != len(expected) {
		t.Fatalf("getServerParams len expected: %v, actual: %v", len(expected), len(actual))
	}

	for name, params := range expected {
		actualParams, ok := actual[name]
		if !ok {
			t.Errorf("getServerParams expected: %v, actual: missing", name)
			continue
		}
		if !reflect.DeepEqual(params, actualParams) {
			t.Errorf("getServerParams server %+v expected: %+v, actual: %+v", name, params, actualParams)
		}
	}
}

func ExpectedGetAllServers(params map[string]ServerParams, ipIsService bool, ip6IsService bool) map[string]ServerUnion {
	expected := map[string]ServerUnion{}
	for name, param := range params {
		s := ServerUnion{
			APIPort:                  param.APIPort,
			SecureAPIPort:            param.SecureAPIPort,
			CRConfigTrafficOpsServer: randServer(ipIsService, ip6IsService),
		}
		i := int(*param.Weight * *param.WeightMultiplier)
		s.HashCount = &i
		if !ipIsService {
			s.Ip = util.StrPtr("")
		}
		if !ip6IsService {
			s.Ip6 = util.StrPtr("")
		}
		expected[name] = s
	}
	return expected
}

func MockGetAllServers(mock sqlmock.Sqlmock, expected map[string]ServerUnion, cdn string, ipIsService bool, ip6IsService bool) {
	serverRows := sqlmock.NewRows([]string{"id", "host_name", "cachegroup", "fqdn", "hashid", "https_port", "tcp_port", "profile_name", "routing_disabled", "status", "type", "capabilities"})
	interfaceRows := sqlmock.NewRows([]string{"max_bandwidth", "monitor", "mtu", "name", "server", "router_host_name", "router_port_name"})
	ipRows := sqlmock.NewRows([]string{"address", "gateway", "service_address", "interface", "server"})
	i := 1
	for name, s := range expected {
		capabilities := "{" + strings.Join(s.Capabilities, ",") + "}"
		serverRows = serverRows.AddRow(i, name, *s.CacheGroup, *s.Fqdn, *s.HashId, *s.HttpsPort, *s.Port, *s.Profile, s.RoutingDisabled, *s.ServerStatus, *s.ServerType, capabilities)
		if s.InterfaceName == nil {
			i++
			continue
		}
		interfaceRows = interfaceRows.AddRow(nil, true, nil, *s.InterfaceName, i, "", "")

		if s.Ip != nil {
			ipRows = ipRows.AddRow(*s.Ip, nil, ipIsService, *s.InterfaceName, i)
		}
		if s.Ip6 != nil {
			ipRows = ipRows.AddRow(*s.Ip6, nil, ip6IsService, *s.InterfaceName, i)
		}
		i++
	}
	mock.ExpectQuery("SELECT").WithArgs(cdn).WillReturnRows(serverRows)
	mock.ExpectQuery("SELECT").WillReturnRows(interfaceRows)
	mock.ExpectQuery("SELECT").WillReturnRows(ipRows)
}

func compare(expected map[string]ServerUnion, actual map[string]ServerUnion, t *testing.T) {
	for name, server := range expected {
		actualServer, ok := actual[name]
		if !ok {
			t.Errorf("getAllServers expected: %v, actual: missing", name)
			continue
		}

		if actualServer.APIPort == nil && server.APIPort != nil {
			t.Errorf("expected server '%s' to have APIPort '%s', actual: <nil>", name, *server.APIPort)
		} else if server.APIPort == nil && actualServer.APIPort != nil {
			t.Errorf("expected server '%s' to have nil APIPort, actual: '%s'", name, *actualServer.APIPort)
		} else if (server.APIPort != nil || actualServer.APIPort != nil) && *server.APIPort != *actualServer.APIPort {
			t.Errorf("expected server '%s' to have APIPort '%s', actual: '%s'", name, *server.APIPort, *actualServer.APIPort)
		}

		if actualServer.SecureAPIPort == nil && server.SecureAPIPort != nil {
			t.Errorf("expected server '%s' to have SecureAPIPort '%s', actual: <nil>", name, *server.SecureAPIPort)
		} else if server.SecureAPIPort == nil && actualServer.SecureAPIPort != nil {
			t.Errorf("expected server '%s' to have nil SecureAPIPort, actual: '%s'", name, *actualServer.SecureAPIPort)
		} else if (server.SecureAPIPort != nil || actualServer.SecureAPIPort != nil) && *server.SecureAPIPort != *actualServer.SecureAPIPort {
			t.Errorf("expected server '%s' to have SecureAPIPort '%s', actual: '%s'", name, *server.SecureAPIPort, *actualServer.SecureAPIPort)
		}

		if actualServer.CacheGroup == nil && server.CacheGroup != nil {
			t.Errorf("expected server '%s' to have CacheGroup '%s', actual: <nil>", name, *server.CacheGroup)
		} else if server.CacheGroup == nil && actualServer.CacheGroup != nil {
			t.Errorf("expected server '%s' to have nil CacheGroup, actual: '%s'", name, *actualServer.CacheGroup)
		} else if (server.CacheGroup != nil || actualServer.CacheGroup != nil) && *server.CacheGroup != *actualServer.CacheGroup {
			t.Errorf("expected server '%s' to have CacheGroup '%s', actual: '%s'", name, *server.CacheGroup, *actualServer.CacheGroup)
		}

		if actualServer.Fqdn == nil && server.Fqdn != nil {
			t.Errorf("expected server '%s' to have Fqdn '%s', actual: <nil>", name, *server.Fqdn)
		} else if server.Fqdn == nil && actualServer.Fqdn != nil {
			t.Errorf("expected server '%s' to have nil Fqdn, actual: '%s'", name, *actualServer.Fqdn)
		} else if (server.Fqdn != nil || actualServer.Fqdn != nil) && *server.Fqdn != *actualServer.Fqdn {
			t.Errorf("expected server '%s' to have Fqdn '%s', actual: '%s'", name, *server.Fqdn, *actualServer.Fqdn)
		}

		if actualServer.HashCount == nil && server.HashCount != nil {
			t.Errorf("expected server '%s' to have HashCount '%v', actual: <nil>", name, *server.HashCount)
		} else if server.HashCount == nil && actualServer.HashCount != nil {
			t.Errorf("expected server '%s' to have nil HashCount, actual: '%v'", name, *actualServer.HashCount)
		} else if (server.HashCount != nil || actualServer.HashCount != nil) && *server.HashCount != *actualServer.HashCount {
			t.Errorf("expected server '%s' to have HashCount '%v', actual: '%v'", name, *server.HashCount, *actualServer.HashCount)
		}

		if actualServer.HashId == nil && server.HashId != nil {
			t.Errorf("expected server '%s' to have HashId '%v', actual: <nil>", name, *server.HashId)
		} else if server.HashId == nil && actualServer.HashId != nil {
			t.Errorf("expected server '%s' to have nil HashId, actual: '%v'", name, *actualServer.HashId)
		} else if (server.HashId != nil || actualServer.HashId != nil) && *server.HashId != *actualServer.HashId {
			t.Errorf("expected server '%s' to have HashId '%v', actual: '%v'", name, *server.HashId, *actualServer.HashId)
		}

		if actualServer.HttpsPort == nil && server.HttpsPort != nil {
			t.Errorf("expected server '%s' to have HttpsPort '%v', actual: <nil>", name, *server.HttpsPort)
		} else if server.HttpsPort == nil && actualServer.HttpsPort != nil {
			t.Errorf("expected server '%s' to have nil HttpsPort, actual: '%v'", name, *actualServer.HttpsPort)
		} else if (server.HttpsPort != nil || actualServer.HttpsPort != nil) && *server.HttpsPort != *actualServer.HttpsPort {
			t.Errorf("expected server '%s' to have HttpsPort '%v', actual: '%v'", name, *server.HttpsPort, *actualServer.HttpsPort)
		}

		if actualServer.InterfaceName == nil && server.InterfaceName != nil {
			t.Errorf("expected server '%s' to have InterfaceName '%v', actual: <nil>", name, *server.InterfaceName)
		} else if server.InterfaceName == nil && actualServer.InterfaceName != nil {
			t.Errorf("expected server '%s' to have nil InterfaceName, actual: '%v'", name, *actualServer.InterfaceName)
		} else if (server.InterfaceName != nil || actualServer.InterfaceName != nil) && *server.InterfaceName != *actualServer.InterfaceName {
			t.Errorf("expected server '%s' to have InterfaceName '%v', actual: '%v'", name, *server.InterfaceName, *actualServer.InterfaceName)
		}

		if actualServer.Ip == nil && server.Ip != nil {
			t.Errorf("expected server '%s' to have Ip '%v', actual: <nil>", name, *server.Ip)
		} else if server.Ip == nil && actualServer.Ip != nil {
			t.Errorf("expected server '%s' to have nil Ip, actual: '%v'", name, *actualServer.Ip)
		} else if (server.Ip != nil || actualServer.Ip != nil) && *server.Ip != *actualServer.Ip {
			t.Errorf("expected server '%s' to have Ip '%v', actual: '%v'", name, *server.Ip, *actualServer.Ip)
		}

		if actualServer.Ip6 == nil && server.Ip6 != nil {
			t.Errorf("expected server '%s' to have Ip6 '%v', actual: <nil>", name, *server.Ip6)
		} else if server.Ip6 == nil && actualServer.Ip6 != nil {
			t.Errorf("expected server '%s' to have nil Ip6, actual: '%v'", name, *actualServer.Ip6)
		} else if (server.Ip6 != nil || actualServer.Ip6 != nil) && *server.Ip6 != *actualServer.Ip6 {
			t.Errorf("expected server '%s' to have Ip6 '%v', actual: '%v'", name, *server.Ip6, *actualServer.Ip6)
		}

		if actualServer.LocationId == nil && server.LocationId != nil {
			t.Errorf("expected server '%s' to have LocationId '%v', actual: <nil>", name, *server.LocationId)
		} else if server.LocationId == nil && actualServer.LocationId != nil {
			t.Errorf("expected server '%s' to have nil LocationId, actual: '%v'", name, *actualServer.LocationId)
		} else if (server.LocationId != nil || actualServer.LocationId != nil) && *server.LocationId != *actualServer.LocationId {
			t.Errorf("expected server '%s' to have LocationId '%v', actual: '%v'", name, *server.LocationId, *actualServer.LocationId)
		}

		if actualServer.Port == nil && server.Port != nil {
			t.Errorf("expected server '%s' to have Port '%v', actual: <nil>", name, *server.Port)
		} else if server.Port == nil && actualServer.Port != nil {
			t.Errorf("expected server '%s' to have nil Port, actual: '%v'", name, *actualServer.Port)
		} else if (server.Port != nil || actualServer.Port != nil) && *server.Port != *actualServer.Port {
			t.Errorf("expected server '%s' to have Port '%v', actual: '%v'", name, *server.Port, *actualServer.Port)
		}

		if actualServer.Profile == nil && server.Profile != nil {
			t.Errorf("expected server '%s' to have Profile '%v', actual: <nil>", name, *server.Profile)
		} else if server.Profile == nil && actualServer.Profile != nil {
			t.Errorf("expected server '%s' to have nil Profile, actual: '%v'", name, *actualServer.Profile)
		} else if (server.Profile != nil || actualServer.Profile != nil) && *server.Profile != *actualServer.Profile {
			t.Errorf("expected server '%s' to have Profile '%v', actual: '%v'", name, *server.Profile, *actualServer.Profile)
		}

		if actualServer.ServerStatus == nil && server.ServerStatus != nil {
			t.Errorf("expected server '%s' to have ServerStatus '%v', actual: <nil>", name, *server.ServerStatus)
		} else if server.ServerStatus == nil && actualServer.ServerStatus != nil {
			t.Errorf("expected server '%s' to have nil ServerStatus, actual: '%v'", name, *actualServer.ServerStatus)
		} else if (server.ServerStatus != nil || actualServer.ServerStatus != nil) && *server.ServerStatus != *actualServer.ServerStatus {
			t.Errorf("expected server '%s' to have ServerStatus '%v', actual: '%v'", name, *server.ServerStatus, *actualServer.ServerStatus)
		}

		if actualServer.ServerType == nil && server.ServerType != nil {
			t.Errorf("expected server '%s' to have ServerType '%v', actual: <nil>", name, *server.ServerType)
		} else if server.ServerType == nil && actualServer.ServerType != nil {
			t.Errorf("expected server '%s' to have nil ServerType, actual: '%v'", name, *actualServer.ServerType)
		} else if (server.ServerType != nil || actualServer.ServerType != nil) && *server.ServerType != *actualServer.ServerType {
			t.Errorf("expected server '%s' to have ServerType '%v', actual: '%v'", name, *server.ServerType, *actualServer.ServerType)
		}

		if actualServer.RoutingDisabled != server.RoutingDisabled {
			t.Errorf("expected server '%s' to have RoutingDisabled '%d', actual: '%d'", name, server.RoutingDisabled, actualServer.RoutingDisabled)
		}

		if len(actualServer.DeliveryServices) != len(server.DeliveryServices) {
			t.Errorf("expected server '%s' to have %d DeliveryServices, actual: %d", name, len(server.DeliveryServices), len(actualServer.DeliveryServices))
			continue
		}

		for dsName, dses := range server.DeliveryServices {
			actualDSes, ok := actualServer.DeliveryServices[dsName]
			if !ok {
				t.Errorf("expected Delivery Service '%s' to be in server '%s', but it wasn't", dsName, name)
				continue
			}

			if len(dses) != len(actualDSes) {
				t.Errorf("expected Delivery Service '%s' in server '%s' to have %d entries, actual: %d", dsName, name, len(dses), len(actualDSes))
				continue
			}

			for i, ds := range dses {
				if ds != actualDSes[i] {
					t.Errorf("expected the %dth entry in Delivery Service '%s' in server '%s' to be '%s', actual: '%s'", i, dsName, name, ds, actualDSes[i])
				}
			}
		}
	}
}

func TestGetAllServers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	cdn := "mycdn"

	mock.ExpectBegin()
	getServerParamsExpected := ExpectedGetServerParams()
	MockGetServerParams(mock, getServerParamsExpected, cdn)

	expected := ExpectedGetAllServers(getServerParamsExpected, true, true)
	MockGetAllServers(mock, expected, cdn, true, true)
	mock.ExpectCommit()

	dbCtx, cancelTx := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancelTx()
	tx, err := db.BeginTx(dbCtx, nil)
	if err != nil {
		t.Fatalf("creating transaction: %v", err)
	}
	defer tx.Commit()

	actual, err := getAllServers(cdn, tx)

	if err != nil {
		t.Fatalf("getAllServers expected: nil error, actual: %v", err)
	}

	if len(actual) != len(expected) {
		t.Errorf("getAllServers len expected: %v, actual: %v", len(expected), len(actual))
	}
	compare(expected, actual, t)
}

func TestGetAllServersNonService(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	cdn := "mycdn"

	mock.ExpectBegin()
	getServerParamsExpected := ExpectedGetServerParams()
	MockGetServerParams(mock, getServerParamsExpected, cdn)

	expected := ExpectedGetAllServers(getServerParamsExpected, true, false)
	MockGetAllServers(mock, expected, cdn, true, false)
	mock.ExpectCommit()

	dbCtx, cancelTx := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancelTx()
	tx, err := db.BeginTx(dbCtx, nil)
	if err != nil {
		t.Fatalf("creating transaction: %v", err)
	}
	defer tx.Commit()

	actual, err := getAllServers(cdn, tx)

	if err != nil {
		t.Fatalf("getAllServers expected: nil error, actual: %v", err)
	}

	if len(actual) != len(expected) {
		t.Errorf("getAllServers len expected: %v, actual: %v", len(expected), len(actual))
	}

	compare(expected, actual, t)
}

func ExpectedGetServerDSNames() map[tc.CacheName][]tc.DeliveryServiceName {
	return map[tc.CacheName][]tc.DeliveryServiceName{
		"cache0": []tc.DeliveryServiceName{"ds0", "ds1"},
		"cache1": []tc.DeliveryServiceName{"ds0", "ds1"},
	}
}

func MockGetServerDSNames(mock sqlmock.Sqlmock, expected map[tc.CacheName][]tc.DeliveryServiceName, cdn string) {
	rows := sqlmock.NewRows([]string{"host_name", "xml_id"})
	for cache, dses := range expected {
		for _, ds := range dses {
			rows = rows.AddRow(cache, ds)
		}
	}
	mock.ExpectQuery("SELECT").WithArgs(cdn).WillReturnRows(rows)
}

func TestGetServerDSNames(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	cdn := "mycdn"

	mock.ExpectBegin()
	expected := ExpectedGetServerDSNames()
	MockGetServerDSNames(mock, expected, cdn)
	mock.ExpectCommit()

	dbCtx, cancelTx := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancelTx()
	tx, err := db.BeginTx(dbCtx, nil)
	if err != nil {
		t.Fatalf("creating transaction: %v", err)
	}
	defer tx.Commit()

	actual, err := getServerDSNames(cdn, tx)

	if err != nil {
		t.Fatalf("getServerDSNames expected: nil error, actual: %v", err)
	}

	if len(actual) != len(expected) {
		t.Errorf("getServerDSNames len expected: %v, actual: %v", len(expected), len(actual))
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("getServerDSNames expected: %v, actual: %v", expected, actual)
	}
}

func ExpectedGetServerDSes(expectedGetServerDSNames map[tc.CacheName][]tc.DeliveryServiceName) map[tc.CacheName]map[string][]string {
	e := map[tc.CacheName]map[string][]string{}
	for cache, dses := range expectedGetServerDSNames {
		e[cache] = map[string][]string{}
		for _, ds := range dses {
			e[cache][string(ds)] = []string{string(ds) + "regex0", string(ds) + "regex1"}
		}
	}
	return e
}

func MockGetServerDSes(mock sqlmock.Sqlmock, expected map[tc.CacheName]map[string][]string, cdn string) {
	rows := sqlmock.NewRows([]string{"ds", "ds_type", "routing_name", "pattern", "hasTopology"})
	dsmap := map[string][]string{}
	for _, dses := range expected {
		for ds, patterns := range dses {
			dsmap[ds] = patterns
		}
	}

	for ds, patterns := range dsmap {
		for _, pattern := range patterns {
			rows = rows.AddRow(ds, "DNS", "", pattern, false)
		}
	}
	mock.ExpectQuery("SELECT").WithArgs(cdn).WillReturnRows(rows)
}

func TestGetServerDSes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	cdn := "mycdn"
	domain := "mydomain"

	mock.ExpectBegin()
	expectedGetServerDSNames := ExpectedGetServerDSNames()
	MockGetServerDSNames(mock, expectedGetServerDSNames, cdn)

	expected := ExpectedGetServerDSes(expectedGetServerDSNames)
	MockGetServerDSes(mock, expected, cdn)
	mock.ExpectCommit()

	dbCtx, cancelTx := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancelTx()
	tx, err := db.BeginTx(dbCtx, nil)
	if err != nil {
		t.Fatalf("creating transaction: %v", err)
	}
	defer tx.Commit()

	actual, err := getServerDSes(cdn, tx, domain)

	if err != nil {
		t.Fatalf("getServerDSes expected: nil error, actual: %v", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("getServerDSes expected: %v, actual: %v", expected, actual)
	}
}

func ExpectedGetCDNInfo() (string, bool) {
	return *randStr(), *randBool()
}

func MockGetCDNInfo(mock sqlmock.Sqlmock, expectedDomain string, expectedDNSSECEnabled bool, cdn string) {
	rows := sqlmock.NewRows([]string{"domain_name", "dnssec_enabled"})
	rows = rows.AddRow(expectedDomain, expectedDNSSECEnabled)
	mock.ExpectQuery("select").WithArgs(cdn).WillReturnRows(rows)
}

func TestGetCDNInfo(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	cdn := "mycdn"

	mock.ExpectBegin()
	expectedDomain, expectedDNSSECEnabled := ExpectedGetCDNInfo()
	MockGetCDNInfo(mock, expectedDomain, expectedDNSSECEnabled, cdn)
	mock.ExpectCommit()

	dbCtx, cancelTx := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancelTx()
	tx, err := db.BeginTx(dbCtx, nil)
	if err != nil {
		t.Fatalf("creating transaction: %v", err)
	}
	defer tx.Commit()

	actualDomain, actualDNSSECEnabled, err := getCDNInfo(cdn, tx)
	if err != nil {
		t.Fatalf("getCDNInfo expected: nil error, actual: %v", err)
	}

	if expectedDomain != actualDomain {
		t.Errorf("getCDNInfo expected: %v, actual: %v", expectedDomain, actualDomain)
	}
	if expectedDNSSECEnabled != actualDNSSECEnabled {
		t.Errorf("getCDNInfo expected: %v, actual: %v", expectedDNSSECEnabled, actualDNSSECEnabled)
	}
}

func ExpectedGetCDNNameFromID() string {
	return *randStr()
}

func MockGetCDNNameFromID(mock sqlmock.Sqlmock, expected string, cdnID int) {
	rows := sqlmock.NewRows([]string{"name"})
	rows = rows.AddRow(expected)
	mock.ExpectQuery("select").WithArgs(cdnID).WillReturnRows(rows)
}

func TestGetCDNNameFromID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	cdnID := 42

	mock.ExpectBegin()
	expected := ExpectedGetCDNNameFromID()
	MockGetCDNNameFromID(mock, expected, cdnID)
	mock.ExpectCommit()

	dbCtx, cancelTx := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancelTx()
	tx, err := db.BeginTx(dbCtx, nil)
	if err != nil {
		t.Fatalf("creating transaction: %v", err)
	}
	defer tx.Commit()

	actual, exists, err := getCDNNameFromID(cdnID, tx)
	if err != nil {
		t.Fatalf("getCDNNameFromID expected: nil error, actual: %v", err)
	}
	if !exists {
		t.Fatalf("getCDNNameFromID exists expected: true, actual: false")
	}

	if expected != actual {
		t.Errorf("getCDNNameFromID expected: %v, actual: %v", expected, actual)
	}
}
