package steering

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
	"database/sql"
	"errors"
	"net/http"
	"sort"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"

	"github.com/lib/pq"
)

func Get(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	steering, err := findSteering(inf.Tx.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("steering.Get finding: "+err.Error()))
		return
	}
	api.WriteResp(w, r, steering)
}

func findSteering(tx *sql.Tx) ([]tc.Steering, error) {
	steeringData, err := getSteeringData(tx)
	if err != nil {
		return nil, err
	}
	targetIDs := steeringDataTargetIDs(steeringData)
	steeringFilters, err := getSteeringFilters(tx, targetIDs)
	if err != nil {
		return nil, err
	}
	primaryOriginCoords, err := getPrimaryOriginCoords(tx, targetIDs)
	if err != nil {
		return nil, err
	}

	steerings := map[tc.DeliveryServiceName]tc.Steering{}

	for _, data := range steeringData {
		if _, ok := steerings[data.DeliveryService]; !ok {
			steerings[data.DeliveryService] = tc.Steering{
				DeliveryService: data.DeliveryService,
				ClientSteering:  data.DSType == tc.DSTypeClientSteering,
				Filters:         []tc.SteeringFilter{},         // Initialize, so JSON produces `[]` not `null` if there are no filters.
				Targets:         []tc.SteeringSteeringTarget{}, // Initialize, so JSON produces `[]` not `null` if there are no targets.
			}
		}
		steering := steerings[data.DeliveryService]

		if filters, ok := steeringFilters[data.TargetID]; ok {
			steering.Filters = append(steering.Filters, filters...)
		}

		target := tc.SteeringSteeringTarget{DeliveryService: data.TargetName}
		switch data.Type {
		case tc.SteeringTypeOrder:
			target.Order = int32(data.Value)
		case tc.SteeringTypeWeight:
			target.Weight = int32(data.Value)
		case tc.SteeringTypeGeoOrder:
			target.GeoOrder = util.IntPtr(data.Value)
			target.Latitude = util.FloatPtr(primaryOriginCoords[data.TargetID].Lat)
			target.Longitude = util.FloatPtr(primaryOriginCoords[data.TargetID].Lon)
		case tc.SteeringTypeGeoWeight:
			target.Weight = int32(data.Value)
			target.GeoOrder = util.IntPtr(0)
			target.Latitude = util.FloatPtr(primaryOriginCoords[data.TargetID].Lat)
			target.Longitude = util.FloatPtr(primaryOriginCoords[data.TargetID].Lon)
		}
		steering.Targets = append(steering.Targets, target)
		steerings[data.DeliveryService] = steering
	}

	arr := []tc.Steering{}
	for _, steering := range steerings {
		arr = append(arr, steering)
	}

	sort.Slice(arr, func(i, j int) bool {
		return arr[i].DeliveryService < arr[j].DeliveryService
	})

	return arr, nil
}

type SteeringData struct {
	DeliveryService tc.DeliveryServiceName
	SteeringID      int
	TargetName      tc.DeliveryServiceName
	TargetID        int
	Value           int
	Type            tc.SteeringType
	DSType          tc.DSType
}

func steeringDataTargetIDs(data []SteeringData) []int {
	ids := []int{}
	for _, d := range data {
		ids = append(ids, d.TargetID)
	}
	return ids
}

func getSteeringData(tx *sql.Tx) ([]SteeringData, error) {
	qry := `
SELECT
  ds.xml_id as steering_xml_id,
  ds.id as steering_id,
  t.xml_id as target_xml_id,
  t.id as target_id,
  st.value,
  tp.name as steering_type,
  dt.name as ds_type
FROM
  steering_target st
  JOIN deliveryservice ds on ds.id = st.deliveryservice
  JOIN deliveryservice t on t.id = st.target
  JOIN type tp on tp.id = st.type
  JOIN type dt on dt.id = ds.type
ORDER BY
  steering_xml_id,
  target_xml_id
`
	rows, err := tx.Query(qry)
	if err != nil {
		return nil, errors.New("querying steering: " + err.Error())
	}
	defer rows.Close()
	data := []SteeringData{}
	for rows.Next() {
		sd := SteeringData{}
		if err := rows.Scan(&sd.DeliveryService, &sd.SteeringID, &sd.TargetName, &sd.TargetID, &sd.Value, &sd.Type, &sd.DSType); err != nil {
			return nil, errors.New("get steering data scanning: " + err.Error())
		}
		data = append(data, sd)
	}
	return data, nil
}

// getSteeringFilters takes a slice of ds ids, and returns a map of delivery service ids to patterns and delivery service names.
func getSteeringFilters(tx *sql.Tx, dsIDs []int) (map[int][]tc.SteeringFilter, error) {
	qry := `
SELECT
  ds.id,
  ds.xml_id,
  r.pattern
FROM
  deliveryservice ds
  JOIN deliveryservice_regex dsr ON dsr.deliveryservice = ds.id
  JOIN regex r ON dsr.regex = r.id
  JOIN type t ON r.type = t.id
WHERE
  ds.id = ANY($1)
  AND t.name = $2
ORDER BY
  r.pattern,
  ds.type,
  dsr.set_number
`
	rows, err := tx.Query(qry, pq.Array(dsIDs), tc.DSMatchTypeSteeringRegex)
	if err != nil {
		return nil, errors.New("querying steering regexes: " + err.Error())
	}
	defer rows.Close()
	filters := map[int][]tc.SteeringFilter{}
	for rows.Next() {
		dsID := 0
		f := tc.SteeringFilter{}
		if err := rows.Scan(&dsID, &f.DeliveryService, &f.Pattern); err != nil {
			return nil, errors.New("scanning steering filters: " + err.Error())
		}
		filters[dsID] = append(filters[dsID], f)
	}
	return filters, nil
}

type Coord struct {
	Lat float64
	Lon float64
}

// getPrimaryOriginCoords takes a slice of ds ids, and returns a map of delivery service ids to their primary origin coordinates.
func getPrimaryOriginCoords(tx *sql.Tx, dsIDs []int) (map[int]Coord, error) {
	qry := `
SELECT
  o.deliveryservice,
  c.latitude,
  c.longitude
FROM
  origin o
  JOIN coordinate c ON c.id = o.coordinate
WHERE
  o.deliveryservice = ANY($1)
  AND o.is_primary
`
	rows, err := tx.Query(qry, pq.Array(dsIDs))
	if err != nil {
		return nil, errors.New("querying steering primary origin coords: " + err.Error())
	}
	defer rows.Close()
	coords := map[int]Coord{}
	for rows.Next() {
		dsID := 0
		c := Coord{}
		if err := rows.Scan(&dsID, &c.Lat, &c.Lon); err != nil {
			return nil, errors.New("scanning steering primary origin coords: " + err.Error())
		}
		coords[dsID] = c
	}
	return coords, nil
}
