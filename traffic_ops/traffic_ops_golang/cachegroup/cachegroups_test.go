package cachegroup

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
	"net/http"
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

func getTestCacheGroups() []tc.CacheGroup {
	cgs := []tc.CacheGroup{}
	testCG1 := tc.CacheGroup{
		ID:                          1,
		Name:                        "cachegroup1",
		ShortName:                   "cg1",
		Latitude:                    38.7,
		Longitude:                   90.7,
		ParentCachegroupID:          2,
		SecondaryParentCachegroupID: 2,
		LocalizationMethods: []tc.LocalizationMethod{
			tc.LocalizationMethodDeepCZ,
			tc.LocalizationMethodCZ,
			tc.LocalizationMethodGeo,
		},
		Type:        "EDGE_LOC",
		TypeID:      6,
		LastUpdated: tc.TimeNoMod{Time: time.Now()},
		Fallbacks: []string{
			"cachegroup2",
			"cachegroup3",
		},
		FallbackToClosest: true,
	}
	cgs = append(cgs, testCG1)

	testCG2 := tc.CacheGroup{
		ID:                          1,
		Name:                        "parentCacheGroup",
		ShortName:                   "pg1",
		Latitude:                    38.7,
		Longitude:                   90.7,
		ParentCachegroupID:          1,
		SecondaryParentCachegroupID: 1,
		LocalizationMethods: []tc.LocalizationMethod{
			tc.LocalizationMethodDeepCZ,
			tc.LocalizationMethodCZ,
			tc.LocalizationMethodGeo,
		},
		Type:        "MID_LOC",
		TypeID:      7,
		LastUpdated: tc.TimeNoMod{Time: time.Now()},
	}
	cgs = append(cgs, testCG2)

	return cgs
}

func TestReadCacheGroups(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testCGs := getTestCacheGroups()
	rows := sqlmock.NewRows([]string{
		"id",
		"name",
		"short_name",
		"latitude",
		"longitude",
		"localization_methods",
		"parent_cachegroup_id",
		"parent_cachegroup_name",
		"secondary_parent_cachegroup_id",
		"secondary_parent_cachegroup_name",
		"type_name",
		"type_id",
		"last_updated",
		"fallbacks",
		"fallbackToClosest",
	})

	for _, ts := range testCGs {
		rows = rows.AddRow(
			ts.ID,
			ts.Name,
			ts.ShortName,
			ts.Latitude,
			ts.Longitude,
			[]byte("{DEEP_CZ,CZ,GEO}"),
			ts.ParentCachegroupID,
			ts.ParentName,
			ts.SecondaryParentCachegroupID,
			ts.SecondaryParentName,
			ts.Type,
			ts.TypeID,
			ts.LastUpdated,
			[]byte("{cachegroup2,cachegroup3}"),
			ts.FallbackToClosest,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin(), Params: map[string]string{"id": "1"}}
	obj := TOCacheGroup{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.CacheGroupNullable{},
	}
	cachegroups, userErr, sysErr, _, _ := obj.Read(nil, false)

	if userErr != nil || sysErr != nil {
		t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
	}

	if len(cachegroups) != 2 {
		t.Errorf("cdn.Read expected: len(cachegroups) == 2, actual: %v", len(cachegroups))
	}
}

func TestFuncs(t *testing.T) {
	if strings.Index(SelectQuery(), "SELECT") != 0 {
		t.Errorf("expected SelectQuery to start with SELECT")
	}
	if strings.Index(InsertQuery(), "INSERT") != 0 {
		t.Errorf("expected InsertQuery to start with INSERT")
	}
	if strings.Index(UpdateQuery(), "UPDATE") != 0 {
		t.Errorf("expected UpdateQuery to start with UPDATE")
	}
	if strings.Index(DeleteQuery(), "DELETE") != 0 {
		t.Errorf("expected DeleteQuery to start with DELETE")
	}
}

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TOCacheGroup{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("cachegroup must be creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("cachegroup must be reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("cachegroup must be updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("cachegroup must be deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("cachegroup must be Identifier")
	}
}

func TestValidate(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	rows := sqlmock.NewRows([]string{"name", "use_in_table"})
	rows.AddRow("EDGE_LOC", "cachegroup")

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT\\s+name,\\s+use_in_table").WillReturnRows(rows)
	tx := db.MustBegin()
	reqInfo := api.Info{Tx: tx}

	// invalid name, shortname, loattude, and longitude
	id := 1
	nm := "not!a!valid!cachegroup"
	sn := "not!a!valid!shortname"
	la := -190.0
	lo := -190.0
	lm := []tc.LocalizationMethod{tc.LocalizationMethodGeo, tc.LocalizationMethodInvalid}
	ty := "EDGE_LOC"
	ti := 6
	lu := tc.TimeNoMod{Time: time.Now()}
	c := TOCacheGroup{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.CacheGroupNullable{
			ID:                  &id,
			Name:                &nm,
			ShortName:           &sn,
			Latitude:            &la,
			Longitude:           &lo,
			LocalizationMethods: &lm,
			Type:                &ty,
			TypeID:              &ti,
			LastUpdated:         &lu,
		},
	}
	errs := util.JoinErrsStr(test.SortErrors(test.SplitErrors(c.Validate())))

	expectedErrs := util.JoinErrsStr([]error{
		errors.New(`'latitude' Must be a floating point number within the range +-90`),
		errors.New(`'localizationMethods' 'invalid' is not one of [CZ DEEP_CZ GEO]`),
		errors.New(`'longitude' Must be a floating point number within the range +-180`),
		errors.New(`'name' invalid characters found - Use alphanumeric . or - or _ .`),
		errors.New(`'shortName' invalid characters found - Use alphanumeric . or - or _ .`),
		errors.New(`'type' unable to check whether cachegroup not!a!valid!cachegroup is used in any topologies`),
	})

	if !reflect.DeepEqual(expectedErrs, errs) {
		t.Errorf("expected %s, got %s", expectedErrs, errs)
	}

	rows.AddRow("EDGE_LOC", "cachegroup")
	mock.ExpectQuery("SELECT\\s+name,\\s+use_in_table").WillReturnRows(rows)

	rows = sqlmock.NewRows([]string{"id", "name"})
	mock.ExpectQuery("SELECT\\s+t\\.id\\s+FROM\\s+cachegroup").WillReturnRows(rows)

	//  valid name, shortName latitude, longitude
	nm = "This.is.2.a-Valid---Cachegroup."
	sn = `awesome-cachegroup`
	la = 90.0
	lo = 90.0
	lm = []tc.LocalizationMethod{tc.LocalizationMethodGeo, tc.LocalizationMethodCZ, tc.LocalizationMethodDeepCZ}
	c = TOCacheGroup{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.CacheGroupNullable{
			ID:                  &id,
			Name:                &nm,
			ShortName:           &sn,
			Latitude:            &la,
			Longitude:           &lo,
			LocalizationMethods: &lm,
			Type:                &ty,
			TypeID:              &ti,
			LastUpdated:         &lu,
		},
	}
	err = c.Validate()
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
}

func TestBadTypeParamCacheGroups(t *testing.T) {
	detail := `cachegroup read: converting cachegroup type to integer strconv.Atoi: parsing "wrong": invalid syntax`
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testCGs := getTestCacheGroups()
	rows := sqlmock.NewRows([]string{
		"id",
		"name",
		"short_name",
		"latitude",
		"longitude",
		"localization_methods",
		"parent_cachegroup_id",
		"parent_cachegroup_name",
		"secondary_parent_cachegroup_id",
		"secondary_parent_cachegroup_name",
		"type_name",
		"type_id",
		"last_updated",
		"fallbacks",
		"fallbackToClosest",
	})

	for _, ts := range testCGs {
		rows = rows.AddRow(
			ts.ID,
			ts.Name,
			ts.ShortName,
			ts.Latitude,
			ts.Longitude,
			[]byte("{DEEP_CZ,CZ,GEO}"),
			ts.ParentCachegroupID,
			ts.ParentName,
			ts.SecondaryParentCachegroupID,
			ts.SecondaryParentName,
			ts.Type,
			ts.TypeID,
			ts.LastUpdated,
			[]byte("{cachegroup2,cachegroup3}"),
			ts.FallbackToClosest,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin(), Params: map[string]string{"type": "wrong"}}
	obj := TOCacheGroup{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.CacheGroupNullable{},
	}
	_, userErr, _, sc, _ := obj.Read(nil, false)
	if sc != http.StatusBadRequest {
		t.Errorf("Read expected status code: Bad Request, actual: %v ", sc)
	}
	if userErr == nil {
		t.Fatalf("Read expected: user error with details %v, actual: nil", detail)
	}
	if userErr.Error() != detail {
		t.Fatalf("Read expected: user error with details %v, actual: %v", detail, userErr.Error())
	}
}
