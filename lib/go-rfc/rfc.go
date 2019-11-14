package rfc

/*
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"time"
)

// ParseHTTPDate parses the given RFC7231§7.1.1 HTTP-date
func ParseHTTPDate(d string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC1123, d); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC850, d); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.ANSIC, d); err == nil {
		return t, true
	}
	return time.Time{}, false
}
