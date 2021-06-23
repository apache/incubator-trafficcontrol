package deliveryservice

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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestDeliveryServicesRequiredCapabilityInterfaces(t *testing.T) {
	var i interface{}
	i = &RequiredCapability{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("DeliveryServicesRequiredCapability must be Creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("DeliveryServicesRequiredCapability must be Reader")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("DeliveryServicesRequiredCapability must be Deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("DeliveryServicesRequiredCapability must be Identifier")
	}
}

func TestCreateDeliveryServicesRequiredCapability(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	mock.ExpectBegin()
	rc := RequiredCapability{
		api.InfoerImpl{
			ReqInfo: &api.Info{
				Tx:   db.MustBegin(),
				User: &auth.CurrentUser{PrivLevel: 30},
			},
		},
		tc.DeliveryServicesRequiredCapability{
			DeliveryServiceID: util.IntPtr(1),
			XMLID:             util.StrPtr("ds1"),
		},
	}

	mockTenantID(t, mock, 1)

	typeRows := sqlmock.NewRows([]string{"name", "required_capabilities", "topology"}).AddRow(
		"HTTP", "{}", nil,
	)
	mock.ExpectQuery("SELECT t.name.*").WillReturnRows(typeRows)

	scRows := sqlmock.NewRows([]string{"name"}).AddRow(
		"mem",
	)
	mock.ExpectQuery("SELECT name FROM server_capability").WillReturnRows(scRows)

	arrayRows := sqlmock.NewRows([]string{"array"})
	mock.ExpectQuery("SELECT ds.server FROM deliveryservice_server").WillReturnRows(arrayRows)

	rows := sqlmock.NewRows([]string{"required_capability", "deliveryservice_id", "last_updated"}).AddRow(
		util.StrPtr("mem"),
		util.IntPtr(1),
		time.Now(),
	)
	mock.ExpectQuery("INSERT INTO deliveryservices_required_capability").WillReturnRows(rows)

	userErr, sysErr, errCode := rc.Create()
	if userErr != nil {
		t.Fatalf(userErr.Error())
	}
	if sysErr != nil {
		t.Fatalf(sysErr.Error())
	}
	if got, want := errCode, http.StatusOK; got != want {
		t.Fatalf(fmt.Sprintf("got %d; expected http status code %d", got, want))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err.Error())
	}
}

func TestUnauthorizedCreateDeliveryServicesRequiredCapability(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	mock.ExpectBegin()
	rc := RequiredCapability{
		api.InfoerImpl{
			ReqInfo: &api.Info{
				Tx:   db.MustBegin(),
				User: &auth.CurrentUser{PrivLevel: 1},
			},
		},
		tc.DeliveryServicesRequiredCapability{
			DeliveryServiceID: util.IntPtr(1),
			XMLID:             util.StrPtr("ds1"),
		},
	}

	mockTenantID(t, mock, 0)

	userErr, sysErr, errCode := rc.Create()

	expErr := "not authorized on this tenant"
	if userErr.Error() != expErr {
		t.Fatalf("got %s; expected %s", userErr, expErr)
	}

	if sysErr != nil {
		t.Fatalf(userErr.Error())
	}

	if got, want := errCode, http.StatusForbidden; got != want {
		t.Fatalf(fmt.Sprintf("got %d; expected http status code %d", got, want))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err.Error())
	}
}

func TestReadDeliveryServicesRequiredCapability(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	capability := tc.DeliveryServicesRequiredCapability{
		RequiredCapability: util.StrPtr("mem"),
		DeliveryServiceID:  util.IntPtr(1),
		XMLID:              util.StrPtr("ds1"),
	}

	mock.ExpectBegin()
	rc := RequiredCapability{
		api.InfoerImpl{
			ReqInfo: &api.Info{
				Tx:   db.MustBegin(),
				User: &auth.CurrentUser{PrivLevel: 30},
			},
		},
		capability,
	}

	tenantRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery("SELECT id FROM user_tenant_children;").WillReturnRows(tenantRows)

	rows := sqlmock.NewRows([]string{"required_capability", "deliveryservice_id", "xml_id", "last_updated"}).AddRow(
		capability.RequiredCapability,
		capability.DeliveryServiceID,
		capability.XMLID,
		time.Now(),
	)
	mock.ExpectQuery("SELECT .* FROM deliveryservices_required_capability").WillReturnRows(rows)

	results, userErr, sysErr, errCode, _ := rc.Read(nil, false)
	if userErr != nil {
		t.Fatalf(userErr.Error())
	}
	if sysErr != nil {
		t.Fatalf(sysErr.Error())
	}
	if got, want := errCode, http.StatusOK; got != want {
		t.Fatalf(fmt.Sprintf("got %d; expected http status code %d", got, want))
	}
	if got, want := len(results), 1; got != want {
		t.Errorf("got %d; expected %d required capabilities assigned to deliveryservices", got, want)
	}

	for _, result := range results {
		cap, ok := result.(tc.DeliveryServicesRequiredCapability)
		if ok {
			if got, want := *cap.DeliveryServiceID, 1; got != want {
				t.Errorf("got %d; expected %d ", got, want)
			}
			if got, want := *cap.XMLID, "ds1"; got != want {
				t.Errorf("got %s; expected %s ", got, want)
			}
			if got, want := *cap.RequiredCapability, "mem"; got != want {
				t.Errorf("got %s; expected %s ", got, want)
			}
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err.Error())
	}
}

func TestDeleteDeliveryServicesRequiredCapability(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	mock.ExpectBegin()
	mockTenantID(t, mock, 1)

	mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(1, 1))

	rc := RequiredCapability{
		api.InfoerImpl{
			ReqInfo: &api.Info{
				Tx:   db.MustBegin(),
				User: &auth.CurrentUser{PrivLevel: 30},
			},
		},
		tc.DeliveryServicesRequiredCapability{
			RequiredCapability: util.StrPtr("mem"),
			DeliveryServiceID:  util.IntPtr(1),
			XMLID:              util.StrPtr("ds1"),
		},
	}

	userErr, sysErr, errCode := rc.Delete()
	if userErr != nil {
		t.Fatalf(userErr.Error())
	}
	if sysErr != nil {
		t.Fatalf(sysErr.Error())
	}
	if got, want := errCode, http.StatusOK; got != want {
		t.Fatalf(fmt.Sprintf("got %d; expected http status code %d", got, want))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err.Error())
	}
}

func TestUnauthorizedDeleteDeliveryServicesRequiredCapability(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	mock.ExpectBegin()
	rc := RequiredCapability{
		api.InfoerImpl{
			ReqInfo: &api.Info{
				Tx:   db.MustBegin(),
				User: &auth.CurrentUser{PrivLevel: 1},
			},
		},
		tc.DeliveryServicesRequiredCapability{
			DeliveryServiceID: util.IntPtr(1),
			XMLID:             util.StrPtr("ds1"),
		},
	}

	mockTenantID(t, mock, 0)

	userErr, sysErr, errCode := rc.Delete()

	expErr := "not authorized on this tenant"
	if userErr.Error() != expErr {
		t.Fatalf("got %s; expected %s", userErr, expErr)
	}

	if sysErr != nil {
		t.Fatalf(userErr.Error())
	}

	if got, want := errCode, http.StatusForbidden; got != want {
		t.Fatalf(fmt.Sprintf("got %d; expected http status code %d", got, want))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err.Error())
	}
}

func mockTenantID(t *testing.T, mock sqlmock.Sqlmock, x int) {
	t.Helper()

	tenantRows := sqlmock.NewRows([]string{"id"}).AddRow(
		x,
	)
	mock.ExpectQuery("SELECT tenant_id FROM deliveryservice").WillReturnRows(tenantRows)

	rows := sqlmock.NewRows([]string{"id", "active"}).AddRow(
		1,
		true,
	)
	mock.ExpectQuery("WITH RECURSIVE user_tenant_id as").WillReturnRows(rows)
}

func TestCreateDeliveryServicesRequiredCapabilityInvalidDSType(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	mock.ExpectBegin()
	rc := RequiredCapability{
		api.InfoerImpl{
			ReqInfo: &api.Info{
				Tx:   db.MustBegin(),
				User: &auth.CurrentUser{PrivLevel: 30},
			},
		},
		tc.DeliveryServicesRequiredCapability{
			DeliveryServiceID: util.IntPtr(1),
			XMLID:             util.StrPtr("ds1"),
		},
	}

	mockTenantID(t, mock, 1)

	typeRows := sqlmock.NewRows([]string{"name", "required_capabilities", "topology"}).AddRow(
		"ANY_MAP", "{}", nil,
	)
	mock.ExpectQuery("SELECT t.name.*").WillReturnRows(typeRows)

	userErr, sysErr, errCode := rc.Create()
	if userErr == nil {
		t.Fatal("Expected to get user error with invalid ds type")
	}
	if sysErr != nil {
		t.Fatalf(sysErr.Error())
	}
	if got, want := errCode, http.StatusBadRequest; got != want {
		t.Fatalf(fmt.Sprintf("got %d; expected http status code %d", got, want))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err.Error())
	}
}
