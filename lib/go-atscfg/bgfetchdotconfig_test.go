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

func TestMakeBGFetchDotConfig(t *testing.T) {
	cdnName := "mycdn"

	server := makeGenericServer()
	server.CDNName = &cdnName
	hdr := "myHeaderComment"

	cfg, err := MakeBGFetchDotConfig(server, hdr)
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	if !strings.Contains(txt, hdr) {
		t.Errorf("expected: header comment '" + hdr + "', actual: missing")
	}

	if !strings.HasPrefix(strings.TrimSpace(txt), "#") {
		t.Errorf("expected: header comment, actual: missing")
	}
}
