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

var TableRoleCapabilitiesController = function(roles, $scope, $state, $uibModal, locationUtils, capabilityService, roleService, messageModel) {

	$scope.role = roles[0];

	$scope.editCapability = function(name) {
		locationUtils.navigateToPath('/capabilities/' + name);
	};

	$scope.selectCapabilities = function() {
		var modalInstance = $uibModal.open({
			templateUrl: 'common/modules/table/roleCapabilities/table.assignCapabilities.tpl.html',
			controller: 'TableAssignCapabilitiesController',
			size: 'lg',
			resolve: {
				role: function() {
					return $scope.role;
				},
				capabilities: function(capabilityService) {
					return capabilityService.getCapabilities();
				},
				assignedCapabilities: function() {
					return $scope.role.capabilities;
				}
			}
		});
		modalInstance.result.then(function(selectedCapabilities) {
			$scope.role.capabilities = selectedCapabilities;
			roleService.updateRole($scope.role)
				.then(
					function(result) {
						messageModel.setMessages(result.alerts, false);
						$scope.refresh();
					}
				);
		}, function () {
			// do nothing
		});
	};

	$scope.confirmRemoveCapability = function(capToRemove, $event) {
		$event.stopPropagation(); // this kills the click event so it doesn't trigger anything else

		var params = {
			title: 'Remove Capabilty from Role?',
			message: 'Are you sure you want to remove ' + capToRemove + ' from this role?'
		};
		var modalInstance = $uibModal.open({
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
			$scope.role.capabilities = $scope.role.capabilities.filter(function(cap)  { return cap !== capToRemove; });
			roleService.updateRole($scope.role)
				.then(
					function(result) {
						messageModel.setMessages(result.alerts, false);
						$scope.refresh();
					}
				);
		}, function () {
			// do nothing
		});
	};

	$scope.refresh = function() {
		$state.reload(); // reloads all the resolves for the view
	};

	$scope.navigateToPath = locationUtils.navigateToPath;

	angular.element(document).ready(function () {
		$('#capabilitiesTable').dataTable({
			"aLengthMenu": [[25, 50, 100, -1], [25, 50, 100, "All"]],
			"iDisplayLength": 25,
			"columnDefs": [
				{ 'orderable': false, 'targets': 1 }
			],
			"aaSorting": []
		});
	});

};

TableRoleCapabilitiesController.$inject = ['roles', '$scope', '$state', '$uibModal', 'locationUtils', 'capabilityService', 'roleService', 'messageModel'];
module.exports = TableRoleCapabilitiesController;
