package api

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
	"strconv"
)

// Version represents an API version.
type Version struct {
	// API major version - '3' in '3.1'.
	Major uint64
	// API minor version - '1' in '3.1'.
	Minor uint64
}

// Equal determines if an API Version is exactly equal to some other Version.
func (v Version) Equal(other Version) bool {
	return v.Major == other.Major && v.Minor == other.Minor
}

// LessThan determines if an API Version is a lower version than some other Version.
func (v Version) LessThan(other Version) bool {
	return v.Major < other.Major || (v.Major == other.Major && v.Minor < other.Minor)
}

// GreaterThan determines if an API Version is a higher version than some other Version.
func (v Version) GreaterThan(other Version) bool {
	return v.Major > other.Major || (v.Major == other.Major && v.Minor > other.Minor)
}

// String returns a string representation of the Version.
func (v Version) String() string {
	return strconv.FormatUint(v.Major, 10) + "." + strconv.FormatUint(v.Minor, 10)
}
