package apitenant

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
	"net/http"
	"strconv"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"
	"github.com/jmoiron/sqlx"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TOTenant{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("Tenant must be Creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("Tenant must be Reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("Tenant must be Updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("Tenant must be Deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("Tenant must be Identifier")
	}
	if _, ok := i.(api.Tenantable); !ok {
		t.Errorf("Tenant must be Tenantable")
	}
}

// TestIsUpdateable is a test to ensure attempts to update a tenant
// pass validation beforehand.
func TestIsUpdateable(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	// First test, attempt to change root
	root := getRootTestTenant()
	userErr, _, statusCode := root.isUpdatable()
	if userErr == nil && statusCode != http.StatusBadRequest {
		t.Errorf("Should not be able to update root tenant. userErr = %s, statuscode = %d", userErr, statusCode)
	}

	// Second test, attempt to change Child's (ID:7) parent from ID:3 to ID:1 (root)
	child := getValidChildTenant()
	updateParentIDValue := 1
	child.ParentID = &updateParentIDValue // set parent ID as would be done through the GenericUpdate and Keys call

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT")

	child.ReqInfo = &api.Info{Tx: db.MustBegin(), Params: map[string]string{"id": strconv.Itoa(*child.ID)}}
	userErr, _, statusCode = child.isUpdatable()
	if userErr != nil && statusCode != http.StatusOK {
		t.Errorf("Should be able to update child to new parent (from Parent to Root). userErr = %s, statuscode = %d", userErr, statusCode)
	}

	// Third test, attempt to change Parent from ID:3 to ID:7 (Child)
	parent := getValidParentTenant()
	updateParentIDValue = 7
	parent.ParentID = &updateParentIDValue // set parent ID as would be done through the GenericUpdate and Keys call
	cols := test.ColsFromStructByTag("db", tc.TenantNullable{})
	rows := sqlmock.NewRows(cols).AddRow(
		child.ID,
		child.Name,
		child.Active,
		child.LastUpdated,
		child.ParentID,
		child.ParentName,
	)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	parent.ReqInfo = &api.Info{Tx: db.MustBegin(), Params: map[string]string{"id": strconv.Itoa(*parent.ID)}}
	userErr, _, statusCode = parent.isUpdatable()
	if userErr == nil && statusCode != http.StatusBadRequest {
		t.Errorf("Should NOT be able to update parent to own child (from Parent to Child). userErr = %s, statuscode = %d", userErr, statusCode)
	}

}

func getValidChildTenant() *TOTenant {
	ten := &TOTenant{}
	var tenid int = 7
	var tenname string = "Child"
	var tenact bool = true
	var tenpid int = 3
	var tenpaname string = "Parent"
	ten.TenantNullable = tc.TenantNullable{
		ID:          &tenid,
		Name:        &tenname,
		Active:      &tenact,
		LastUpdated: tc.NewTimeNoMod(),
		ParentID:    &tenpid,
		ParentName:  &tenpaname,
	}
	ten.ReqInfo = &api.Info{}
	return ten
}

func getValidParentTenant() *TOTenant {
	ten := &TOTenant{}
	var tenid int = 3
	var tenname string = "Parent"
	var tenact bool = true
	var tenpid int = 1
	var tenpaname string = "root"
	ten.TenantNullable = tc.TenantNullable{
		ID:          &tenid,
		Name:        &tenname,
		Active:      &tenact,
		LastUpdated: tc.NewTimeNoMod(),
		ParentID:    &tenpid,
		ParentName:  &tenpaname,
	}
	ten.ReqInfo = &api.Info{}
	return ten
}

func getRootTestTenant() *TOTenant {
	ten := &TOTenant{}
	var tenid int = 1
	var tenname string = "root"
	var tenact bool = true
	ten.TenantNullable = tc.TenantNullable{
		ID:          &tenid,
		Name:        &tenname,
		Active:      &tenact,
		LastUpdated: tc.NewTimeNoMod(),
		ParentID:    nil,
		ParentName:  nil,
	}
	ten.ReqInfo = &api.Info{}
	return ten
}
