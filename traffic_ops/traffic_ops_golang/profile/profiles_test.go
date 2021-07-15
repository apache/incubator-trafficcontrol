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

func getTestProfiles() []tc.ProfileNullable {
	profiles := []tc.ProfileNullable{}

	lastUpdated := tc.TimeNoMod{}
	lastUpdated.Scan(time.Now())
	ID := 1
	name := "profile1"
	description := "desc1"
	pt := "TR_PROFILE"
	cdnID := 1
	cdnName := "cdn1"
	rd := true

	testCase := tc.ProfileNullable{
		ID:              &ID,
		LastUpdated:     &lastUpdated,
		Name:            &name,
		Description:     &description,
		CDNName:         &cdnName,
		CDNID:           &cdnID,
		RoutingDisabled: &rd,
		Type:            &pt,
	}
	profiles = append(profiles, testCase)

	testCase2 := testCase
	name = "profile2"
	testCase2.Name = &name
	profiles = append(profiles, testCase2)

	return profiles
}

func TestGetProfiles(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testCase := getTestProfiles()
	cols := test.ColsFromStructByTag("db", tc.ProfileNullable{})
	rows := sqlmock.NewRows(cols)

	for _, ts := range testCase {
		rows = rows.AddRow(
			ts.ID,
			ts.LastUpdated,
			ts.Name,
			ts.Description,
			ts.CDNName,
			ts.CDNID,
			ts.RoutingDisabled,
			ts.Type,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.APIInfo{Tx: db.MustBegin(), Params: map[string]string{"name": "1"}}

	obj := TOProfile{
		api.APIInfoImpl{ReqInfo: &reqInfo},
		tc.ProfileNullable{},
	}
	profiles, userErr, sysErr, _, _ := obj.Read(nil, false)
	if userErr != nil || sysErr != nil {
		t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
	}

	if len(profiles) != 2 {
		t.Errorf("profile.Read expected: len(profiles) == 2, actual: %v", len(profiles))
	}

}

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TOProfile{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("Profile must be Creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("Profile must be Reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("Profile must be Updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("Profile must be Deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("Profile must be Identifier")
	}
}

func TestValidate(t *testing.T) {
	p := TOProfile{}
	errs := util.JoinErrsStr(test.SortErrors(test.SplitErrors(p.Validate())))
	expected := util.JoinErrsStr(test.SortErrors([]error{
		errors.New("'cdn' cannot be blank"),
		errors.New("'description' cannot be blank"),
		errors.New("'name' required and cannot be blank"),
		errors.New("'type' cannot be blank"),
	}))

	if !reflect.DeepEqual(expected, errs) {
		t.Errorf("expected %++v,  got %++v", expected, errs)
	}

	p.CDNID = new(int)
	*p.CDNID = 1
	p.Description = new(string)
	*p.Description = "description"
	p.Name = new(string)
	*p.Name = "A name with spaces"
	p.Type = new(string)
	*p.Type = "type"

	err := p.Validate()
	if !strings.Contains(err.Error(), "cannot contain spaces") {
		t.Error("Expected an error about the Profile name containing spaces")
	}

}
