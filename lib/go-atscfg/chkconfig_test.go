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

	"github.com/apache/trafficcontrol/lib/go-tc"
)

func TestMakeChkconfig(t *testing.T) {
	serverProfile := "sp0"
	params := []tc.Parameter{
		{
			Name:       "p0",
			ConfigFile: ChkconfigParamConfigFile,
			Value:      "p0v0",
			Profiles:   []byte(`["` + serverProfile + `"]`),
		},
		{
			Name:       "p0",
			ConfigFile: ChkconfigParamConfigFile,
			Value:      "p0v1",
			Profiles:   []byte(`["` + serverProfile + `"]`),
		},
		{
			Name:       "1",
			ConfigFile: ChkconfigParamConfigFile,
			Value:      "p1v0",
			Profiles:   []byte(`["` + serverProfile + `"]`),
		},
	}

	cfg, err := MakeChkconfig(params, &ChkconfigOpts{})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	chkconfig := []chkConfigEntry{}
	if err := json.Unmarshal([]byte(txt), &chkconfig); err != nil {
		t.Fatalf("MakePackages expected a JSON array of objects, actual: " + err.Error())
	}

	paramsMap := paramsToMultiMap(params)

	for _, chkConfigEntry := range chkconfig {
		vals, ok := paramsMap[chkConfigEntry.Name]
		if !ok {
			t.Errorf("expected %+v actual %v\n", params, chkConfigEntry.Name)
		}

		if !strArrContains(vals, chkConfigEntry.Val) {
			t.Errorf("expected %+v actual %v\n", vals, chkConfigEntry.Val)
		}

		paramsMap[chkConfigEntry.Name] = strArrRemove(vals, chkConfigEntry.Val)
	}
}
