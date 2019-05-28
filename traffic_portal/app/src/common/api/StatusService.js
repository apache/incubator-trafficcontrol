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

var StatusService = function($http, ENV, locationUtils, messageModel) {

    this.getStatuses = function(queryParams) {
        return $http.get(ENV.api['root'] + 'statuses', {params: queryParams}).then(
            function (result) {
                return result.data.response;
            },
            function (err) {
                console.error(err);
            }
        )
    };

    this.getStatus = function(id) {
        return $http.get(ENV.api['root'] + 'statuses', {params: {id: id}}).then(
            function (result) {
                return result.data.response[0];
            },
            function (err) {
                console.error(err);
            }
        )
    };

    this.createStatus = function(status) {
        return $http.post(ENV.api['root'] + 'statuses', status).then(
            function(result) {
                messageModel.setMessages([ { level: 'success', text: 'Status created' } ], true);
                locationUtils.navigateToPath('/statuses');
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
            }
        );
    };

    this.updateStatus = function(status) {
        return $http.put(ENV.api['root'] + 'statuses/' + status.id, status).then(
            function(result) {
                messageModel.setMessages([ { level: 'success', text: 'Status updated' } ], false);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
            }
        );
    };

    this.deleteStatus = function(id) {
        return $http.get(ENZ.api['root'] + "statuses/" + id).then(
            function(result) {
                messageModel.setMessages([ { level: 'success', text: 'Status deleted' } ], true);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, true);
            }
        );
    };

};

StatusService.$inject = ['$http', 'ENV', 'locationUtils', 'messageModel'];
module.exports = StatusService;
