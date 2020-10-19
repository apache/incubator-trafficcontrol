package topology

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
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/cachegroup"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// TOTopology is a type alias on which we can define functions.
type TOTopology struct {
	api.APIInfoImpl `json:"-"`
	Alerts          tc.Alerts `json:"-"`
	tc.Topology
}

// GetAlerts implements the AlertsResponse interface.
func (topology *TOTopology) GetAlerts() tc.Alerts {
	return topology.Alerts
}

// DeleteQueryBase holds a delete query with no WHERE clause and is a
// requirement of the api.GenericOptionsDeleter interface.
func (topology *TOTopology) DeleteQueryBase() string {
	return deleteQueryBase()
}

// ParamColumns maps query parameters to their respective database columns.
func (topology *TOTopology) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"name":        {Column: "t.name"},
		"description": {Column: "t.description"},
		"lastUpdated": {Column: "t.last_updated"},
	}
}

// GenericOptionsDeleter is required by the api.GenericOptionsDeleter interface
// and is called by api.GenericOptionsDelete().
func (topology *TOTopology) DeleteKeyOptions() map[string]dbhelpers.WhereColumnInfo {
	return topology.ParamColumns()
}

func (topology *TOTopology) SetLastUpdated(time tc.TimeNoMod) { topology.LastUpdated = &time }

// GetKeyFieldsInfo is a requirement of the api.Updater interface.
func (topology TOTopology) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"name", api.GetStringKey}}
}

// GetType returns the human-readable type of TOTopology as a string.
func (topology *TOTopology) GetType() string {
	return "topology"
}

// Validate is a requirement of the api.Validator interface.
func (topology *TOTopology) Validate() error {
	nameRule := validation.NewStringRule(tovalidate.IsAlphanumericUnderscoreDash, "must consist of only alphanumeric, dash, or underscore characters.")
	rules := validation.Errors{}
	rules["name"] = validation.Validate(topology.Name, validation.Required, nameRule)

	nodeCount := len(topology.Nodes)
	if nodeCount < 1 {
		rules["length"] = fmt.Errorf("must provide 1 or more node, %v found", nodeCount)
	}
	var (
		cacheGroups      = make([]tc.CacheGroupNullable, nodeCount)
		cacheGroupsExist = true
		err              error
		cacheGroupMap    map[string]tc.CacheGroupNullable
		exists           bool
	)
	_ = err
	cacheGroupNames := make([]string, len(topology.Nodes))
	for index, node := range topology.Nodes {
		rules[fmt.Sprintf("node %v parents size", index)] = validation.Validate(node.Parents, validation.Length(0, 2))
		rules[fmt.Sprintf("node %v duplicate parents", index)] = checkForDuplicateParents(topology.Nodes, index)
		rules[fmt.Sprintf("node %v self parent", index)] = checkForSelfParents(topology.Nodes, index)
		cacheGroupNames[index] = node.Cachegroup
	}

	cacheGroupMap, errs := cachegroup.GetCacheGroupsByName(cacheGroupNames, topology.APIInfoImpl.ReqInfo.Tx)
	if errs.Occurred() {
		// TODO: looks like system errors will go undetected here
		var err error
		message := "could not get cachegroups"
		if errs.UserError != nil {
			err = fmt.Errorf("%s: %s", message, errs.UserError.Error())
		}
		return err
	}
	cacheGroups = make([]tc.CacheGroupNullable, len(topology.Nodes))
	for index, node := range topology.Nodes {
		if cacheGroups[index], exists = cacheGroupMap[node.Cachegroup]; !exists {
			rules[fmt.Sprintf("cachegroup %s not found", node.Cachegroup)] = fmt.Errorf("node %d references nonexistent cachegroup %s", index, node.Cachegroup)
			cacheGroupsExist = false
		}
	}
	rules["duplicate cachegroup name"] = checkUniqueCacheGroupNames(topology.Nodes)
	if !cacheGroupsExist {
		return util.JoinErrs(tovalidate.ToErrors(rules))
	}

	for index, node := range topology.Nodes {
		rules[fmt.Sprintf("parent '%v' edge type", node.Cachegroup)] = topology.checkForEdgeParents(cacheGroups, index)
	}

	cacheGroupIds := make([]int, len(cacheGroupNames))
	for index, cacheGroup := range cacheGroups {
		cacheGroupIds[index] = *cacheGroup.ID
	}
	rules["empty cachegroups"] = CheckForEmptyCacheGroups(topology.ReqInfo.Tx, cacheGroupIds, false, nil)

	/* Only perform further checks if everything so far is valid */
	if err = util.JoinErrs(tovalidate.ToErrors(rules)); err != nil {
		return err
	}

	for _, leafMid := range checkForLeafMids(topology.Nodes, cacheGroups) {
		rules[fmt.Sprintf("node %v leaf mid", leafMid.Cachegroup)] = fmt.Errorf("cachegroup %v's type is %v; it cannot be a leaf (it must have at least 1 child)", leafMid.Cachegroup, tc.CacheGroupMidTypeName)
	}
	_, rules["topology cycles"] = checkForCycles(topology.Nodes)
	rules["super-topology cycles"] = topology.checkForCyclesAcrossTopologies()

	return util.JoinErrs(tovalidate.ToErrors(rules))
}

func CheckForEmptyCacheGroups(tx *sqlx.Tx, cacheGroupIds []int, cachegroupsInTopology bool, excludeServerIds []int) error {
	if excludeServerIds == nil {
		excludeServerIds = []int{}
	}
	var (
		baseError   = errors.New("unable to check for cachegroups with no servers")
		systemError = "checking for cachegroups with no servers: %s"
		query       = selectEmptyCacheGroupsQuery(cachegroupsInTopology)
		parameters  = map[string]interface{}{
			"cachegroup_ids":     pq.Array(cacheGroupIds),
			"exclude_server_ids": pq.Array(excludeServerIds),
		}
	)

	rows, err := tx.NamedQuery(query, parameters)
	if err != nil {
		log.Errorf(systemError, err.Error())
		return baseError
	}

	var (
		serverCount int
		cacheGroup  string
		cacheGroups []string
		topologies  []string
	)
	defer log.Close(rows, "unable to close DB connection when checking for cachegroups with no servers")
	for rows.Next() {
		var scanTo = []interface{}{&cacheGroup, &serverCount}
		var topologiesForRow []string
		if cachegroupsInTopology {
			scanTo = append(scanTo, pq.Array(&topologiesForRow))
		}
		if err := rows.Scan(scanTo...); err != nil {
			log.Errorf(systemError, err.Error())
			return baseError
		}
		if serverCount != 0 {
			break
		}
		cacheGroups = append(cacheGroups, cacheGroup)
		if cachegroupsInTopology {
			topologies = append(topologies, topologiesForRow...)
		}
	}

	if len(cacheGroups) > 0 {
		errMessage := "cachegroups with no servers in them: " + strings.Join(cacheGroups, ", ")
		if cachegroupsInTopology {
			errMessage += " in topologies: " + strings.Join(topologies, ", ")
		}
		err = errors.New(errMessage)
	}
	return err
}

func (topology *TOTopology) nodesInOtherTopologies() ([]tc.TopologyNode, map[string][]string, error) {
	baseError := errors.New("unable to verify that there are no cycles across all topologies")
	where := `WHERE name != :topology_name`
	query := selectQueryWithParentNames() + where + `
		UNION ` + selectNonTopologyCacheGroupsQuery() + `
		UNION ` + selectNonTopologyParentCacheGroupsQuery()

	parameters := map[string]interface{}{
		"topology_name":    topology.Name,
		"edge_type_prefix": strings.ToLower(tc.EdgeTypePrefix) + "%",
		"mid_type_prefix":  strings.ToLower(tc.MidTypePrefix) + "%",
	}
	rows, err := topology.ReqInfo.Tx.NamedQuery(query, parameters)
	if err != nil {
		return nil, nil, baseError
	}
	defer log.Close(rows, "unable to close DB connection")

	parentMapMap := map[string]map[string]bool{}
	topologiesMapByCacheGroup := map[string]map[string]bool{}
	for index := 0; rows.Next(); index++ {
		var (
			topologyName string
			cacheGroup   string
			parents      []string
		)
		topologyNode := tc.TopologyNode{}
		topologyNode.Parents = []int{}
		if err = rows.Scan(
			&topologyName,
			&cacheGroup,
			pq.Array(&parents),
		); err != nil {
			return nil, nil, baseError
		}
		if _, exists := parentMapMap[cacheGroup]; !exists {
			parentMapMap[cacheGroup] = map[string]bool{}
		}
		if _, exists := topologiesMapByCacheGroup[cacheGroup]; !exists {
			topologiesMapByCacheGroup[cacheGroup] = map[string]bool{}
		}
		for _, parent := range parents {
			parentMapMap[cacheGroup][parent] = true
		}
		topologiesMapByCacheGroup[cacheGroup][topologyName] = true
	}

	topologiesByCacheGroup := map[string][]string{}
	// Build the list of topologies containing each cache group
	for cacheGroup, topologiesMap := range topologiesMapByCacheGroup {
		var topologies []string
		for topology, _ := range topologiesMap {
			topologies = append(topologies, topology)
		}
		topologiesByCacheGroup[cacheGroup] = topologies
	}

	// Add nodes for the topology we are validating
	for _, node := range topology.Nodes {
		if _, exists := parentMapMap[node.Cachegroup]; !exists {
			parentMapMap[node.Cachegroup] = map[string]bool{}
		}
		for _, parentCacheGroupIndex := range node.Parents {
			parentMapMap[node.Cachegroup][topology.Nodes[parentCacheGroupIndex].Cachegroup] = true
		}
	}

	indexByCachegroup := map[string]int{}
	var cacheGroups []string
	index := 0
	// Get an index for each cachegroup
	for cacheGroup, _ := range parentMapMap {
		cacheGroups = append(cacheGroups, cacheGroup)
		indexByCachegroup[cacheGroup] = index
		index++
	}

	var nodes []tc.TopologyNode
	// Reduce parentMapMap to an array of TopologyNodes
	// We can't rely on iterating through parentMapMap in the same order twice, so we iterate through
	// cacheGroups instead
	for _, cacheGroup := range cacheGroups {
		parentMap := parentMapMap[cacheGroup]
		node := tc.TopologyNode{Cachegroup: cacheGroup}
		for parent, _ := range parentMap {
			node.Parents = append(node.Parents, indexByCachegroup[parent])
		}
		nodes = append(nodes, node)
	}

	return nodes, topologiesByCacheGroup, nil
}

// Implementation of the Identifier, Validator interface functions
func (topology TOTopology) GetKeys() (map[string]interface{}, bool) {
	return map[string]interface{}{"name": topology.Name}, true
}

// SetKeys is a requirement of the api.Updater interface and is called by
// api.UpdateHandler().
func (topology *TOTopology) SetKeys(keys map[string]interface{}) {
	topology.Name, _ = keys["name"].(string)
}

// GetAuditName is a requirement of the api.Identifier interface.
func (topology *TOTopology) GetAuditName() string {
	return topology.Name
}

// Create is a requirement of the api.Creator interface.
func (topology *TOTopology) Create() apierrors.Errors {
	tx := topology.APIInfo().Tx.Tx
	err := tx.QueryRow(insertQuery(), topology.Name, topology.Description).Scan(&topology.Name, &topology.Description, &topology.LastUpdated)
	if err != nil {
		return api.ParseDBError(err)
	}

	if errs := topology.addNodes(); errs.Occurred() {
		return errs
	}

	if errs := topology.addParents(); errs.Occurred() {
		return errs
	}

	return apierrors.New()
}

// Read is a requirement of the api.Reader interface and is called by api.ReadHandler().
func (topology *TOTopology) Read(h http.Header, useIMS bool) ([]interface{}, apierrors.Errors, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	interfaces := make([]interface{}, 0)
	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(topology.ReqInfo.Params, topology.ParamColumns())
	if len(errs) > 0 {
		return nil, apierrors.Errors{UserError: util.JoinErrs(errs), Code: http.StatusBadRequest}, nil
	}
	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(topology.ReqInfo.Tx, h, queryValues, selectMaxLastUpdatedQuery(where))
		if !runSecond {
			log.Debugln("IMS HIT")
			return interfaces, apierrors.Errors{Code: http.StatusNotModified}, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}
	// Case where we need to run the second query
	query := selectQuery() + where + orderBy + pagination
	rows, err := topology.ReqInfo.Tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, apierrors.Errors{SystemError: errors.New("topology read: querying: " + err.Error()), Code: http.StatusInternalServerError}, nil
	}
	defer log.Close(rows, "unable to close DB connection")
	topologies := map[string]*tc.Topology{}
	indices := map[int]int{}
	for index := 0; rows.Next(); index++ {
		var (
			name, description string
			lastUpdated       tc.TimeNoMod
		)
		topologyNode := tc.TopologyNode{}
		topologyNode.Parents = []int{}
		var parents pq.Int64Array
		if err = rows.Scan(
			&name,
			&description,
			&lastUpdated,
			&topologyNode.Id,
			&topologyNode.Cachegroup,
			&parents,
		); err != nil {
			return nil, apierrors.Errors{SystemError: errors.New("topology read: scanning: " + err.Error()), Code: http.StatusInternalServerError}, nil
		}
		for _, id := range parents {
			topologyNode.Parents = append(topologyNode.Parents, int(id))
		}
		indices[topologyNode.Id] = index
		if _, exists := topologies[name]; !exists {
			topology := tc.Topology{Nodes: []tc.TopologyNode{}}
			topologies[name] = &topology
			topology.Name = name
			topology.Description = description
			topology.LastUpdated = &lastUpdated
		}
		topologies[name].Nodes = append(topologies[name].Nodes, topologyNode)
	}

	for _, topology := range topologies {
		nodeMap := map[int]int{}
		for index, node := range topology.Nodes {
			nodeMap[node.Id] = index
		}
		for _, node := range topology.Nodes {
			for parentIndex := 0; parentIndex < len(node.Parents); parentIndex++ {
				node.Parents[parentIndex] = nodeMap[node.Parents[parentIndex]]
			}
		}
		interfaces = append(interfaces, *topology)
	}
	return interfaces, apierrors.New(), &maxTime
}

func (topology *TOTopology) removeParents() error {
	_, err := topology.ReqInfo.Tx.Exec(deleteParentsQuery(), topology.Name)
	if err != nil {
		return errors.New("topology update: error deleting old parents: " + err.Error())
	}
	return nil
}

func (topology *TOTopology) removeNodes(cachegroups *[]string) error {
	_, err := topology.ReqInfo.Tx.Exec(deleteNodesQuery(), topology.Name, pq.Array(*cachegroups))
	if err != nil {
		return errors.New("topology update: error removing old unused nodes: " + err.Error())
	}
	return nil
}

func (topology *TOTopology) addNodes() apierrors.Errors {
	var cachegroupsToInsert []string
	var indices = make([]int, 0)
	for index, node := range topology.Nodes {
		if node.Id == 0 {
			cachegroupsToInsert = append(cachegroupsToInsert, node.Cachegroup)
			indices = append(indices, index)
		}
	}
	if len(cachegroupsToInsert) == 0 {
		return apierrors.New()
	}
	rows, err := topology.ReqInfo.Tx.Query(nodeInsertQuery(), topology.Name, pq.Array(cachegroupsToInsert))
	if err != nil {
		return apierrors.Errors{
			Code:        http.StatusInternalServerError,
			SystemError: errors.New("error adding nodes: " + err.Error()),
		}
	}
	defer log.Close(rows, "unable to close DB connection")
	for _, index := range indices {
		rows.Next()
		err = rows.Scan(&topology.Nodes[index].Id, &topology.Name, &topology.Nodes[index].Cachegroup)
		if err != nil {
			return api.ParseDBError(err)
		}
	}
	return apierrors.New()
}

func (topology *TOTopology) addParents() apierrors.Errors {
	var (
		children []int
		parents  []int
		ranks    []int
	)
	for _, node := range topology.Nodes {
		for rank := 1; rank <= len(node.Parents); rank++ {
			parent := topology.Nodes[node.Parents[rank-1]]
			children = append(children, node.Id)
			parents = append(parents, parent.Id)
			ranks = append(ranks, rank)
		}
	}
	rows, err := topology.ReqInfo.Tx.Query(nodeParentInsertQuery(), pq.Array(children), pq.Array(parents), pq.Array(ranks))
	if err != nil {
		return api.ParseDBError(err)
	}
	defer log.Close(rows, "unable to close DB connection")
	for _, node := range topology.Nodes {
		for rank := 1; rank <= len(node.Parents); rank++ {
			rows.Next()
			parent := topology.Nodes[node.Parents[rank-1]]
			err = rows.Scan(&node.Id, &parent.Id, &rank)
			if err != nil {
				return api.ParseDBError(err)
			}
		}
	}
	return apierrors.New()
}

func (topology *TOTopology) setDescription() apierrors.Errors {
	errs := apierrors.New()
	rows, err := topology.ReqInfo.Tx.Query(updateQuery(), topology.Description, topology.Name)
	if err != nil {
		errs.SystemError = fmt.Errorf("topology update: error setting the description for topology %v: %v", topology.Name, err.Error())
		errs.Code = http.StatusInternalServerError
		return errs
	}
	defer log.Close(rows, "unable to close DB connection")
	for rows.Next() {
		err = rows.Scan(&topology.Name, &topology.Description, &topology.LastUpdated)
		if err != nil {
			return api.ParseDBError(err)
		}
	}
	return errs
}

// Update is a requirement of the api.Updater interface.
func (topology *TOTopology) Update() apierrors.Errors {
	topologies, errs, _ := topology.Read(nil, false)
	if errs.Occurred() {
		return errs
	}
	if len(topologies) != 1 {
		errs.SetUserError("cannot find exactly 1 topology with the query string provided")
		errs.Code = http.StatusBadRequest
		return errs
	}
	oldTopology := TOTopology{APIInfoImpl: topology.APIInfoImpl, Topology: topologies[0].(tc.Topology)}
	if errs = topology.setDescription(); errs.Occurred() {
		return errs
	}

	if err := oldTopology.removeParents(); err != nil {
		errs.SystemError = err
		errs.Code = http.StatusInternalServerError
		return errs
	}
	var oldNodes, newNodes = map[string]int{}, map[string]int{}
	for index, node := range oldTopology.Nodes {
		oldNodes[node.Cachegroup] = index
	}
	for index, node := range topology.Nodes {
		newNodes[node.Cachegroup] = index
	}
	var toRemove []string
	for cachegroupName := range oldNodes {
		if _, exists := newNodes[cachegroupName]; !exists {
			toRemove = append(toRemove, cachegroupName)
		} else {
			topology.Nodes[newNodes[cachegroupName]].Id = oldTopology.Nodes[oldNodes[cachegroupName]].Id
		}
	}
	if len(toRemove) > 0 {
		if err := oldTopology.removeNodes(&toRemove); err != nil {
			errs.SystemError = err
			errs.Code = http.StatusInternalServerError
			return errs
		}
	}
	if errs = topology.addNodes(); errs.Occurred() {
		return errs
	}
	return topology.addParents()
}

// Delete is unused and simply satisfies the Deleter interface
// (although TOTOpology is used as an OptionsDeleter)
func (topology *TOTopology) Delete() apierrors.Errors {
	return apierrors.New()
}

// OptionsDelete is a requirement of the OptionsDeleter interface.
func (topology *TOTopology) OptionsDelete() apierrors.Errors {
	topologies, errs, _ := topology.Read(nil, false)
	if errs.Occurred() {
		return errs
	}
	if len(topologies) != 1 {
		errs.UserError = fmt.Errorf("cannot find exactly 1 topology with the query string provided")
		errs.Code = http.StatusBadRequest
		return errs
	}
	topology.Topology = topologies[0].(tc.Topology)

	var cachegroups []string
	for _, node := range topology.Nodes {
		cachegroups = append(cachegroups, node.Cachegroup)
	}
	return api.GenericOptionsDelete(topology)
}

func insertQuery() string {
	query := `
INSERT INTO topology (name, description)
VALUES ($1, $2)
RETURNING name, description, last_updated
`
	return query
}

func nodeInsertQuery() string {
	query := `
INSERT INTO topology_cachegroup (topology, cachegroup)
VALUES ($1, unnest($2::text[]))
RETURNING id, topology, cachegroup
`
	return query
}

func nodeParentInsertQuery() string {
	query := `
INSERT INTO topology_cachegroup_parents (child, parent, rank)
VALUES (unnest($1::int[]), unnest($2::int[]), unnest($3::int[]))
RETURNING child, parent, rank
`
	return query
}

func selectQuery() string {
	query := `
SELECT t.name, t.description, t.last_updated,
tc.id, tc.cachegroup,
	(SELECT COALESCE (ARRAY_AGG (CAST (tcp.parent as INT) ORDER BY tcp.rank ASC)) AS parents
	FROM topology_cachegroup tc2
	INNER JOIN topology_cachegroup_parents tcp ON tc2.id = tcp.child
	WHERE tc2.topology = tc.topology
	AND tc2.cachegroup = tc.cachegroup
	)
FROM topology t
JOIN topology_cachegroup tc on t.name = tc.topology
`
	return query
}

func selectQueryWithParentNames() string {
	query := `
SELECT t.name, tc.cachegroup,
	(SELECT COALESCE (ARRAY_AGG (tcpc.cachegroup)) AS parents
	FROM topology_cachegroup tc2
	INNER JOIN topology_cachegroup_parents tcp ON tc2.id = tcp.child
	INNER JOIN topology_cachegroup tcpc ON tcp.parent = tcpc.id
	WHERE tc2.topology = tc.topology
	AND tc2.cachegroup = tc.cachegroup
	)
FROM topology t
JOIN topology_cachegroup tc on t.name = tc.topology
`
	return query
}

func selectEmptyCacheGroupsQuery(cachegroupsInTopology bool) string {
	var joinTopologyCachegroups string
	var topologyNames string
	if cachegroupsInTopology {
		// language=SQL
		topologyNames = `
		, ARRAY_AGG(tc.topology)
`
		// language=SQL
		joinTopologyCachegroups = `
		JOIN topology_cachegroup tc ON c."name" = tc.cachegroup
`
	}
	// language=SQL
	query := fmt.Sprintf(`
		SELECT
			c."name",
			COUNT(*) FILTER (
			    WHERE s.id IS NOT NULL
			    AND NOT(s."id" = ANY(CAST(:exclude_server_ids AS INT[])))
			) AS server_count %s
		FROM cachegroup c
		%s
		LEFT JOIN "server" s ON c.id = s.cachegroup
		WHERE c."id" = ANY(CAST(:cachegroup_ids AS BIGINT[]))
		GROUP BY c."name"
		ORDER BY server_count
	`, topologyNames, joinTopologyCachegroups)
	return query
}

func selectNonTopologyCacheGroupsQuery() string {
	query := `
SELECT 'non-topology cachegroups' AS name, c."name" AS cachegroup,
	(SELECT COALESCE (ARRAY_AGG (pc."name")) AS parents
	FROM cachegroup pc
	WHERE pc.id = c.parent_cachegroup_id
	OR pc.id = c.secondary_parent_cachegroup_id
	)
FROM cachegroup c
JOIN "type" t ON c."type" = t.id
WHERE (LOWER(t.name) LIKE :edge_type_prefix
OR LOWER(t.name) LIKE :mid_type_prefix)
AND (c.parent_cachegroup_id IS NOT NULL
OR c.secondary_parent_cachegroup_id IS NOT NULL)
`
	return query
}

func selectNonTopologyParentCacheGroupsQuery() string {
	query := `
SELECT 'non-topology cachegroups' AS name, pc2."name" AS cachegroup,
	(SELECT COALESCE (ARRAY_AGG (pc."name")) AS parents
	FROM cachegroup pc
	WHERE pc.id = pc2.parent_cachegroup_id
	OR pc.id = pc2.secondary_parent_cachegroup_id
	)
FROM cachegroup c
JOIN "type" t ON c."type" = t.id
JOIN cachegroup pc2 ON c.parent_cachegroup_id = pc2.id
	OR c.secondary_parent_cachegroup_id = pc2.id
WHERE (LOWER(t.name) LIKE :edge_type_prefix
OR LOWER(t.name) LIKE :mid_type_prefix)
AND (c.parent_cachegroup_id IS NOT NULL
OR c.secondary_parent_cachegroup_id IS NOT NULL)
`
	return query
}

func deleteQueryBase() string {
	query := `
DELETE FROM topology t
`
	return query
}

func deleteNodesQuery() string {
	query := `
DELETE FROM topology_cachegroup tc
WHERE tc.topology = $1
AND tc.cachegroup = ANY ($2::text[])
`
	return query
}

func deleteParentsQuery() string {
	query := `
DELETE FROM topology_cachegroup_parents tcp
WHERE tcp.child IN
    (SELECT tc.id
    FROM topology t
    JOIN topology_cachegroup tc on t.name = tc.topology
    WHERE t.name = $1)
`
	return query
}

func updateQuery() string {
	query := `
UPDATE topology t SET
description = $1
WHERE t.name = $2
RETURNING t.name, t.description, t.last_updated
`
	return query
}

func selectMaxLastUpdatedQuery(where string) string {
	return `SELECT max(ti) from (
		SELECT max(t.last_updated) as ti from topology t JOIN topology_cachegroup tc on t.name = tc.topology` + where +
		` UNION ALL
	select max(last_updated) as ti from last_deleted l where l.table_name='topology') as res`
}
