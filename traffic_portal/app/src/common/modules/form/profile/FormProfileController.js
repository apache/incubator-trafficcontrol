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

var FormProfileController = function(profile, $scope, $location, $uibModal, fileUtils, formUtils, locationUtils, cdnService, profileService) {

    var getCDNs = function() {
        cdnService.getCDNs(true)
            .then(function(result) {
                $scope.cdns = result;
            });
    };

    $scope.profile = profile;

    $scope.types = [
        { value: 'ATS_PROFILE' },
        { value: 'TR_PROFILE' },
        { value: 'TM_PROFILE' },
        { value: 'TS_PROFILE' },
        { value: 'TP_PROFILE' },
        { value: 'INFLUXDB_PROFILE' },
        { value: 'RIAK_PROFILE' },
        { value: 'SPLUNK_PROFILE' },
        { value: 'DS_PROFILE' },
        { value: 'ORG_PROFILE' },
        { value: 'KAFKA_PROFILE' },
        { value: 'LOGSTASH_PROFILE' },
        { value: 'ES_PROFILE' },
        { value: 'UNK_PROFILE' },
        { value: 'GROVE_PROFILE' }
    ];

    $scope.falseTrue = [
        { value: true, label: 'true' },
        { value: false, label: 'false' }
    ];

    $scope.viewParams = function() {
        $location.path($location.path() + '/parameters');
    };

    $scope.viewServers = function() {
        $location.path($location.path() + '/servers');
    };

    $scope.viewDeliveryServices = function() {
        $location.path($location.path() + '/delivery-services');
    };

    $scope.cloneProfile = function() {
        var params = {
            title: 'Clone Profile',
            message: "Your are about to clone the " + profile.name + " profile. Your clone will have the same attributes and parameter assignments as the " + profile.name + " profile.<br><br>Please enter a name for your cloned profile."
        };
        var modalInstance = $uibModal.open({
            templateUrl: 'common/modules/dialog/input/dialog.input.tpl.html',
            controller: 'DialogInputController',
            size: 'md',
            resolve: {
                params: function () {
                    return params;
                }
            }
        });
        modalInstance.result.then(function(clonedProfileName) {
            profileService.cloneProfile(profile.name, clonedProfileName);
        }, function () {
            // do nothing
        });
    };

    $scope.exportProfile = function(profile) {
        profileService.exportProfile(profile.id).
            then(
                function(result) {
                    fileUtils.exportJSON(result, profile.name, 'traffic_ops');
                }
        );

    };

    $scope.queueUpdatesByProfile = function() {
        profileService.queueServerUpdatesByProfile($scope.profile.cdn, $scope.profile.name).then($scope.refresh);
    };

    $scope.clearUpdatesByProfile = function() {
        profileService.clearServerUpdatesByProfile($scope.profile.cdn, $scope.profile.name).then($scope.refresh);
    };

    $scope.navigateToPath = locationUtils.navigateToPath;

    $scope.hasError = formUtils.hasError;

    $scope.hasPropertyError = formUtils.hasPropertyError;

    var init = function () {
        getCDNs();
    };
    init();

};

FormProfileController.$inject = ['profile', '$scope', '$location', '$uibModal', 'fileUtils', 'formUtils', 'locationUtils', 'cdnService', 'profileService'];
module.exports = FormProfileController;
