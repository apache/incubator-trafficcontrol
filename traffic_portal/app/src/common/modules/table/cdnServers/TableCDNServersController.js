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

var TableCDNServersController = function(cdn, servers, filter, $controller, $scope) {

	// extends the TableServersController to inherit common methods
	angular.extend(this, $controller('TableServersController', { tableName: 'cdnServers', servers: servers, filter: filter, $scope: $scope }));

	/** @type CGC.TitleBreadCrumbs */
	$scope.breadCrumbs = [{
		text: "CDNs",
		href: "#!/cdns"
	},
	{
		getText: function() { return $scope.cdn.name; },
		getHref: function() { return "#!/cdns/" + $scope.cdn.id; }
	},
	{
		text: "Servers"
	}];

	$scope.cdn = cdn;
};

TableCDNServersController.$inject = ['cdn', 'servers', 'filter', '$controller', '$scope'];
module.exports = TableCDNServersController;
