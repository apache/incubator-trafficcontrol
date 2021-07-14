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

var WidgetCDNChartController = function(cdn, $scope, $timeout, $filter, $q, $interval, cdnService, cacheStatsService, dateUtils, locationUtils, numberUtils, propertiesModel) {

	var chartSeries,
		chartOptions;

	var chartInterval,
		autoRefresh = propertiesModel.properties.dashboard.autoRefresh;

	var bandwidthChartData = [],
		connectionsChartData = [];

	var getCDN = function(id) {
		cdnService.getCDN(id)
			.then(function(result) {
				$scope.cdn = result;
				registerResizeListener();
				getCurrentStats($scope.cdn.name);
				getChartData($scope.cdn.name, moment().subtract(1, 'days'), moment().subtract(10, 'seconds'));
			});
	};

	var getCurrentStats = function(cdnName) {
		cdnService.getCurrentStats()
			.then(function(result) {
				$scope.currentStats = result.currentStats.find(function(item)  {
					return item.cdn === cdnName;
				});
			});
	};

	var getChartData = function(cdnName, start, end) {
		var promises = [];

		// get cdn bandwidth
		promises.push(cacheStatsService.getBandwidth(cdnName, start, end));

		// get cdn connections
		promises.push(cacheStatsService.getConnections(cdnName, start, end));

		$q.all(promises)
			.then(
				function(responses) {
						// set chart data
						bandwidthChartData = (responses[0].series) ? buildBandwidthChartData(responses[0].series, start) : bandwidthChartData;
						connectionsChartData = (responses[1].series) ? buildConnectionsChartData(responses[1].series, start) : connectionsChartData;

					$timeout(function () {
						buildChart(bandwidthChartData, connectionsChartData);
					}, 100);
				},
				function(fault) {
					buildChart([], []); // build an empty chart
				});

	};

	var buildBandwidthChartData = function(series, start) {
		const normalizedChartData = [];

		if (angular.isDefined(series)) {
			series.values.forEach(function(seriesItem)  {
				if (moment(seriesItem[0]).isSame(start) || moment(seriesItem[0]).isAfter(start)) {
					normalizedChartData.push([ moment(seriesItem[0]).valueOf(), numberUtils.convertTo(seriesItem[1], $scope.unitSize) ]); // converts data to appropriate unit
				}
			});
		}

		return normalizedChartData;
	};

	var buildConnectionsChartData = function(series, start) {
		const normalizedChartData = [];

		if (angular.isDefined(series)) {
			series.values.forEach(function(seriesItem) {
				if (moment(seriesItem[0]).isSame(start) || moment(seriesItem[0]).isAfter(start)) {
					if (typeof(seriesItem[1]) === "number" || seriesItem[1] instanceof Number) {
						normalizedChartData.push([ moment(seriesItem[0]).valueOf(), seriesItem[1] ]);
					}
				}
			});
		}

		return normalizedChartData;
	};

	var buildChart = function(bandwidthChartData, connectionsChartData) {

		chartOptions = {
			xaxis: {
				mode: "time",
				timezone: "utc",
				twelveHourClock: false,
				timeBase: "milliseconds"
			},
			yaxes: [
				{
					position: "left",
					axisLabel: "Bandwidth (Gbps)",
					axisLabelUseCanvas: true,
					axisLabelFontSizePixels: 12,
					axisLabelFontFamily: 'Verdana, Arial',
					axisLabelPadding: 3
				},
				{
					position: "right",
					axisLabel: "Connections",
					axisLabelUseCanvas: true,
					axisLabelFontSizePixels: 12,
					axisLabelFontFamily: 'Verdana, Arial',
					axisLabelPadding: 3
				}
			],
			legend: {
				position: "nw"
			},
			grid: {
				hoverable: true,
				axisMargin: 20
			},
			tooltip: {
				show: true,
				content: function(label, xval, yval, flotItem){
					var tooltipString = dateUtils.dateFormat(xval, "UTC: ddd mmm d yyyy H:MM:ss tt (Z)") + '<br>';
					tooltipString += '<span>' + label + ': ' + $filter('number')(yval, 2) + (flotItem.series.label === "Bandwidth" ? ' Gbps' : '') + '</span><br>';
					return tooltipString;
				}
			}
		};

		chartSeries = [
			{ label: "Bandwidth", yaxis: 1, color: '#3498DB', data: bandwidthChartData },
			{ label: "Connections", yaxis: 2, color: '#E74C3C', data: connectionsChartData }
		];

		plotChart();

	};

	var createIntervals = function(cdnId) {
		killIntervals();
		chartInterval = $interval(function() { getCDN(cdnId) }, propertiesModel.properties.dashboard.cdnChart.refreshRateInMS );
	};

	var killIntervals = function() {
		if (angular.isDefined(chartInterval)) {
			$interval.cancel(chartInterval);
			chartInterval = undefined;
		}
	};

	var registerResizeListener = function() {
		$(window).bind("resize", plotChart);
	};

	var unregisterResizeListener = function() {
		$(window).unbind("resize", plotChart);
	};

	var plotChart = function() {
		if (chartOptions && chartSeries) {
			$.plot($("#cdn-chart-" + $scope.cdn.id), chartSeries, chartOptions);
		}
	};

	$scope.cdn;

	$scope.unitSize = 'Gb';

	$scope.randomId = '_' + Math.random().toString(36).substr(2, 9);

	$scope.navigateToPath = locationUtils.navigateToPath;

	$scope.$on("$destroy", function() {
		killIntervals();
		unregisterResizeListener();
	});

	angular.element(document).ready(function () {
		var cdnId;
		if (cdn) {
			cdnId = cdn.id;
		} else {
			// cdn wasn't passed in. we need to figure it out on our own
			cdnId = $('#' + $scope.randomId).closest('.chartContainer').data('cdnid');
		}
		getCDN(cdnId);
		if (autoRefresh) {
			createIntervals(cdnId);
		}
	});

};

WidgetCDNChartController.$inject = ['cdn', '$scope', '$timeout', '$filter', '$q', '$interval', 'cdnService', 'cacheStatsService', 'dateUtils', 'locationUtils', 'numberUtils', 'propertiesModel'];
module.exports = WidgetCDNChartController;
