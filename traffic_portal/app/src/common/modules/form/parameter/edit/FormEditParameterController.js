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

var FormEditParameterController = function(parameter, $scope, $controller, $uibModal, $anchorScroll, locationUtils, parameterService, profileService, messageModel) {

    // extends the FormParameterController to inherit common methods
    angular.extend(this, $controller('FormParameterController', { parameter: parameter, $scope: $scope }));

    var deleteParameter = function(parameter) {
        parameterService.deleteParameter(parameter.id)
            .then(function(result) {
                messageModel.setMessages(result.alerts, true);
                locationUtils.navigateToPath('/parameters');
            });
    };

    var save = function(parameter) {
        parameterService.updateParameter(parameter).
            then(function() {
                $scope.parameterName = angular.copy(parameter.name);
                $anchorScroll(); // scrolls window to top
            });
    };

    $scope.parameterName = angular.copy(parameter.name);

    $scope.settings = {
        isNew: false,
        saveLabel: 'Update'
    };

    $scope.confirmSave = function(parameter) {
        profileService.getParameterProfiles(parameter.id).
            then(function(result) {
                const params = {
                    title: 'Update Parameter?',
                    message: result.length + ' profiles use this parameter.<br/><br/>'
                };
                if (result.length > 0) {
                    params.message += result.map(function(x) {return x.name;}).join('<br/>') + '<br/><br/>';
                }
                params.message += 'Are you sure you want to update the parameter?';

                const modalInstance = $uibModal.open({
                    templateUrl: 'common/modules/dialog/confirm/dialog.confirm.tpl.html',
                    controller: 'DialogConfirmController',
                    size: 'md',
                    resolve: {
                        params: function () {
                            return params;
                        }
                    }
                });
                modalInstance.result.then(function() {
                    save(parameter);
                }, function () {
                    // do nothing
                });
            }
        );
    };

    $scope.confirmDelete = function(parameter) {
        profileService.getParameterProfiles(parameter.id).
            then(function(result) {
                const params = {
                    title: 'Delete Parameter?',
                    message: result.length + ' profiles use this parameter.<br/><br/>'
                };
                if (result.length > 0) {
                    params.message += result.map(function(x) {return x.name;}).join('<br/>') + '<br/><br/>';
                }
                params.message += 'Are you sure you want to delete the parameter?';

                const modalInstance = $uibModal.open({
                    templateUrl: 'common/modules/dialog/confirm/dialog.confirm.tpl.html',
                    controller: 'DialogConfirmController',
                    size: 'md',
                    resolve: {
                        params: function () {
                            return params;
                        }
                    }
                });
                modalInstance.result.then(function() {
                    params = {
                        title: 'Delete Parameter: ' + parameter.name,
                        key: parameter.name
                    };
                    modalInstance = $uibModal.open({
                        templateUrl: 'common/modules/dialog/delete/dialog.delete.tpl.html',
                        controller: 'DialogDeleteController',
                        size: 'md',
                        resolve: {
                            params: function () {
                                return params;
                            }
                        }
                    });
                    modalInstance.result.then(function() {
                        deleteParameter(parameter);
                    }, function () {
                        // do nothing
                    });
                }, function () {
                    // do nothing
                });
            });
    };

};

FormEditParameterController.$inject = ['parameter', '$scope', '$controller', '$uibModal', '$anchorScroll', 'locationUtils', 'parameterService', 'profileService', 'messageModel'];
module.exports = FormEditParameterController;
