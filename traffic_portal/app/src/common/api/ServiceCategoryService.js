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

var ServiceCategoryService = function($http, ENV, locationUtils, messageModel) {

    this.getServiceCategories = function(queryParams) {
        return $http.get(ENV.api['root'] + 'service_categories', {params: queryParams}).then(
            function(result) {
                return result.data.response;
            },
            function(err) {
                throw err;
            }
        );
    };

    this.getServiceCategory = function(id) {
        return $http.get(ENV.api['root'] + 'service_categories', {params: {id: id}}).then(
            function(result) {
                return result.data.response[0];
            },
            function(err) {
                throw err;
            }
        );
    };

    this.createServiceCategory = function(serviceCategory) {
        return $http.post(ENV.api['root'] + 'service_categories', serviceCategory).then(
            function(result) {
                messageModel.setMessages(result.data.alerts, true);
                locationUtils.navigateToPath('/service-categories');
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

    this.updateServiceCategory = function(serviceCategory) {
        return $http.put(ENV.api['root'] + 'service_categories/' + serviceCategory.id, serviceCategory).then(
            function(result) {
                messageModel.setMessages(result.data.alerts, false);
                return result;            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

    this.deleteServiceCategory = function(id) {
        return $http.delete(ENV.api['root'] + 'service_categories/' + id).then(
            function(result) {
                messageModel.setMessages(result.data.alerts, true);
                return result;
            },
            function(err) {
                messageModel.setMessages(err.data.alerts, false);
                throw err;
            }
        );
    };

};

ServiceCategoryService.$inject = ['$http', 'ENV', 'locationUtils', 'messageModel'];
module.exports = ServiceCategoryService;
