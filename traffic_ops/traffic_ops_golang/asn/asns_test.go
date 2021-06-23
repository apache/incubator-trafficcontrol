package asn

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
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"
	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/apache/trafficcontrol/lib/go-util"
)

func getTestASNs() []tc.ASNNullable {
	asns := []tc.ASNNullable{}
	i := 1
	c := "Yukon"
	testCase := tc.ASNNullable{
		ASN:          &i,
		Cachegroup:   &c,
		CachegroupID: &i,
		ID:           &i,
		LastUpdated:  &tc.TimeNoMod{Time: time.Now()},
	}
	asns = append(asns, testCase)

	testCase2 := testCase
	*testCase2.ASN = 2
	asns = append(asns, testCase2)

	return asns
}

func TestGetASNs(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testCase := getTestASNs()
	cols := test.ColsFromStructByTag("db", tc.ASNNullable{})
	rows := sqlmock.NewRows(cols)

	for _, ts := range testCase {
		rows = rows.AddRow(
			*ts.ASN,
			*ts.Cachegroup,
			*ts.CachegroupID,
			*ts.ID,
			*ts.LastUpdated,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()
	reqInfo := api.Info{Tx: db.MustBegin(), Params: map[string]string{"dsId": "1"}}

	obj := TOASNV11{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.ASNNullable{},
	}
	asns, userErr, sysErr, _, _ := obj.Read(nil, false)
	if userErr != nil || sysErr != nil {
		t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
	}

	if len(asns) != 2 {
		t.Errorf("asn.Read expected: len(asns) == 2, actual: %v", len(asns))
	}

}

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TOASNV11{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("asn must be creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("asn must be reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("asn must be updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("asn must be deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("asn must be Identifier")
	}
}

func TestValidate(t *testing.T) {
	i := -99
	asn := TOASNV11{
		api.InfoerImpl{},
		tc.ASNNullable{ASN: &i, CachegroupID: &i},
	}
	errs := util.JoinErrsStr(test.SortErrors(test.SplitErrors(asn.Validate())))
	expected := util.JoinErrsStr([]error{
		errors.New(`'asn' must be no less than 0`),
		errors.New(`'cachegroupId' must be no less than 0`),
	})
	if !reflect.DeepEqual(expected, errs) {
		t.Errorf(`expected %v,  got %v`, expected, errs)
	}
}

func TestCheckNumberForUpdate(t *testing.T) {
	expected := `another asn exists for this number`
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	cols := []string{"id"}
	rows := sqlmock.NewRows(cols)
	rows = rows.AddRow(
		2,
	)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin()}
	asnNum := 2
	cachegroupID := 10
	id := 1
	asn := TOASNV11{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.ASNNullable{ASN: &asnNum, CachegroupID: &cachegroupID, ID: &id},
	}
	err = asn.ASNExists(false)
	if err == nil {
		t.Fatalf("expected error but got none")
	}
	if err.Error() != expected {
		t.Errorf("Expected error detail to be %v, got %v", expected, err.Error())
	}
}

func TestASNExistsForUpdateFailure(t *testing.T) {
	expected := `another asn exists for this number`
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	cols := []string{"id"}
	rows := sqlmock.NewRows(cols)
	rows = rows.AddRow(
		2,
	)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin()}
	asnNum := 2
	cachegroupID := 10
	id := 1
	asn := TOASNV11{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.ASNNullable{ASN: &asnNum, CachegroupID: &cachegroupID, ID: &id},
	}
	err = asn.ASNExists(false)
	if err == nil {
		t.Fatalf("expected error but got none")
	}
	if err.Error() != expected {
		t.Errorf("Expected error detail to be %v, got %v", expected, err.Error())
	}
}

func TestASNExistsForUpdateSuccess(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	cols := []string{"id"}
	rows := sqlmock.NewRows(cols)
	rows = rows.AddRow(
		1,
	)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin()}
	asnNum := 2
	cachegroupID := 10
	id := 1
	asn := TOASNV11{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.ASNNullable{ASN: &asnNum, CachegroupID: &cachegroupID, ID: &id},
	}
	err = asn.ASNExists(false)
	if err != nil {
		t.Errorf("expected no error but got %v", err.Error())
	}
}

func TestASNExists(t *testing.T) {
	expected := `an asn with the specified number already exists`
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	cols := []string{"id"}
	rows := sqlmock.NewRows(cols)
	rows = rows.AddRow(
		1,
	)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin()}
	asnNum := 2
	cachegroupID := 10
	asn := TOASNV11{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.ASNNullable{ASN: &asnNum, CachegroupID: &cachegroupID},
	}
	err = asn.ASNExists(true)
	if err == nil {
		t.Fatalf("expected error but got none")
	}
	if err.Error() != expected {
		t.Errorf("Expected error detail to be %v, got %v", expected, err.Error())
	}
}
