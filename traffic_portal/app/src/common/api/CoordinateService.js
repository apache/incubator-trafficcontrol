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

var CoordinateService = function($http, locationUtils, messageModel, ENV) {

    this.getCoordinates = function(queryParams) {
        return $http.get(ENV.api['root'] + 'coordinates', {params: queryParams}).then(
            function(result) {
                return result.data.response[0];
            },
            function (err) {
                console.error(error);
            }
        );
    };

    this.createCoordinate = function(coordinate) {
        return $http.post(ENV.api['root'] + "coordinates", coordinate).then(
            function(response) {
                messageModel.setMessages(response.data.alerts, true);
                locationUtils.navigateToPath('/coordinates');
                return response;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false)
                return err;
            }
        );
    };

    this.updateCoordinate = function(id, coordinate) {
        return $http.put(ENV.api['root'] + "coordinates", coordinate, {params: {id: id}}).then(
            function(response) {
                messageModel.setMessages(response.data.alerts, false);
                return response;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                return err;
            }
        );
    };

    this.deleteCoordinate = function(id) {
        return $http.delete(ENV.api['root'] + "coordinates", {params: {id: id}}).then(
            function(response) {
                messageModel.setMessages(response.data.alerts, true);
                return response;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                return err;
            }
        );
    };

};

CoordinateService.$inject = ['$http', 'locationUtils', 'messageModel', 'ENV'];
module.exports = CoordinateService;
