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

var FormOriginController = function(origin, $scope, $window, $location, formUtils, locationUtils, tenantUtils, deliveryServiceService, profileService, tenantService, coordinateService, cacheGroupService, userModel) {

    var getProfiles = function() {
        profileService.getProfiles({ orderby: 'name' })
            .then(function(result) {
                $scope.profiles = result.filter( function(profile)  {
                    return profile.type === 'ORG_PROFILE';
                });
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

    var getCacheGroups = function() {
        cacheGroupService.getCacheGroups({ orderby: 'name' })
            .then(function(result) {
                $scope.cacheGroups = result.filter( function(cachegroup)  {
                    return cachegroup.typeName === 'ORG_LOC';
                });
            });
    };

    var getCoordinates = function() {
        coordinateService.getCoordinates({ orderby: 'name' })
            .then(function(result) {
                $scope.coordinates = result;
            });
    };

    var getDeliveryServices = function() {
        deliveryServiceService.getDeliveryServices()
            .then(function(result) {
                $scope.deliveryServices = result.sort(
                    function(d1, d2)  {
                        if (d1.xmlId < d2.xmlId) {
                            return -1;
                        } else if (d1.xmlId > d2.xmlId) {
                            return 1;
                        }
                        return 0;
                    }
                )
            }
        );
    };

    $scope.origin = origin;

    $scope.protocols = [
        { value: 'http', label: 'http' },
        { value: 'https', label: 'https' }
    ];

    $scope.tenantLabel = function(tenant) {
        return '-'.repeat(tenant.level) + ' ' + tenant.name;
    };

    $scope.nullifyIfEmptyIP = function(origin) {
        origin.ipAddress = origin.ipAddress == '' ? null : origin.ipAddress;
        origin.ip6Address = origin.ip6Address == '' ? null : origin.ip6Address;
    }

    $scope.navigateToPath = locationUtils.navigateToPath;

    $scope.editDeliveryService = function(deliveryServiceId) {
        ds = $scope.deliveryServices.find( function(d)  {return d.hasOwnProperty("id") && d.id === deliveryServiceId;});
        $window.open('/#!/delivery-services/' + ds.id + '?type=' + ds.type, '_blank');
    };

    $scope.hasError = formUtils.hasError;

    $scope.hasPropertyError = formUtils.hasPropertyError;

    var init = function () {
        getProfiles();
        getTenants();
        getCacheGroups();
        getCoordinates();
        getDeliveryServices();
    };
    init();

};

FormOriginController.$inject = ['origin', '$scope', '$window', '$location', 'formUtils', 'locationUtils', 'tenantUtils', 'deliveryServiceService', 'profileService', 'tenantService', 'coordinateService', 'cacheGroupService', 'userModel'];
module.exports = FormOriginController;
