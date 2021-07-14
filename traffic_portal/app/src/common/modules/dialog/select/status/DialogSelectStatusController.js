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

var DialogSelectStatusController = function(server, statuses, $scope, $uibModalInstance) {

	$scope.server = server;

	$scope.statuses = statuses;

	$scope.selectedStatusId = null;

	$scope.status = {
		id: null,
		name: null,
		offlineReason: null
	};

	$scope.select = function() {
		const selectedStatusId = parseInt($scope.selectedStatusId);
		const selectedStatus = statuses.find( function(status)  { return parseInt(status.id) === selectedStatusId });
		$scope.status.id = selectedStatus.id;
		$scope.status.name = selectedStatus.name;
		$uibModalInstance.close($scope.status);
	};

	$scope.needsUpdates = function(server) {
		return (server.type.indexOf('EDGE') != -1) || (server.type.indexOf('MID') != -1);
	};

	$scope.cancel = function () {
		$uibModalInstance.dismiss('cancel');
	};

	$scope.offline = function () {
		const selectedStatusId = parseInt(selectedStatusId);
		const selectedStatus = statuses.find( function(status)  { return parseInt(status.id) === selectedStatusId });
		return selectedStatus && (selectedStatus.name === "ADMIN_DOWN" || selectedStatus.name === "OFFLINE");
	};

};

DialogSelectStatusController.$inject = ['server', 'statuses', '$scope', '$uibModalInstance'];
module.exports = DialogSelectStatusController;
