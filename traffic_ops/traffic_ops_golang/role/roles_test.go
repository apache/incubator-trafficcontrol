package role

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

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"
)

func stringAddr(s string) *string {
	return &s
}
func intAddr(i int) *int {
	return &i
}

func getTestRoles() []tc.Role {
	roles := []tc.Role{
		{
			ID:          intAddr(1),
			Name:        stringAddr("role1"),
			Description: stringAddr("the first role"),
		},
		{
			ID:          intAddr(2),
			Name:        stringAddr("role2"),
			Description: stringAddr("the second role"),
		},
	}
	return roles
}

//removed sqlmock based ReadRoles test due to sqlmock / pq.Array() type incompatibility issue.

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
	i = &TORole{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("role must be creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("role must be reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("role must be updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("role must be deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("role must be Identifier")
	}
}

func TestValidate(t *testing.T) {
	// invalid name, empty domainname
	n := "not_a_valid_role"
	reqInfo := api.APIInfo{}
	r := TORole{ReqInfo: &reqInfo, Role: tc.Role{Name: &n}}
	errs := util.JoinErrsStr(test.SortErrors(test.SplitErrors(r.Validate())))

	expectedErrs := util.JoinErrsStr([]error{
		errors.New(`'description' cannot be blank`),
	})

	if !reflect.DeepEqual(expectedErrs, errs) {
		t.Errorf("expected %s, got %s", expectedErrs, errs)
	}

	//  name,  domainname both valid
	r = TORole{ReqInfo: &reqInfo, Role: tc.Role{Name: stringAddr("this is a valid name"), Description: stringAddr("this is a description")}}
	err := r.Validate()
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}

}
