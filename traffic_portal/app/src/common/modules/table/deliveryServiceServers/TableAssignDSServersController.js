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

var TableAssignDSServersController = function(deliveryService, servers, assignedServers, $scope, $uibModalInstance) {

	var selectedServers = [];

	var addAll = function() {
		markVisibleServers(true);
	};

	var removeAll = function() {
		markVisibleServers(false);
	};

	var markVisibleServers = function(selected) {
		var visibleServerIds = $('#dsServersUnassignedTable tr.server-row').map(
			function() {
				return parseInt($(this).attr('id'));
			}).get();
		$scope.servers = servers.map(
			function(server)  {
				if (visibleServerIds.includes(server.id)) {
					server['selected'] = selected;
				}
				return server;
			}
		);
		updateSelectedCount();
	};

	var updateSelectedCount = function() {
		selectedServers = $scope.servers.filter(function(server)  { return server['selected'] === true; } );
		$('div.selected-count').html('<b>' + selectedServers.length + ' servers selected</b>');
	};

	$scope.deliveryService = deliveryService;

	$scope.servers = servers.map(
		function(server)  {
			const isAssigned = assignedServers.find(function(assignedServer)  { return assignedServer.id === server.id });
			if (isAssigned) {
				server['selected'] = true;
			}
			return server;
		}
	);

	$scope.selectAll = function($event) {
		var checkbox = $event.target;
		if (checkbox.checked) {
			addAll();
		} else {
			removeAll();
		}
	};

	$scope.onChange = function() {
		updateSelectedCount();
	};

	$scope.submit = function() {
		var selectedServerIds = selectedServers.map(function(s) {return s.id;});
		$uibModalInstance.close(selectedServerIds);
	};

	$scope.cancel = function () {
		$uibModalInstance.dismiss('cancel');
	};

	angular.element(document).ready(function () {
		var dsServersUnassignedTable = $('#dsServersUnassignedTable').dataTable({
			"scrollY": "60vh",
			"paging": false,
			"order": [[ 1, 'asc' ]],
			"dom": '<"selected-count">frtip',
			"columnDefs": [
				{ 'orderable': false, 'targets': 0 },
				{ "width": "5%", "targets": 0 }
			],
			"stateSave": false
		});
		dsServersUnassignedTable.on( 'search.dt', function () {
			$("#selectAllCB").removeAttr("checked"); // uncheck the all box when filtering
		} );
		updateSelectedCount();
	});

};

TableAssignDSServersController.$inject = ['deliveryService', 'servers', 'assignedServers', '$scope', '$uibModalInstance'];
module.exports = TableAssignDSServersController;
