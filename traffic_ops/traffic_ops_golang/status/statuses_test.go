package status

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
	"database/sql"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"
	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func getTestStatuses() []tc.Status {
	cdns := []tc.Status{}
	testStatus := tc.Status{
		Description: "description",
		ID:          1,
		Name:        "cdn1",
		LastUpdated: tc.TimeNoMod{Time: time.Now()},
	}
	cdns = append(cdns, testStatus)

	testStatus2 := testStatus
	testStatus2.Name = "cdn2"
	testStatus2.Description = "description2"
	cdns = append(cdns, testStatus2)

	return cdns
}

func TestReadStatuses(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testStatuses := getTestStatuses()
	cols := test.ColsFromStructByTag("db", tc.Status{})
	rows := sqlmock.NewRows(cols)

	for _, ts := range testStatuses {
		rows = rows.AddRow(
			ts.Description,
			ts.ID,
			ts.LastUpdated,
			ts.Name,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin(), Params: map[string]string{"dsId": "1"}}

	obj := TOStatus{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.StatusNullable{},
		sql.NullString{},
	}
	statuses, userErr, sysErr, _, _ := obj.Read(nil, false)
	if userErr != nil || sysErr != nil {
		t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
	}

	if len(statuses) != 2 {
		t.Errorf("status.Read expected: len(statuses) == 2, actual: %v", len(statuses))
	}
}

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TOStatus{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("Status must be Creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("Status must be Reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("Status must be Updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("Status must be Deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("Status must be Identifier")
	}
}
