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

var ProfileService = function($http, locationUtils, messageModel, ENV) {

    this.getProfiles = function(queryParams) {
        return $http.get(ENV.api['root'] + 'profiles', {params: queryParams}).then(
            function(result) {
                return result.data.response;
            },
            function(err) {
                throw err;
            }
        );
    };

    this.getProfile = function(id) {
        return $http.get(ENV.api['root'] + 'profiles', {params: {id: id}}).then(
            function (result) {
                return result.data.response[0];
            },
            function (err) {
                throw err;
            }
        );
    };

    this.createProfile = function(profile) {
        return $http.post(ENV.api['root'] + 'profiles', profile).then(
            function(result) {
                messageModel.setMessages([ { level: 'success', text: 'Profile created' } ], true);
                locationUtils.navigateToPath('/profiles/' + result.data.response.id + '/parameters');
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

    // todo: change to use query param when it is supported
    this.updateProfile = function(profile) {
        return $http.put(ENV.api['root'] + 'profiles/' + profile.id, profile).then(
            function(result) {
                messageModel.setMessages([ { level: 'success', text: 'Profile updated' } ], false);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

    // todo: change to use query param when it is supported
    this.deleteProfile = function(id) {
        return $http.delete(ENV.api['root'] + "profiles/" + id).then(
            function(result) {
                return result.data;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

    this.getParameterProfiles = function(paramId) {
        return $http.get(ENV.api['root'] + 'profiles', {params: {param: paramId}}).then(
            function (result) {
                return result.data.response;
            },
            function (err) {
                throw err;
            }
        );
    };

    this.cloneProfile = function(sourceName, cloneName) {
        return $http.post(ENV.api['root'] + "profiles/name/" + cloneName + "/copy/" + sourceName).then(
            function(result) {
                messageModel.setMessages(result.data.alerts, true);
                locationUtils.navigateToPath('/profiles/' + result.data.response.id);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

    this.exportProfile = function(id) {
        return $http.get(ENV.api['root'] + "profiles/" + id + "/export").then(
            function(result) {
                return result.data;
            },
            function(err) {
                throw err;
            }
        );
    };

    this.importProfile = function(importJSON) {
        return $http.post(ENV.api['root'] + "profiles/import", importJSON).then(
            function(result) {
                messageModel.setMessages(result.data.alerts, true);
                locationUtils.navigateToPath('/profiles/' + result.data.response.id);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

    this.queueServerUpdatesByProfile = function(cdnID, profileName) {
        return $http.post(ENV.api['root'] + 'cdns/' + cdnID + '/queue_update?profile=' + profileName, {action: "queue"}).then(
            function(result) {
                messageModel.setMessages([{level: 'success', text: 'Queued server updates by profile'}], false);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

    this.clearServerUpdatesByProfile = function(cdnID, profileName) {
        return $http.post(ENV.api['root'] + 'cdns/' + cdnID + '/queue_update?profile=' + profileName, {action: "dequeue"}).then(
            function(result) {
                messageModel.setMessages([{level: 'success', text: 'Cleared server updates by profile'}], false);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

};

ProfileService.$inject = ['$http', 'locationUtils', 'messageModel', 'ENV'];
module.exports = ProfileService;
