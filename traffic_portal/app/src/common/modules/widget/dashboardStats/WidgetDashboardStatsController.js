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

var WidgetDashboardStatsController = function($scope, $interval, $filter, locationUtils, cacheGroupService, cdnService, serverService, propertiesModel) {

	var cacheGroupHealthInterval,
		currentStatsInterval,
		serverCountInterval,
		autoRefresh = propertiesModel.properties.dashboard.autoRefresh;

	var serverCount = new Map([
		["ONLINE", "Loading..."],
		["OFFLINE", "Loading..."],
		["REPORTED", "Loading..."],
		["ADMIN_DOWN", "Loading..."]
	]);

	var getCacheGroupHealth = function() {
		cacheGroupService.getCacheGroupHealth()
			.then(
				function(result) {
					$scope.totalOnline = $filter('number')(result.totalOnline, 0);
					$scope.totalOffline = $filter('number')(result.totalOffline, 0);
				},
				function() {
					$scope.totalOnline = 'Error';
					$scope.totalOffline = 'Error';
				}
			);
	};

	var getCurrentStats = function() {
		cdnService.getCurrentStats()
			.then(
				function(result) {
					const totalStats = result.currentStats.find(function(item)  {
						// total stats are buried in a hash where cdn = total
						return item.cdn === 'total';
					});
					$scope.totalBandwidth = $filter('number')(totalStats.bandwidth, 2) + ' Gbps';
					$scope.totalConnections = $filter('number')(totalStats.connections, 0);
				},
				function() {
					$scope.totalBandwidth = 'Error';
					$scope.totalConnections = 'Error';
				}
			);
	};

	var getServerCount = function() {
		serverService.getServers({type: "EDGE"})
			.then(function(result) {
				serverCount.set("ONLINE", 0);
				serverCount.set("OFFLINE", 0);
				serverCount.set("REPORTED", 0);
				serverCount.set("ADMIN_DOWN", 0);
				for (let s in result) {
					const server = result[s]; // webpack won't handle a for...of loop
					if (serverCount.has(server.status)) {
						serverCount.set(server.status, serverCount.get(server.status) + 1);
					}
				}
			});
	};

	var createIntervals = function() {
		killIntervals();
		cacheGroupHealthInterval = $interval(function() { getCacheGroupHealth() }, propertiesModel.properties.dashboard.healthyCacheCount.refreshRateInMS );
		currentStatsInterval = $interval(function() { getCurrentStats() }, propertiesModel.properties.dashboard.currentStats.refreshRateInMS );
		serverCountInterval = $interval(function() { getServerCount() }, propertiesModel.properties.dashboard.cacheStatusCount.refreshRateInMS );
	};

	var killIntervals = function() {
		if (angular.isDefined(cacheGroupHealthInterval)) {
			$interval.cancel(cacheGroupHealthInterval);
			cacheGroupHealthInterval = undefined;
		}
		if (angular.isDefined(currentStatsInterval)) {
			$interval.cancel(currentStatsInterval);
			currentStatsInterval = undefined;
		}
		if (angular.isDefined(serverCountInterval)) {
			$interval.cancel(serverCountInterval);
			serverCountInterval = undefined;
		}
	};

	$scope.totalBandwidth = 'Loading...';

	$scope.totalConnections = 'Loading...';

	$scope.totalOnline = 'Loading...';

	$scope.totalOffline = 'Loading...';

	$scope.online = function() {
		return serverCount.get("ONLINE");
	};

	$scope.offline = function() {
		return serverCount.get("OFFLINE");
	};

	$scope.reported = function() {
		return serverCount.get("REPORTED");
	};

	$scope.adminDown = function() {
		return serverCount.get("ADMIN_DOWN");
	};

	$scope.navigateToPath = locationUtils.navigateToPath;

	$scope.$on("$destroy", function() {
		killIntervals();
	});

	var init = function () {
		getServerCount();
		getCurrentStats();
		getCacheGroupHealth();
		if (autoRefresh) {
			createIntervals();
		}
	};
	init();

};

WidgetDashboardStatsController.$inject = ['$scope', '$interval', '$filter', 'locationUtils', 'cacheGroupService', 'cdnService', 'serverService', 'propertiesModel'];
module.exports = WidgetDashboardStatsController;
