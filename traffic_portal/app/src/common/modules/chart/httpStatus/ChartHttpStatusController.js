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

var ChartHttpStatusController = function(deliveryService, $scope, $state, $timeout, $filter, $q, $interval, deliveryServiceService, deliveryServiceStatsService, dateUtils, locationUtils, numberUtils, propertiesModel) {

	var chartSeries,
		chartOptions;

	var chartInterval,
		autoRefresh = propertiesModel.properties.deliveryServices.charts.autoRefresh;

	var status2xxChartData = [],
		status3xxChartData = [],
		status4xxChartData = [],
		status5xxChartData = [];

	var refreshHttpStatus = function() {
		registerResizeListener();
		getChartData($scope.deliveryService.xmlId, moment().subtract(1, 'days'), moment().subtract(10, 'seconds'));
	};

	var getChartData = function(xmlId, start, end) {
		var promises = [];

		promises.push(deliveryServiceStatsService.getHttpStatusByGroup(xmlId, '2xx', start, end));
		promises.push(deliveryServiceStatsService.getHttpStatusByGroup(xmlId, '3xx', start, end));
		promises.push(deliveryServiceStatsService.getHttpStatusByGroup(xmlId, '4xx', start, end));
		promises.push(deliveryServiceStatsService.getHttpStatusByGroup(xmlId, '5xx', start, end));

		$q.all(promises)
			.then(
				function(responses) {
					// set date range text
					$scope.dateRangeText = dateUtils.dateFormat(start.toDate(), "UTC: ddd mmm d yyyy H:MM:ss tt (Z)") + ' to ' + dateUtils.dateFormat(end.toDate(), "UTC: ddd mmm d yyyy H:MM:ss tt (Z)");

					// set chart data
					status2xxChartData = (responses[0].series) ? buildHttpStatusChartData(responses[0], start) : status2xxChartData;
					status3xxChartData = (responses[1].series) ? buildHttpStatusChartData(responses[1], start) : status3xxChartData;
					status4xxChartData = (responses[2].series) ? buildHttpStatusChartData(responses[2], start) : status4xxChartData;
					status5xxChartData = (responses[3].series) ? buildHttpStatusChartData(responses[3], start) : status5xxChartData;

					// set summary data
					$scope.summaryData2xx = responses[0].summary;
					$scope.summaryData3xx = responses[1].summary;
					$scope.summaryData4xx = responses[2].summary;
					$scope.summaryData5xx = responses[3].summary;

					$timeout(function () {
						buildChart(status2xxChartData, status3xxChartData, status4xxChartData, status5xxChartData);
					}, 100);
				},
				function(fault) {
					buildChart([], [], [], []);
				});
	};


	var buildHttpStatusChartData = function(result, start) {
		const normalizedData = new Array();
		const series = result.series;

		if (angular.isDefined(series)) {
			series.values.forEach(function(seriesItem) {
				if (moment(seriesItem[0]).isSame(start) || moment(seriesItem[0]).isAfter(start)) {
					if (_.isNumber(seriesItem[1])) {
						normalizedChartData.push([ moment(seriesItem[0]).valueOf(), seriesItem[1] ]);
					}
				}
			});
		}
		return normalizedData;
	};

	var buildChart = function(status2xxChartData, status3xxChartData, status4xxChartData, status5xxChartData) {

		chartOptions = {
			xaxis: {
				mode: "time",
				timezone: "browser",
				twelveHourClock: true,
				timeBase: "milliseconds"
			},
			yaxes: [
				{
					position: "left",
					axisLabel: "Success (2xx and 3xx)",
					axisLabelUseCanvas: true,
					axisLabelFontSizePixels: 12,
					axisLabelFontFamily: 'Verdana, Arial',
					axisLabelPadding: 3
				},
				{
					position: "right",
					axisLabel: "Client Error (4xx)",
					axisLabelUseCanvas: true,
					axisLabelFontSizePixels: 12,
					axisLabelFontFamily: 'Verdana, Arial',
					axisLabelPadding: 3
				},
				{
					position: "right",
					axisLabel: "Server Error (5xx)",
					axisLabelUseCanvas: true,
					axisLabelFontSizePixels: 12,
					axisLabelFontFamily: 'Verdana, Arial',
					axisLabelPadding: 3
				}
			],
			grid: {
				hoverable: true,
				axisMargin: 20
			},
			tooltip: {
				show: true,
				content: function(label, xval, yval, flotItem){
					var tooltipString = dateUtils.dateFormat(xval, "UTC: ddd mmm d yyyy H:MM:ss tt (Z)") + '<br>';
					tooltipString += '<span>' + label + ': ' + $filter('number')(yval, 0) + '</span><br>'
					return tooltipString;
				}
			}
		};

		chartSeries = [
			{ label: "2xx", yaxis: 1, color: "#91ca32", data: status2xxChartData },
			{ label: "3xx", yaxis: 1, color: "#5897fb", data: status3xxChartData },
			{ label: "4xx", yaxis: 2, color: "#6859a3", data: status4xxChartData },
			{ label: "5xx", yaxis: 3, color: "#a94442", data: status5xxChartData }
		];

		plotChart();

	};

	var createIntervals = function() {
		killIntervals();
		chartInterval = $interval(function() { refreshHttpStatus() }, propertiesModel.properties.deliveryServices.charts.refreshRateInMS );
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
			$.plot($("#ds-httpStatus-chart-" + $scope.deliveryService.id), chartSeries, chartOptions);
		}
	};

	$scope.deliveryService = deliveryService;

	$scope.summaryData2xx = {};
	$scope.summaryData3xx = {};
	$scope.summaryData4xx = {};
	$scope.summaryData5xx = {};

	$scope.dateRangeText;

	$scope.$on("$destroy", function() {
		killIntervals();
		unregisterResizeListener();
	});

	angular.element(document).ready(function () {
		refreshHttpStatus();
		if (autoRefresh) {
			createIntervals();
		}
	});

};

ChartHttpStatusController.$inject = ['deliveryService', '$scope', '$state', '$timeout', '$filter', '$q', '$interval', 'deliveryServiceService', 'deliveryServiceStatsService', 'dateUtils', 'locationUtils', 'numberUtils', 'propertiesModel'];
module.exports = ChartHttpStatusController;
