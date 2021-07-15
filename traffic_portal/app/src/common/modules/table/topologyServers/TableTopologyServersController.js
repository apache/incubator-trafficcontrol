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

var TableTopologyServersController = function(topologies, servers, filter, $controller, $scope, $uibModal, cdnService, topologyService) {

	// extends the TableServersController to inherit common methods
	angular.extend(this, $controller('TableServersController', { tableName: 'topologyServers', servers: servers, filter: filter, $scope: $scope }));

	$scope.topology = topologies[0];

	/** @type CGC.TitleBreadCrumbs[] */
	$scope.breadCrumbs = [{
		text: "Topologies",
		href: "#!/topologies"
	},
	{
		getText: function() { return $scope.topology.name; },
		getHref: function() { return "#!/topologies/edit?name=" + $scope.topology.name; }
	},
	{
		text: "Servers"
	}];

	$scope.confirmTopologyQueueServerUpdates = function(topology) {
		const params = {
			title: 'Queue Server Updates: ' + topology.name,
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
			topologyService.queueServerUpdates(topology.name, cdn.id).then($scope.refresh);
		}, function () {
			console.log('Queue server updated cancelled');
		});
	};

	$scope.confirmTopologyClearServerUpdates = function(topology) {
		const params = {
			title: 'Clear Server Updates: ' + topology.name,
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
			topologyService.clearServerUpdates(topology.name, cdn.id).then($scope.refresh);
		}, function () {
			console.log('Clear server updated cancelled');
		});
	};

	this.$onInit = function() {
		let i;
		for(i = 0; i < $scope.dropDownOptions.length; ++i) {
			const ddo = $scope.dropDownOptions[i];
			if (ddo.text !== undefined){
				if (ddo.text === "Queue Server Updates") {
					$scope.dropDownOptions[i].onClick = function(entry) { $scope.confirmTopologyQueueServerUpdates($scope.topology); };
				} else if (ddo.text === "Clear Server Updates") {
					$scope.dropDownOptions[i].onClick = function(entry) { $scope.confirmTopologyClearServerUpdates($scope.topology); };
				}
			}
		}
	};

};

TableTopologyServersController.$inject = ['topologies', 'servers', 'filter', '$controller', '$scope', '$uibModal', 'cdnService', 'topologyService'];
module.exports = TableTopologyServersController;
