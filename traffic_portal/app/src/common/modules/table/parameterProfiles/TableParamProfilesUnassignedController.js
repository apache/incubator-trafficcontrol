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

var TableParamProfilesUnassignedController = function(parameter, allProfiles, assignedProfiles, $scope, $uibModalInstance) {

	var selectedProfiles = [];

	var addAll = function() {
		markVisibleProfiles(true);
	};

	var removeAll = function() {
		markVisibleProfiles(false);
	};

	var markVisibleProfiles = function(selected) {
		var visibleProfileIds = $('#assignProfilesTable tr.profile-row').map(
			function() {
				return parseInt($(this).attr('id'));
			}).get();
		$scope.selectedProfiles = allProfiles.map(
			function(profile)  {
				if (visibleProfileIds.includes(profile.id)) {
					profile['selected'] = selected;
				}
				return profile;
			}
		);
		updateSelectedCount();
	};

	var updateSelectedCount = function() {
		selectedProfiles = $scope.selectedProfiles.filter(function(profile)  { return profile['selected'] === true; } );
		$('div.selected-count').html('<b>' + selectedProfiles.length + ' profiles selected</b>');
	};

	$scope.parameter = parameter;

	$scope.selectedProfiles = allProfiles.map(function(profile)  {
		const isAssigned = assignedProfiles.find(function(assignedProfile)  { return assignedProfile.id === profile.id });
		if (isAssigned) {
			profile['selected'] = true;
		}
		return profile;
	});

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
		var selectedProfileIds = selectedProfiles.map(function(p) {return p.id;});
		$uibModalInstance.close(selectedProfileIds);
	};

	$scope.cancel = function () {
		$uibModalInstance.dismiss('cancel');
	};

	angular.element(document).ready(function () {
		var assignProfilesTable = $('#assignProfilesTable').dataTable({
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
		assignProfilesTable.on( 'search.dt', function () {
			$("#selectAllCB").removeAttr("checked"); // uncheck the all box when filtering
		} );
		updateSelectedCount();
	});

};

TableParamProfilesUnassignedController.$inject = ['parameter', 'allProfiles', 'assignedProfiles', '$scope', '$uibModalInstance'];
module.exports = TableParamProfilesUnassignedController;
