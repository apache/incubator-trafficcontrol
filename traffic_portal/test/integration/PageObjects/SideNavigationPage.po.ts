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
import { browser, by, element, ExpectedConditions } from 'protractor';
import { BasePage } from './BasePage.po';

export class SideNavigationPage extends BasePage{
    //Navigation for Configure
    private propConfigure  = "//div[@id='sidebar-menu']//a[contains(text(),'Configure')]";
    private mnuConfigure = element(by.xpath( this.propConfigure ))
    private lnkServers = element(by.xpath("(//a[@href='/#!/servers'])[1]"));
    private lnkServerCapabilities = element(by.xpath("//a[@href='/#!/server-capabilities']"));
    private lnkOrigins = element(by.xpath("//a[@href='/#!/origins']"));
    private lnkProfiles = element(by.xpath("//a[@href='/#!/profiles']"));
    private lnkParameters = element(by.xpath("//a[@href='/#!/parameters']"))
    private lnkTypes = element(by.xpath("//a[@href='/#!/types']"))
    private lnkStatuses = element(by.xpath("//a[@href='/#!/statuses']"))
    //Navigation for Services
    private propServices  = "//div[@id='sidebar-menu']//a[text()=' Services']"
    private mnuServices = element(by.xpath( this.propServices ));
    private lnkDeliveryServices = element(by.xpath("//a[@href='/#!/delivery-services']"));
    private lnkDeliveryServiceRequest = element(by.xpath("//a[@href='/#!/delivery-service-requests']"));
    private lnkServiceCategories = element(by.xpath("//a[@href='/#!/service-categories']"));
    //Navigation for Users Admin
    private propUserAdmin  = "//div[@id='sidebar-menu']//a[contains(text(),'User Admin')]"
    private mnuUserAdmin = element(by.xpath( this.propUserAdmin ))
    private lnkTenants = element(by.xpath("//a[@href='/#!/tenants']"));
    private lnkUsers = element(by.xpath("//a[@href='/#!/users']"));
    //Navigation for CDNs
    private propCDN  = "//div[@id='sidebar-menu']//a[contains(text(),'CDNs')]";
    private mnuCDN = element(by.xpath(this.propCDN))
    //Navigation for Topology
    private propTopology  = "//div[@id='sidebar-menu']//a[contains(text(),'Topology')]"
    private mnuTopology = element(by.xpath( this.propTopology ))
    private lnkPhysLocations = element(by.linkText('Phys Locations'))
    private lnkDivisions = element(by.xpath("//a[@href='/#!/divisions']"));
    private lnkTopologies = element(by.xpath("//a[@href='/#!/topologies']"));
    private lnkCacheGroups = element(by.xpath("//a[@href='/#!/cache-groups']"));
    private lnkCoordinates = element(by.xpath("//a[@href='/#!/coordinates']"));
    private lnkRegions = element(by.xpath("//a[@href='/#!/regions']"));
    private lnkASNs = element(by.xpath("//a[@href='/#!/asns']"));
    //Navigation for Jobs
    private mnuTools = element(by.cssContainingText("#sidebar-menu a", "Tools"))
    private lnkJobs = element(by.linkText("Invalidate Content"))
    async ClickConfigureMenu(){
        await browser.wait(ExpectedConditions.visibilityOf(this.mnuConfigure), 2000);
        await this.mnuConfigure.click();
    }
    async ClickUserAdminMenu(){
        await browser.wait(ExpectedConditions.visibilityOf(this.mnuUserAdmin), 2000);
        await this.mnuUserAdmin.click();
    }
    async ClickServicesMenu(){
        await browser.wait(ExpectedConditions.visibilityOf(this.mnuServices), 2000);
        await this.mnuServices.click();
    }
    async ClickTopologyMenu(){
        await browser.wait(ExpectedConditions.visibilityOf(this.mnuTopology), 2000);
        await this.mnuTopology.click();
    }
    async ClickToolsMenu(){
        await browser.wait(ExpectedConditions.visibilityOf(this.mnuTools), 2000);
        await this.mnuTools.click();
    }
    async NavigateToTopologiesPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkTopologies), 2000);
        await browser.actions().mouseMove(this.lnkTopologies).perform();
        await browser.actions().click(this.lnkTopologies).perform();
    }
    async NavigateToDivisionsPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkDivisions), 2000);
        await browser.actions().mouseMove(this.lnkDivisions).perform();
        await browser.actions().click(this.lnkDivisions).perform();
    }
    async NavigateToRegionsPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkRegions), 2000);
        await browser.actions().mouseMove(this.lnkRegions).perform();
        await browser.actions().click(this.lnkRegions).perform();
    }
    async NavigateToASNsPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkASNs), 2000);
        await browser.actions().mouseMove(this.lnkASNs).perform();
        await browser.actions().click(this.lnkASNs).perform();
    }
    async NavigateToCacheGroupsPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkCacheGroups), 2000);
        await browser.actions().mouseMove(this.lnkCacheGroups).perform();
        await browser.actions().click(this.lnkCacheGroups).perform();
    }
    async NavigateToCoordinatesPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkCoordinates), 2000);
        await browser.actions().mouseMove(this.lnkCoordinates).perform();
        await browser.actions().click(this.lnkCoordinates).perform();
    }
    async NavigateToServersPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkServers), 2000);
        await browser.actions().mouseMove(this.lnkServers).perform();
        await browser.actions().click(this.lnkServers).perform();
    }
    async NavigateToServerCapabilitiesPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkServerCapabilities), 2000);
        await browser.actions().mouseMove(this.lnkServerCapabilities).perform();
        await browser.actions().click(this.lnkServerCapabilities).perform();
    }
    async NavigateToOriginsPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkOrigins), 2000);
        await browser.actions().mouseMove(this.lnkOrigins).perform();
        await browser.actions().click(this.lnkOrigins).perform();
    }
    async NavigateToProfilesPage() {
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkProfiles), 2000);
        await browser.actions().mouseMove(this.lnkProfiles).perform();
        await browser.actions().click(this.lnkProfiles).perform();
    }
    async NavigateToParametersPage() {
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkParameters), 2000);
        await browser.actions().mouseMove(this.lnkParameters).perform();
        await browser.actions().click(this.lnkParameters).perform();
    }
    async NavigateToTypesPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkTypes), 2000);
        await browser.actions().mouseMove(this.lnkTypes).perform();
        await browser.actions().click(this.lnkTypes).perform();
    }
    async NavigateToStatusesPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkStatuses), 2000);
        await browser.actions().mouseMove(this.lnkStatuses).perform();
        await browser.actions().click(this.lnkStatuses).perform();
    }
    async NavigateToDeliveryServicesPage() {
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkDeliveryServices), 2000);
        await browser.actions().mouseMove(this.lnkDeliveryServices).perform();
        await browser.actions().click(this.lnkDeliveryServices).perform();
    }
    async NavigateToDeliveryServicesRequestsPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkDeliveryServiceRequest), 2000);
        await browser.actions().mouseMove(this.lnkDeliveryServiceRequest).perform();
        await browser.actions().click(this.lnkDeliveryServiceRequest).perform();
    }
    async NavigateToServiceCategoriesPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkServiceCategories), 2000);
        await browser.actions().mouseMove(this.lnkServiceCategories).perform();
        await browser.actions().click(this.lnkServiceCategories).perform();
    }
    async NavigateToUsersPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkUsers), 2000);
        await browser.actions().mouseMove(this.lnkUsers).perform();
        await browser.actions().click(this.lnkUsers).perform();
    }
    async NavigateToTenantsPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkTenants), 2000);
        await browser.actions().mouseMove(this.lnkTenants).perform();
        await browser.actions().click(this.lnkTenants).perform();
    }
    async NavigateToCDNPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.mnuCDN), 2000);
        await browser.actions().mouseMove(this.mnuCDN).perform();
        await browser.actions().click(this.mnuCDN).perform();
    }
    async NavigateToPhysLocation(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkPhysLocations), 2000);
        await browser.actions().mouseMove(this.lnkPhysLocations).perform();
        await browser.actions().click(this.lnkPhysLocations).perform();
    }
    async NavigateToJobsPage(){
        await browser.wait(ExpectedConditions.visibilityOf(this.lnkJobs), 2000);
        await browser.actions().mouseMove(this.lnkJobs).perform();
        await browser.actions().click(this.lnkJobs).perform();
    }
}
