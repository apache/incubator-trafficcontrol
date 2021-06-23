package coordinate

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

func getTestCoordinates() []tc.Coordinate {
	coords := []tc.Coordinate{}
	testCoord1 := tc.Coordinate{
		ID:          1,
		Name:        "coordinate1",
		Latitude:    38.7,
		Longitude:   90.7,
		LastUpdated: tc.TimeNoMod{Time: time.Now()},
	}
	coords = append(coords, testCoord1)

	testCoord2 := tc.Coordinate{
		ID:          2,
		Name:        "coordinate2",
		Latitude:    38.7,
		Longitude:   90.7,
		LastUpdated: tc.TimeNoMod{Time: time.Now()},
	}
	coords = append(coords, testCoord2)

	return coords
}

func TestReadCoordinates(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testCoords := getTestCoordinates()
	cols := test.ColsFromStructByTag("db", tc.Coordinate{})
	rows := sqlmock.NewRows(cols)

	for _, ts := range testCoords {
		rows = rows.AddRow(
			ts.ID,
			ts.Name,
			ts.Latitude,
			ts.Longitude,
			ts.LastUpdated,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin(), Params: map[string]string{"id": "1"}}
	obj := TOCoordinate{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.CoordinateNullable{},
	}
	coordinates, userErr, sysErr, _, _ := obj.Read(nil, false)
	if userErr != nil || sysErr != nil {
		t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
	}
	if len(coordinates) != 2 {
		t.Errorf("coordinate.Read expected: len(coordinates) == 2, actual: %v", len(coordinates))
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
	i = &TOCoordinate{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("coordinate must be creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("coordinate must be reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("coordinate must be updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("coordinate must be deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("coordinate must be Identifier")
	}
}

func TestValidate(t *testing.T) {
	// invalid name, latitude, and longitude
	id := 1
	nm := "not!a!valid!name"
	la := -190.0
	lo := -190.0
	lu := tc.TimeNoMod{Time: time.Now()}
	c := TOCoordinate{CoordinateNullable: tc.CoordinateNullable{ID: &id,
		Name:        &nm,
		Latitude:    &la,
		Longitude:   &lo,
		LastUpdated: &lu,
	}}
	errs := util.JoinErrsStr(test.SortErrors(test.SplitErrors(c.Validate())))

	expectedErrs := util.JoinErrsStr([]error{
		errors.New(`'latitude' Must be a floating point number within the range +-90`),
		errors.New(`'longitude' Must be a floating point number within the range +-180`),
		errors.New(`'name' invalid characters found - Use alphanumeric . or - or _ .`),
	})

	if !reflect.DeepEqual(expectedErrs, errs) {
		t.Errorf("expected %s, got %s", expectedErrs, errs)
	}

	//  valid name, latitude, longitude
	nm = "This.is.2.a-Valid---Coordinate."
	la = 90.0
	lo = 90.0
	c = TOCoordinate{CoordinateNullable: tc.CoordinateNullable{ID: &id,
		Name:        &nm,
		Latitude:    &la,
		Longitude:   &lo,
		LastUpdated: &lu,
	}}
	err := c.Validate()
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
}
