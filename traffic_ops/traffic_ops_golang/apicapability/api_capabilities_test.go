package apicapability

import (
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/jmoiron/sqlx"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

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

func TestGetAPICapabilities(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	caps := getTestCapabilities()

	rows := sqlmock.NewRows([]string{
		"id",
		"http_method",
		"route",
		"capability",
		"last_updated",
	})

	for _, c := range caps {
		rows = rows.AddRow(
			c.ID,
			c.HTTPMethod,
			c.Route,
			c.Capability,
			c.LastUpdated,
		)
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	results, errs := getAPICapabilities(db.MustBegin(), map[string]string{})

	if errs.Occurred() {
		t.Errorf("Read expected: no errors, actual: %v", errs)
	}

	if len(results) != 4 {
		t.Errorf("cdn.Read expected: len(results) == 4, actual: %v", len(results))
	}
}

func getTestCapabilities() []tc.APICapability {
	return []tc.APICapability{
		tc.APICapability{
			ID:          1,
			HTTPMethod:  "GET",
			Route:       "asns",
			Capability:  "asns-read",
			LastUpdated: tc.TimeNoMod{Time: time.Now()},
		},
		tc.APICapability{
			ID:          2,
			HTTPMethod:  "POST",
			Route:       "asns",
			Capability:  "asns-write",
			LastUpdated: tc.TimeNoMod{Time: time.Now()},
		},
		tc.APICapability{
			ID:          3,
			HTTPMethod:  "PUT",
			Route:       "asns/*",
			Capability:  "asns-write",
			LastUpdated: tc.TimeNoMod{Time: time.Now()},
		},
		tc.APICapability{
			ID:          4,
			HTTPMethod:  "DELETE",
			Route:       "asns/*",
			Capability:  "asns-write",
			LastUpdated: tc.TimeNoMod{Time: time.Now()},
		},
	}
}
