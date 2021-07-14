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

var TenantUtils = function () {

	/*
	 * recurse thru a hierarchical structure and
	 * add a 'level' property to each item
	 */
	var applyLevels = function (arr, level) {
		arr.forEach(function(item)  {
			item.level = level;
			applyLevels(item.children, level + 1);
		});
	};

	var hierarchySortFunc = function (a, b) {
		return a.name > b.name;
	};

	/*
	 * sort parent groups into an array
	 * representative of the hierarchy order
	 */
	this.hierarchySort = function (parentGroups, rootParentId, result) {

		if (parentGroups[rootParentId] == undefined) return;
		var arr = parentGroups[rootParentId].sort(hierarchySortFunc);
		for (var i = 0; i < arr.length; i++) {
			result.push(arr[i]);
			this.hierarchySort(parentGroups, arr[i].id, result);
		}

		return result;
	};

	/*
	 * take a flat tenant array and
	 * group the tenants by parentId
	 */
	this.groupTenantsByParent = function (tenants) {

		var parentGroups = {};

		for (var i = 0; i < tenants.length; i++) {
			if (parentGroups[tenants[i].parentId] == undefined) parentGroups[tenants[i].parentId] = [];
			parentGroups[tenants[i].parentId].push(tenants[i]);
		}

		return parentGroups;
	};

	/*
	 * Takes a flat tenant array and turns it into a hierarchical
	 * representation of tenants based on their parent/child relationships
	 */
	this.convertToHierarchy = function (tenantsArr) {
		var map = {};

		for (var i = 0; i < tenantsArr.length; i++) {
			var obj = tenantsArr[i];
			obj.children = [];

			map[obj.id] = obj;

			var parent = (i == 0) ? '-' : obj.parentId; // the first item is always the top-level tenant
			if (!map[parent]) {
				map[parent] = {
					children: []
				};
			}
			map[parent].children.push(obj);
		}

		return map['-'].children;
	};

	this.addLevels = function (tenants) {
		// convert a flat tenant list into a hierarchy
		var tenantHierarchy = this.convertToHierarchy(tenants);
		// and then walk down the hierarchy to find out how deeply nested each tenant is and add a 'level' property to each tenant
		applyLevels(tenantHierarchy, 1);
	};

};

TenantUtils.$inject = [];
module.exports = TenantUtils;
