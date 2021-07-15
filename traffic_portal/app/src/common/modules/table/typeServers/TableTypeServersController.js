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
/** @typedef { import('../agGrid/CommonGridController').CGC } CGC */

var TableTypeServersController = function(type, servers, filter, $controller, $scope) {

	// extends the TableServersController to inherit common methods
	angular.extend(this, $controller('TableServersController', { tableName: 'typeServers', servers: servers, filter: filter, $scope: $scope }));

	/** @type CGC.TitleBreadCrumbs[] */
	$scope.breadCrumbs = [{
		href: "#!/types",
		text: "Types"
	},
	{
		getHref: function() {
			return "#!/types/" + $scope.type.id;
		},
		getText: function() {
			return $scope.type.name;
		}
	},
	{
		text: "Servers"
	}];

	$scope.type = type;
};

TableTypeServersController.$inject = ['type', 'servers', 'filter', '$controller', '$scope'];
module.exports = TableTypeServersController;
