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

var TypeService = function($http, ENV, locationUtils, messageModel) {

    this.getTypes = function(queryParams) {
        return $http.get(ENV.api['root'] + 'types', {params: queryParams}).then(
            function (result) {
                return result.data.response;
            },
            function (err) {
                throw err;
            }
        )
    };

    this.getType = function(id) {
        return $http.get(ENV.api['root'] + 'types', {params: {id: id}}).then(
            function (result) {
                return result.data.response[0];
            },
            function (err) {
                throw err;
            }
        )
    };

    this.createType = function(type) {
        return $http.post(ENV.api['root'] + 'types', type).then(
            function(result) {
                messageModel.setMessages([ { level: 'success', text: 'Type created' } ], true);
                locationUtils.navigateToPath('/types');
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

    // todo: change to use query param when it is supported
    this.updateType = function(type) {
        return $http.put(ENV.api['root'] + 'types/' + type.id, type).then(
            function(result) {
                messageModel.setMessages([ { level: 'success', text: 'Type updated' } ], false);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

    // todo: change to use query param when it is supported
    this.deleteType = function(id) {
        return $http.delete(ENV.api['root'] + "types/" + id).then(
            function(result) {
                messageModel.setMessages([ { level: 'success', text: 'Type deleted' } ], true);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, true);
                throw err;
            }
        );
    };

    this.queueServerUpdates = function(cdnID, typeName) {
        return $http.post(ENV.api['root'] + 'cdns/' + cdnID +'/queue_update?type=' + typeName, {action: "queue"}).then(
            function(result) {
                messageModel.setMessages([{level: 'success', text: 'Queued server updates by type'}], false);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

    this.clearServerUpdates = function(cdnID, typeName) {
        return $http.post(ENV.api['root'] + 'cdns/' + cdnID + '/queue_update?type=' + typeName, {action: "dequeue"}).then(
            function(result) {
                messageModel.setMessages([{level: 'success', text: 'Cleared server updates by type'}], false);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };
};

TypeService.$inject = ['$http', 'ENV', 'locationUtils', 'messageModel'];
module.exports = TypeService;
