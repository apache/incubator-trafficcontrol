package cachegroup

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
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type TOCacheGroup struct {
	api.InfoerImpl `json:"-"`
	tc.CacheGroupNullable
}

func (cg TOCacheGroup) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (cg TOCacheGroup) GetKeys() (map[string]interface{}, bool) {
	if cg.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *cg.ID}, true
}

func (cg *TOCacheGroup) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	cg.ID = &i
}

//Implementation of the Identifier, Validator interface functions
func (cg TOCacheGroup) GetID() (int, bool) {
	if cg.ID == nil {
		return 0, false
	}
	return *cg.ID, true
}

func (cg TOCacheGroup) GetAuditName() string {
	if cg.Name != nil {
		return *cg.Name
	}
	id, _ := cg.GetID()
	return strconv.Itoa(id)
}

func (cg TOCacheGroup) GetType() string {
	return "cachegroup"
}

func (cg *TOCacheGroup) SetID(i int) {
	cg.ID = &i
}

// Is the cachegroup being used?
func isUsed(tx *sqlx.Tx, ID int) (bool, error) {

	var usedByTopology bool
	var usedByServer bool
	var usedByParent bool
	var usedBySecondaryParent bool
	var usedByASN bool

	query := `SELECT
    (SELECT id FROM topology_cachegroup WHERE topology_cachegroup.cachegroup = (SELECT name FROM cachegroup WHERE id = $1) LIMIT 1) IS NOT NULL,
    (SELECT id FROM server WHERE server.cachegroup = $1 LIMIT 1) IS NOT NULL,
    (SELECT id FROM cachegroup WHERE cachegroup.parent_cachegroup_id = $1 LIMIT 1) IS NOT NULL,
    (SELECT id FROM cachegroup WHERE cachegroup.secondary_parent_cachegroup_id = $1 LIMIT 1) IS NOT NULL,
    (SELECT id FROM asn WHERE cachegroup = $1 LIMIT 1) IS NOT NULL;`

	err := tx.QueryRow(query, ID).Scan(&usedByTopology, &usedByServer, &usedByParent, &usedBySecondaryParent, &usedByASN)
	if err != nil {
		log.Errorf("received error: %++v from query execution", err)
		return false, err
	}
	//Only return the immediate error
	if usedByTopology {
		return true, errors.New("cachegroup is in use by one or more topologies")
	}
	if usedByServer {
		return true, errors.New("cachegroup is in use by one or more servers")
	}
	if usedByParent {
		return true, errors.New("cachegroup is in use as a parent cachegroup")
	}
	if usedBySecondaryParent {
		return true, errors.New("cachegroup is in use as a secondary parent cachegroup")
	}
	if usedByASN {
		return true, errors.New("cachegroup is in use in one or more ASNs")
	}

	return false, nil
}

func isValidCacheGroupChar(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	if r == '.' || r == '-' || r == '_' {
		return true
	}
	return false
}

// IsValidCacheGroupName returns true if the name contains only characters valid for a CacheGroup name
func IsValidCacheGroupName(str string) bool {
	i := strings.IndexFunc(str, func(r rune) bool { return !isValidCacheGroupChar(r) })
	return i == -1
}

func IsValidParentCachegroupID(id *int) bool {
	if id == nil || *id > 0 {
		return true
	}
	return false
}

// ValidateTypeInTopology validates cachegroup updates to ensure the type of the cachegroup does not change
// if it is assigned to a topology.
func (cg *TOCacheGroup) ValidateTypeInTopology() error {
	userErr := fmt.Errorf("unable to check whether cachegroup %s is used in any topologies", *cg.Name)

	// language=SQL
	const previousTypeQuery = `
	SELECT t.id
	FROM cachegroup c
	JOIN "type" t ON c."type" = t.id
	WHERE c.id = $1
	`
	var previousTypeID int
	// We only run this validation on PUT, not POST
	if cg.ID == nil {
		return nil
	}
	err := cg.ReqInfo.Tx.QueryRow(previousTypeQuery, *cg.ID).Scan(&previousTypeID)
	// Cachegroup does not exist in the database yet
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		log.Errorf("%s: getting the previous type of cachegroup %s: %s", userErr.Error(), *cg.Name, err.Error())
		return userErr
	}
	if cg.TypeID == nil || *cg.TypeID == previousTypeID {
		return nil
	}

	// language=SQL
	const usedInTopologyQuery = `
	SELECT EXISTS (SELECT
	FROM topology_cachegroup tc
	WHERE tc.cachegroup = $1)
	`
	var usedInTopology bool
	err = cg.ReqInfo.Tx.QueryRow(usedInTopologyQuery, *cg.Name).Scan(&usedInTopology)
	if err != nil {
		log.Errorf("%s: querying topology_cachegroup by cachegroup name: %s", userErr.Error(), err.Error())
		return userErr
	}
	if !usedInTopology {
		return nil
	}

	// language=SQL
	const readableTypesQuery = `
	SELECT t.id, t."name"
	FROM "type" t
	WHERE id = $1 OR id = $2
	`
	rows, err := cg.ReqInfo.Tx.Query(readableTypesQuery, previousTypeID, *cg.TypeID)
	if err != nil {
		log.Errorf("%s: querying type names: %s", userErr.Error(), err.Error())
		return userErr
	}
	typeNameByID := map[int]string{}
	for rows.Next() {
		var typeID int
		var typeName string
		err = rows.Scan(&typeID, &typeName)
		if err != nil {
			log.Errorf("%s: scanning type names: %s", userErr.Error(), err.Error())
			return userErr
		}
		typeNameByID[typeID] = typeName
	}
	log.Close(rows, "error closing rows for type names")

	return fmt.Errorf("cannot change type of cachegroup %s from %s to %s because it is in use by a topology", *cg.Name, typeNameByID[previousTypeID], typeNameByID[*cg.TypeID])
}

// Validate fulfills the api.Validator interface
func (cg TOCacheGroup) Validate() error {
	if _, err := tc.ValidateTypeID(cg.ReqInfo.Tx.Tx, cg.TypeID, "cachegroup"); err != nil {
		return err
	}

	if cg.Fallbacks != nil && len(*cg.Fallbacks) > 0 {
		isValid, err := cg.isAllowedToFallback(*cg.TypeID)
		if err != nil {
			return err
		}
		if !isValid {
			return errors.New("the cache group " + *cg.Name + " is not allowed to have fallbacks.  It must be of type EDGE_LOC.")
		}

		for _, fallback := range *cg.Fallbacks {
			isValid, err = cg.isValidCacheGroupFallback(fallback)
			if err != nil {
				return err
			}
			if !isValid {
				return errors.New("the cache group " + fallback + " is not valid as a fallback.  It must exist as a cache group and be of type EDGE_LOC.")
			}
		}
	}

	validName := validation.NewStringRule(IsValidCacheGroupName, "invalid characters found - Use alphanumeric . or - or _ .")
	validShortName := validation.NewStringRule(IsValidCacheGroupName, "invalid characters found - Use alphanumeric . or - or _ .")
	latitudeErr := "Must be a floating point number within the range +-90"
	longitudeErr := "Must be a floating point number within the range +-180"
	errs := validation.Errors{
		"name":                        validation.Validate(cg.Name, validation.Required, validName),
		"shortName":                   validation.Validate(cg.ShortName, validation.Required, validShortName),
		"latitude":                    validation.Validate(cg.Latitude, validation.Min(-90.0).Error(latitudeErr), validation.Max(90.0).Error(latitudeErr)),
		"longitude":                   validation.Validate(cg.Longitude, validation.Min(-180.0).Error(longitudeErr), validation.Max(180.0).Error(longitudeErr)),
		"parentCacheGroupID":          validation.Validate(cg.ParentCachegroupID, validation.Min(1)),
		"secondaryParentCachegroupID": validation.Validate(cg.SecondaryParentCachegroupID, validation.Min(1)),
		"localizationMethods":         validation.Validate(cg.LocalizationMethods, validation.By(tovalidate.IsPtrToSliceOfUniqueStringersICase("CZ", "DEEP_CZ", "GEO"))),
		"type":                        cg.ValidateTypeInTopology(),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

//The TOCacheGroup implementation of the Creator interface
//The insert sql returns the id and lastUpdated values of the newly inserted cachegroup and have
//to be added to the struct
func (cg *TOCacheGroup) Create() (error, error, int) {

	if cg.LocalizationMethods == nil {
		cg.LocalizationMethods = &[]tc.LocalizationMethod{}
	}

	if cg.Fallbacks == nil {
		cg.Fallbacks = &[]string{}
	}

	if cg.FallbackToClosest == nil {
		fbc := true
		cg.FallbackToClosest = &fbc
	}

	err := cg.ReqInfo.Tx.Tx.QueryRow(
		InsertQuery(),
		cg.Name,
		cg.ShortName,
		cg.TypeID,
		cg.ParentCachegroupID,
		cg.SecondaryParentCachegroupID,
		cg.FallbackToClosest,
	).Scan(
		&cg.ID,
		&cg.Type,
		&cg.ParentName,
		&cg.SecondaryParentName,
	)
	if err != nil {
		return api.ParseDBError(err)
	}

	coordinateID, err := cg.createCoordinate()
	if err != nil {
		return nil, errors.New("cachegroup create: creating coord:" + err.Error()), http.StatusInternalServerError
	}

	err = cg.ReqInfo.Tx.Tx.QueryRow(
		`UPDATE cachegroup SET coordinate=$1 WHERE id=$2 RETURNING last_updated`,
		coordinateID,
		cg.ID,
	).Scan(&cg.LastUpdated)
	if err != nil {
		return nil, fmt.Errorf("followup update during cachegroup create: %v", err), http.StatusInternalServerError
	}

	if err = cg.createLocalizationMethods(); err != nil {
		return nil, errors.New("creating cachegroup: creating localization methods: " + err.Error()), http.StatusInternalServerError
	}

	if err = cg.createCacheGroupFallbacks(); err != nil {
		return nil, errors.New("creating cachegroup: creating cache group fallbacks: " + err.Error()), http.StatusInternalServerError
	}

	return nil, nil, http.StatusOK
}

func (cg *TOCacheGroup) createLocalizationMethods() error {
	q := `DELETE FROM cachegroup_localization_method WHERE cachegroup = $1`
	if _, err := cg.ReqInfo.Tx.Tx.Exec(q, *cg.ID); err != nil {
		return fmt.Errorf("unable to delete cachegroup_localization_methods for cachegroup %d: %s", *cg.ID, err.Error())
	}
	if cg.LocalizationMethods != nil {
		q = `INSERT INTO cachegroup_localization_method (cachegroup, method) VALUES ($1, $2)`
		for _, method := range *cg.LocalizationMethods {
			if _, err := cg.ReqInfo.Tx.Tx.Exec(q, *cg.ID, method.String()); err != nil {
				return fmt.Errorf("unable to insert cachegroup_localization_methods for cachegroup %d: %s", *cg.ID, err.Error())
			}
		}
	}
	return nil
}

func (cg *TOCacheGroup) createCacheGroupFallbacks() error {
	deleteCgfQuery := `DELETE FROM cachegroup_fallbacks WHERE primary_cg = $1`
	if _, err := cg.ReqInfo.Tx.Tx.Exec(deleteCgfQuery, *cg.ID); err != nil {
		return fmt.Errorf("unable to delete cachegroup_fallbacks for cachegroup %d: %s", *cg.ID, err.Error())
	}
	if cg.Fallbacks == nil {
		return nil
	}
	insertCgfQuery := `INSERT INTO cachegroup_fallbacks (primary_cg, backup_cg, set_order) VALUES ($1, (SELECT cachegroup.id FROM cachegroup WHERE cachegroup.name = $2), $3)`
	for orderIndex, fallback := range *cg.Fallbacks {
		if _, err := cg.ReqInfo.Tx.Tx.Exec(insertCgfQuery, *cg.ID, fallback, orderIndex); err != nil {
			return fmt.Errorf("unable to insert cachegroup_fallbacks for cachegroup %d: %s", *cg.ID, err.Error())
		}
	}
	return nil
}

func (cg *TOCacheGroup) isValidCacheGroupFallback(fallbackName string) (bool, error) {
	var isValid bool
	query := `SELECT(
SELECT cachegroup.id
FROM cachegroup
JOIN type on type.id = cachegroup.type
WHERE cachegroup.name = $1
AND (type.name = 'EDGE_LOC')
) IS NOT NULL;`

	err := cg.ReqInfo.Tx.Tx.QueryRow(query, fallbackName).Scan(&isValid)
	if err != nil {
		log.Errorf("received error: %++v from cachegroup fallback validation query execution", err)
		return false, err
	}
	return isValid, nil
}

func (cg *TOCacheGroup) isAllowedToFallback(cacheGroupType int) (bool, error) {
	var isValid bool
	query := `SELECT(
SELECT type.name
FROM type
WHERE type.id = $1
AND (type.name = 'EDGE_LOC')
) IS NOT NULL;`

	err := cg.ReqInfo.Tx.Tx.QueryRow(query, cacheGroupType).Scan(&isValid)
	if err != nil {
		log.Errorf("received error: %++v from cachegroup fallback validation query execution", err)
		return false, err
	}
	return isValid, nil
}

func (cg *TOCacheGroup) createCoordinate() (*int, error) {
	var coordinateID *int
	if cg.Latitude != nil && cg.Longitude != nil {
		q := `INSERT INTO coordinate (name, latitude, longitude) VALUES ($1, $2, $3) RETURNING id`
		if err := cg.ReqInfo.Tx.Tx.QueryRow(q, tc.CachegroupCoordinateNamePrefix+*cg.Name, *cg.Latitude, *cg.Longitude).Scan(&coordinateID); err != nil {
			return nil, err
		}
	}
	return coordinateID, nil
}

func (cg *TOCacheGroup) updateCoordinate() error {
	if cg.Latitude != nil && cg.Longitude != nil {
		q := `UPDATE coordinate SET name = $1, latitude = $2, longitude = $3 WHERE id = (SELECT coordinate FROM cachegroup WHERE id = $4)`
		result, err := cg.ReqInfo.Tx.Tx.Exec(q, tc.CachegroupCoordinateNamePrefix+*cg.Name, *cg.Latitude, *cg.Longitude, *cg.ID)
		if err != nil {
			return fmt.Errorf("updating coordinate for cachegroup '%s': %s", *cg.Name, err.Error())
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("updating coordinate for cachegroup '%s', getting rows affected: %s", *cg.Name, err.Error())
		}
		if rowsAffected == 0 {
			return fmt.Errorf("updating coordinate for cachegroup '%s', zero rows affected", *cg.Name)
		}
	}
	return nil
}

func (cg *TOCacheGroup) deleteCoordinate(coordinateID int) error {
	q := `UPDATE cachegroup SET coordinate = NULL WHERE id = $1`
	result, err := cg.ReqInfo.Tx.Tx.Exec(q, *cg.ID)
	if err != nil {
		return fmt.Errorf("updating cachegroup %d coordinate to null: %s", *cg.ID, err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updating cachegroup %d coordinate to null, getting rows affected: %s", *cg.ID, err.Error())
	}
	if rowsAffected == 0 {
		return fmt.Errorf("updating cachegroup %d coordinate to null, zero rows affected", *cg.ID)
	}

	q = `DELETE FROM coordinate WHERE id = $1`
	result, err = cg.ReqInfo.Tx.Tx.Exec(q, coordinateID)
	if err != nil {
		return fmt.Errorf("delete coordinate %d for cachegroup %d: %s", coordinateID, *cg.ID, err.Error())
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete coordinate %d for cachegroup %d, getting rows affected: %s", coordinateID, *cg.ID, err.Error())
	}
	if rowsAffected == 0 {
		return fmt.Errorf("delete coordinate %d for cachegroup %d, zero rows affected", coordinateID, *cg.ID)
	}
	return nil
}

func GetCacheGroupsByName(names []string, Tx *sqlx.Tx) (map[string]tc.CacheGroupNullable, error, error, int) {
	query := SelectQuery() + multipleCacheGroupsWhere()
	namesPqArray := pq.Array(names)
	rows, err := Tx.Query(query, namesPqArray)
	if err != nil {
		userErr, sysErr, errCode := api.ParseDBError(err)
		return nil, userErr, sysErr, errCode
	}
	defer log.Close(rows, "unable to close DB connection")
	cacheGroupMap := map[string]tc.CacheGroupNullable{}
	for rows.Next() {
		var s tc.CacheGroupNullable
		lms := make([]tc.LocalizationMethod, 0)
		cgfs := make([]string, 0)
		if err = rows.Scan(
			&s.ID,
			&s.Name,
			&s.ShortName,
			&s.Latitude,
			&s.Longitude,
			pq.Array(&lms),
			&s.ParentCachegroupID,
			&s.ParentName,
			&s.SecondaryParentCachegroupID,
			&s.SecondaryParentName,
			&s.Type,
			&s.TypeID,
			&s.LastUpdated,
			pq.Array(&cgfs),
			&s.FallbackToClosest,
		); err != nil {
			return nil, nil, errors.New("cachegroup read: scanning: " + err.Error()), http.StatusInternalServerError
		}
		s.LocalizationMethods = &lms
		s.Fallbacks = &cgfs
		cacheGroupMap[*s.Name] = s
	}
	return cacheGroupMap, nil, nil, http.StatusOK
}

func (cg *TOCacheGroup) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	cacheGroups := []interface{}{}
	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		"id":        {Column: "cachegroup.id", Checker: api.IsInt},
		"name":      {Column: "cachegroup.name"},
		"shortName": {Column: "cachegroup.short_name"},
		"type":      {Column: "cachegroup.type"},
		"topology":  {Column: "topology_cachegroup.topology"},
	}
	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(cg.ReqInfo.Params, queryParamsToQueryCols)
	if len(errs) > 0 {
		return nil, util.JoinErrs(errs), nil, http.StatusBadRequest, nil
	}

	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(cg.Info().Tx, h, queryValues, selectMaxLastUpdatedQuery(where))
		if !runSecond {
			log.Debugln("IMS HIT")
			return cacheGroups, nil, nil, http.StatusNotModified, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}
	baseSelect := SelectQuery()
	if _, ok := cg.ReqInfo.Params["topology"]; ok {
		baseSelect += `
LEFT JOIN topology_cachegroup ON cachegroup.name = topology_cachegroup.cachegroup
`
	}
	// If the type cannot be converted to an int, return 400
	if cgType, ok := cg.ReqInfo.Params["type"]; ok {
		_, err := strconv.Atoi(cgType)
		if err != nil {
			return nil, errors.New("cachegroup read: converting cachegroup type to integer " + err.Error()), nil, http.StatusBadRequest, nil
		}
	}
	query := baseSelect + where + orderBy + pagination
	rows, err := cg.ReqInfo.Tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, nil, errors.New("cachegroup read: querying: " + err.Error()), http.StatusInternalServerError, nil
	}
	defer rows.Close()

	for rows.Next() {
		var s TOCacheGroup
		lms := make([]tc.LocalizationMethod, 0)
		cgfs := make([]string, 0)
		if err = rows.Scan(
			&s.ID,
			&s.Name,
			&s.ShortName,
			&s.Latitude,
			&s.Longitude,
			pq.Array(&lms),
			&s.ParentCachegroupID,
			&s.ParentName,
			&s.SecondaryParentCachegroupID,
			&s.SecondaryParentName,
			&s.Type,
			&s.TypeID,
			&s.LastUpdated,
			pq.Array(&cgfs),
			&s.FallbackToClosest,
		); err != nil {
			return nil, nil, errors.New("cachegroup read: scanning: " + err.Error()), http.StatusInternalServerError, nil
		}
		s.LocalizationMethods = &lms
		s.Fallbacks = &cgfs
		cacheGroups = append(cacheGroups, s)
	}
	return cacheGroups, nil, nil, http.StatusOK, &maxTime
}

func selectMaxLastUpdatedQuery(where string) string {
	return `SELECT max(t) from (
		SELECT max(cachegroup.last_updated) as t FROM cachegroup
LEFT JOIN coordinate ON coordinate.id = cachegroup.coordinate
INNER JOIN type ON cachegroup.type = type.id
LEFT JOIN cachegroup AS cgp ON cachegroup.parent_cachegroup_id = cgp.id
LEFT JOIN cachegroup AS cgs ON cachegroup.secondary_parent_cachegroup_id = cgs.id ` + where +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='cachegroup') as res`
}

//The TOCacheGroup implementation of the Updater interface
func (cg *TOCacheGroup) Update(h http.Header) (error, error, int) {

	if cg.LocalizationMethods == nil {
		cg.LocalizationMethods = &[]tc.LocalizationMethod{}
	}

	if cg.Fallbacks == nil {
		cg.Fallbacks = &[]string{}
	}

	if cg.FallbackToClosest == nil {
		fbc := true
		cg.FallbackToClosest = &fbc
	}

	userErr, sysErr, errCode := api.CheckIfUnModified(h, cg.ReqInfo.Tx, *cg.ID, "cachegroup")
	if userErr != nil || sysErr != nil {
		return userErr, sysErr, errCode
	}

	coordinateID, userErr, sysErr, errCode := cg.handleCoordinateUpdate()
	if userErr != nil || sysErr != nil {
		return userErr, sysErr, errCode
	}

	err := cg.ReqInfo.Tx.Tx.QueryRow(
		UpdateQuery(),
		cg.Name,
		cg.ShortName,
		coordinateID,
		cg.ParentCachegroupID,
		cg.SecondaryParentCachegroupID,
		cg.TypeID,
		cg.FallbackToClosest,
		cg.ID,
	).Scan(
		&cg.Type,
		&cg.ParentName,
		&cg.SecondaryParentName,
		&cg.LastUpdated,
	)
	if err != nil {
		return api.ParseDBError(err)
	}

	if err = cg.createLocalizationMethods(); err != nil {
		return nil, errors.New("cachegroup update: creating localization methods: " + err.Error()), http.StatusInternalServerError
	}

	if err = cg.createCacheGroupFallbacks(); err != nil {
		return nil, errors.New("cachegroup update: creating cache group fallbacks: " + err.Error()), http.StatusInternalServerError
	}

	return nil, nil, http.StatusOK
}

func (cg *TOCacheGroup) handleCoordinateUpdate() (*int, error, error, int) {

	coordinateID, err := cg.getCoordinateID()

	// This is not a logic error. Because the coordinate id is recieved from the
	// cachegroup table, not being able to find the coordinate is equivalent to
	// not being able to find the cachegroup.
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no cachegroup with id %d found", *cg.ID), nil, http.StatusNotFound
	}
	if err != nil {
		return nil, nil, tc.DBError, http.StatusInternalServerError
	}

	// If partial coordinate information is given or the coordinate information is wholly
	// nullified, it is invalid and we zero the reference to the coordinate in the database.
	// For now I am also nullifying both the longitude and latitude if one of them is nil
	// because a non-nil returned value would have no meaning.
	//
	// TODO: Find references that talk about longitude and latidude as required fields
	//	- Making longitude and latitude required would prevent this odd case
	//	- Longitude and latitude probably aren't required for versioning reasons
	//	- We've recently had a discussion on versioning that may be related
	// TODO: In the meantime should an error be returned for partial coordinate information?
	//	- Probably not
	//
	if cg.Latitude == nil || cg.Longitude == nil {
		if err = cg.deleteCoordinate(*coordinateID); err != nil {
			return nil, nil, tc.DBError, http.StatusInternalServerError
		}
		cg.Latitude = nil
		cg.Longitude = nil
		return nil, nil, nil, http.StatusOK
	}

	if err = cg.updateCoordinate(); err != nil {
		return nil, nil, tc.DBError, http.StatusInternalServerError
	}
	return coordinateID, nil, nil, http.StatusOK
}

func (cg *TOCacheGroup) getCoordinateID() (*int, error) {
	q := `SELECT coordinate FROM cachegroup WHERE id = $1`
	var coordinateID *int
	if err := cg.ReqInfo.Tx.Tx.QueryRow(q, *cg.ID).Scan(&coordinateID); err != nil {
		return nil, err
	}
	return coordinateID, nil
}

//The CacheGroup implementation of the Deleter interface
//all implementations of Deleter should use transactions and return the proper errorType
func (cg *TOCacheGroup) Delete() (error, error, int) {
	inUse, err := isUsed(cg.ReqInfo.Tx, *cg.ID)
	if inUse {
		return err, nil, http.StatusBadRequest
	}
	if err != nil {
		return nil, errors.New("cachegroup delete: checking use: " + err.Error()), http.StatusInternalServerError
	}

	coordinateID, err := cg.getCoordinateID()
	if err == sql.ErrNoRows {
		return errors.New("no cachegroup with that id found"), nil, http.StatusNotFound
	}
	if err != nil {
		return nil, errors.New("cachegroup delete: getting coord: " + err.Error()), http.StatusInternalServerError
	}

	if err = cg.deleteCoordinate(*coordinateID); err != nil {
		return nil, errors.New("cachegroup delete: deleting coord: " + err.Error()), http.StatusInternalServerError
	}

	result, err := cg.ReqInfo.Tx.Exec(DeleteQuery(), *cg.ID)
	if err != nil {
		return nil, errors.New("cachegroup delete: " + err.Error()), http.StatusInternalServerError
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("getting rows affected: %v", err), http.StatusInternalServerError
	}

	// In the zero case, either err != nil occurs (from the Exec) or we got sql.ErrNoRows above
	if rowsAffected != 1 {
		return nil, errors.New("cachegroup delete affected multiple rows"), http.StatusInternalServerError
	}

	return nil, nil, http.StatusOK
}

func InsertQuery() string {
	return `INSERT INTO cachegroup (
name,
short_name,
type,
parent_cachegroup_id,
secondary_parent_cachegroup_id,
fallback_to_closest
) VALUES($1,$2,$3,$4,$5,$6)
RETURNING
id,
(SELECT name FROM type WHERE cachegroup.type = type.id),
(SELECT name FROM cachegroup parent
	WHERE cachegroup.parent_cachegroup_id = parent.id),
(SELECT name FROM cachegroup secondary_parent
	WHERE cachegroup.secondary_parent_cachegroup_id = secondary_parent.id)`
}

func SelectQuery() string {
	// the 'type_name' and 'type_id' aliases on the 'type.name'
	// and cachegroup.type' fields are needed
	// to disambiguate the struct scan, see also the
	// tc.CacheGroupNullable struct 'db' metadata
	return `SELECT
cachegroup.id,
cachegroup.name,
cachegroup.short_name,
coordinate.latitude,
coordinate.longitude,
(SELECT COALESCE(array_agg(CAST(method as text)), '{}') AS localization_methods FROM cachegroup_localization_method clm WHERE clm.cachegroup = cachegroup.id),
cachegroup.parent_cachegroup_id,
cgp.name AS parent_cachegroup_name,
cachegroup.secondary_parent_cachegroup_id,
cgs.name AS secondary_parent_cachegroup_name,
type.name AS type_name,
cachegroup.type AS type_id,
cachegroup.last_updated,
(SELECT COALESCE(array_agg(CAST(cg2.name as text) ORDER BY cgf.set_order ASC), '{}') AS fallbacks FROM cachegroup cg2 INNER JOIN cachegroup_fallbacks cgf ON cgf.backup_cg = cg2.id WHERE cgf.primary_cg = cachegroup.id),
cachegroup.fallback_to_closest
FROM cachegroup
LEFT JOIN coordinate ON coordinate.id = cachegroup.coordinate
INNER JOIN type ON cachegroup.type = type.id
LEFT JOIN cachegroup AS cgp ON cachegroup.parent_cachegroup_id = cgp.id
LEFT JOIN cachegroup AS cgs ON cachegroup.secondary_parent_cachegroup_id = cgs.id
`
}

func multipleCacheGroupsWhere() string {
	return `
WHERE
cachegroup.name = ANY ($1)
`
}

func UpdateQuery() string {
	// to disambiguate struct scans, the named
	// parameter 'type_id' is an alias to cachegroup.type
	//see also the tc.CacheGroupNullable struct 'db' metadata
	return `UPDATE
cachegroup SET
name=$1,
short_name=$2,
coordinate=$3,
parent_cachegroup_id=$4,
secondary_parent_cachegroup_id=$5,
type=$6,
fallback_to_closest=$7
WHERE id=$8
RETURNING
(SELECT name FROM type WHERE cachegroup.type = type.id),
(SELECT name FROM cachegroup parent
	WHERE cachegroup.parent_cachegroup_id = parent.id),
(SELECT name FROM cachegroup secondary_parent
	WHERE cachegroup.secondary_parent_cachegroup_id = secondary_parent.id),
last_updated`
}

func DeleteQuery() string {
	return `DELETE FROM cachegroup WHERE id=$1`
}
