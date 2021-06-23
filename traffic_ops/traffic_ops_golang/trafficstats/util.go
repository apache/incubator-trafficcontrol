package trafficstats

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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

var (
	jsonWithRFCTimestamps = rfc.MimeType{
		Name:       "application/json",
		Parameters: map[string]string{"timestamp": "rfc"},
	}

	jsonWithUnixTimestamps = rfc.MimeType{
		Name:       "application/json",
		Parameters: map[string]string{"timestamp": "unix"},
	}
)

const (
	defaultInterval = "1m"
)

func tsConfigFromRequest(r *http.Request, i *api.Info) (tc.TrafficStatsConfig, int, error) {
	c := tc.TrafficStatsConfig{}
	var e error
	if accept := r.Header.Get("Accept"); accept != "" {

		mimes, err := rfc.MimeTypesFromAccept(accept)
		if err != nil {
			log.Warnf("Failed to negotiate content, Accept line '%s', error: %v", accept, err)
		} else {

			found := false
			for _, m := range mimes {
				if jsonWithRFCTimestamps.Satisfy(m) {
					found = true
					break
				}

				if jsonWithUnixTimestamps.Satisfy(m) {
					found = true
					c.Unix = true
					break
				}
			}

			if !found {
				e = fmt.Errorf("Failed to negotiate content; cannot produce output satisfying %s", accept)
				return c, http.StatusNotAcceptable, e
			}
		}
	}

	if limit, ok := i.Params["limit"]; ok {
		lim, err := strconv.ParseUint(limit, 10, 64)
		if err != nil {
			e = errors.New("Invalid limit!")
			return c, http.StatusBadRequest, e
		}
		c.Limit = &lim
	}

	if offset, ok := i.Params["offset"]; ok {
		off, err := strconv.ParseUint(offset, 10, 64)
		if err != nil {
			e = errors.New("Invalid offset!")
			return c, http.StatusBadRequest, e
		}
		c.Offset = &off
	}

	if orderby, ok := i.Params["orderby"]; ok {
		if c.OrderBy = tc.OrderableFromString(orderby); c.OrderBy == nil {
			e = errors.New("Invalid orderby! Can only be 'time'")
			return c, http.StatusBadRequest, e
		}
	}

	if c.Start, e = parseTime(i.Params["startDate"]); e != nil {
		log.Errorf("Parsing startDate: %v", e)
		e = errors.New("Invalid startDate!")
		return c, http.StatusBadRequest, e
	}

	if c.End, e = parseTime(i.Params["endDate"]); e != nil {
		log.Errorf("Parsing endDate: %v", e)
		e = errors.New("Invalid endDate!")
		return c, http.StatusBadRequest, e
	}

	if interval, ok := i.Params["interval"]; !ok {
		c.Interval = defaultInterval
	} else if !tc.TrafficStatsDurationPattern.MatchString(interval) {
		e = errors.New("interval: must be a valid InfluxQL duration literal (resolution no less than minute)")
		return c, http.StatusBadRequest, e
	} else {
		c.Interval = interval
	}

	if ex, ok := i.Params["exclude"]; ok {
		switch tc.ExcludeFromString(ex) {
		case tc.ExcludeSummary:
			c.ExcludeSummary = true
		case tc.ExcludeSeries:
			c.ExcludeSeries = true
		default:
			e = errors.New("Invalid exclude! Must be 'series' or 'summary'")
			return c, http.StatusBadRequest, e
		}
	}
	return c, http.StatusOK, nil
}

func parseTime(raw string) (time.Time, error) {
	t, e := time.Parse(time.RFC3339Nano, raw)
	if e == nil {
		return t, nil
	}

	if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
		t = time.Unix(0, i)
		return t, nil
	}

	return time.Parse(tc.TimeLayout, raw)
}

func extractFloat64(k string, m map[string]interface{}) (float64, error) {
	tmp, ok := m[k]
	if !ok {
		return 0, fmt.Errorf("response has no value for column %s", k)
	}

	switch t := tmp.(type) {
	case float64:
		return tmp.(float64), nil
	case json.Number:
		ret, err := tmp.(json.Number).Float64()
		if err != nil {
			return 0, fmt.Errorf("Error parsing value for column '%s' as a float64: %v", k, err)
		}
		return ret, nil
	case nil:
		// This is the only field that can be nil - because sometimes there isn't enough data for
		// 5% of it to be below a value. Could probably coalesce this in the query at some point, though.
		if k == "fifthPercentile" {
			return 0, nil
		}
		return 0, fmt.Errorf("column '%s' was null/blank", k)
	default:
		return 0, fmt.Errorf("invalid type for column '%s' - expected 'float64' or 'json.Number', got %T (%v)", k, t, tmp)
	}
}

func extractUInt(k string, m map[string]interface{}) (uint, error) {
	tmp, ok := m[k]
	if !ok {
		return 0, fmt.Errorf("response has no value for column %s", k)
	}
	switch t := tmp.(type) {
	case float64:
		return uint(tmp.(float64)), nil
	case json.Number:
		ret, err := tmp.(json.Number).Int64()
		if err != nil {
			return 0, fmt.Errorf("Error parsing value for column '%s' as an int64: %v", k, err)
		}
		return uint(ret), nil
	default:
		return 0, fmt.Errorf("invalid type for column '%s' - expected unsigned integer, got %T (%v)", k, t, tmp)
	}
}

func buildExtraClauses(conf *tc.TrafficStatsConfig) string {
	extraClauses := strings.Builder{}
	if conf.OrderBy != nil {
		extraClauses.Write([]byte(" ORDER BY "))
		extraClauses.WriteString(string(*conf.OrderBy))
	}

	if conf.Limit != nil {
		extraClauses.WriteString(fmt.Sprintf(" LIMIT %d", *conf.Limit))
	}

	if conf.Offset != nil {
		extraClauses.WriteString(fmt.Sprintf(" OFFSET %d", *conf.Offset))
	}
	return extraClauses.String()
}
