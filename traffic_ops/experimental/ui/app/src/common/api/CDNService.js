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

var CDNService = function(Restangular, locationUtils, messageModel) {

    this.getCDNs = function() {
        return Restangular.all('cdns').getList();
    };

    this.getCDN = function(id) {
        return Restangular.one("cdns", id).get();
    };

    this.createCDN = function(cdn) {
        return Restangular.service('cdns').post(cdn)
            .then(
                function() {
                    messageModel.setMessages([ { level: 'success', text: 'CDN created' } ], true);
                    locationUtils.navigateToPath('/admin/cdns');
                },
                function(fault) {
                    messageModel.setMessages(fault.data.alerts, false);
                }
            );
    };

    this.updateCDN = function(cdn) {
        return cdn.put()
            .then(
                function() {
                    messageModel.setMessages([ { level: 'success', text: 'CDN updated' } ], false);
                },
                function(fault) {
                    messageModel.setMessages(fault.data.alerts, false);
                }
            );
    };

    this.deleteCDN = function(id) {
        return Restangular.one("cdns", id).remove()
            .then(
                function() {
                    messageModel.setMessages([ { level: 'success', text: 'CDN deleted' } ], true);
                },
                function(fault) {
                    messageModel.setMessages(fault.data.alerts, true);
                }
            );
    };

};

CDNService.$inject = ['Restangular', 'locationUtils', 'messageModel'];
module.exports = CDNService;
