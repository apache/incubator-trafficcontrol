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
)

func TestMakePluginDotConfig(t *testing.T) {
	profileName := "myProfile"
	hdr := "myHeaderComment"

	paramData := makeParamsFromMap("serverProfile", PluginFileName, map[string]string{
		"param0": "val0",
		"param1": "val1",
		"param2": "val2",
	})

	server := makeGenericServer()
	server.Profile = &profileName

	cfg, err := MakePluginDotConfig(server, paramData, &PluginDotConfigOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	testComment(t, txt, hdr)

	if !strings.Contains(txt, "param0 val0") {
		t.Errorf("expected config to contain paramData 'param0 val0', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "param1 val1") {
		t.Errorf("expected config to contain paramData 'param1 val1', actual: '%v'", txt)
	}
	if !strings.Contains(txt, "param2 val2") {
		t.Errorf("expected config to contain paramData 'param2 val2', actual: '%v'", txt)
	}
}
