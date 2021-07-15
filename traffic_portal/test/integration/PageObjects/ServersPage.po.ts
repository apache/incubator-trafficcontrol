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

import randomIpv6 from "random-ipv6";

import { BasePage } from './BasePage.po';
import {SideNavigationPage} from '../PageObjects/SideNavigationPage.po';
import { randomize } from '../config';

interface CreateServer {
  Status: string;
  Hostname: string;
  Domainname: string;
  CDN: string;
  CacheGroup: string;
  Type: string;
  Profile: string;
  PhysLocation: string;
  InterfaceName: string;
  validationMessage?: string;
}

interface ServerCapability {
  ServerCapability: string;
  validationMessage?: string;
}

interface UpdateServer {
  description: string;
  CDN: string;
  Profile: string;
  validationMessage?: string;
}

interface DeleteServer {
  Name: string;
  validationMessage?: string;
}

export class ServersPage extends BasePage {

  private btnMore = element(by.xpath("//button[contains(text(),'More')]"));

  private btnDelete = element(by.buttonText('Delete'));

  private txtStatus = element(by.name('status'));
  private txtHostName = element(by.xpath("//ol[@class='breadcrumb pull-left']//li[@class='active ng-binding']"))
  private txtDomainName = element(by.name('domainName'));
  private txtProfile = element(by.name('profile'));
  private txtPhysLocation = element(by.name('physLocation'));
  private lblInputError = element(by.className("input-error"));

  private txtHostname = element(by.name('hostName'));
  private txtCDN = element(by.name('cdn'));
  private txtCacheGroup = element(by.name('cachegroup'));
  private txtType = element(by.name('type'));
  private txtConfirmServerName = element(by.name('confirmWithNameInput'));

  private btnYesRemoveSC = element(by.buttonText("Yes"))
  private btnManageCapabilities = element(by.linkText('Manage Capabilities'));
  private btnAddCapabilities = element(by.name('addCapabilityBtn'));
  private selectCapabilities = element(by.name('selectFormDropdown'));
  private searchFilter = element(by.id('serverCapabilitiesTable_filter')).element(by.css('label input'));
  private btnManageDeliveryService = element(by.linkText('Manage Delivery Services'));
  private btnLinkDStoServer = element(by.xpath("//button[@title='Link Delivery Services to Server']"));
  private txtDSSearch = element(by.id('assignDSTable_filter')).element(by.css('label input'));
  private txtInterfaceName = element(by.id("-name"));
  private btnMoreCreateServer = element(by.name("moreBtn"))
  private btnCreateServer = element(by.name("createServerMenuItem"))
  private txtQuickSearch = element(by.id("quickSearch"));
  private randomize = randomize;

  async OpenServerPage(){
    let snp = new SideNavigationPage();
    await snp.NavigateToServersPage();
   }
   async OpenConfigureMenu(){
    let snp = new SideNavigationPage();
    await snp.ClickConfigureMenu();
   }
  GetInputErrorDisplayed() {
    return this.lblInputError.getText()
  }

  public IsServersItemPresent(): PromiseLike<boolean> {
    return element(by.xpath("//table[@id='serversTable']//tr/td[text()='" + "']")).isPresent()
  }

  async ClickAddServer() {
    await this.btnCreateServer.click()
  }

  public async CreateServer(server: CreateServer): Promise<boolean> {
    let result = false;
    let basePage = new BasePage();
    let ipv6 = randomIpv6();
    await this.btnMoreCreateServer.click();
    await this.btnCreateServer.click();
    await this.txtStatus.sendKeys(server.Status);
    await this.txtHostname.sendKeys(server.Hostname+this.randomize);
    await this.txtDomainName.sendKeys(server.Domainname);
    await this.txtCDN.sendKeys("ALL");
    await this.txtCDN.sendKeys(server.CDN + this.randomize);
    await this.txtCacheGroup.sendKeys(server.CacheGroup + this.randomize);
    await this.txtType.sendKeys(server.Type);
    await this.txtProfile.sendKeys(server.Profile + this.randomize);
    await this.txtPhysLocation.sendKeys(server.PhysLocation);
    await this.txtInterfaceName.sendKeys(server.InterfaceName);
    await element(by.id(""+server.InterfaceName+"-")).sendKeys(ipv6.toString());
    if (!await basePage.ClickCreate())
        result = false;
    await basePage.GetOutputMessage().then(function(value){
      if(server.validationMessage == value){
        result = true;
      }else{
        result = false;
      }
    })
    await this.OpenServerPage();
    return result;
  }

  async SearchServer(nameServer:string){
    let name = nameServer+this.randomize;
    await this.txtQuickSearch.clear();
    await this.txtQuickSearch.sendKeys(name);
    await browser.actions().click(element(by.xpath("//span[text()='"+name+"']"))).perform();
  }

  public async SearchDeliveryServiceFromServerPage(name:string): Promise<boolean> {
    await this.txtDSSearch.clear();
    await this.txtDSSearch.sendKeys(name);
    if(await browser.isElementPresent(element(by.xpath("//td[@data-search='^"+name+"$']"))) == true){
      await element(by.xpath("//td[@data-search='^"+name+"$']")).click();
      return true;
    }
    return false;
  }

  public async AddDeliveryServiceToServer(deliveryServiceName:string,outputMessage:string): Promise<boolean | undefined> {
    let result: boolean | undefined = false;
    let basePage = new BasePage();
    let deliveryService = deliveryServiceName+this.randomize;
    const serverNameRandomized = await this.txtHostName.getText();
    let serverName = serverNameRandomized.replace(this.randomize,"")
    if(outputMessage.includes("delivery services assigned")){
      outputMessage = outputMessage.replace(serverName,serverNameRandomized)
    }
    if(outputMessage.includes("cannot assign")){
      let dsCapRequired = outputMessage.slice(112,118);
      outputMessage = outputMessage.replace(dsCapRequired,dsCapRequired+this.randomize)
      outputMessage = outputMessage.replace(serverName,serverNameRandomized)
    }
    await this.btnMore.click();
    if(await this.btnManageDeliveryService.isPresent() == true){
      await this.btnManageDeliveryService.click();
      await this.btnLinkDStoServer.click();
      if(await this.SearchDeliveryServiceFromServerPage(deliveryService) == true){
        await basePage.ClickSubmit();
         result = await basePage.GetOutputMessage().then(function(value){
          if(value == outputMessage){
            return true;
          }else{
            return false;
          }
        })
      }
    }else{
      result = undefined;
    }
    return result;

  }

  public async AddServerCapabilitiesToServer(serverCapabilities: ServerCapability): Promise<boolean | undefined> {
    let result: boolean | undefined = false;
    let basePage = new BasePage();
    let serverCapabilitiesName = serverCapabilities.ServerCapability + this.randomize;
    await this.btnMore.click();
    if((await this.btnManageCapabilities.isPresent()) == true){
      await this.btnManageCapabilities.click();
      await this.btnAddCapabilities.click();
      await this.selectCapabilities.sendKeys(serverCapabilitiesName);
      await basePage.ClickSubmit();
      result = await basePage.GetOutputMessage().then(function(value){
        if(serverCapabilities.validationMessage === value || serverCapabilities.validationMessage && value.includes(serverCapabilities.validationMessage)){
          result = true;
        }else{
          result = false;
        }
        return result;
      })
    }else{
      result = undefined;
    }
    await this.OpenServerPage();
    return result;
   }

   async SearchServerServerCapabilities(name:string){
    let result = false;
    await this.searchFilter.clear();
    await this.searchFilter.sendKeys(name);
    result = await element.all(by.repeater('sc in ::serverCapabilities')).filter(function(row){
      return row.element(by.name('name')).getText().then(function(val){
        return val === name;
      });
    }).first().getText().then(function(value){
      if(value == name){
        return true;
      }else{
        return false;
      }
    })
    return result;
   }


   public async RemoveServerCapabilitiesFromServer(serverCapabilities:string,outputMessage:string): Promise<boolean | undefined> {
    let result: boolean | undefined = false;
    let basePage = new BasePage();
    let serverCapabilitiesname = serverCapabilities+this.randomize;
    const url = (await browser.getCurrentUrl()).toString();
    let serverNumber = url.substring(url.lastIndexOf('/') + 1);
    if(outputMessage.includes("cannot remove")){
      outputMessage = outputMessage.replace(serverCapabilities,serverCapabilitiesname)
      outputMessage = outputMessage.slice(0,56) + serverNumber + " " + outputMessage.slice(56);
    }
    await this.btnMore.click();
    if((await this.btnManageCapabilities.isPresent()) == true){
      await this.btnManageCapabilities.click();
      if(await this.SearchServerServerCapabilities(serverCapabilitiesname) == true){
        await element(by.xpath("//td[text()='"+ serverCapabilitiesname +"']/following-sibling::td/a[@title='Remove Server Capability']")).click();
      }
      await this.btnYesRemoveSC.click();
      result = await basePage.GetOutputMessage().then(function(value){
        if(outputMessage == value){
          return true;
        }else if(value.includes(outputMessage)){
          return true;
        }else{
          return false;
        }
      })
    }else{
      result = undefined;
    }
    await this.OpenServerPage();
    return result;
   }

  public async UpdateServer(server: UpdateServer): Promise<boolean | undefined> {
    let result = false;
    let basePage = new BasePage();
    if(server.description.includes('change the cdn of a Server')){
      await this.txtCDN.sendKeys(server.CDN + this.randomize);
      await this.txtProfile.sendKeys(server.Profile + this.randomize)
      await basePage.ClickUpdate();
      result = await basePage.GetOutputMessage().then(function (value) {
          if (server.validationMessage == value) {
            return true;
          } else {
            return false;
          }
        })
        return result;
    }
  }

  async DeleteServer(server: DeleteServer){
    let result = false;
    let basePage = new BasePage();
    let name = server.Name + this.randomize
    await this.btnDelete.click();
    await browser.wait(ExpectedConditions.visibilityOf(this.txtConfirmServerName), 1000);
    await this.txtConfirmServerName.sendKeys(name);
    if(await basePage.ClickDeletePermanently() == true){
      result = await basePage.GetOutputMessage().then(function(value){
        if(server.validationMessage == value){
          return true
        }else{
          return false;
        }
      })
    }else{
      await basePage.ClickCancel();
    }
    await this.OpenServerPage();
    return result;
   }
}
