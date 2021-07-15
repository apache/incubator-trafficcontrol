// Package cachegroupparameter is deprecated and will be removed with API v1-3.
package cachegroupparameter

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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var (
	cgpRows = []string{
		"config_file",
		"id",
		"last_updated",
		"name",
		"value",
		"secure",
	}
	cgRows = []string{
		"name",
	}
)

func TestReadCacheGroupParameters(t *testing.T) {

	var testCases = []struct {
		description          string
		storageError         error
		expectedUserError    bool
		params               map[string]string
		cgParams             []tc.CacheGroupParameterNullable
		cgExists             bool
		cgExistsStorageError error
		expectedReturnCode   int
	}{
		{
			description:       "Success: Read Cache Group Parameters",
			storageError:      nil,
			expectedUserError: false,
			params: map[string]string{
				"id": "1",
			},
			cgParams: []tc.CacheGroupParameterNullable{
				generateParameter("global", "param1", "val1", false, 1),
				generateParameter("global", "param2", "val2", false, 2),
			},
			cgExists:             true,
			cgExistsStorageError: nil,
			expectedReturnCode:   http.StatusOK,
		},
		{
			description:       "Success: Read Cache Group Parameters with parameter id",
			storageError:      nil,
			expectedUserError: false,
			params: map[string]string{
				"id":          "1",
				"parameterId": "1",
			},
			cgParams: []tc.CacheGroupParameterNullable{
				generateParameter("global", "param1", "val1", false, 1),
			},
			cgExists:             true,
			cgExistsStorageError: nil,
			expectedReturnCode:   http.StatusOK,
		},
		{
			description:       "Success: Read Cache Group Parameters no data",
			storageError:      nil,
			expectedUserError: false,
			params: map[string]string{
				"id": "1",
			},
			cgParams:             []tc.CacheGroupParameterNullable{},
			cgExists:             true,
			cgExistsStorageError: nil,
			expectedReturnCode:   http.StatusOK,
		},
		{
			description:       "Failure: Storage Error reading Cache Group Parameters",
			storageError:      errors.New("failure getting cache group parameters"),
			expectedUserError: false,
			params: map[string]string{
				"id": "1",
			},
			cgParams:             []tc.CacheGroupParameterNullable{},
			cgExists:             true,
			cgExistsStorageError: nil,
			expectedReturnCode:   http.StatusInternalServerError,
		},
		{
			description:       "Failure: User Error invalid params",
			storageError:      nil,
			expectedUserError: true,
			params: map[string]string{
				"id": "not_an_id",
			},
			cgParams:             []tc.CacheGroupParameterNullable{},
			cgExists:             true,
			cgExistsStorageError: nil,
			expectedReturnCode:   http.StatusBadRequest,
		},
		{
			description:       "Failure: System Error getting cache group",
			storageError:      nil,
			expectedUserError: false,
			params: map[string]string{
				"id": "1",
			},
			cgParams:             []tc.CacheGroupParameterNullable{},
			cgExists:             true,
			cgExistsStorageError: errors.New("error getting cache group"),
			expectedReturnCode:   http.StatusInternalServerError,
		},
		{
			description:       "Failure: Cache group does not exist",
			storageError:      nil,
			expectedUserError: true,
			params: map[string]string{
				"id": "1",
			},
			cgParams:             []tc.CacheGroupParameterNullable{},
			cgExists:             false,
			cgExistsStorageError: nil,
			expectedReturnCode:   http.StatusNotFound,
		},
	}
	toParameterReaders := map[string]api.Reader{
		"Parameters": &TOCacheGroupParameter{},
	}
	for _, testCase := range testCases {
		for toParameterKey, toParameterReader := range toParameterReaders {
			testCaseKey := fmt.Sprintf("%s - %s", testCase.description, toParameterKey)
			t.Run(testCaseKey, func(t *testing.T) {
				t.Log("Starting test scenario: ", testCaseKey)
				mockDB, mock, err := sqlmock.New()
				if err != nil {
					t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
				}
				defer mockDB.Close()
				db := sqlx.NewDb(mockDB, "sqlmock")
				defer db.Close()
				rows := sqlmock.NewRows(cgpRows)
				for _, cgParam := range testCase.cgParams {
					rows = rows.AddRow(
						cgParam.ConfigFile,
						cgParam.ID,
						cgParam.LastUpdated,
						cgParam.Name,
						cgParam.Value,
						cgParam.Secure,
					)
				}
				mock.ExpectBegin()
				cgr := sqlmock.NewRows(cgRows)
				if testCase.cgExistsStorageError != nil {
					mock.ExpectQuery("cachegroup").WillReturnError(testCase.cgExistsStorageError)
				} else {
					if testCase.cgExists {
						cgr = cgr.AddRow("cachegroup_name")
					}
					mock.ExpectQuery("cachegroup").WillReturnRows(cgr)
				}

				if testCase.storageError != nil {
					mock.ExpectQuery("cachegroup_parameter").WillReturnError(testCase.storageError)
				} else {
					mock.ExpectQuery("cachegroup_parameter").WillReturnRows(rows)
				}
				mock.ExpectCommit()

				reqInfo := api.APIInfo{Tx: db.MustBegin(), Params: testCase.params}
				toParameterReader.SetInfo(&reqInfo)

				parameters, userErr, sysErr, returnCode, _ := toParameterReader.Read(nil, false)

				if testCase.storageError != nil {
					if sysErr == nil {
						t.Errorf("Read error expected: received no sysErr")
					}
				} else if testCase.expectedUserError {
					if userErr == nil {
						t.Errorf("User error expected: received no userErr")
					}
				} else if testCase.cgExistsStorageError != nil {
					if sysErr == nil {
						t.Errorf("Read error expected: received no sysErr")
					}
				} else {
					if userErr != nil || sysErr != nil {
						t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
					}
					if len(parameters) != len(testCase.cgParams) {
						t.Errorf("cdn.Read expected: len(parameters) == %v, actual: %v", len(testCase.cgParams), len(parameters))
					}
				}
				if testCase.expectedReturnCode != returnCode {
					t.Errorf("Expected return code: %d, actual %d", testCase.expectedReturnCode, returnCode)
				}
			})
		}
	}
}

func generateParameter(configFile, param, val string, secureFlag bool, id int) tc.CacheGroupParameterNullable {
	lastUpdated := tc.TimeNoMod{}
	lastUpdated.Scan(time.Now())
	testParameter := tc.CacheGroupParameterNullable{
		ConfigFile:  &configFile,
		ID:          &id,
		LastUpdated: &lastUpdated,
		Name:        &param,
		Secure:      &secureFlag,
		Value:       &val,
	}
	return testParameter
}

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TOCacheGroupParameter{}

	if _, ok := i.(api.Reader); !ok {
		t.Errorf("CacheGroupParameter must be Reader")
	}
}
