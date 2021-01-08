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

package v3

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/apache/trafficcontrol/lib/go-log"
)

// LoadNegativeFixtures unmarshals the JSON file provided in the negativeFixtures
// option (tc-negative-fixtures.js by default) into negativeTestData.
func LoadNegativeFixtures(negativeFixturesPath string) {

	f, err := ioutil.ReadFile(negativeFixturesPath)
	if err != nil {
		log.Errorf("Cannot unmarshal negative fixtures json %s", err)
		os.Exit(1)
	}
	err = json.Unmarshal(f, &negativeTestData)
	if err != nil {
		log.Errorf("Cannot unmarshal negative fixtures json %v", err)
		os.Exit(1)
	}
}
