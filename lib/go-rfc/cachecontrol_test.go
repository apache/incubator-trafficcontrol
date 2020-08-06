package rfc

import (
	"testing"
	"time"
)

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

func TestETag(t *testing.T) {
	layout := "2006-01-02 15:04:05.000000-07"
	updatedAt, err := time.Parse(layout, "2020-08-06 12:11:22.278418-06")
	if err != nil {
		t.Errorf("Expected no error parsing the time, but got %v", err.Error())
	}
	if updatedAt.String() != "2020-08-06 12:11:22.278418 -0600 MDT" {
		t.Errorf("Expected time %v, actual %v", "2020-08-06 12:11:22.278418 -0600 MDT", updatedAt.String())
	}
	etag := ETag(updatedAt)
	if etag != `"v1-c4q474ughgls"` {
		t.Errorf("Expected Etag to be %v, actual %v", `"v1-c4q474ughgls"`, etag)
	}
	ans, err := ParseETag(etag)
	if err != nil {
		t.Errorf("Expected no error parsing the time, but got %v", err.Error())
	}
	if ans.String() != "2020-08-06 12:11:22.278418 -0600 MDT" {
		t.Errorf("Expected time %v, actual %v", "2020-08-06 12:11:22.278418 -0600 MDT", ans.String())
	}
}