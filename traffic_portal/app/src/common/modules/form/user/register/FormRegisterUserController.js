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

var FormRegisterUserController = function($scope, $location, formUtils, tenantUtils, locationUtils, roleService, tenantService, userService, userModel) {

	var getRoles = function() {
		roleService.getRoles()
			.then(function(result) {
				$scope.roles = result.sort(
					function(r1, r2)  {
						if (r1.privLevel < r2.privLevel) {
							return 1;
						} else if (r1.privLevel > r2.privLevel) {
							return -1;
						}
						return 0;
					}
				);
			});
	};

	var getTenants = function() {
		tenantService.getTenant(userModel.user.tenantId)
			.then(function(tenant) {
				tenantService.getTenants()
					.then(function(tenants) {
						$scope.tenants = tenantUtils.hierarchySort(tenantUtils.groupTenantsByParent(tenants), tenant.parentId, []);
						tenantUtils.addLevels($scope.tenants);
					});
			});
	};

	$scope.registration = {};

	$scope.register = function(registration) {
		userService.registerUser(registration);
	};

	$scope.tenantLabel = function(tenant) {
		return '-'.repeat(tenant.level) + ' ' + tenant.name;
	};

	$scope.roleLabel = function(role) {
		return role.name + ' (' + role.privLevel + ')';
	};

	$scope.navigateToPath = locationUtils.navigateToPath;

	$scope.hasError = formUtils.hasError;

	$scope.hasPropertyError = formUtils.hasPropertyError;

	var init = function () {
		getRoles();
		getTenants();
	};
	init();

};

FormRegisterUserController.$inject = ['$scope', '$location', 'formUtils', 'tenantUtils', 'locationUtils', 'roleService', 'tenantService', 'userService', 'userModel'];
module.exports = FormRegisterUserController;
