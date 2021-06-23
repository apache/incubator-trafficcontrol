package staticdnsentry

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

	util "github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"
	"github.com/jmoiron/sqlx"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

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
	i = &TOStaticDNSEntry{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("staticDNSEntry must be creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("staticDNSEntry must be reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("staticDNSEntry must be updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("staticDNSEntry must be deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("staticDNSEntry must be Identifier")
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
	rows.AddRow("A_RECORD", "staticdnsentry")

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	tx := db.MustBegin()

	reqInfo := api.Info{Tx: tx}
	// invalid name, empty domainname
	ts := TOStaticDNSEntry{InfoerImpl: api.InfoerImpl{ReqInfo: &reqInfo}}
	errs := util.JoinErrsStr(test.SortErrors(test.SplitErrors(ts.Validate())))

	expectedErrs := util.JoinErrsStr([]error{
		errors.New(`'address' cannot be blank`),
		errors.New(`'deliveryserviceId' cannot be blank`),
		errors.New(`'host' cannot be blank`),
		errors.New(`'ttl' cannot be blank`),
		errors.New(`'typeId' cannot be blank`),
	})

	if !reflect.DeepEqual(expectedErrs, errs) {
		t.Errorf("expected %s, GOT %s", expectedErrs, errs)
	}
}
