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

var TableServersController = function(tableName, servers, filter, $scope, $state, $uibModal, $window, dateUtils, locationUtils, serverUtils, cdnService, serverService, statusService, propertiesModel, messageModel, userModel, $document) {

	/**** Constants, scope data, etc. ****/

	/** The columns of the ag-grid table */
	$scope.columns = [
		{
			headerName: "Cache Group",
			field: "cachegroup",
			hide: false
		},
		{
			headerName: "CDN",
			field: "cdnName",
			hide: false
		},
		{
			headerName: "Domain",
			field: "domainName",
			hide: false
		},
		{
			headerName: "Host",
			field: "hostName",
			hide: false
		},
		{
			headerName: "HTTPS Port",
			field: "httpsPort",
			hide: true,
			filter: "agNumberColumnFilter"
		},
		{
			headerName: "Hash ID",
			field: "xmppId",
			hide: true
		},
		{
			headerName: "ID",
			field: "id",
			hide: true,
			filter: "agNumberColumnFilter"
		},
		{
			headerName: "ILO IP Address",
			field: "iloIpAddress",
			hide: true,
			cellRenderer: "httpsCellRenderer",
			onCellClicked: null
		},
		{
			headerName: "ILO IP Gateway",
			field: "iloIpGateway",
			hide: true
		},
		{
			headerName: "ILO IP Netmask",
			field: "iloIpNetmask",
			hide: true
		},
		{
			headerName: "ILO Username",
			field: "iloUsername",
			hide: true
		},
		{
			headerName: "Interface Name",
			field: "interfaceName",
			hide: true
		},
		{
			headerName: "IPv6 Address",
			field: "ip6Address",
			hide: false,
			cellRenderer: "sshCellRenderer",
			onCellClicked: null
		},
		{
			headerName: "IPv6 Gateway",
			field: "ip6Gateway",
			hide: true
		},
		{
			headerName: "Last Updated",
			field: "lastUpdated",
			hide: true,
			filter: "agDateColumnFilter",
			relative: true
		},
		{
			headerName: "Mgmt IP Address",
			field: "mgmtIpAddress",
			hide: true,
			cellRenderer: "sshCellRenderer",
			onCellClicked: null
		},
		{
			headerName: "Mgmt IP Gateway",
			field: "mgmtIpGateway",
			hide: true,
			filter: true
		},
		{
			headerName: "Mgmt IP Netmask",
			field: "mgmtIpNetmask",
			hide: true,
			filter: true
		},
		{
			headerName: "IPv4 Gateway",
			field: "ipGateway",
			hide: true,
			filter: true
		},
		{
			headerName: "IPv4 Address",
			field: "ipAddress",
			hide: false,
			filter: true,
			cellRenderer: "sshCellRenderer",
			onCellClicked: null
		},
		{
			headerName: "Network MTU",
			field: "interfaceMtu",
			hide: true,
			filter: "agNumberColumnFilter"
		},
		{
			headerName: "IPv4 Subnet",
			field: "ipNetmask",
			hide: true
		},
		{
			headerName: "Offline Reason",
			field: "offlineReason",
			hide: true
		},
		{
			headerName: "Phys Location",
			field: "physLocation",
			hide: true
		},
		{
			headerName: "Profile",
			field: "profile",
			hide: false
		},
		{
			headerName: "Rack",
			field: "rack",
			hide: true
		},
		{
			headerName: "Reval Pending",
			field: "revalPending",
			hide: true,
			filter: true,
			cellRenderer: "updateCellRenderer"
		},
		{
			headerName: "Router Hostname",
			field: "routerHostName",
			hide: true
		},
		{
			headerName: "Router Port Name",
			field: "routerPortName",
			hide: true
		},
		{
			headerName: "Status",
			field: "status",
			hide: false,
			tooltipValueGetter: function(params) {
				if (!params.value || !serverUtils.isOffline(params.value)) {
					return;
				}
				return params.value + ': ' + params.data.offlineReason;
			}
		},
		{
			headerName: "TCP Port",
			field: "tcpPort",
			hide: true
		},
		{
			headerName: "Type",
			field: "type",
			hide: false
		},
		{
			headerName: "Update Pending",
			field: "updPending",
			hide: false,
			filter: true,
			cellRenderer: "updateCellRenderer"
		},
		{
			headerName: "Status Last Updated",
			field: "statusLastUpdated",
			hide: true,
			filter: "agDateColumnFilter",
			relative: true
		}
	];

	/** All of the statuses (populated on init). */
	let statuses = [];

	/** @type CGC.DropDownOption[] */
	$scope.dropDownOptions = [{
		name: "createServerMenuItem",
		href: "#!/servers/new",
		text: "Create New Server",
		type: 2
	}, {
		type: 0
	}, {
		onClick: function (entry) {
			$scope.confirmCDNQueueServerUpdates(entry);
		},
		text: "Queue Server Updates",
		type: 1
	}, {
		onClick: function (entry) {
			$scope.confirmCDNClearServerUpdates(entry);
		},
		text: "Clear Server Updates",
		type: 1
	}];

	/** @type CGC.ContextMenuOption[] */
	$scope.contextMenuOptions = [
		{
			type: 2,
			getHref: function(entry) {
				return "#!/servers/" + entry.id;
			},
			getText: function (entry) {
				return "Open " + entry.hostName + " in New Tab";
			},
			newTab: true
		},
		{
			type: 2,
			getHref: function (entry) {
				return "http://" + entry.hostName + "." + entry.domainName;
			},
			text: "Navigate To Server FQDN"
		},
		{
			type: 0
		},
		{
			type: 2,
			getHref: function (entry) {
				return "#!/servers/" + entry.id;
			},
			text: "Edit"
		},
		{
			type: 1,
			onClick: function (entry) {
				$scope.confirmDelete(entry, null);
			},
            text: "Delete"
		},
		{
			type: 0
		},
		{
			type: 1,
			text: "Update Status",
			onClick: function (entry) {
				$scope.confirmStatusUpdate(entry, null);
			}
		},
		{
			type: 1,
			isDisabled: function(entry){
				return !$scope.isCache(entry) || entry.updPending;
			},
			onClick: function (entry){
			    $scope.queueServerUpdates(entry, null);
			},
			text: "Queue Server Updates",
		},
		{
			type: 1,
			isDisabled: function(entry){
				return !$scope.isCache(entry) || !entry.updPending;
			},
            onClick: function (entry){
				$scope.clearServerUpdates(entry,  null);
			},
			text: "Clear Server Updates",
		},
		{
			type: 0,
			shown: function (entry) {
				return $scope.showCharts;
			}
		},
		{
			type: 2,
			shown: function (entry) {
				return $scope.showCharts;
			},
			text: "Show Charts",
			getHref: function (entry) {
				return $scope.chartsBase + entry.hostName;
			}
		},
		{
			type: 0,
			shown: function (entry) {
				return $scope.isEdge(entry) || $scope.isCache(entry);
			}
		},
		{
			type: 2,
			shown: function (entry) {
				return $scope.isCache(entry);
			},
			text: "Manage Capabilities",
			getHref: function (entry) {
				return "#!/servers/" + entry.id + "/capabilities";
			}
		},
		{
			type: 2,
			shown: function (entry) {
				return $scope.isEdge(entry) || $scope.isCache(entry);
			},
            text: "Manage Delivery Services",
			getHref: function (entry) {
				return "#!/servers/" + entry.id + "/delivery-services";
			}
		},
	];

	/** All of the servers - lastUpdated fields converted to actual Dates, ip fields populated from interfaces */
	$scope.servers = servers.map(
		function(x) {
			x.lastUpdated = x.lastUpdated ? new Date(x.lastUpdated.replace("+00", "Z")) : x.lastUpdated;
			x.statusLastUpdated = x.statusLastUpdated ? new Date(x.statusLastUpdated): x.statusLastUpdated;
			Object.assign(x, serverUtils.toLegacyIPInfo(x.interfaces));
			return x;
	});

	/** The base URL to use for constructing links to server charts. */
	$scope.chartsBase = propertiesModel.properties.servers.charts.baseUrl;

	/** Options, configuration, data and callbacks for the ag-grid table. */
	/** @type CGC.GridSettings */
	$scope.gridOptions = {
		onRowClick: function(row) {
			locationUtils.navigateToPath("/servers/" + row.data.id);
		}
	};

	$scope.defaultData = {
		hostName: "",
		domainName: "",
		id: -1
	};

	/** These three functions are used by the context menu to determine what functionality to provide for a server. */
	$scope.isCache = serverUtils.isCache;
	$scope.isEdge = serverUtils.isEdge;
	$scope.isOrigin = serverUtils.isOrigin;

	/** Used by the context menu to determine whether or not to include links to server charts. */
	$scope.showCharts = propertiesModel.properties.servers.charts.show;

	/**** Miscellaneous scope functions ****/

	/** Reloads all 'resolve'd data for the view. */
	$scope.refresh = function() {
		$state.reload();
	};

	/**** Context menu functions ****/

	$scope.queueServerUpdates = function(server) {
		serverService.queueServerUpdates(server.id).then($scope.refresh);
	};

	$scope.clearServerUpdates = function(server) {
		serverService.clearServerUpdates(server.id).then($scope.refresh);
	};

	$scope.confirmDelete = function(server) {

		const params = {
			title: 'Delete Server: ' + server.hostName,
			key: server.hostName
		};
		const modalInstance = $uibModal.open({
			templateUrl: 'common/modules/dialog/delete/dialog.delete.tpl.html',
			controller: 'DialogDeleteController',
			size: 'md',
			resolve: {
				params: function () {
					return params;
				}
			}
		});
		modalInstance.result.then(
			function() {
				serverService.deleteServer(server.id).then(
					function(result) {
						messageModel.setMessages(result.alerts, false);
						$scope.refresh();
					},
					function(err) {
						// TODO: use template strings once the build can handle them.
						console.error("Error deleting server", server.hostName + "." + server.domainName, "(#" + String(server.id) + "):", err);
					}
				);
			},
			function() {
				// This is just a cancel event from closing the dialog, do nothing.
			}
		);
	};

	/**
	 * updateStatus sets the status of the given server to the given status value.
	 *
	 * @param {{id: number, offlineReason?: string}} status The numeric ID of the status to set along with a reason why it was set offline, if applicable.
	 * @param {{id: number}} server The server (or at least its numeric ID) which will have its status set.
	 */
	function updateStatus(status, server) {
		const params = {
			status: status.id,
			offlineReason: status.offlineReason
		};

		serverService.updateStatus(server.id, params).then(
			function(result) {
				messageModel.setMessages(result.data.alerts, false);
				$scope.refresh();
			},
			function(fault) {
				messageModel.setMessages(fault.data.alerts, false);
			}
		);
	}

	$scope.confirmStatusUpdate = function(server) {

		const modalInstance = $uibModal.open({
			templateUrl: 'common/modules/dialog/select/status/dialog.select.status.tpl.html',
			controller: 'DialogSelectStatusController',
			size: 'md',
			resolve: {
				server: function() {
					return server;
				},
				statuses: function() {
					return statuses;
				}
			}
		});
		modalInstance.result.then(
			function(status) {
				updateStatus(status, server);
			},
			function () {
				// this is just a cancel event from closing the dialog, do nothing
			}
		);
	};

	$scope.confirmCDNQueueServerUpdates = function(entry) {
		const params = {
			title: 'Queue Server Updates',
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
			cdnService.queueServerUpdates(cdn.id).then($scope.refresh);
		}, function () {
			// do nothing
		});
	};

	$scope.confirmCDNClearServerUpdates = function(entry) {
		const params = {
			title: 'Clear Server Updates',
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
			cdnService.clearServerUpdates(cdn.id).then($scope.refresh);
		}, function () {
			// do nothing
		});
	};


	this.$onInit = function(){
		statusService.getStatuses().then(
			function(result) {
				statuses = result;
			}
		);
	}
};

TableServersController.$inject = ['tableName', 'servers', 'filter', '$scope', '$state', '$uibModal', '$window', 'dateUtils', 'locationUtils', 'serverUtils', 'cdnService', 'serverService', 'statusService', 'propertiesModel', 'messageModel', "userModel", "$document"];
module.exports = TableServersController;
