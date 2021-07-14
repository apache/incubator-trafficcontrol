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

var TableParametersController = function(parameters, $scope, $state, $uibModal, $window, locationUtils, parameterService, profileService, messageModel) {

    let parametersTable;

    var deleteParameter = function(parameter) {
        parameterService.deleteParameter(parameter.id)
            .then(function(result) {
                messageModel.setMessages(result.alerts, false);
                $scope.refresh();
            });
    };

    var confirmDelete = function(parameter) {
        profileService.getParameterProfiles(parameter.id).
        then(function(result) {
            var params = {
                title: 'Delete Parameter?',
                message: result.length + ' profiles use this parameter.<br/><br/>'
            };
            if (result.length > 0) {
                params.message += result.map(function(r) {return r.name;}).join('<br/>') + '<br/><br/>';
            }
            params.message += 'Are you sure you want to delete the parameter?';

            var modalInstance = $uibModal.open({
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

    $scope.parameters = parameters;

    $scope.columns = [
        { "name": "Name", "visible": true, "searchable": true },
        { "name": "Config File", "visible": true, "searchable": true },
        { "name": "Value", "visible": true, "searchable": true },
        { "name": "Secure", "visible": true, "searchable": true },
        { "name": "Profiles", "visible": true, "searchable": true }
    ];

    $scope.contextMenuItems = [
        {
            text: 'Open in New Tab',
            click: function ($itemScope) {
                $window.open('/#!/parameters/' + $itemScope.p.id, '_blank');
            }
        },
        null, // Dividier
        {
            text: 'Edit',
            click: function ($itemScope) {
                $scope.editParameter($itemScope.p.id);
            }
        },
        {
            text: 'Delete',
            click: function ($itemScope) {
                confirmDelete($itemScope.p);
            }
        },
        null, // Dividier
        {
            text: 'Manage Profiles',
            click: function ($itemScope) {
                locationUtils.navigateToPath('/parameters/' + $itemScope.p.id + '/profiles');
            }
        }
    ];

    $scope.editParameter = function(id) {
        locationUtils.navigateToPath('/parameters/' + id);
    };

    $scope.createParameter = function() {
        locationUtils.navigateToPath('/parameters/new');
    };

    $scope.refresh = function() {
        $state.reload(); // reloads all the resolves for the view
    };

    $scope.toggleVisibility = function(colName) {
        const col = parametersTable.column(colName + ':name');
        col.visible(!col.visible());
        parametersTable.rows().invalidate().draw();
    };

    $scope.columnFilterFn = function(column) {
        if (column.name === 'Action') {
            return false;
        }
        return true;
    };

    angular.element(document).ready(function () {
        parametersTable = $('#parametersTable').DataTable({
            "aLengthMenu": [[25, 50, 100, -1], [25, 50, 100, "All"]],
            "iDisplayLength": 25,
            "aaSorting": [],
            "columnDefs": [
                { "width": "50%", "targets": 2 }
            ],
            "columns": $scope.columns,
            "initComplete": function(settings, json) {
                try {
                    // need to create the show/hide column checkboxes and bind to the current visibility
                    $scope.columns = JSON.parse(localStorage.getItem('DataTables_parametersTable_/')).columns;
                } catch (e) {
                    console.error("Failure to retrieve required column info from localStorage (key=DataTables_parametersTable_/):", e);
                }
            }
        });
    });

};

TableParametersController.$inject = ['parameters', '$scope', '$state', '$uibModal', '$window', 'locationUtils', 'parameterService', 'profileService', 'messageModel'];
module.exports = TableParametersController;
