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

var TableDeliveryServiceServersController = function(deliveryService, servers, filter, $controller, $scope, $uibModal, deliveryServiceService, locationUtils) {

	// extends the TableServersController to inherit common methods
	angular.extend(this, $controller('TableServersController', { tableName: 'deliveryServiceServers', servers: servers, filter: filter, $scope: $scope }));

	let removeServer = function(serverId) {
		deliveryServiceService.deleteDeliveryServiceServer($scope.deliveryService.id, serverId)
			.then(
				function() {
					$scope.refresh();
				}
			);
	};

	$scope.deliveryService = deliveryService;

	this.$onInit = function() {
	    $scope.contextMenuOptions.push({
			type: 0
		});
		$scope.contextMenuOptions.push({
			type: 1,
			shown: function(row) {
			    return !row.topology;
			},
			onClick: function(row) {
				$scope.confirmRemoveServer(row);
			},
			isDisabled: function(row) {
				return !$scope.isEdge(row) && !$scope.isOrigin(row);
			},
			getText: function(row) {
				return "Remove " + row.type + "Server";
			}
		});
		$scope.contextMenuOptions.push({
			type: 1,
			shown: function(row) {
				return row.topology;
			},
			onClick: function(row) {
				$scope.confirmRemoveServer(row);
			},
			isDisabled: function(row) {
				return !$scope.isOrigin(row);
			},
			getText: function(row) {
				return "Remove " + row.type + "Server";
			}
		});

		$scope.dropDownOptions.push({
			type: 0
		});
		$scope.dropDownOptions.push({
			type: 1,
            name: "selectServersMenuItem",
			onClick: function() {
				$scope.selectServers();
			},
			getText: function () {
				return "Assign " + ($scope.deliveryService.topology ? "ORG " : "") + "Servers";
			}
		});
	};

	$scope.selectServers = function() {
		var modalInstance = $uibModal.open({
			templateUrl: 'common/modules/table/deliveryServiceServers/table.assignDSServers.tpl.html',
			controller: 'TableAssignDSServersController',
			size: 'lg',
			resolve: {
				deliveryService: function() {
					return deliveryService;
				},
				servers: function(serverService) {
					if (deliveryService.topology) {
						// topology-based ds's can only have ORG servers from the same CDN directly assigned
						return serverService.getServers({ type: 'ORG', cdn: deliveryService.cdnId });
					} else {
						return serverService.getEligibleDeliveryServiceServers(deliveryService.id);
					}
				},
				assignedServers: function() {
					return servers;
				}
			}
		});
		modalInstance.result.then(function(selectedServerIds) {
			deliveryServiceService.assignDeliveryServiceServers(deliveryService.id, selectedServerIds)
				.then(
					function() {
						$scope.refresh();
					}
				);
		}, function () {
			// do nothing
		});
	};

	$scope.confirmRemoveServer = function(server) {
		const params = {
			title: 'Remove Server from Delivery Service?',
			message: 'Are you sure you want to remove ' + server.hostName + ' from this delivery service?'
		};
		const modalInstance = $uibModal.open({
			templateUrl: 'common/modules/dialog/confirm/dialog.confirm.tpl.html',
			controller: 'DialogConfirmController',
			size: 'md',
			resolve: {
				params: function () {
					return params;
				}
			}
		});
		modalInstance.result.then(function() {
			removeServer(server.id);
		}, function () {
			// do nothing
		});
	};

	/** @type CGC.TitleButton */
	$scope.titleButton = {
	    onClick: function() {
	    	locationUtils.navigateToPath("topologies/edit?name=" + $scope.deliveryService.topology);
		},
		getText: function() {
	    	return "[ " + $scope.deliveryService.topology + " topology ]";
		}
	};

	/** @type CGC.TitleBreadCrumbs */
	$scope.breadCrumbs = [{
		href: "#!/delivery-services",
		text: "Delivery Services"
	},
		{
			getHref: function() {
				return "#!/delivery-services/" + $scope.deliveryService.id + "?type=" + $scope.deliveryService.type;
			},
			getText: function() {
				return $scope.deliveryService.xmlId;
			}
		},
		{
			text: "Servers"
		}];

};

TableDeliveryServiceServersController.$inject = ['deliveryService', 'servers', 'filter', '$controller', '$scope', '$uibModal', 'deliveryServiceService', 'locationUtils'];
module.exports = TableDeliveryServiceServersController;
