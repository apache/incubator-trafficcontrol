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

describe('Traffic Portal Parameters Test Suite', function() {
	const pageData = new pd();
	const commonFunctions = new cfunc();
	const myNewParameter = {
        name: 'parameter-' + commonFunctions.shuffle('abcdefghijklmonpqrstuvwxyz0123456789'),
		configFile: 'config-' + commonFunctions.shuffle('abcdefghijklmonpqrstuvwxyz0123456789'),
		secure: true
	};

	it('should go to the parameters page', async () => {
		console.log("Go to the parameters page");
		await browser.get(browser.baseUrl + "/#!/parameters");
		expect(browser.getCurrentUrl().then(commonFunctions.urlPath)).toEqual(commonFunctions.urlPath(browser.baseUrl)+"#!/parameters");
	});

	it('should open new parameter form page', async () => {
		console.log("Open new parameter form page");
		await pageData.createParameterButton.click();
		expect(browser.getCurrentUrl().then(commonFunctions.urlPath)).toEqual(commonFunctions.urlPath(browser.baseUrl)+"#!/parameters/new");
	});

	it('should fill out form, create button is enabled and submit', async () => {
		console.log("Filling out form, check create button is enabled and submit");
		expect(pageData.createButton.isEnabled()).toBe(false);
		await pageData.name.sendKeys(myNewParameter.name);
		await pageData.configFile.sendKeys(myNewParameter.configFile);
		await commonFunctions.selectDropdownByLabel(pageData.secure, myNewParameter.secure.toString());
		await pageData.value.sendKeys(myNewParameter.name);
		expect(pageData.createButton.isEnabled()).toBe(true);
		await pageData.createButton.click();
		expect(pageData.successMsg.isPresent()).toBe(true);
        expect(pageData.parameterCreatedText.isPresent()).toBe(true, 'Actual message does not match expected message');
		expect(browser.getCurrentUrl().then(commonFunctions.urlPath)).toMatch(commonFunctions.urlPath(browser.baseUrl)+"#!/parameters/[0-9]+/profiles");
	});

});