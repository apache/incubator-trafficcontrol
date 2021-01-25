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

module.exports = function(){
    this.username=element(by.name('uName'));
    this.fullName=element(by.name('fullName'));
    this.email=element(by.name('email'));
    this.roleName=element(by.name('role'));
    this.tenantId=element(by.name('tenantId'));
    this.localPasswd=element(by.name('uPass'));
    this.confirmLocalPasswd=element(by.name('confirmPassword'));
    this.registerSent=element(by.name('registrationSent'));
    this.createButton=element(by.buttonText('Create'));
    this.registerNewUserButton=element(by.buttonText('Register User'));
    this.registerEmailButton=element(by.buttonText('Send Registration'));
    this.updateButton=element(by.buttonText('Update'));
    this.searchFilter=element(by.id('usersTable_filter')).element(by.css('label input'));
};
