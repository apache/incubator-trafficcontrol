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

var FormDeliveryServiceController = function(deliveryService, dsCurrent, origin, topologies, type, types, $scope, $location, $uibModal, $window, formUtils, locationUtils, tenantUtils, deliveryServiceUtils, cdnService, profileService, tenantService, propertiesModel, userModel, serviceCategoryService) {

    var getCDNs = function() {
        cdnService.getCDNs()
            .then(function(result) {
                $scope.cdns = result;
            });
    };

    var getProfiles = function() {
        profileService.getProfiles({ orderby: 'name' })
            .then(function(result) {
                $scope.profiles = result.filter(
                    function(profile)  {
                        return profile.type === 'DS_PROFILE';
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

    var getServiceCategories = function() {
        serviceCategoryService.getServiceCategories({dsId: deliveryService.id })
            .then(function(result) {
                $scope.serviceCategories = result;
            });
    };

    $scope.deliveryService = deliveryService;

    $scope.showGeneralConfig = true;

    $scope.showCacheConfig = true;

    $scope.showRoutingConfig = true;

    $scope.dsCurrent = dsCurrent; // this ds is used primarily for showing the diff between a ds request and the current DS

    $scope.origin = origin[0];

    $scope.topologies = topologies;

    $scope.showChartsButton = propertiesModel.properties.deliveryServices.charts.customLink.show;

    $scope.openCharts = deliveryServiceUtils.openCharts;

    $scope.dsRequestsEnabled = propertiesModel.properties.dsRequests.enabled;

    $scope.edgeFQDNs = function(ds) {
        const urls = new Array();
        if (ds.exampleURLs instanceof Array && ds.exampleURLs.length > 0) {
            for (var i = 0; i < ds.exampleURLs.length; ++i) {
                urls.push(ds.exampleURLs[i]);
            }
        }
        return urls.join('\n');
    };

    $scope.DRAFT = 0;
    $scope.SUBMITTED = 1;
    $scope.REJECTED = 2;
    $scope.PENDING = 3;
    $scope.COMPLETE = 4;

    $scope.saveable = function() {
        // this may be overriden in a child class. i.e. FormEditDeliveryServiceController
        return true;
    };

    $scope.deletable = function() {
        // this may be overriden in a child class. i.e. FormEditDeliveryServiceController
        return true;
    };

    $scope.types = types.filter( function(currentType)  {
        let category;
        if (type.indexOf('ANY_MAP') !== -1) {
            category = 'ANY_MAP';
        } else if (type.indexOf('DNS') !== -1) {
            category = 'DNS';
        } else if (type.indexOf('HTTP') !== -1) {
            category = 'HTTP';
        } else if (type.indexOf('STEERING') !== -1) {
            category = 'STEERING';
        }
        return currentType.name.indexOf(category) !== -1;
    });

    $scope.clientSteeringType = types.find( function(t)  {return t.hasOwnProperty("name") && t.name === "CLIENT_STEERING";});
    $scope.isClientSteering = function(ds) {
        if (ds.typeId == $scope.clientSteeringType.id) {
            return true;
        } else {
            ds.trResponseHeaders = "";
            return false;
        }
    };

    $scope.falseTrue = [
        { value: true, label: 'true' },
        { value: false, label: 'false' }
    ];

    $scope.activeInactive = [
        { value: true, label: 'Active' },
        { value: false, label: 'Not Active'}
    ];

    $scope.signingAlgos = [
        { value: null, label: 'None' },
        { value: 'url_sig', label: 'URL Signature Keys' },
        { value: 'uri_signing', label: 'URI Signing Keys' }
    ];

    $scope.protocols = [
        { value: 0, label: 'HTTP' },
        { value: 1, label: 'HTTPS' },
        { value: 2, label: 'HTTP AND HTTPS' },
        { value: 3, label: 'HTTP TO HTTPS' }
    ];

    $scope.qStrings = [
        { value: 0, label: 'Use query parameter strings in cache key and pass in upstream requests' },
        { value: 1, label: 'Do not use query parameter strings in cache key, but do pass in upstream requests' },
        { value: 2, label: 'Neither use query parameter strings in cache key, nor pass in upstream requests' }
    ];

    $scope.geoLimits = [
        { value: 0, label: 'None' },
        { value: 1, label: 'Coverage Zone File only' },
        { value: 2, label: 'Coverage Zone File and Country Code(s)' }
    ];

    $scope.geoProviders = [
        { value: 0, label: 'Maxmind' },
        { value: 1, label: 'Neustar' }
    ];

    $scope.dscps = [
        { value: 0, label: '0 - Best Effort' },
        { value: 10, label: '10 - AF11' },
        { value: 12, label: '12 - AF12' },
        { value: 14, label: '14 - AF13' },
        { value: 18, label: '18 - AF21' },
        { value: 20, label: '20 - AF22' },
        { value: 22, label: '22 - AF23' },
        { value: 26, label: '26 - AF31' },
        { value: 28, label: '28 - AF32' },
        { value: 30, label: '30 - AF33' },
        { value: 34, label: '34 - AF41' },
        { value: 36, label: '36 - AF42' },
        { value: 37, label: '37 - ' },
        { value: 38, label: '38 - AF43' },
        { value: 8, label: '8 - CS1' },
        { value: 16, label: '16 - CS2' },
        { value: 24, label: '24 - CS3' },
        { value: 32, label: '32 - CS4' },
        { value: 40, label: '40 - CS5' },
        { value: 48, label: '48 - CS6' },
        { value: 56, label: '56 - CS7' }
    ];

    $scope.deepCachingTypes = [
        { value: 'NEVER', label: 'NEVER' },
        { value: 'ALWAYS', label: 'ALWAYS' }
    ];

    $scope.dispersions = [
        { value: 1, label: '1 - OFF' },
        { value: 2, label: '2' },
        { value: 3, label: '3' },
        { value: 4, label: '4' },
        { value: 5, label: '5' },
        { value: 6, label: '6' },
        { value: 7, label: '7' },
        { value: 8, label: '8' },
        { value: 9, label: '9' },
        { value: 10, label: '10' }
    ];

    $scope.rrhs = [
        { value: 0, label: "Don't cache Range Requests" },
        { value: 1, label: "Use the background_fetch plugin" },
        { value: 2, label: "Use the cache_range_requests plugin" },
        { value: 3, label: "Use the slice plugin" }
    ];

    $scope.msoAlgos = [
        { value: 0, label: "0 - Consistent Hash" },
        { value: 1, label: "1 - Primary/Backup" },
        { value: 2, label: "2 - Strict Round Robin" },
        { value: 3, label: "3 - IP-based Round Robin" },
        { value: 4, label: "4 - Latch on Failover" }
    ];

    $scope.tenantLabel = function(tenant) {
        return '-'.repeat(tenant.level) + ' ' + tenant.name;
    };

    $scope.clone = function(ds) {
        locationUtils.navigateToPath('/delivery-services/' + ds.id + '/clone?type=' + ds.type);
    };

    $scope.changeSigningAlgorithm = function(signingAlgorithm) {
        if (signingAlgorithm == null) {
            deliveryService.signed = false;
        } else {
            deliveryService.signed = true;
        }
    };

    $scope.encodeRegex = function(consistentHashRegex) {
        if (consistentHashRegex != undefined) {
            $scope.encodedRegex = encodeURIComponent(consistentHashRegex);
        } else {
            scope.encodedRegex = "";
        }
    };

    $scope.addQueryParam = function() {
        $scope.deliveryService.consistentHashQueryParams.push('');
    };

    $scope.removeQueryParam = function(index) {
        if ($scope.deliveryService.consistentHashQueryParams.length > 1) {
            $scope.deliveryService.consistentHashQueryParams.splice(index, 1);
        } else {
            // if only one query param is left, don't remove the item from the array. instead, just blank it out
            // so the dynamic form widget will still be visible. empty strings get stripped out on save anyhow.
            $scope.deliveryService.consistentHashQueryParams[index] = '';
        }
        $scope.deliveryServiceForm.$pristine = false; // this enables the 'update' button in the ds form
    };

    $scope.viewTargets = function() {
        $location.path($location.path() + '/targets');
    };

    $scope.viewCapabilities = function() {
        $location.path($location.path() + '/required-server-capabilities');
    };

    $scope.viewOrigins = function() {
        $location.path($location.path() + '/origins');
    };

    $scope.viewServers = function() {
        $location.path($location.path() + '/servers');
    };

    $scope.viewRegexes = function() {
        $location.path($location.path() + '/regexes');
    };

    $scope.viewJobs = function() {
        $location.path($location.path() + '/jobs');
    };

    $scope.manageSslKeys = function() {
        $location.path($location.path() + '/ssl-keys');
    };

    $scope.manageUrlSigKeys = function() {
        $location.path($location.path() + '/url-sig-keys');
    };

    $scope.manageUriSigningKeys = function() {
        $location.path($location.path() + '/uri-signing-keys');
    };

    $scope.viewStaticDnsEntries = function() {
        $location.path($location.path() + '/static-dns-entries');
    };

    $scope.viewCharts = function() {
        $location.path($location.path() + '/charts');
    };

    $scope.navigateToPath = locationUtils.navigateToPath;

    $scope.hasError = formUtils.hasError;

    $scope.hasPropertyError = formUtils.hasPropertyError;

    $scope.rangeRequestSelected = function() {
        if ($scope.deliveryService.rangeRequestHandling != 3) {
            $scope.deliveryService.rangeSliceBlockSize = null;
        }
    };

    var init = function () {
        getCDNs();
        getProfiles();
        getTenants();
        getServiceCategories();
        if (!deliveryService.consistentHashQueryParams || deliveryService.consistentHashQueryParams.length < 1) {
            // add an empty one so the dynamic form widget is visible. empty strings get stripped out on save anyhow.
            $scope.deliveryService.consistentHashQueryParams = [ '' ];
        }
    };
    init();

};

FormDeliveryServiceController.$inject = ['deliveryService', 'dsCurrent', 'origin', 'topologies', 'type', 'types', '$scope', '$location', '$uibModal', '$window', 'formUtils', 'locationUtils', 'tenantUtils', 'deliveryServiceUtils', 'cdnService', 'profileService', 'tenantService', 'propertiesModel', 'userModel', 'serviceCategoryService'];
module.exports = FormDeliveryServiceController;
