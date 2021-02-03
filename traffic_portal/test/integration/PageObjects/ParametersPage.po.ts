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
import { by, element } from 'protractor'
import { BasePage } from './BasePage.po';
import { SideNavigationPage } from './SideNavigationPage.po';
export class ParametersPage extends BasePage {

  private btnCreateNewParameter = element(by.xpath("//button[@title='Create Parameter']"));
  private txtName = element(by.name('name'));
  private txtConfigFile = element(by.name('configFile'));
  private txtValue = element((by.name("value")));
  private txtSecure = element(by.name('secure'));

  private txtSearch = element(by.id('parametersTable_filter')).element(by.css('label input'));
  private mnuParametersTable = element(by.id('parametersTable'));
  private btnDelete = element(by.buttonText('Delete'));
  private btnYes = element(by.buttonText('Yes'));
  private txtConfirmName = element(by.name('confirmWithNameInput'));
  private config = require('../config');
  private randomize = this.config.randomize;

  async OpenParametersPage() {
    let snp = new SideNavigationPage();
    await snp.NavigateToParametersPage();
  }
  async OpenConfigureMenu() {
    let snp = new SideNavigationPage();
    await snp.ClickConfigureMenu();
  }
  async CreateParameter(parameter) {
    let result = false;
    let basePage = new BasePage();
    await this.btnCreateNewParameter.click();
    await this.txtName.sendKeys(parameter.Name + this.randomize);
    await this.txtConfigFile.sendKeys(parameter.ConfigFile);
    await this.txtValue.sendKeys(parameter.Value)
    await this.txtSecure.sendKeys(parameter.Secure);
    await basePage.ClickCreate();
    result = await basePage.GetOutputMessage().then(function (value) {
      if (parameter.validationMessage == value) {
        return true;
      } else {
        return false;
      }
    })
    return result;
  }
  async SearchParameter(nameParameter: string) {
    let name = nameParameter + this.randomize;
    await this.txtSearch.clear();
    await this.txtSearch.sendKeys(name);
    await element.all(by.repeater('p in ::parameters')).filter(function (row) {
      return row.element(by.name('name')).getText().then(function (val) {
        return val === name;
      });
    }).first().click();
  }
  async UpdateParameter(parameter) {
    let result = false;
    let basePage = new BasePage();
    switch (parameter.description) {
      case "update parameter configfile":
        await this.txtConfigFile.clear();
        await this.txtConfigFile.sendKeys(parameter.ConfigFile);
        await basePage.ClickUpdate();
        await this.btnYes.click();
        break;
      default:
        result = undefined;
    }
    if (result = !undefined) {
      result = await basePage.GetOutputMessage().then(function (value) {
        if (parameter.validationMessage == value) {
          return true;
        } else {
          return false;
        }
      })

    }
    return result;
  }
  async DeleteParameter(parameter) {
    let result = false;
    let basePage = new BasePage();
    await this.btnDelete.click();
    await this.btnYes.click();
    await this.txtConfirmName.sendKeys(parameter.Name + this.randomize);
    await basePage.ClickDeletePermanently();
    result = await basePage.GetOutputMessage().then(function (value) {
      if (parameter.validationMessage == value) {
        return true;
      } else {
        return false;
      }
    })
    return result;
  }
}