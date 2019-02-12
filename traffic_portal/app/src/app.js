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

'use strict';
require('app-templates');

var App = function($urlRouterProvider) {
    $urlRouterProvider.otherwise('/');
};

App.$inject = ['$urlRouterProvider'];

var trafficPortal = angular.module('trafficPortal', [
        'config',
        'ngAnimate',
        'ngResource',
        'ngSanitize',
        'ngRoute',
        'ui.router',
        'ui.bootstrap',
        'restangular',
        'app.templates',
        'angular-jwt',
        'chart.js',
        'angular-loading-bar',
        'moment-picker',

        // public modules
        require('./modules/public').name,
        require('./modules/public/login').name,

        // private modules
        require('./modules/private').name,
        require('./modules/private/asns').name,
        require('./modules/private/asns/edit').name,
        require('./modules/private/asns/list').name,
        require('./modules/private/asns/new').name,
        require('./modules/private/cacheGroups').name,
        require('./modules/private/cacheGroups/edit').name,
        require('./modules/private/cacheGroups/list').name,
        require('./modules/private/cacheGroups/new').name,
        require('./modules/private/cacheGroups/asns').name,
        require('./modules/private/cacheGroups/parameters').name,
        require('./modules/private/cacheGroups/servers').name,
        require('./modules/private/cacheGroups/staticDnsEntries').name,
        require('./modules/private/cacheChecks').name,
        require('./modules/private/cacheStats').name,
        require('./modules/private/capabilities').name,
        require('./modules/private/capabilities/new').name, // this must be defined before edit for the url matcher to work
        require('./modules/private/capabilities/list').name,
        require('./modules/private/capabilities/edit').name,
        require('./modules/private/capabilities/endpoints').name,
        require('./modules/private/capabilities/users').name,
        require('./modules/private/cdns').name,
        require('./modules/private/cdns/config').name,
        require('./modules/private/cdns/deliveryServices').name,
        require('./modules/private/cdns/dnssecKeys').name,
        require('./modules/private/cdns/dnssecKeys/generate').name,
        require('./modules/private/cdns/dnssecKeys/regenerateKsk').name,
        require('./modules/private/cdns/dnssecKeys/view').name,
        require('./modules/private/cdns/edit').name,
        require('./modules/private/cdns/federations').name,
        require('./modules/private/cdns/federations/deliveryServices').name,
        require('./modules/private/cdns/federations/edit').name,
        require('./modules/private/cdns/federations/list').name,
        require('./modules/private/cdns/federations/new').name,
        require('./modules/private/cdns/federations/users').name,
        require('./modules/private/cdns/list').name,
        require('./modules/private/cdns/new').name,
        require('./modules/private/cdns/profiles').name,
        require('./modules/private/cdns/servers').name,
        require('./modules/private/changeLogs').name,
        require('./modules/private/changeLogs/list').name,
        require('./modules/private/coordinates').name,
        require('./modules/private/coordinates/edit').name,
        require('./modules/private/coordinates/list').name,
        require('./modules/private/coordinates/new').name,
        require('./modules/private/dashboard').name,
        require('./modules/private/dashboard/view').name,
        require('./modules/private/deliveryServiceRequests').name,
        require('./modules/private/deliveryServiceRequests/compare').name,
        require('./modules/private/deliveryServiceRequests/comments').name,
        require('./modules/private/deliveryServiceRequests/edit').name,
        require('./modules/private/deliveryServiceRequests/list').name,
        require('./modules/private/deliveryServices').name,
        require('./modules/private/deliveryServices/clone').name,
        require('./modules/private/deliveryServices/charts').name,
        require('./modules/private/deliveryServices/charts/view').name,
        require('./modules/private/deliveryServices/compare').name,
        require('./modules/private/deliveryServices/edit').name,
        require('./modules/private/deliveryServices/list').name,
        require('./modules/private/deliveryServices/new').name,
        require('./modules/private/deliveryServices/jobs').name,
        require('./modules/private/deliveryServices/jobs/list').name,
        require('./modules/private/deliveryServices/jobs/new').name,
        require('./modules/private/deliveryServices/origins').name,
        require('./modules/private/deliveryServices/regexes').name,
        require('./modules/private/deliveryServices/regexes/edit').name,
        require('./modules/private/deliveryServices/regexes/list').name,
        require('./modules/private/deliveryServices/regexes/new').name,
        require('./modules/private/deliveryServices/servers').name,
        require('./modules/private/deliveryServices/staticDnsEntries').name,
        require('./modules/private/deliveryServices/staticDnsEntries/edit').name,
        require('./modules/private/deliveryServices/staticDnsEntries/list').name,
        require('./modules/private/deliveryServices/staticDnsEntries/new').name,
        require('./modules/private/deliveryServices/targets').name,
        require('./modules/private/deliveryServices/targets/edit').name,
        require('./modules/private/deliveryServices/targets/list').name,
        require('./modules/private/deliveryServices/targets/new').name,
        require('./modules/private/deliveryServices/users').name,
        require('./modules/private/deliveryServices/urlSigKeys').name,
        require('./modules/private/deliveryServices/uriSigningKeys').name,
        require('./modules/private/deliveryServices/sslKeys').name,
        require('./modules/private/deliveryServices/sslKeys/view').name,
        require('./modules/private/deliveryServices/sslKeys/generate').name,
        require('./modules/private/divisions').name,
        require('./modules/private/divisions/edit').name,
        require('./modules/private/divisions/list').name,
        require('./modules/private/divisions/new').name,
        require('./modules/private/divisions/regions').name,
        require('./modules/private/endpoints').name,
        require('./modules/private/endpoints/edit').name,
        require('./modules/private/endpoints/list').name,
        require('./modules/private/endpoints/new').name,
        require('./modules/private/iso').name,
        require('./modules/private/jobs').name,
        require('./modules/private/jobs/list').name,
        require('./modules/private/jobs/new').name,
        require('./modules/private/origins').name,
        require('./modules/private/origins/edit').name,
        require('./modules/private/origins/list').name,
        require('./modules/private/origins/new').name,
        require('./modules/private/physLocations').name,
        require('./modules/private/physLocations/edit').name,
        require('./modules/private/physLocations/list').name,
        require('./modules/private/physLocations/new').name,
        require('./modules/private/physLocations/servers').name,
        require('./modules/private/parameters').name,
        require('./modules/private/parameters/cacheGroups').name,
        require('./modules/private/parameters/edit').name,
        require('./modules/private/parameters/list').name,
        require('./modules/private/parameters/new').name,
        require('./modules/private/parameters/profiles').name,
        require('./modules/private/profiles').name,
        require('./modules/private/profiles/compare').name,
        require('./modules/private/profiles/deliveryServices').name,
        require('./modules/private/profiles/edit').name,
        require('./modules/private/profiles/list').name,
        require('./modules/private/profiles/new').name,
        require('./modules/private/profiles/parameters').name,
        require('./modules/private/profiles/servers').name,
        require('./modules/private/regions').name,
        require('./modules/private/regions/edit').name,
        require('./modules/private/regions/list').name,
        require('./modules/private/regions/physLocations').name,
        require('./modules/private/regions/new').name,
        require('./modules/private/roles').name,
        require('./modules/private/roles/capabilities').name,
        require('./modules/private/roles/edit').name,
        require('./modules/private/roles/list').name,
        require('./modules/private/roles/new').name,
        require('./modules/private/roles/users').name,
        require('./modules/private/servers').name,
        require('./modules/private/servers/configFiles').name,
        require('./modules/private/servers/deliveryServices').name,
        require('./modules/private/servers/edit').name,
        require('./modules/private/servers/new').name,
        require('./modules/private/servers/list').name,
        require('./modules/private/statuses').name,
        require('./modules/private/statuses/edit').name,
        require('./modules/private/statuses/list').name,
        require('./modules/private/statuses/new').name,
        require('./modules/private/statuses/servers').name,
        require('./modules/private/tenants').name,
        require('./modules/private/tenants/deliveryServices').name,
        require('./modules/private/tenants/edit').name,
        require('./modules/private/tenants/list').name,
        require('./modules/private/tenants/new').name,
        require('./modules/private/tenants/users').name,
        require('./modules/private/types').name,
        require('./modules/private/types/edit').name,
        require('./modules/private/types/list').name,
        require('./modules/private/types/new').name,
        require('./modules/private/types/servers').name,
        require('./modules/private/types/cacheGroups').name,
        require('./modules/private/types/deliveryServices').name,
        require('./modules/private/types/staticDnsEntries').name,
        require('./modules/private/users').name,
        require('./modules/private/users/deliveryServices').name,
        require('./modules/private/users/edit').name,
        require('./modules/private/users/list').name,
        require('./modules/private/users/new').name,
        require('./modules/private/users/register').name,

        // current user
        require('./modules/private/user').name,
        require('./modules/private/user/edit').name,

        // custom
        require('./modules/private/custom').name,

        // common modules
        require('./common/modules/chart/bps').name,
        require('./common/modules/chart/httpStatus').name,
        require('./common/modules/chart/tps').name,
        require('./common/modules/compare').name,
        require('./common/modules/dialog/compare').name,
        require('./common/modules/dialog/confirm').name,
        require('./common/modules/dialog/confirm/enter').name,
        require('./common/modules/dialog/delete').name,
        require('./common/modules/dialog/deliveryServiceRequest').name,
        require('./common/modules/dialog/federationResolver').name,
        require('./common/modules/dialog/import').name,
        require('./common/modules/dialog/input').name,
        require('./common/modules/dialog/reset').name,
        require('./common/modules/dialog/select').name,
        require('./common/modules/dialog/select/status').name,
        require('./common/modules/dialog/text').name,
        require('./common/modules/dialog/textarea').name,
        require('./common/modules/header').name,
        require('./common/modules/message').name,
        require('./common/modules/navigation').name,
        require('./common/modules/release').name,

        // forms
        require('./common/modules/form/asn').name,
        require('./common/modules/form/asn/edit').name,
        require('./common/modules/form/asn/new').name,
        require('./common/modules/form/cacheGroup').name,
        require('./common/modules/form/cacheGroup/edit').name,
        require('./common/modules/form/cacheGroup/new').name,
        require('./common/modules/form/capability').name,
        require('./common/modules/form/capability/edit').name,
        require('./common/modules/form/capability/new').name,
        require('./common/modules/form/cdn').name,
        require('./common/modules/form/cdn/edit').name,
        require('./common/modules/form/cdn/new').name,
        require('./common/modules/form/cdnDnssecKeys').name,
        require('./common/modules/form/cdnDnssecKeys/generate').name,
        require('./common/modules/form/cdnDnssecKeys/regenerateKsk').name,
        require('./common/modules/form/coordinate').name,
        require('./common/modules/form/coordinate/edit').name,
        require('./common/modules/form/coordinate/new').name,
        require('./common/modules/form/deliveryService').name,
        require('./common/modules/form/deliveryService/clone').name,
        require('./common/modules/form/deliveryService/edit').name,
        require('./common/modules/form/deliveryService/new').name,
        require('./common/modules/form/deliveryServiceRegex').name,
        require('./common/modules/form/deliveryServiceRegex/edit').name,
        require('./common/modules/form/deliveryServiceRegex/new').name,
        require('./common/modules/form/deliveryServiceSslKeys').name,
        require('./common/modules/form/deliveryServiceSslKeys/generate').name,
        require('./common/modules/form/deliveryServiceStaticDnsEntry').name,
        require('./common/modules/form/deliveryServiceStaticDnsEntry/edit').name,
        require('./common/modules/form/deliveryServiceStaticDnsEntry/new').name,
        require('./common/modules/form/deliveryServiceTarget').name,
        require('./common/modules/form/deliveryServiceTarget/edit').name,
        require('./common/modules/form/deliveryServiceTarget/new').name,
        require('./common/modules/form/deliveryServiceJob').name,
        require('./common/modules/form/deliveryServiceJob/new').name,
        require('./common/modules/form/division').name,
        require('./common/modules/form/division/edit').name,
        require('./common/modules/form/division/new').name,
        require('./common/modules/form/endpoint').name,
        require('./common/modules/form/endpoint/edit').name,
        require('./common/modules/form/endpoint/new').name,
        require('./common/modules/form/federation').name,
        require('./common/modules/form/federation/edit').name,
        require('./common/modules/form/federation/new').name,
        require('./common/modules/form/iso').name,
        require('./common/modules/form/job').name,
        require('./common/modules/form/job/new').name,
        require('./common/modules/form/origin').name,
        require('./common/modules/form/origin/edit').name,
        require('./common/modules/form/origin/new').name,
        require('./common/modules/form/physLocation').name,
        require('./common/modules/form/physLocation/edit').name,
        require('./common/modules/form/physLocation/new').name,
        require('./common/modules/form/parameter').name,
        require('./common/modules/form/parameter/edit').name,
        require('./common/modules/form/parameter/new').name,
        require('./common/modules/form/profile').name,
        require('./common/modules/form/profile/edit').name,
        require('./common/modules/form/profile/new').name,
        require('./common/modules/form/region').name,
        require('./common/modules/form/region/edit').name,
        require('./common/modules/form/region/new').name,
        require('./common/modules/form/role').name,
        require('./common/modules/form/role/edit').name,
        require('./common/modules/form/role/new').name,
        require('./common/modules/form/server').name,
        require('./common/modules/form/server/edit').name,
        require('./common/modules/form/server/new').name,
        require('./common/modules/form/status').name,
        require('./common/modules/form/status/edit').name,
        require('./common/modules/form/status/new').name,
        require('./common/modules/form/tenant').name,
        require('./common/modules/form/tenant/edit').name,
        require('./common/modules/form/tenant/new').name,
        require('./common/modules/form/type').name,
        require('./common/modules/form/type/edit').name,
        require('./common/modules/form/type/new').name,
        require('./common/modules/form/user').name,
        require('./common/modules/form/user/edit').name,
        require('./common/modules/form/user/new').name,
        require('./common/modules/form/user/register').name,

        // tables
        require('./common/modules/table/asns').name,
        require('./common/modules/table/cacheGroups').name,
        require('./common/modules/table/cacheGroupAsns').name,
        require('./common/modules/table/cacheGroupParameters').name,
        require('./common/modules/table/cacheGroupServers').name,
        require('./common/modules/table/cacheGroupStaticDnsEntries').name,
        require('./common/modules/table/capabilities').name,
        require('./common/modules/table/capabilityEndpoints').name,
        require('./common/modules/table/capabilityUsers').name,
        require('./common/modules/table/changeLogs').name,
        require('./common/modules/table/cdns').name,
        require('./common/modules/table/cdnDeliveryServices').name,
        require('./common/modules/table/cdnFederations').name,
        require('./common/modules/table/cdnFederationDeliveryServices').name,
        require('./common/modules/table/cdnFederationUsers').name,
        require('./common/modules/table/cdnProfiles').name,
        require('./common/modules/table/cdnServers').name,
        require('./common/modules/table/coordinates').name,
        require('./common/modules/table/deliveryServices').name,
        require('./common/modules/table/deliveryServiceJobs').name,
        require('./common/modules/table/deliveryServiceOrigins').name,
        require('./common/modules/table/deliveryServiceRegexes').name,
        require('./common/modules/table/deliveryServiceRequests').name,
        require('./common/modules/table/deliveryServiceRequestComments').name,
        require('./common/modules/table/deliveryServiceServers').name,
        require('./common/modules/table/deliveryServiceStaticDnsEntries').name,
        require('./common/modules/table/deliveryServiceTargets').name,
        require('./common/modules/table/deliveryServiceUsers').name,
        require('./common/modules/table/divisions').name,
        require('./common/modules/table/divisionRegions').name,
        require('./common/modules/table/endpoints').name,
        require('./common/modules/table/federationResolvers').name,
        require('./common/modules/table/jobs').name,
        require('./common/modules/table/origins').name,
        require('./common/modules/table/physLocations').name,
        require('./common/modules/table/physLocationServers').name,
        require('./common/modules/table/parameters').name,
        require('./common/modules/table/parameterCacheGroups').name,
        require('./common/modules/table/parameterProfiles').name,
        require('./common/modules/table/profileDeliveryServices').name,
        require('./common/modules/table/profileParameters').name,
        require('./common/modules/table/profileServers').name,
        require('./common/modules/table/profiles').name,
        require('./common/modules/table/regions').name,
        require('./common/modules/table/regionPhysLocations').name,
        require('./common/modules/table/roles').name,
        require('./common/modules/table/roleCapabilities').name,
        require('./common/modules/table/roleUsers').name,
        require('./common/modules/table/servers').name,
        require('./common/modules/table/serverConfigFiles').name,
        require('./common/modules/table/serverDeliveryServices').name,
        require('./common/modules/table/statuses').name,
        require('./common/modules/table/statusServers').name,
        require('./common/modules/table/tenants').name,
        require('./common/modules/table/tenantDeliveryServices').name,
        require('./common/modules/table/tenantUsers').name,
        require('./common/modules/table/types').name,
        require('./common/modules/table/typeCacheGroups').name,
        require('./common/modules/table/typeDeliveryServices').name,
        require('./common/modules/table/typeServers').name,
        require('./common/modules/table/typeStaticDnsEntries').name,
        require('./common/modules/table/users').name,
        require('./common/modules/table/userDeliveryServices').name,

        // widgets
        require('./common/modules/widget/cacheGroups').name,
        require('./common/modules/widget/capacity').name,
        require('./common/modules/widget/cdnChart').name,
        require('./common/modules/widget/changeLogs').name,
        require('./common/modules/widget/dashboardStats').name,
        require('./common/modules/widget/deliveryServices').name,
        require('./common/modules/widget/routing').name,

        // models
        require('./common/models').name,
        require('./common/api').name,

        // directives
        require('./common/directives/match').name,
        require('./common/directives/dragAndDrop').name,

        // services
        require('./common/service/application').name,
        require('./common/service/utils').name,

        // filters
        require('./common/filters').name

    ], App)

        .config(function($stateProvider, $logProvider, $controllerProvider, RestangularProvider, momentPickerProvider, ENV) {

            momentPickerProvider.options({
                minutesStep: 1,
                maxView: 'hour'
            });

            RestangularProvider.setBaseUrl(ENV.api['root']);

            RestangularProvider.setResponseInterceptor(function(data, operation, what) {

                if (angular.isDefined(data.response)) { // todo: this should not be needed. need better solution.
                    if (operation == 'getList') {
                        return data.response;
                    }
                    return data.response[0];
                } else {
                    return data;
                }
            });

            $logProvider.debugEnabled(true);
            $stateProvider
                .state('trafficPortal', {
                    url: '/',
                    abstract: true,
                    templateUrl: 'common/templates/master.tpl.html',
                        resolve: {
                                properties: function(trafficPortalService, propertiesModel) {
                                        return trafficPortalService.getProperties()
                                            .then(function(result) {
                                                    propertiesModel.setProperties(result);
                                            });
                                }
                        }

                });
        })

        .run(function($log, applicationService) {
            $log.debug("Application run...");
            applicationService.startup();
        })
    ;

trafficPortal.factory('authInterceptor', function ($rootScope, $q, $window, $location, $timeout, messageModel, userModel) {
    return {
        responseError: function (rejection) {
            var url = $location.url(),
                alerts = [];

            try { alerts = rejection.data.alerts; }
            catch(e) {}

            // 401, 403, 404 and 5xx errors handled globally; all others handled in fault handler
            if (rejection.status === 401) {
                $rootScope.$broadcast('trafficPortal::exit');
                userModel.resetUser();
                if (url == '/login' || $location.search().redirect) {
                    messageModel.setMessages(alerts, false);
                } else {
                    $timeout(function () {
                        messageModel.setMessages(alerts, true);
                        // forward the to the login page with ?redirect=page/they/were/trying/to/reach
                        $location.url('/login').search({ redirect: encodeURIComponent(url) });
                    }, 200);
                }
            } else if (rejection.status === 403 || rejection.status === 404) {
                $timeout(function () {
                    messageModel.setMessages(alerts, false);
                }, 200);
            } else if (rejection.status.toString().match(/^5\d[01356789]$/)) {
                // matches 5xx EXCEPT for 502's and 504's which indicate a timeout and will be handled by each service call accordingly
                $timeout(function () {
                    if (alerts && alerts.length > 0) {
                            messageModel.setMessages(alerts, false);
                    } else {
                            messageModel.setMessages([ { level: 'error', text: rejection.status.toString() + ': ' + rejection.statusText } ], false);
                    }
                }, 200);
            }

            return $q.reject(rejection);
        }
    };
});

trafficPortal.config(function ($httpProvider) {
    $httpProvider.interceptors.push('authInterceptor');
});


