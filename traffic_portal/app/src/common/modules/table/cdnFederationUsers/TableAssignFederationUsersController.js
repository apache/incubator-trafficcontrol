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

var TableAssignFederationUsersController = function(federation, users, assignedUsers, $scope, $uibModalInstance) {

	var selectedUsers = [];

	var addAll = function() {
		markVisibleUsers(true);
	};

	var removeAll = function() {
		markVisibleUsers(false);
	};

	var markVisibleUsers = function(selected) {
		var visibleUserIds = $('#assignFederationUsersTable tr.user-row').map(
			function() {
				return parseInt($(this).attr('id'));
			}).get();
		$scope.users = users.map(
			function(user)  {
				if (visibleUserIds.includes(user.id)) {
					user['selected'] = selected;
				}
				return user;
			}
		);
		updateSelectedCount();
	};

	var updateSelectedCount = function() {
		selectedUsers = $scope.users.filter(function(user)  { return user['selected'] === true; } );
		$('div.selected-count').html('<b>' + selectedUsers.length + ' selected</b>');
	};

	$scope.federation = federation;

	$scope.users = users.map(
		function(user)  {
			const isAssigned = assignedUsers.find(function(assignedUser)  { return assignedUser.id === user.id });
			if (isAssigned) {
				user['selected'] = true;
			}
			return user;
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
		var selectedUserIds = selectedUsers.map(function(u) {return u.id;});
		$uibModalInstance.close(selectedUserIds);
	};

	$scope.cancel = function () {
		$uibModalInstance.dismiss('cancel');
	};

	angular.element(document).ready(function () {
		var assignFederationUsersTable = $('#assignFederationUsersTable').dataTable({
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
		assignFederationUsersTable.on( 'search.dt', function () {
			$("#selectAllCB").removeAttr("checked"); // uncheck the all box when filtering
		} );
		updateSelectedCount();
	});

};

TableAssignFederationUsersController.$inject = ['federation', 'users', 'assignedUsers', '$scope', '$uibModalInstance'];
module.exports = TableAssignFederationUsersController;
