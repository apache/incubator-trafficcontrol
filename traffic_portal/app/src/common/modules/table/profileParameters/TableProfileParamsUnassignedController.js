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

var TableProfileParamsUnassignedController = function(profile, allParams, assignedParams, $scope, $uibModalInstance) {

	var selectedParams = [];

	var addAll = function() {
		markVisibleParams(true);
	};

	var removeAll = function() {
		markVisibleParams(false);
	};

	var markVisibleParams = function(selected) {
		var visibleParamIds = $('#assignParamsTable tr.param-row').map(
			function() {
				return parseInt($(this).attr('id'));
			}).get();
		$scope.selectedParams = allParams.map(
			function(param)  {
				if (visibleParamIds.includes(param.id)) {
					param['selected'] = selected;
				}
				return param;
			}
		);
		updateSelectedCount();
	};

	var updateSelectedCount = function() {
		selectedParams = $scope.selectedParams.filter(function(param)  { return param['selected'] === true; } );
		$('div.selected-count').html('<b>' + selectedParams.length + ' parameters selected</b>');
	};

	$scope.profile = profile;

	$scope.selectedParams = allParams.map(
		function(param)  {
			const isAssigned = assignedParams.find(function(assignedParam)  { return assignedParam.id === param.id });
			if (isAssigned) {
				param['selected'] = true;
			}
			return param;
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
		const selectedParamIds = selectedParams.map(function(p) {return p.id;});
		$uibModalInstance.close(selectedParamIds);
	};

	$scope.cancel = function () {
		$uibModalInstance.dismiss('cancel');
	};

	angular.element(document).ready(function () {
		var assignParamsTable = $('#assignParamsTable').dataTable({
			"scrollY": "60vh",
			"paging": false,
			"order": [[ 1, 'asc' ]],
			"dom": '<"selected-count">frtip',
			"columnDefs": [
				{ 'orderable': false, 'targets': 0 },
				{ "width": "5%", "targets": 0 },
				{ "width": "50%", "targets": 3 }
			],
			"stateSave": false
		});
		assignParamsTable.on( 'search.dt', function () {
			$("#selectAllCB").removeAttr("checked"); // uncheck the all box when filtering
		} );
		updateSelectedCount();
	});

};

TableProfileParamsUnassignedController.$inject = ['profile', 'allParams', 'assignedParams', '$scope', '$uibModalInstance'];
module.exports = TableProfileParamsUnassignedController;
