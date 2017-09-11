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

var TableServerDeliveryServicesController = function(server, serverDeliveryServices, $scope, $state, dateUtils, deliveryServiceUtils, locationUtils, deliveryServiceService) {

	var protocols = deliveryServiceUtils.protocols;

	var qstrings = deliveryServiceUtils.qstrings;

	$scope.server = server;

	$scope.serverDeliveryServices = serverDeliveryServices;

	$scope.protocol = function(ds) {
		return protocols[ds.protocol];
	};

	$scope.qstring = function(ds) {
		return qstrings[ds.qstringIgnore];
	};

	$scope.getRelativeTime = dateUtils.getRelativeTime;

	$scope.cloneDsAssignments = function() {
		alert('not hooked up yet: cloneDsAssignments');
	};

	$scope.addDeliveryService = function() {
		alert('not hooked up yet: addDeliveryService to server');
	};

	$scope.removeDeliveryService = function(dsId, serverId) {
		deliveryServiceService.deleteDeliveryServiceServer(dsId, serverId)
			.then(
				function() {
					$scope.refresh();
				}
			);
	};

	$scope.refresh = function() {
		$state.reload(); // reloads all the resolves for the view
	};

	$scope.navigateToPath = locationUtils.navigateToPath;

	angular.element(document).ready(function () {
		$('#deliveryServicesTable').dataTable({
			"aLengthMenu": [[25, 50, 100, -1], [25, 50, 100, "All"]],
			"iDisplayLength": 25,
			"aaSorting": []
		});
	});

};

TableServerDeliveryServicesController.$inject = ['server', 'serverDeliveryServices', '$scope', '$state', 'dateUtils', 'deliveryServiceUtils', 'locationUtils', 'deliveryServiceService'];
module.exports = TableServerDeliveryServicesController;
