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

var TableCapabilityUsersController = function(capability, capUsers, $controller, $scope, $state, dateUtils, locationUtils) {

	// extends the TableUsersController to inherit common methods
	angular.extend(this, $controller('TableUsersController', { users: capUsers, $scope: $scope }));

	let capUsersTable;

	$scope.capability = capability[0];

	$scope.relativeLoginTime = dateUtils.relativeLoginTime;

	$scope.editUser = function(id) {
		locationUtils.navigateToPath('/users/' + id);
	};

	$scope.refresh = function() {
		$state.reload(); // reloads all the resolves for the view
	};

	$scope.toggleVisibility = function(colName) {
		const col = capUsersTable.column(colName + ':name');
		col.visible(!col.visible());
		capUsersTable.rows().invalidate().draw();
	};

	$scope.navigateToPath = locationUtils.navigateToPath;

	angular.element(document).ready(function () {
		capUsersTable = $('#capUsersTable').DataTable({
			"aLengthMenu": [[25, 50, 100, -1], [25, 50, 100, "All"]],
			"iDisplayLength": 25,
			"aaSorting": [],
			"columns": $scope.columns,
			"initComplete": function() {
				try {
					// need to create the show/hide column checkboxes and bind to the current visibility
					$scope.columns = JSON.parse(localStorage.getItem('DataTables_capUsersTable_/')).columns;
				} catch (e) {
					console.error("Failure to retrieve required column info from localStorage (key=DataTables_capUsersTable_/):", e);
				}
			}
		});
	});

};

TableCapabilityUsersController.$inject = ['capability', 'capUsers', '$controller', '$scope', '$state', 'dateUtils', 'locationUtils'];
module.exports = TableCapabilityUsersController;
