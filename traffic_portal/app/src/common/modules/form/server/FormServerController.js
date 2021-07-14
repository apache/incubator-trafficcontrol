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

var FormServerController = function(server, $scope, $location, $state, $uibModal, formUtils, locationUtils, serverUtils, serverService, cacheGroupService, cdnService, physLocationService, profileService, typeService, messageModel, propertiesModel) {

    $scope.IPPattern = serverUtils.IPPattern;
    $scope.IPWithCIDRPattern = serverUtils.IPWithCIDRPattern;
    $scope.IPv4Pattern = serverUtils.IPv4Pattern;

    var getPhysLocations = function() {
        physLocationService.getPhysLocations()
            .then(function(result) {
                $scope.physLocations = result;
            });
    };

    var getCacheGroups = function() {
        cacheGroupService.getCacheGroups({ orderby: 'name' })
            .then(function(result) {
                $scope.cacheGroups = result;
            });
    };

    var getTypes = function() {
        typeService.getTypes({ useInTable: 'server' })
            .then(function(result) {
                $scope.types = result;
            });
    };

    var getCDNs = function() {
        cdnService.getCDNs(true)
            .then(function(result) {
                $scope.cdns = result;
            });
    };

    var getProfiles = function(cdnId) {
        profileService.getProfiles({ orderby: 'name', cdn: cdnId })
            .then(function(result) {
                $scope.profiles = result.filter(
                    function(profile)  {
                        return profile.type !== 'DS_PROFILE'; // DS profiles are not intended for servers
                    }
                );
            });
    };

    var updateStatus = function(status) {
        serverService.updateStatus(server.id, { status: status.id, offlineReason: status.offlineReason })
            .then(
                function(result) {
                    messageModel.setMessages(result.data.alerts, false);
                    $scope.refresh();
                },
	            function(fault) {
		            messageModel.setMessages(fault.data.alerts, false);
	            }
            );
    };

    $scope.refresh = function() {
        $state.reload(); // reloads all the resolves for the view
    };

    $scope.server = server;

    $scope.falseTrue = [
        { value: true, label: 'true' },
        { value: false, label: 'false' }
    ];

    $scope.isCache = serverUtils.isCache;

    $scope.isEdge = serverUtils.isEdge;

    $scope.isOrigin = serverUtils.isOrigin;

    $scope.openCharts = serverUtils.openCharts;

    $scope.showChartsButton = propertiesModel.properties.servers.charts.show;

    $scope.addIP = function(interface) {
        $scope.serverForm.$setDirty();
        const newIP = {
            address: "",
            gateway: null,
            serviceAddress: false
        };

        if (!interface.ipAddresses) {
            interface.ipAddresses = [newIP];
        } else {
            interface.ipAddresses.push(newIP);
        }
    }

    $scope.deleteIP = function(interface, ip) {
        $scope.serverForm.$setDirty();
        interface.ipAddresses.splice(interface.ipAddresses.indexOf(ip), 1);
    }

    $scope.addInterface = function() {
        $scope.serverForm.$setDirty();
        const newInf = {
            mtu: 1500,
            maxBandwidth: null,
            monitor: false,
            ipAddresses: []
        };

        if (!$scope.server.interfaces) {
           $scope.server.interfaces = [newInf];
        } else {
           $scope.server.interfaces.push(newInf);
        }
    }

    $scope.deleteInterface = function(interface) {
        $scope.serverForm.$setDirty();
        $scope.server.interfaces.splice($scope.server.interfaces.indexOf(interface, 1));
    }

    $scope.onCDNChange = function() {
        $scope.server.profileId = null; // the cdn of the server changed, so we need to blank out the selected server profile (if any)
        getProfiles($scope.server.cdnId); // and get a new list of profiles (for the selected cdn)
    };

    $scope.isLargeCIDR = function(address) {
        const matches = /^(.+)\/(\d+)$/.exec(address);
        if (matches && matches.length === 3) {
            const ip = matches[1];
            const cidr = parseInt(matches[2], 10);
            if ($scope.IPv4Pattern.test(ip)) {
                if (cidr < 24) {
                    return true;
                }
            } else {
                if (cidr < 64) {
                    return true;
                }
            }
        }
        return false;
    };

    $scope.queueServerUpdates = function(server) {
        serverService.queueServerUpdates(server.id)
            .then(
                function() {
                    $scope.refresh();
                }
            );
    };

    $scope.clearServerUpdates = function(server) {
        serverService.clearServerUpdates(server.id)
            .then(
                function() {
                    $scope.refresh();
                }
            );
    };

    $scope.confirmStatusUpdate = function() {
        var modalInstance = $uibModal.open({
            templateUrl: 'common/modules/dialog/select/status/dialog.select.status.tpl.html',
            controller: 'DialogSelectStatusController',
            size: 'md',
            resolve: {
                server: function() {
                    return server;
                },
                statuses: function() {
                    return $scope.statuses;
                }
            }
        });
        modalInstance.result.then(function(status) {
            updateStatus(status);
        }, function () {
            // do nothing
        });
    };

    $scope.viewCapabilities = function() {
        $location.path($location.path() + '/capabilities');
    };

    $scope.viewDeliveryServices = function() {
        $location.path($location.path() + '/delivery-services');
    };

    $scope.navigateToPath = locationUtils.navigateToPath;

    $scope.hasError = formUtils.hasError;

    $scope.hasPropertyError = formUtils.hasPropertyError;

    var init = function () {
        getPhysLocations();
        getCacheGroups();
        getTypes();
        getCDNs();
        getProfiles(($scope.server.cdnId) ? $scope.server.cdnId : 0); // hacky but does the job. only when a cdn is selected can we fetch the appropriate profiles. otherwise, show no profiles.
    };
    init();

};

FormServerController.$inject = ['server', '$scope', '$location', '$state', '$uibModal', 'formUtils', 'locationUtils', 'serverUtils', 'serverService', 'cacheGroupService', 'cdnService', 'physLocationService', 'profileService', 'typeService', 'messageModel', 'propertiesModel'];
module.exports = FormServerController;
