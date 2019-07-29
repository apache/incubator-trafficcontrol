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

describe('Traffic Portal Regions Test Suite', function() {
	const pageData = new pd();
	const commonFunctions = new cfunc();
	const myNewRegion = {
		name: 'region-' + commonFunctions.shuffle('abcdefghijklmonpqrstuvwxyz0123456789'),
	};

	it('should go to the regions page', async () => {
		console.log("Go to the regions page");
		await browser.setLocation("regions");
		expect(browser.getCurrentUrl().then(commonFunctions.urlPath)).toEqual(commonFunctions.urlPath(browser.baseUrl)+"#!/regions");
	});

	it('should open new region form page', async () => {
		console.log("Open new region form page");
		await pageData.createRegionButton.click();
		expect(browser.getCurrentUrl().then(commonFunctions.urlPath)).toEqual(commonFunctions.urlPath(browser.baseUrl)+"#!/regions/new");
	});

	it('should fill out form, create button is enabled and submit', async () => {
		console.log("Filling out form, check create button is enabled and submit");
		expect(pageData.createButton.isEnabled()).toBe(false);
		await pageData.name.sendKeys(myNewRegion.name);
		await commonFunctions.selectDropdownByNum(pageData.division, 1);
		expect(pageData.createButton.isEnabled()).toBe(true);
		await pageData.createButton.click();
		expect(pageData.successMsg.isPresent()).toBe(true);
        expect(pageData.regionCreatedText.isPresent()).toBe(true, 'Actual message does not match expected message');
		expect(browser.getCurrentUrl().then(commonFunctions.urlPath)).toEqual(commonFunctions.urlPath(browser.baseUrl)+"#!/regions");
	});

});
