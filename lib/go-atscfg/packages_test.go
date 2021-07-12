package atscfg

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
	"encoding/json"
	"testing"
)

func TestMakePackages(t *testing.T) {
	params := map[string][]string{
		"p0": []string{"p0v0", "p0v1"},
		"1":  []string{"p1v0"},
	}
	paramData := makeParamsFromMapArr("serverProfile", LogsXMLFileName, params)

	cfg, err := MakePackages(paramData, nil)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	packages := []pkg{}
	if err := json.Unmarshal([]byte(txt), &packages); err != nil {
		t.Fatalf("MakePackages expected a JSON array of objects, actual: " + err.Error())
	}

	for _, pkg := range packages {
		vals, ok := params[pkg.Name]
		if !ok {
			t.Errorf("MakePackages expected %+v actual %v\n", params, pkg.Name)
		}

		if !strArrContains(vals, pkg.Version) {
			t.Errorf("MakePackages expected %+v actual %v\n", vals, pkg.Version)
		}

		params[pkg.Name] = strArrRemove(vals, pkg.Version)
	}
}

func strArrContains(arr []string, str string) bool {
	for _, as := range arr {
		if as == str {
			return true
		}
	}
	return false
}

func strArrRemove(arr []string, str string) []string {
	// this is terribly inefficient, but it's just for testing, so it doesn't matter
	newArr := []string{}
	for _, as := range arr {
		if as == str {
			continue
		}
		newArr = append(newArr, as)
	}
	return newArr
}
