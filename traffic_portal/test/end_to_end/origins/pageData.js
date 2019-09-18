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

module.exports = function () {
    this.createOriginButton = element(by.css('button[title="Create Origin"]'));
    this.name = element(by.name("name"));
    this.tenant = element(by.name("tenantId"));
    this.fqdn = element(by.name("fqdn"));
    this.protocol = element(by.name("protocol"));
    this.ds = element(by.name('deliveryServiceId'));
    this.createButton = element(by.buttonText('Create'));
    this.successMsg = element(by.css('.alert-success'));
    this.originCreatedText = element(by.cssContainingText('div', 'origin was created.'));
    this.searchFilter = element(by.id('originsTable_filter')).element(by.css('label')).element(by.css('input'));
    this.updateButton = element(by.buttonText('Update'));
    this.originUpdatedText = element(by.cssContainingText('div', 'origin was updated.'));
    this.deleteButton = element(by.buttonText('Delete'));
    this.confirmWithNameInput = element(by.name('confirmWithNameInput'));
    this.deletePermanentlyButton = element(by.buttonText('Delete Permanently'));
    this.originDeletedText = element(by.cssContainingText('div', 'origin was deleted.'));
};
