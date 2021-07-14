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

var FormDeliveryServiceTargetController = function(deliveryService, currentTargets, target, $scope, formUtils, locationUtils, deliveryServiceService, typeService) {

	var getDeliveryServices = function() {
		deliveryServiceService.getDeliveryServices({ cdn: deliveryService.cdnId })
			.then(function(result) {
				$scope.deliveryServices = result.filter( function(ds)  {
					return ds.type.startsWith('HTTP') && currentTargets.find( function(t)  {return t.hasOwnProperty("targetId") && t.targetId === ds.id;}) === undefined;
				});
			});
	};

	var getTypes = function() {
		typeService.getTypes({ useInTable: 'steering_target' })
			.then(function(result) {
				$scope.types = result;
			});
	};

	$scope.deliveryService = deliveryService;

	$scope.targetId = angular.copy(target.targetId);

	$scope.target = target;

	$scope.navigateToPath = locationUtils.navigateToPath;

	$scope.hasError = formUtils.hasError;

	$scope.hasPropertyError = formUtils.hasPropertyError;

	var init = function () {
		getDeliveryServices();
		getTypes();
	};
	init();

};

FormDeliveryServiceTargetController.$inject = ['deliveryService', 'currentTargets', 'target', '$scope', 'formUtils', 'locationUtils', 'deliveryServiceService', 'typeService'];
module.exports = FormDeliveryServiceTargetController;
