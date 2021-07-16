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

var TableUsersController = function(users, $scope, $state, dateUtils, locationUtils) {

    let usersTable;

    $scope.users = users;

    $scope.relativeLoginTime = dateUtils.relativeLoginTime;

    $scope.columns = [
        { "name": "Full Name", "visible": true, "searchable": true },
        { "name": "Username", "visible": true, "searchable": true },
        { "name": "Email", "visible": true, "searchable": true },
        { "name": "Tenant", "visible": true, "searchable": true },
        { "name": "Role", "visible": true, "searchable": true },
        { "name": "Registration Sent", "visible": false, "searchable": true },
        { "name": "Last Authenticated", "visible": false, "searchable": false },
        { "name": "Change Log Count", "visible": false, "searchable": false },
    ];

    $scope.editUser = function(id) {
        locationUtils.navigateToPath('/users/' + id);
    };

    $scope.create = function() {
        locationUtils.navigateToPath('/users/new');
    };

    $scope.register = function() {
        locationUtils.navigateToPath('/users/register');
    };

    $scope.refresh = function() {
        $state.reload(); // reloads all the resolves for the view
    };

    $scope.toggleVisibility = function(colName) {
        const col = usersTable.column(colName + ':name');
        col.visible(!col.visible());
        usersTable.rows().invalidate().draw();
    };

    angular.element(document).ready(function () {
        usersTable = $('#usersTable').DataTable({
            "aLengthMenu": [[25, 50, 100, -1], [25, 50, 100, "All"]],
            "iDisplayLength": 25,
            "aaSorting": [],
            "columns": $scope.columns,
            "initComplete": function() {
                try {
                    // need to create the show/hide column checkboxes and bind to the current visibility
                    $scope.columns = JSON.parse(localStorage.getItem('DataTables_usersTable_/')).columns;
                } catch (e) {
                    console.error("Failure to retrieve required column info from localStorage (key=DataTables_usersTable_/):", e);
                }
            }
        });
    });

};

TableUsersController.$inject = ['users', '$scope', '$state', 'dateUtils', 'locationUtils'];
module.exports = TableUsersController;
