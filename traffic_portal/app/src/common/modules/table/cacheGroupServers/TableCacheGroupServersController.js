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

var TableCacheGroupsServersController = function(cacheGroup, servers, filter, $controller, $scope, $state, $uibModal, cacheGroupService) {

	// extends the TableServersController to inherit common methods
	angular.extend(this, $controller('TableServersController', { tableName: 'cacheGroupServers', servers: servers, filter: filter, $scope: $scope }));

	$scope.cacheGroup = cacheGroup;

	/** @type CGC.TitleBreadCrumbs */
	$scope.breadCrumbs = [{
		text: "Cache Groups",
		href: "#!/cache-groups"
	},
	{
		getText: function() {
			return $scope.cacheGroup.name;
		},
		getHref: function() {
			return "#!/cache-groups/" + $scope.cacheGroup.id;
		}
	},
	{
		text: "Servers"
	}];

	let queueCacheGroupServerUpdates = function(cacheGroup, cdnId) {
		cacheGroupService.queueServerUpdates(cacheGroup.id, cdnId)
			.then(
				function() {
					$scope.refresh();
				}
			);
	};

	let clearCacheGroupServerUpdates = function(cacheGroup, cdnId) {
		cacheGroupService.clearServerUpdates(cacheGroup.id, cdnId)
			.then(
				function() {
					$scope.refresh();
				}
			);
	};

	$scope.confirmCacheGroupQueueServerUpdates = function(cacheGroup) {
		const params = {
			title: 'Queue Server Updates: ' + cacheGroup.name,
			message: "Please select a CDN"
		};
		const modalInstance = $uibModal.open({
			templateUrl: 'common/modules/dialog/select/dialog.select.tpl.html',
			controller: 'DialogSelectController',
			size: 'md',
			resolve: {
				params: function () {
					return params;
				},
				collection: function(cdnService) {
					return cdnService.getCDNs();
				}
			}
		});
		modalInstance.result.then(function(cdn) {
			queueCacheGroupServerUpdates(cacheGroup, cdn.id);
		}, function () {
			// do nothing
		});
	};

	$scope.confirmCacheGroupClearServerUpdates = function(cacheGroup) {
		const params = {
			title: 'Clear Server Updates: ' + cacheGroup.name,
			message: "Please select a CDN"
		};
		const modalInstance = $uibModal.open({
			templateUrl: 'common/modules/dialog/select/dialog.select.tpl.html',
			controller: 'DialogSelectController',
			size: 'md',
			resolve: {
				params: function () {
					return params;
				},
				collection: function(cdnService) {
					return cdnService.getCDNs();
				}
			}
		});
		modalInstance.result.then(function(cdn) {
			clearCacheGroupServerUpdates(cacheGroup, cdn.id);
		}, function () {
			// do nothing
		});
	};

	this.$onInit = function() {
		let i;
		for(i = 0; i < $scope.dropDownOptions.length; ++i) {
			const ddo = $scope.dropDownOptions[i];
			if (ddo.text !== undefined){
				if (ddo.text === "Queue Server Updates") {
					$scope.dropDownOptions[i].onClick = function(entry) { $scope.confirmCacheGroupQueueServerUpdates($scope.cacheGroup); };
				} else if (ddo.text === "Clear Server Updates") {
					$scope.dropDownOptions[i].onClick = function(entry) { $scope.confirmCacheGroupClearServerUpdates($scope.cacheGroup); };
				}
			}
		}
	};
};

TableCacheGroupsServersController.$inject = ['cacheGroup', 'servers', 'filter', '$controller', '$scope', '$state', '$uibModal', 'cacheGroupService'];
module.exports = TableCacheGroupsServersController;
