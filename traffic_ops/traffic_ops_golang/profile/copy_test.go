package profile

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
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault/backends/disabled"

	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestCopyProfileInvalidExistingProfile(t *testing.T) {
	testCases := []struct {
		description     string
		profile         tc.ProfileCopyResponse
		existingProfile tc.ProfileNullable
		newProfileID    int

		mockProfileExists int // How many profiles should SQL return when finding new profile
		mockReadProfile   int // How many profiles should SQL return when reading existing profile

		sysErr  string
		userErr string
	}{
		{
			description: "multiple profiles with existing name returned",
			profile: tc.ProfileCopyResponse{
				Response: tc.ProfileCopy{
					ExistingName: "existingProfile",
					Name:         "newProfile",
				},
			},
			existingProfile: tc.ProfileNullable{
				ID:              util.IntPtr(1),
				Name:            util.StrPtr("existingProfile"),
				Description:     util.StrPtr("desc1"),
				CDNID:           util.IntPtr(1),
				RoutingDisabled: util.BoolPtr(true),
				Type:            util.StrPtr("TEST_PROFILE"),
			},
			mockReadProfile: 2,
			sysErr:          "multiple profiles with name existingProfile returned",
		},
		{
			description: "existing profile does not exist",
			profile: tc.ProfileCopyResponse{
				Response: tc.ProfileCopy{
					ExistingName: "existingProfile",
					Name:         "newProfile",
				},
			},
			userErr: "profile with name existingProfile does not exist",
		},
	}

	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {

			mockDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf(err.Error())
			}

			db := sqlx.NewDb(mockDB, "sqlmock")
			defer db.Close()

			mock.ExpectBegin()

			mockFindProfile(t, mock, c.profile.Response.Name, c.mockProfileExists)
			mockReadProfile(t, mock, c.existingProfile, c.mockReadProfile)

			inf := api.Info{
				Tx: db.MustBegin(),
				Params: map[string]string{
					"existing_profile": c.profile.Response.ExistingName,
					"new_profile":      c.profile.Response.Name,
				},
			}

			errs := copyProfile(&inf, &c.profile.Response)
			if c.userErr != "" { // Check if we expect a user error for this test
				if got, want := errs.userErr.Error(), c.userErr; got != want {
					t.Fatalf("got err=%s; expected err=%s", got, want)
				}
			} else if errs.userErr != nil {
				t.Fatalf(errs.userErr.Error())
			}

			if c.sysErr != "" { // Check if we expect a sys error for this test
				if got, want := errs.sysErr.Error(), c.sysErr; got != want {
					t.Fatalf("got err=%s; expected err=%s", got, want)
				}
			} else if errs.sysErr != nil {
				t.Fatalf(errs.sysErr.Error())
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expections: %s", err)
			}
		})
	}
}

func TestCopyNewProfileExists(t *testing.T) {

	profile := tc.ProfileCopyResponse{
		Response: tc.ProfileCopy{
			ExistingName: "existingProfile",
			Name:         "newProfile",
		},
	}

	expectedErr := "profile with name newProfile already exists"

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(err.Error())
	}

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	mock.ExpectBegin()

	mockFindProfile(t, mock, profile.Response.Name, 1)

	inf := api.Info{
		Tx: db.MustBegin(),
		Params: map[string]string{
			"existing_profile": profile.Response.ExistingName,
			"new_profile":      profile.Response.Name,
		},
	}

	errs := copyProfile(&inf, &profile.Response)
	if got, want := errs.userErr.Error(), expectedErr; got != want {
		t.Fatalf("got err=%s; expected err=%s", got, want)
	}

	if errs.sysErr != nil {
		t.Fatalf(errs.sysErr.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestCopyProfile(t *testing.T) {
	profile := tc.ProfileCopyResponse{
		Response: tc.ProfileCopy{
			ExistingID:   1,
			ExistingName: "existingProfile",
			ID:           2,
			Name:         "newProfile",
		},
	}

	existingProfile := tc.ProfileNullable{
		ID:              util.IntPtr(1),
		Name:            util.StrPtr("existingProfile"),
		Description:     util.StrPtr("desc1"),
		CDNID:           util.IntPtr(1),
		RoutingDisabled: util.BoolPtr(true),
		Type:            util.StrPtr("TEST_PROFILE"),
	}

	expectedID := 2

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(err.Error())
	}

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	mock.ExpectBegin()
	mockFindProfile(t, mock, profile.Response.Name, 0)
	mockReadProfile(t, mock, existingProfile, 1)
	mockInsertProfile(t, mock, expectedID)
	mockFindParams(t, mock, profile.Response.ExistingName)
	mockInsertParams(t, mock, profile.Response.ID)

	req := mockHTTPReq(t, "profiles/name/{new_profile}/copy/{existing_profile}", db)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CopyProfileHandler)
	handler.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("hanlder returned wrong status code: got %v want %v", got, want)
	}

	expErr := "created new profile [newProfile] from existing profile [existingProfile]"
	if !strings.Contains(rr.Body.String(), expErr) {
		t.Fatalf("got %s; expected %s", rr.Body.String(), expErr)
	}
}

func mockFindProfile(t *testing.T, mock sqlmock.Sqlmock, name string, results int) {
	t.Helper()
	query := regexp.QuoteMeta(`SELECT count(*) from profile where name = $1`)
	profileExists := sqlmock.NewRows([]string{"count"}).AddRow(results)
	mock.ExpectQuery(query).WithArgs(name).WillReturnRows(profileExists)
}

func mockReadProfile(t *testing.T, mock sqlmock.Sqlmock, profile tc.ProfileNullable, results int) {
	t.Helper()

	existingRow := sqlmock.NewRows([]string{
		"id",
		"name",
		"description",
		"cdn",
		"routing_disabled",
		"type",
	})

	for i := 0; i < results; i++ {
		existingRow.AddRow(
			profile.ID,
			profile.Name,
			profile.Description,
			profile.CDNID,
			profile.RoutingDisabled,
			profile.Type,
		)
	}

	mock.ExpectQuery("SELECT .* FROM profile").WillReturnRows(existingRow)
}

func mockInsertProfile(t *testing.T, mock sqlmock.Sqlmock, id int) {
	newRow := sqlmock.NewRows([]string{
		"id",
		"last_updated",
	}).AddRow(
		id,
		time.Now(),
	)
	mock.ExpectQuery("INSERT INTO profile").WillReturnRows(newRow)
}

func mockFindParams(t *testing.T, mock sqlmock.Sqlmock, name string) {
	t.Helper()

	existingRow := sqlmock.NewRows([]string{
		"profile",
		"parameter_id",
		"last_updated",
	}).AddRow(
		name,
		1,
		time.Now(),
	)

	mock.ExpectQuery("SELECT .* FROM profile_parameter").WillReturnRows(existingRow)
}

func mockInsertParams(t *testing.T, mock sqlmock.Sqlmock, id int) {
	t.Helper()

	existingRow := sqlmock.NewRows([]string{
		"profile",
		"parameter",
		"last_updated",
	}).AddRow(
		id,
		1,
		time.Now(),
	)

	mock.ExpectQuery("INSERT INTO profile_parameter").WillReturnRows(existingRow)
}

func mockHTTPReq(t *testing.T, path string, db *sqlx.DB) *http.Request {
	req, err := http.NewRequest("POST", path, strings.NewReader(
		`{"existing_profile", "existingProfile", "new_profile", "newProfile"}`))
	if err != nil {
		t.Error("Error creating new request")
	}

	cfg := config.Config{ConfigTrafficOpsGolang: config.ConfigTrafficOpsGolang{DBQueryTimeoutSeconds: 20}}
	ctx := req.Context()
	ctx = context.WithValue(ctx, auth.CurrentUserKey,
		auth.CurrentUser{UserName: "username", ID: 1, PrivLevel: auth.PrivLevelAdmin})
	ctx = context.WithValue(ctx, "db", db)
	ctx = context.WithValue(ctx, "context", &cfg)
	ctx = context.WithValue(ctx, "reqid", uint64(0))
	var tv trafficvault.TrafficVault = &disabled.Disabled{}
	ctx = context.WithValue(ctx, api.TrafficVaultContextKey, tv)
	ctx = context.WithValue(ctx, "pathParams", map[string]string{"existing_profile": "existingProfile", "new_profile": "newProfile"})

	// Add our context to the request
	req = req.WithContext(ctx)
	return req
}
