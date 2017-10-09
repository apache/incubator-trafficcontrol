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

var FormUserController = function(user, $scope, $location, formUtils, stringUtils, locationUtils, tenantUtils, roleService, tenantService) {

    var getRoles = function() {
        roleService.getRoles({ orderby: 'priv_level DESC' })
            .then(function(result) {
                $scope.roles = result;
            });
    };

    var getTenants = function() {
        tenantService.getTenants()
            .then(function(result) {
                $scope.tenants = result;
                tenantUtils.addLevels($scope.tenants);
            });
    };

    $scope.user = user;

    $scope.label = function(role) {
        return role.name + ' (' + role.privLevel + ')';
    };

    $scope.tenantLabel = function(tenant) {
        return '-'.repeat(tenant.level) + ' ' + tenant.name;
    };

    $scope.labelize = stringUtils.labelize;

    $scope.viewDeliveryServices = function() {
        $location.path($location.path() + '/delivery-services');
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

FormUserController.$inject = ['user', '$scope', '$location', 'formUtils', 'stringUtils', 'locationUtils', 'tenantUtils', 'roleService', 'tenantService'];
module.exports = FormUserController;
