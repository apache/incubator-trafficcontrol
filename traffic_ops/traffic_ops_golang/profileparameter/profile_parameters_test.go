package profileparameter

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
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"
	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func getTestProfileParameters() []tc.ProfileParameterNullable {
	pps := []tc.ProfileParameterNullable{}
	lastUpdated := tc.TimeNoMod{}
	lastUpdated.Scan(time.Now())
	profileID := 1
	parameterID := 1

	pp := tc.ProfileParameterNullable{
		LastUpdated: &lastUpdated,
		ProfileID:   &profileID,
		ParameterID: &parameterID,
	}
	pps = append(pps, pp)

	pp2 := pp
	pp2.ProfileID = &profileID
	pp2.ParameterID = &parameterID
	pps = append(pps, pp2)

	return pps
}

func TestGetProfileParameters(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testPPs := getTestProfileParameters()
	cols := test.ColsFromStructByTag("db", tc.ProfileParametersNullable{})
	rows := sqlmock.NewRows(cols)

	for _, ts := range testPPs {
		rows = rows.AddRow(
			ts.LastUpdated,
			ts.Profile,
			ts.ParameterID,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	txx := db.MustBegin()
	reqInfo := api.Info{Tx: txx, Params: map[string]string{"profile": "1"}}
	obj := TOProfileParameter{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.ProfileParameterNullable{},
	}
	pps, userErr, sysErr, _, _ := obj.Read(nil, false)
	if userErr != nil || sysErr != nil {
		t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
	}

	if len(pps) != 2 {
		t.Errorf("profileparameter.Read expected: len(pps) == 2, actual: %v", len(pps))
	}

}

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TOProfileParameter{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("ProfileParameter must be Creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("ProfileParameter must be Reader")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("ProfileParameter must be Deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("ProfileParameter must be Identifier")
	}
}
