package parameter

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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"
	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"

	"encoding/json"
)

func getTestParameters() []tc.ParameterNullable {
	parameters := []tc.ParameterNullable{}
	lastUpdated := tc.TimeNoMod{}
	lastUpdated.Scan(time.Now())
	configFile := "global"
	secureFlag := false
	ID := 1
	param := "paramname1"
	val := "val1"

	testParameter := tc.ParameterNullable{
		ConfigFile:  &configFile,
		ID:          &ID,
		LastUpdated: &lastUpdated,
		Name:        &param,
		Profiles:    json.RawMessage(`["foo","bar"]`),
		Secure:      &secureFlag,
		Value:       &val,
	}
	parameters = append(parameters, testParameter)

	testParameter2 := testParameter
	testParameter2.Name = &param
	testParameter2.Value = &val
	testParameter2.ConfigFile = &configFile
	testParameter2.Profiles = json.RawMessage(`["foo","baz"]`)
	parameters = append(parameters, testParameter2)

	return parameters
}

func TestGetParameters(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testParameters := getTestParameters()
	cols := test.ColsFromStructByTag("db", tc.ParameterNullable{})
	rows := sqlmock.NewRows(cols)

	for _, ts := range testParameters {
		rows = rows.AddRow(
			ts.ConfigFile,
			ts.ID,
			ts.LastUpdated,
			ts.Name,
			ts.Profiles,
			ts.Secure,
			ts.Value,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{
		Tx:     db.MustBegin(),
		User:   &auth.CurrentUser{PrivLevel: 30},
		Params: map[string]string{"name": "1"},
	}
	obj := TOParameter{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.ParameterNullable{},
	}
	pps, userErr, sysErr, _, _ := obj.Read(nil, false)
	if userErr != nil || sysErr != nil {
		t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
	}

	if len(pps) != 2 {
		t.Errorf("parameter.Read expected: len(pps) == 2, actual: %v", len(pps))
	}

}

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TOParameter{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("Parameter must be Creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("Parameter must be Reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("Parameter must be Updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("Parameter must be Deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("Parameter must be Identifier")
	}
}
