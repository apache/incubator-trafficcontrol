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

var TableChangeLogsController = function(tableName, changeLogs, $scope, $state, $uibModal, dateUtils, propertiesModel, messageModel) {
	/** @type CGC.TitleButton */
	$scope.titleButton = {
		onClick: function() {
			$scope.changeDays();
		},
		getText: function() {
			return "[ last " + $scope.days + " day" + ($scope.days > 1 ? "s" : "") + " ]";
		}
	};

	/** @type CGC.ColumnDefinition */
	$scope.columns = [
		{
			headerName: "Occurred",
			field: "lastUpdated",
			hide: false,
			filter: "agDateColumnFilter",
			relative: true,
		},
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
			headerName: "Level",
			field: "level",
			hide: true
		},
		{
			headerName: "Message",
			field: "message",
			hide: false
		}
	];

	$scope.days = (propertiesModel.properties.changeLogs) ? propertiesModel.properties.changeLogs.days : 7;

	/** All of the change logs - lastUpdated fields converted to actual Date */
	$scope.changeLogs = changeLogs.map(
		function(x) {
			x.lastUpdated = x.lastUpdated ? new Date(x.lastUpdated.replace("+00", "Z")) : x.lastUpdated;
			return x;
		});

	/** @type CGC.GridSettings */
	$scope.gridOptions = {};

	/** Allows the user to change the number of days queried for change logs. */
	$scope.changeDays = function() {
		const params = {
			title: 'Change Number of Days',
			message: 'Enter the number of days of change logs you need access to (between 1 and 365).',
			type: 'number'
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
		modalInstance.result.then(function(days) {
			let numOfDays = parseInt(days, 10);
			if (numOfDays >= 1 && numOfDays <= 365) {
				propertiesModel.properties.changeLogs.days = numOfDays;
				$state.reload();
			} else {
				messageModel.setMessages([{level: 'error', text: 'Number of days must be between 1 and 365' }], false);
			}
		}, function () {
			console.log('Cancelled');
		});
	};
};

TableChangeLogsController.$inject = ['tableName', 'changeLogs', '$scope', '$state', '$uibModal', 'dateUtils', 'propertiesModel', 'messageModel'];
module.exports = TableChangeLogsController;
