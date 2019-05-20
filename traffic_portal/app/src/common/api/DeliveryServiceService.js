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

var DeliveryServiceService = function($http, locationUtils, httpService, messageModel, ENV) {

    this.getDeliveryServices = function(queryParams) {
        return $http.get(ENV.api['root'] + 'deliveryservices', {params: queryParams}).then(
            function(result) {
                return result.data.response;
            },
            function(err) {
                console.error(err);
            }
        );
    };

    this.getDeliveryService = function(id) {
        return $http.get(ENV.api['root'] + 'deliveryservices', {params: {id: id}}).then(
            function(result) {
                return result.data.response[0];
            },
            function(err) {
                console.error(err);
            }
        );
    };

    this.createDeliveryService = function(ds) {
        $http.post(ENV.api['root'] + "deliveryservices", ds).then(
            function(response) {
                return response;
            },
            function(err) {
                console.error(err);
                return err;
            }
        );
    };

    this.updateDeliveryService = function(ds) {
        return $http.put(ENV.api['root'] + "deliveryservices/" + ds.id, ds).then(
            function(response) {
                return response;
            },
            function(err) {
                console.error(err);
                return err;
            }
        );
    };

    this.deleteDeliveryService = function(ds) {
        return $http.delete(ENV.api['root'] + "deliveryservices/" + ds.id).then(
            function(response) {
                return response;
            },
            function(err) {
                console.error(err);
                return err;
            }
        );
    };

    this.getServerDeliveryServices = function(serverId) {
        return $http.get(ENV.api['root'] + 'servers/' + serverId + '/deliveryservices').then(
            function(result) {
                return result.data.response;
            },
            function(err) {
                console.error(err);
            }
        );
    };

    this.getDeliveryServiceTargets = function(dsId) {
        return $http.get(ENV.api['root'] + 'steering/' + dsId + '/targets').then(
            function(result) {
                return result.data.response;
            },
            function(err) {
                console.error(err);
            }
        );
    };

    this.getDeliveryServiceTarget = function(dsId, targetId) {
        return $http.get(ENV.api['root'] + 'steering/' + dsId + '/targets/' + targetId).then(
            function(result) {
                return result.data.response[0];
            },
            function(err) {
                console.error(err);
            }
        );
    };

    this.updateDeliveryServiceTarget = function(dsId, targetId, target) {
        return $http.put(ENV.api['root'] + "steering/" + dsId + "/targets/" + targetId, target).then(
            function(result) {
                messageModel.setMessages(result.data.alerts, true);
                locationUtils.navigateToPath('/delivery-services/' + dsId + '/targets');
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
            }
        );
    };

    this.createDeliveryServiceTarget = function(dsId, target) {
        return $http.post(ENV.api['root'] + 'steering/' + dsId + '/targets', target).then(
            function(result) {
                messageModel.setMessages(result.data.alerts, true);
                locationUtils.navigateToPath('/delivery-services/' + dsId + '/targets');
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
            }
        );
    };

    this.deleteDeliveryServiceTarget = function(dsId, targetId) {
        return $http.delete(ENV.api['root'] + 'steering/', dsId + '/targets/' + targetId).then(
            function(result) {
                messageModel.setMessages(result.data.alerts, true);
                locationUtils.navigateToPath('/delivery-services/' + dsId + '/targets');
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, true);
            }
        );
    };

    this.getUserDeliveryServices = function(userId) {
        return $http.get(ENV.api['root'] + 'users/' + userId + '/deliveryservices').then(
            function(result) {
                return result.data.response;
            },
            function(err) {
                console.error(err);
            }
        );
    };

    this.deleteDeliveryServiceServer = function(dsId, serverId) {
        return httpService.delete(ENV.api['root'] + 'deliveryservice_server/' + dsId + '/' + serverId).then(
            function(result) {
                messageModel.setMessages(result.data.alerts, false);
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, true);
            }
        );
    };

    this.assignDeliveryServiceServers = function(dsId, servers) {
        return $http.post(ENV.api['root'] + 'deliveryserviceserver',{ dsId: dsId, servers: servers, replace: true } ).then(
            function(result) {
                messageModel.setMessages(result.data.alerts, false);
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
            }
        );
    };

    this.getConsistentHashResult = function (regex, requestPath, cdnId) {
        const url = ENV.api['root'] + "consistenthash";
        const params = {regex: regex, requestPath: requestPath, cdnId: cdnId};

        return $http.post(url, params).then(
            function (result) {
                return result.data;
            },
            function (err) {
                messageModel.setMessages(err.data.alerts, false);
                return err;
            }
        );
    };

};

DeliveryServiceService.$inject = ['$http', 'locationUtils', 'httpService', 'messageModel', 'ENV'];
module.exports = DeliveryServiceService;
