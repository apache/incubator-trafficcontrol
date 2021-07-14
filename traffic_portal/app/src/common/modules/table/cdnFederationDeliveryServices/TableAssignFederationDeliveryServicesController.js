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

var TableAssignFederationDeliveryServicesController = function(federation, deliveryServices, assignedDeliveryServices, $scope, $uibModalInstance) {

	var selectedDeliveryServices = [];

	var addAll = function() {
		markVisibleDeliveryServices(true);
	};

	var removeAll = function() {
		markVisibleDeliveryServices(false);
	};

	var markVisibleDeliveryServices = function(selected) {
		var visibleDsIds = $('#assignFederationDSTable tr.ds-row').map(
			function() {
				return parseInt($(this).attr('id'));
			}).get();
		$scope.deliveryServices = deliveryServices.map(
			function(ds)  {
				if (visibleDsIds.includes(ds.id)) {
					ds['selected'] = selected;
				}
				return ds;
			}
		);
		updateSelectedCount();
	};

	var updateSelectedCount = function() {
		selectedDeliveryServices = $scope.deliveryServices.filter(function(ds)  { return ds['selected'] === true; } );
		$('div.selected-count').html('<b>' + selectedDeliveryServices.length + ' selected</b>');
	};

	$scope.federation = federation;

	$scope.deliveryServices = deliveryServices.map(
		function(ds)  {
			const isAssigned = assignedDeliveryServices.find(function(assignedDS)  { return assignedDS.id === ds.id });
			if (isAssigned) {
				ds['selected'] = true;
			}
			return ds;
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
		const selectedDsIds = selectedDeliveryServices.map(function(d) {return d.id;});
		$uibModalInstance.close(selectedDsIds);
	};

	$scope.cancel = function () {
		$uibModalInstance.dismiss('cancel');
	};

	angular.element(document).ready(function () {
		var assignFederationDSTable = $('#assignFederationDSTable').dataTable({
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
		assignFederationDSTable.on( 'search.dt', function () {
			$("#selectAllCB").removeAttr("checked"); // uncheck the all box when filtering
		} );
		updateSelectedCount();
	});

};

TableAssignFederationDeliveryServicesController.$inject = ['federation', 'deliveryServices', 'assignedDeliveryServices', '$scope', '$uibModalInstance'];
module.exports = TableAssignFederationDeliveryServicesController;
