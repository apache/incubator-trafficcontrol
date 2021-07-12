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
	"strings"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

func TestMakeDropQStringDotConfig(t *testing.T) {
	dropQStringVal := "myDropQStringVal"
	profileName := "myProfile"

	server := makeGenericServer()
	server.Profile = &profileName

	params := []tc.Parameter{
		{
			Name:       DropQStringDotConfigParamName,
			ConfigFile: DropQStringDotConfigFileName,
			Value:      dropQStringVal,
			Profiles:   []byte(`["` + profileName + `"]`),
		},
	}

	hdr := "myHeaderComment"

	cfg, err := MakeDropQStringDotConfig(server, params, &DropQStringDotConfigOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr)

	if !strings.Contains(txt, dropQStringVal) {
		t.Errorf("expected dropQStringVal '"+dropQStringVal+"' actual comment, actual: '%v'", txt)
	}

}
