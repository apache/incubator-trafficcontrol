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

var DialogSelectController = function(params, collection, $scope, $uibModalInstance) {

	$scope.params = params;

	$scope.collection = collection;

	$scope.selectedItemKeyValue = null;

	$scope.select = function() {
		const selectedItemId = parseInt($scope.selectedItemId);
		const selectedItem = collection.find(function(item)  { return parseInt(item.id) === selectedItemId });
		$uibModalInstance.close(selectedItem);
	};

	$scope.cancel = function () {
		$uibModalInstance.dismiss('cancel');
	};

	var init = function() {
		if ($scope.params.label) {
			$scope.label = function(item) { return item[$scope.params.label]; }
		} else if ($scope.params.labelFunction) {
			$scope.label = $scope.params.labelFunction;
		} else {
			$scope.label = function(item) { return item['name']; }
		}

		$scope.key = $scope.params.key || 'id';

		$scope.required = ($scope.params.required !== undefined) ? $scope.params.required : true;

		$scope.selectedItemKeyValue = $scope.params.selectedItemKeyValue || null;
	};
	init();

};

DialogSelectController.$inject = ['params', 'collection', '$scope', '$uibModalInstance'];
module.exports = DialogSelectController;
