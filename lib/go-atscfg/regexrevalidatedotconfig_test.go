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
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
)

func TestMakeRegexRevalidateDotConfig(t *testing.T) {
	cdnName := "mycdn"
	hdr := "myHeaderComment"

	server := makeGenericServer()
	server.CDNName = &cdnName

	ds := makeGenericDS()
	ds.CDNName = &cdnName
	ds.XMLID = util.StrPtr("myds")
	dses := []DeliveryService{*ds}

	params := makeParamsFromMapArr("GLOBAL", RegexRevalidateFileName, map[string][]string{
		RegexRevalidateMaxRevalDurationDaysParamName: []string{"42"},
		"unrelated": []string{"unrelated0", "unrelated1"},
	})

	jobs := []tc.Job{
		tc.Job{
			AssetURL:        "assetURL0",
			StartTime:       time.Now().Add(42*24*time.Hour + time.Hour).Format(tc.JobTimeFormat),
			DeliveryService: "myds",
			CreatedBy:       "me",
			ID:              42,
			Parameters:      "TTL:14h",
			Keyword:         JobKeywordPurge,
		},
		tc.Job{
			AssetURL:        "expiredassetURL0",
			StartTime:       time.Now().Add(-24 * time.Hour).Format(tc.JobTimeFormat),
			DeliveryService: "expiredmyds",
			CreatedBy:       "expiredme",
			ID:              42,
			Parameters:      "TTL:14h",
			Keyword:         JobKeywordPurge,
		},
		tc.Job{
			AssetURL:        "refetchasset##REFETCH##",
			StartTime:       time.Now().Add(24 * time.Hour).Format(tc.JobTimeFormat),
			DeliveryService: "myds",
			CreatedBy:       "want_refetch",
			ID:              42,
			Parameters:      "TTL:24h",
			Keyword:         JobKeywordPurge,
		},
		tc.Job{
			AssetURL:        "refreshasset##REFRESH##",
			StartTime:       time.Now().Add(24 * time.Hour).Format(tc.JobTimeFormat),
			DeliveryService: "myds",
			CreatedBy:       "want_refresh",
			ID:              42,
			Parameters:      "TTL:24h",
			Keyword:         JobKeywordPurge,
		},
	}

	cfg, err := MakeRegexRevalidateDotConfig(server, dses, params, jobs, &RegexRevalidateDotConfigOpts{HdrComment: hdr})
	if err != nil {
		t.Fatal(err)
	}
	txt := cfg.Text

	if !strings.Contains(txt, "assetURL0") {
		t.Errorf("expected 'assetURL0', actual '%v'", txt)
	}
	if strings.Contains(txt, "unrelated") {
		t.Errorf("expected no unrelated param, actual '%v'", txt)
	}
	if strings.Contains(txt, "expired") {
		t.Errorf("expected no expired job, actual '%v'", txt)
	}
	if strings.Contains(txt, "##REFETCH##") || !strings.Contains(txt, "MISS") {
		t.Errorf("##REFETCH## directive not properly handled '%v'", txt)
	}
	if strings.Contains(txt, "##REFRESH##") || !strings.Contains(txt, "STALE") {
		t.Errorf("##REFRESH## directive not properly handled '%v'", txt)
	}
}
