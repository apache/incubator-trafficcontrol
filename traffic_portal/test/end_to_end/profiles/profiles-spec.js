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

var pd = require('./pageData.js');
var cfunc = require('../common/commonFunctions.js');

describe('Traffic Portal Profiles Test Suite', function() {
	const pageData = new pd();
	const commonFunctions = new cfunc();
	const myNewProfile = {
		name: 'profile-' + commonFunctions.shuffle('abcdefghijklmonpqrstuvwxyz0123456789'),
		routingDisabled: false,
		type: 'ATS_PROFILE'
	};

	it('should go to the profiles page', async () => {
		console.log("Go to the profiles page");
		await browser.setLocation("profiles");
		expect(browser.getCurrentUrl().then(commonFunctions.urlPath)).toEqual(commonFunctions.urlPath(browser.baseUrl)+"#!/profiles");
	});

	it('should open new profile form page', async () => {
		console.log("Open new profile form page");
		await pageData.createProfileButton.click();
		expect(browser.getCurrentUrl().then(commonFunctions.urlPath)).toEqual(commonFunctions.urlPath(browser.baseUrl)+"#!/profiles/new");
	});

	it('should fill out form, create button is enabled and submit', async () => {
		console.log("Filling out form, check create button is enabled and submit");
		expect(pageData.createButton.isEnabled()).toBe(false);
		await pageData.name.sendKeys(myNewProfile.name);
		await commonFunctions.selectDropdownByNum(pageData.cdn, 1);
		await commonFunctions.selectDropdownByLabel(pageData.type, myNewProfile.type);
		await commonFunctions.selectDropdownByLabel(pageData.routingDisabled, myNewProfile.routingDisabled.toString());
		await pageData.description.sendKeys(myNewProfile.name);
		expect(pageData.createButton.isEnabled()).toBe(true);
		await pageData.createButton.click();
		expect(pageData.successMsg.isPresent()).toBe(true);
        expect(pageData.profileCreatedText.isPresent()).toBe(true, 'Actual message does not match expected message');
		expect(browser.getCurrentUrl().then(commonFunctions.urlPath)).toMatch(commonFunctions.urlPath(browser.baseUrl)+"#!/profiles/[0-9]+/parameters");
	});

});
