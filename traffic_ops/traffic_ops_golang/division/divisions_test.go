package division

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
	"reflect"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"
	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func getTestDivisions() []tc.Division {
	regions := []tc.Division{}
	testCase := tc.Division{
		ID:          1,
		Name:        "division1",
		LastUpdated: tc.TimeNoMod{Time: time.Now()},
	}
	regions = append(regions, testCase)

	testCase2 := testCase
	testCase2.Name = "region2"
	regions = append(regions, testCase2)

	return regions
}

func TestGetDivisions(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testCase := getTestDivisions()
	cols := test.ColsFromStructByTag("db", tc.Division{})
	rows := sqlmock.NewRows(cols)

	for _, ts := range testCase {
		rows = rows.AddRow(
			ts.ID,
			ts.LastUpdated,
			ts.Name,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin(), Params: map[string]string{"dsId": "1"}}
	obj := TODivision{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.DivisionNullable{},
	}
	vals, userErr, sysErr, _, _ := obj.Read(nil, false)
	if userErr != nil || sysErr != nil {
		t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
	}
	if len(vals) != 2 {
		t.Errorf("read expected: len(vals) == 1, actual: %v", len(vals))
	}

}

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TODivision{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("division must be creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("division must be reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("division must be updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("division must be deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("division must be Identifier")
	}
}

func TestValidation(t *testing.T) {
	div := TODivision{}
	errs := test.SortErrors(test.SplitErrors(div.Validate()))
	expected := []error{}

	if reflect.DeepEqual(expected, errs) {
		t.Errorf(`expected %v,  got %v`, expected, errs)
	}
}
