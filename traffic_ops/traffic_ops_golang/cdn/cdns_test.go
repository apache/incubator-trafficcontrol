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
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"
	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func getTestCDNs() []tc.CDN {
	cdns := []tc.CDN{}
	testCDN := tc.CDN{
		DNSSECEnabled: false,
		DomainName:    "domainName",
		ID:            1,
		Name:          "cdn1",
		LastUpdated:   tc.TimeNoMod{Time: time.Now()},
	}
	cdns = append(cdns, testCDN)

	testCDN2 := testCDN
	testCDN2.Name = "cdn2"
	testCDN2.DomainName = "domain.net"
	cdns = append(cdns, testCDN2)

	return cdns
}

func TestReadCDNs(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testCDNs := getTestCDNs()
	cols := test.ColsFromStructByTag("db", tc.CDN{})
	rows := sqlmock.NewRows(cols)

	for _, ts := range testCDNs {
		rows = rows.AddRow(
			ts.DNSSECEnabled,
			ts.DomainName,
			ts.ID,
			ts.LastUpdated,
			ts.Name,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin(), Params: map[string]string{"dsId": "1"}}
	obj := TOCDN{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.CDNNullable{},
	}
	cdns, userErr, sysErr, _, _ := obj.Read(nil, false)
	if userErr != nil || sysErr != nil {
		t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
	}

	if len(cdns) != 2 {
		t.Errorf("cdn.Read expected: len(cdns) == 2, actual: %v", len(cdns))
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
	i = &TOCDN{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("cdn must be creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("cdn must be reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("cdn must be updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("cdn must be deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("cdn must be Identifier")
	}
}

func TestValidate(t *testing.T) {
	// invalid name, empty domainname
	n := "not_a_valid_cdn"
	c := TOCDN{CDNNullable: tc.CDNNullable{Name: &n}}
	errs := util.JoinErrsStr(test.SortErrors(test.SplitErrors(c.Validate())))

	expectedErrs := util.JoinErrsStr([]error{
		errors.New(`'domainName' cannot be blank`),
		errors.New(`'name' invalid characters found - Use alphanumeric . or - .`),
	})

	if !reflect.DeepEqual(expectedErrs, errs) {
		t.Errorf("expected %s, got %s", expectedErrs, errs)
	}

	//  name,  domainname both valid
	n = "This.is.2.a-Valid---CDNNAME."
	d := `awesome-cdn.example.net`
	c = TOCDN{CDNNullable: tc.CDNNullable{Name: &n, DomainName: &d}}
	err := c.Validate()
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
}
