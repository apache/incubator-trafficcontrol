package federations

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

	"github.com/jmoiron/sqlx"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestCheckFedDSDeletion(t *testing.T) {
	var testCases = []struct {
		description  string
		returnArray  string
		expectUsrErr bool
		expectUsrStr string
		inputDSID    int
	}{
		{
			description:  "Success: Deleted Federation Delivery Service",
			returnArray:  "{1,2}",
			expectUsrErr: false,
			expectUsrStr: "",
			inputDSID:    1,
		},
		{
			description:  "Failure: Remove last Federation Delivery Service",
			returnArray:  "{1}",
			expectUsrErr: true,
			expectUsrStr: "a federation must have at least one delivery service assigned",
			inputDSID:    1,
		},
		{
			description:  "Failure: Federation not found",
			returnArray:  "{}",
			expectUsrErr: true,
			expectUsrStr: "federation 1 not found",
			inputDSID:    1,
		},
		{
			description:  "Failure: Delivery Service not found",
			returnArray:  "{3,4}",
			expectUsrErr: true,
			expectUsrStr: "delivery service 1 is not associated with federation 1",
			inputDSID:    1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Log("Starting test scenario: ", tc.description)
			mockDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer mockDB.Close()
			db := sqlx.NewDb(mockDB, "sqlmock")
			defer db.Close()
			rows := sqlmock.NewRows([]string{"array"})
			rows = rows.AddRow(tc.returnArray)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT").WillReturnRows(rows)
			mock.ExpectCommit()
			errs := checkFedDSDeletion(db.MustBegin().Tx, 1, tc.inputDSID)
			if tc.expectUsrErr {
				if errs.UserError == nil {
					t.Errorf("User error expected: received none")
				}
				// TODO: this segfaults if the above error is reached. Should be an else-if
				if errs.UserError.Error() != tc.expectUsrStr {
					t.Errorf("Expected error with text %v: received %v", tc.expectUsrStr, errs.UserError.Error())
				}
			}
			//TODO: these print the wrong error on failure
			if !tc.expectUsrErr && errs.UserError != nil {
				t.Errorf("User error not expected: received error %v", err)
			}
			if errs.SystemError != nil {
				t.Errorf("System error not expected: received error %v", err)
			}
		})
	}
}
