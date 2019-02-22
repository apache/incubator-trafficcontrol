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

package config

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

const MilitaryTimeFmt = "15:04"

func Validate24HrTimeRange(rng string) error {
	rangeFormat := regexp.MustCompile(`(\S+)-(\S+)`)
	match := rangeFormat.FindStringSubmatch(rng)
	if match == nil {
		return fmt.Errorf("string %v is not a range", rng)
	}

	t1, err := time.Parse(MilitaryTimeFmt, match[1])
	if err != nil {
		return fmt.Errorf("time range must be a 24Hr format")
	}

	t2, err := time.Parse(MilitaryTimeFmt, match[2])
	if err != nil {
		return fmt.Errorf("time range must be a 24Hr format")
	}

	if t1.String() > t2.String() {
		return fmt.Errorf("first time should be smaller than the second")
	}

	return nil
}

func ValidateDHMSTimeFormat(time string) error {

	if time == "" {
		return fmt.Errorf("time string cannot be empty")
	}

	dhms := regexp.MustCompile(`(\d+)([dhms])(\S*)`)
	match := dhms.FindStringSubmatch(time)

	if match == nil {
		return fmt.Errorf("invalid time format")
	}

	var count = map[string]int{
		"d": 0,
		"h": 0,
		"m": 0,
		"s": 0,
	}
	for match != nil {
		if _, err := strconv.Atoi(match[1]); err != nil {
			return err
		}
		if count[match[2]]++; count[match[2]] == 2 {
			return fmt.Errorf("%s unit specified multiple times", match[2])
		}
		match = dhms.FindStringSubmatch(match[3])
	}

	return nil
}
