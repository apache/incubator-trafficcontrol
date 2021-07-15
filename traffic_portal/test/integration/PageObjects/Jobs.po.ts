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

import { by, element } from 'protractor';
import { randomize } from '../config';
import { BasePage } from './BasePage.po';
import { SideNavigationPage } from './SideNavigationPage.po';

interface Job {
    DeliveryService: string;
    Regex: string;
    Ttl: string;
    validationMessage: string;
}
export class JobsPage extends BasePage {
    private moreBtn = element(by.name('moreBtn'));
    private createJobMenuItem = element(by.name('createJobMenuItem'));
    private txtRegex = element(by.name('regex'));
    private txtTtl = element(by.name('ttl'));
    private txtDeliveryservice = element(by.name('deliveryservice'));
    private randomize = randomize;

    public async OpenJobsPage() {
        let snp = new SideNavigationPage();
        await snp.NavigateToJobsPage();
    }

    public async OpenToolsMenu() {
        let snp = new SideNavigationPage();
        await snp.ClickToolsMenu();
    }

    public async CreateJobs(jobs: Job): Promise<boolean> {
        let result = false;
        const basePage = new BasePage();
        const snp = new SideNavigationPage();
        await snp.NavigateToJobsPage();
        await this.moreBtn.click();
        await this.createJobMenuItem.click();
        await this.txtDeliveryservice.sendKeys(jobs.DeliveryService + this.randomize)
        await this.txtRegex.sendKeys(jobs.Regex);
        await this.txtTtl.sendKeys(jobs.Ttl);
        await basePage.ClickCreate();
        result = await basePage.GetOutputMessage().then(value => value.includes(jobs.validationMessage));
        return result;
    }
}
