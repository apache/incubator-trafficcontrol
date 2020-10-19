package origin

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
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"

	"github.com/jmoiron/sqlx"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func getTestOrigins() []tc.Origin {
	origins := []tc.Origin{}
	testOrigin := tc.Origin{
		Cachegroup:        util.StrPtr("Cachegroup"),
		CachegroupID:      util.IntPtr(1),
		Coordinate:        util.StrPtr("originCoordinate"),
		CoordinateID:      util.IntPtr(1),
		DeliveryService:   util.StrPtr("testDS"),
		DeliveryServiceID: util.IntPtr(1),
		FQDN:              util.StrPtr("origin.cdn.net"),
		ID:                util.IntPtr(1),
		IP6Address:        util.StrPtr("dead:beef:cafe::42"),
		IPAddress:         util.StrPtr("10.2.3.4"),
		IsPrimary:         util.BoolPtr(false),
		LastUpdated:       tc.NewTimeNoMod(),
		Name:              util.StrPtr("originName"),
		Port:              util.IntPtr(443),
		Profile:           util.StrPtr("profile"),
		ProfileID:         util.IntPtr(1),
		Protocol:          util.StrPtr("https"),
		Tenant:            util.StrPtr("tenantName"),
		TenantID:          util.IntPtr(1),
	}
	origins = append(origins, testOrigin)

	testOrigin2 := testOrigin
	testOrigin2.FQDN = util.StrPtr("origin2.cdn.com")
	testOrigin2.Name = util.StrPtr("origin2")
	origins = append(origins, testOrigin2)

	testOrigin3 := testOrigin
	testOrigin3.FQDN = util.StrPtr("origin3.cdn.org")
	testOrigin3.Name = util.StrPtr("origin3")
	origins = append(origins, testOrigin3)

	return origins
}

func TestReadOrigins(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testOrigins := getTestOrigins()
	cols := test.ColsFromStructByTag("db", tc.Origin{})
	originRows := sqlmock.NewRows(cols)

	for _, to := range testOrigins {
		originRows = originRows.AddRow(
			to.Cachegroup,
			to.CachegroupID,
			to.Coordinate,
			to.CoordinateID,
			to.DeliveryService,
			to.DeliveryServiceID,
			to.FQDN,
			to.ID,
			to.IP6Address,
			to.IPAddress,
			to.IsPrimary,
			to.LastUpdated,
			to.Name,
			to.Port,
			to.Profile,
			to.ProfileID,
			to.Protocol,
			to.Tenant,
			to.TenantID,
		)
	}

	tenantRows := sqlmock.NewRows([]string{"id"})
	tenantRows.AddRow(1)

	mock.ExpectBegin()
	mock.ExpectQuery("WITH").WillReturnRows(tenantRows)
	mock.ExpectQuery("SELECT").WillReturnRows(originRows)
	v := map[string]string{}

	testUser := auth.CurrentUser{TenantID: 1}
	origins, errs, _ := getOrigins(nil, v, db.MustBegin(), &testUser, false)
	if errs.Occurred() {
		t.Errorf("getOrigins expected: no errors, actual: %s", errs)
	}

	if len(origins) != 3 {
		t.Errorf("getOrigins expected: len(origins) == 3, actual: %v", len(origins))
	}

}

func TestFuncs(t *testing.T) {
	if strings.Index(selectQuery(), "SELECT") != 0 {
		t.Errorf("expected selectQuery to start with SELECT")
	}
	if strings.Index(insertQuery(), "INSERT") != 0 {
		t.Errorf("expected insertQuery to start with INSERT")
	}
	if strings.Index(updateQuery(), "UPDATE") != 0 {
		t.Errorf("expected updateQuery to start with UPDATE")
	}
	if strings.Index(deleteQuery(), "DELETE") != 0 {
		t.Errorf("expected deleteQuery to start with DELETE")
	}
}

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TOOrigin{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("origin must be creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("origin must be reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("origin must be updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("origin must be deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("origin must be Identifier")
	}
	if _, ok := i.(api.Tenantable); !ok {
		t.Errorf("origin must be tenantable")
	}
}

func TestValidate(t *testing.T) {
	const portErr = `'port' must be a valid integer between 1 and 65535`
	const protoErr = `'protocol' must be http or https`
	const fqdnErr = `'fqdn' must be a valid DNS name`
	const ipErr = `'ipAddress' must be a valid IPv4 address`
	const ip6Err = `'ip6Address' must be a valid IPv6 address`

	// verify that non-null fields are invalid
	c := TOOrigin{Origin: tc.Origin{ID: nil,
		Name:              nil,
		DeliveryServiceID: nil,
		FQDN:              nil,
		Protocol:          nil,
	}}
	errs := util.JoinErrsStr(test.SortErrors(test.SplitErrors(c.Validate())))

	expectedErrs := util.JoinErrsStr([]error{
		errors.New(`'deliveryServiceId' is required`),
		errors.New(`'fqdn' cannot be blank`),
		errors.New(`'name' cannot be blank`),
		errors.New(`'protocol' cannot be blank`),
	})

	if !reflect.DeepEqual(expectedErrs, errs) {
		t.Errorf("expected %s, got %s", expectedErrs, errs)
	}

	// all valid fields
	id := 1
	nm := "validname"
	fqdn := "is.a.valid.hostname"
	ip6 := "dead:beef::42"
	ip := "1.2.3.4"
	port := 65535
	pro := "http"
	lu := tc.TimeNoMod{Time: time.Now()}
	c = TOOrigin{Origin: tc.Origin{ID: &id,
		Name:              &nm,
		DeliveryServiceID: &id,
		FQDN:              &fqdn,
		IP6Address:        &ip6,
		IPAddress:         &ip,
		Port:              &port,
		Protocol:          &pro,
		LastUpdated:       &lu,
	}}
	err := c.Validate()
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}

	type testCase struct {
		Int            int
		Str            string
		ExpectedErrors []error
	}

	type typedTestCases struct {
		Type      string
		TestCases []testCase
	}

	nameTestCases := typedTestCases{
		"name",
		[]testCase{
			{Str: "", ExpectedErrors: []error{errors.New(`'name' cannot be blank`)}},
			{Str: "invalid name", ExpectedErrors: []error{errors.New(`'name' cannot contain spaces`)}},
			{Str: "valid-name", ExpectedErrors: []error{}},
		},
	}

	portTestCases := typedTestCases{
		"port",
		[]testCase{
			{Int: -1, ExpectedErrors: []error{errors.New(portErr)}},
			{Int: 0, ExpectedErrors: []error{errors.New(portErr)}},
			{Int: 1, ExpectedErrors: []error{}},
		},
	}

	protoTestCases := typedTestCases{
		"protocol",
		[]testCase{
			{Str: "foo", ExpectedErrors: []error{errors.New(protoErr)}},
			{Str: "", ExpectedErrors: []error{errors.New(`'protocol' cannot be blank`)}},
			{Str: "http", ExpectedErrors: []error{}},
			{Str: "https", ExpectedErrors: []error{}},
		},
	}

	fqdnTestCases := typedTestCases{
		"fqdn",
		[]testCase{
			{Str: "not.@.v@lid.#()stn@me", ExpectedErrors: []error{errors.New(fqdnErr)}},
			{Str: "dead:beef::42", ExpectedErrors: []error{errors.New(fqdnErr)}},
			{Str: "valid.hostname.net", ExpectedErrors: []error{}},
		},
	}

	ipTestCases := typedTestCases{
		"ip",
		[]testCase{
			{Str: "not.@.v@lid.#()stn@me", ExpectedErrors: []error{errors.New(ipErr)}},
			{Str: "dead:beef::42", ExpectedErrors: []error{errors.New(ipErr)}},
			{Str: "1.2.3", ExpectedErrors: []error{errors.New(ipErr)}},
			{Str: "", ExpectedErrors: []error{errors.New(`'ipAddress' cannot be blank`)}},
			{Str: "1.2.3.4", ExpectedErrors: []error{}},
		},
	}

	ip6TestCases := typedTestCases{
		"ip6",
		[]testCase{
			{Str: "not.@.v@lid.#()stn@me", ExpectedErrors: []error{errors.New(ip6Err)}},
			{Str: "1.2.3.4", ExpectedErrors: []error{errors.New(ip6Err)}},
			{Str: "beef", ExpectedErrors: []error{errors.New(ip6Err)}},
			{Str: "", ExpectedErrors: []error{errors.New(`'ip6Address' cannot be blank`)}},
			{Str: "dead:beef::42", ExpectedErrors: []error{}},
		},
	}

	for _, ttc := range []typedTestCases{
		nameTestCases,
		portTestCases,
		protoTestCases,
		fqdnTestCases,
		ipTestCases,
		ip6TestCases,
	} {
		for _, tc := range ttc.TestCases {
			var value interface{}
			switch ttc.Type {
			case "name":
				c.Name = &tc.Str
				value = tc.Str
			case "port":
				c.Port = &tc.Int
				value = tc.Int
			case "protocol":
				c.Protocol = &tc.Str
				value = tc.Str
			case "fqdn":
				c.FQDN = &tc.Str
				value = tc.Str
			case "ip":
				c.IPAddress = &tc.Str
				value = tc.Str
			case "ip6":
				c.IP6Address = &tc.Str
				value = tc.Str
			}
			errStr := util.JoinErrsStr(test.SortErrors(test.SplitErrors(c.Validate())))
			if !reflect.DeepEqual(util.JoinErrsStr(tc.ExpectedErrors), errStr) {
				t.Errorf("given: '%v', expected %s, got %s", value, tc.ExpectedErrors, errStr)
			}
		}
	}

}
