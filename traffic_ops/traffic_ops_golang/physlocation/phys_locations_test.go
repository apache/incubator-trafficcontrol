package physlocation

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
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"
	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func getTestPhysLocations() []tc.PhysLocation {
	physLocations := []tc.PhysLocation{}
	testCase := tc.PhysLocation{
		Address:     "1118 S. Grant St.",
		City:        "Denver",
		Email:       "d.t@gmail.com",
		ID:          1,
		LastUpdated: tc.TimeNoMod{Time: time.Now()},
		Name:        "physLocation1",
		Phone:       "303-210-0000",
		POC:         "Dennis Thompson",
		RegionID:    1,
		RegionName:  "region1",
		ShortName:   "pl1",
		State:       "CO",
		Zip:         "80210",
	}
	physLocations = append(physLocations, testCase)

	testCase2 := testCase
	testCase2.Name = "physLocation2"
	physLocations = append(physLocations, testCase2)

	return physLocations
}

func TestGetPhysLocations(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testCase := getTestPhysLocations()
	cols := test.ColsFromStructByTag("db", tc.PhysLocation{})
	rows := sqlmock.NewRows(cols)

	for _, ts := range testCase {
		rows = rows.AddRow(
			ts.Address,
			ts.City,
			ts.Comments,
			ts.Email,
			ts.ID,
			ts.LastUpdated,
			ts.Name,
			ts.Phone,
			ts.POC,
			ts.RegionID,
			ts.RegionName,
			ts.ShortName,
			ts.State,
			ts.Zip,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin(), Params: map[string]string{"dsId": "1"}}
	obj := TOPhysLocation{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.PhysLocationNullable{},
	}
	physLocations, userErr, sysErr, _, _ := obj.Read(nil, false)
	if userErr != nil || sysErr != nil {
		t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
	}

	if len(physLocations) != 2 {
		t.Errorf("physLocation.Read expected: len(physLocations) == 2, actual: %v", len(physLocations))
	}

}

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TOPhysLocation{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("PhysLocation must be Creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("PhysLocation must be Reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("PhysLocation must be Updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("PhysLocation must be Deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("PhysLocation must be Identifier")
	}
}

func TestValidate(t *testing.T) {
	p := TOPhysLocation{}
	errs := util.JoinErrsStr(test.SortErrors(test.SplitErrors(p.Validate())))
	expected := util.JoinErrsStr(test.SortErrors([]error{
		errors.New("'state' cannot be blank"),
		errors.New("'zip' cannot be blank"),
		errors.New("'address' cannot be blank"),
		errors.New("'city' cannot be blank"),
		errors.New("'name' cannot be blank"),
		errors.New("'regionId' cannot be blank"),
		errors.New("'shortName' cannot be blank"),
	}))

	if !reflect.DeepEqual(expected, errs) {
		t.Errorf("expected %++v,  got %++v", expected, errs)
	}
}
