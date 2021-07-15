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
/** @typedef { import('../agGrid/CommonGridController').CGC } CGC */

var TableDeliveryServiceJobsController = function(deliveryService, jobs, $controller, $scope, $location) {

	// extends the TableJobsController to inherit common methods
	angular.extend(this, $controller('TableJobsController', { tableName: 'dsJobs', jobs: jobs, $scope: $scope }));

	$scope.deliveryService = deliveryService;

	/** @type CGC.TitleBreadCrumbs */
	$scope.breadCrumbs = [{
		href: "#!/delivery-services",
		text: "Delivery Services"
	},
	{
		getText: function () {
			return $scope.deliveryService.xmlId;
		},
		getHref: function () {
			return "#!/delivery-services/" + $scope.deliveryService.id + "?type=" + $scope.deliveryService.type;
		}
	},
	{
		text: "Invalidation Requests"
	}];

	$scope.createJob = function() {
		$location.path($location.path() + '/new');
	};

};

TableDeliveryServiceJobsController.$inject = ['deliveryService', 'jobs', '$controller', '$scope', '$location'];
module.exports = TableDeliveryServiceJobsController;
