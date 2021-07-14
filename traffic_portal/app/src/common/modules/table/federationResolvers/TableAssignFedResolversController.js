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

var TableAssignFedResolversController = function(federation, resolvers, assignedResolvers, $scope, $uibModalInstance) {

    var selectedResolvers = [];

    var addAll = function() {
        markVisibleResolvers(true);
    };

    var removeAll = function() {
        markVisibleResolvers(false);
    };

    var markVisibleResolvers = function(selected) {
        var visibleResolverIds = $('#fedResolversTable tr.resolver-row').map(
                function() {
                    return parseInt($(this).attr('id'));
                }).get();
        $scope.resolvers = resolvers.map(
            function(resolver)  {
                if (visibleResolverIds.includes(resolver.id)) {
                    resolver['selected'] = selected;
                }
                return resolver;
            }
        );
        updateSelectedCount();
    };

    var updateSelectedCount = function() {
        selectedResolvers = $scope.resolvers.filter(function(resolver)  { return resolver['selected'] === true; } );
        $('div.selected-count').html('<b>' + selectedResolvers.length + ' resolvers selected</b>');
    };

    $scope.federation = federation;

    $scope.resolvers = resolvers.map(
        function(resolver)  {
            const isAssigned = assignedResolvers.find(function(assignedResolver)  { return assignedResolver.id === resolver.id });
            if (isAssigned) {
                resolver['selected'] = true;
            }
            return resolver;
        }
    );

    $scope.selectAll = function($event) {
        var checkbox = $event.target;
        if (checkbox.checked) {
            addAll();
        } else {
            removeAll();
        }
    };

    $scope.onChange = function() {
        updateSelectedCount();
    };

    $scope.submit = function() {
        const selectedResolverIds = selectedResolvers.map(function(r) {return r.id;});
        $uibModalInstance.close(selectedResolverIds);
    };

    $scope.cancel = function () {
        $uibModalInstance.dismiss('cancel');
    };

    angular.element(document).ready(function () {
        var fedResolversTable = $('#fedResolversTable').dataTable({
            "scrollY": "60vh",
            "paging": false,
            "order": [[ 1, 'asc' ]],
            "dom": '<"selected-count">frtip',
            "columnDefs": [
                { 'orderable': false, 'targets': 0 },
                { "width": "5%", "targets": 0 }
            ],
            "stateSave": false
        });
        fedResolversTable.on( 'search.dt', function () {
            $("#selectAllCB").removeAttr("checked"); // uncheck the all box when filtering
        } );
        updateSelectedCount();
    });

};

TableAssignFedResolversController.$inject = ['federation', 'resolvers', 'assignedResolvers', '$scope', '$uibModalInstance'];
module.exports = TableAssignFedResolversController;
