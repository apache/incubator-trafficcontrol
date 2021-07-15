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
/** @typedef { import('../agGrid/CommonGridController').CGC } CGC */

var TableNotificationsController = function(tableName, notifications, filter, $scope, $state, $uibModal, $document, dateUtils, cdnService) {
    /** @type CGC.ColumnDefinition */
	$scope.columns = [
		{
			headerName: "Created (UTC)",
			field: "lastUpdated",
			hide: false,
			filter: "agDateColumnFilter",
		},
		{
			headerName: "User",
			field: "user",
			hide: false
		},
		{
			headerName: "CDN",
			field: "cdn",
			hide: false
		},
		{
			headerName: "Notification",
			field: "notification",
			hide: false
		}
	];

	/** @type CGC.ContextMenuOption */
	$scope.contextMenuOptions = [{
		text: "Delete Notification",
		type: 1,
		onClick: function(row) {
			$scope.confirmDeleteNotification(row);
		}
	}];

	/** @type CGC.DropDownOption */
	$scope.dropDownOptions = [{
		text: "Create Notification",
		type: 1,
		onClick: function(row) {
			$scope.selectCDNandCreateNotification();
		}
	}];

	/** @type CGC.GridSettings */
	$scope.gridOptions = {
		refreshable: true
	};

	/** All of the notifications - lastUpdated fields converted to actual Date */
	$scope.notifications = notifications.map(
		function(x) {
			x.lastUpdated = x.lastUpdated ? new Date(x.lastUpdated.replace("+00", "Z")) : x.lastUpdated;
			return x;
		});

	$scope.selectCDNandCreateNotification = function() {
		const params = {
			title: 'Create Notification',
			message: "Please select a CDN"
		};
		const modalInstance = $uibModal.open({
			templateUrl: 'common/modules/dialog/select/dialog.select.tpl.html',
			controller: 'DialogSelectController',
			size: 'md',
			resolve: {
				params: function () {
					return params;
				},
				collection: function(cdnService) {
					return cdnService.getCDNs();
				}
			}
		});
		modalInstance.result.then(function(cdn) {
			$scope.createNotification(cdn);
		}, function () {
			// do nothing
		});
	};

	$scope.createNotification = function(cdn) {
		const params = {
			title: 'Create ' + cdn.name + ' Notification',
			message: 'What is the content of your notification for the ' + cdn.name + ' CDN?'
		};
		const modalInstance = $uibModal.open({
			templateUrl: 'common/modules/dialog/input/dialog.input.tpl.html',
			controller: 'DialogInputController',
			size: 'md',
			resolve: {
				params: function () {
					return params;
				}
			}
		});
		modalInstance.result.then(function(notification) {
			cdnService.createNotification(cdn, notification).
			then(
				function() {
					$state.reload();
				}
			);
		}, function () {
			// do nothing
		});
	};

	$scope.confirmDeleteNotification = function(notification) {
		const params = {
			title: 'Delete Notification',
			message: 'Are you sure you want to delete the notification for the ' + notification.cdn + ' CDN? This will remove the notification from the view of all users.'
		};
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
			cdnService.deleteNotification({ id: notification.id }).
				then(
					function() {
						$state.reload();
					}
				);
		}, function () {
			// do nothing
		});
	};
};

TableNotificationsController.$inject = ['tableName', 'notifications', 'filter', '$scope', '$state', '$uibModal', '$document', 'dateUtils', 'cdnService'];
module.exports = TableNotificationsController;
