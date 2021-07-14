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

var WidgetDeliveryServicesController = function ($scope, $timeout, $filter, $q, $interval, deliveryServiceService, deliveryServiceStatsService, locationUtils, dateUtils, numberUtils, propertiesModel) {

	var interval,
		autoRefresh = propertiesModel.properties.dashboard.autoRefresh;

	$scope.unitSize = 'Gb';
	$scope.isRequested = false;
	$scope.isLoading = false;
	$scope.selectedIndex = 0;

	var createInterval = function() {
		killInterval();
		interval = $interval(function() { $scope.getChartData($scope.selectedDeliveryService); }, propertiesModel.properties.dashboard.deliveryServiceGbps.refreshRateInMS );
	};

	var killInterval = function() {
		if (angular.isDefined(interval)) {
			$interval.cancel(interval);
			interval = undefined;
		}
	};

	$scope.getChartData = function (ds, idx) {
		if (ds.xmlId != $scope.selectedDeliveryService.xmlId) {
			$scope.selectedIndex = idx;
			$scope.isRequested = true;
			$scope.isLoading = true;
			$scope.resetChart();
			if (autoRefresh) {
				createInterval();
			}
		}

		var start = moment().subtract(1, 'days');
		var end = moment().subtract(10, 'seconds');
		$scope.selectedDeliveryService = ds;
		$scope.dateRangeText = dateUtils.dateFormat(start.toDate(), "UTC: ddd mmm d yyyy H:MM:ss tt (Z)") + ' to ' + dateUtils.dateFormat(end.toDate(), "UTC: ddd mmm d yyyy H:MM:ss tt (Z)");

		var promises = [];
		promises.push(deliveryServiceStatsService.getBPS(ds.xmlId, start, end));

		$q.all(promises)
			.then(
				function (responses) {
					if (responses[0].series) {
						$scope.chartData = (responses[0].series) ? $scope.buildBandwidthChartData(responses[0].series, start) : $scope.chartData;
						$scope.finalData = {
							chart: [],
							labels: []
						};
						for (var i = 0; i < $scope.chartData.length; i++) {
							$scope.finalData.chart.push($scope.chartData[i][1]);
							$scope.finalData.labels.push($scope.chartData[i][0]);
						}
						$timeout(function () {
							$scope.buildChart();
						}, 100);
					} else {
						$scope.isLoading = false;
						$scope.finalData.chart = [0,0];
						var d = new Date();
						var c = new Date();
						c.setDate(c.getDate()-1);
						$scope.finalData.labels = [moment( c ).valueOf(), moment( d ).valueOf()];
						$scope.buildChart();
					}
				},
				function (fault) {
					$scope.hasChart = false; // build an empty chart);
				});
	};

	$scope.buildBandwidthChartData = function (series, start) {
		const normalizedChartData = [];
		if (angular.isDefined(series)) {
			series.values.forEach(function(seriesItem)  {
				if (moment(seriesItem[0]).isSame(start) || moment(seriesItem[0]).isAfter(start)) {
					normalizedChartData.push([moment(seriesItem[0]).valueOf(), numberUtils.convertTo(seriesItem[1], $scope.unitSize)]); // converts data to appropriate unit
				}
			});
		}
		return normalizedChartData;
	};

	$scope.buildChart = function () {
		$scope.isLoading = false;
		$scope.dsChart.labels = $scope.finalData.labels;
		$scope.dsChart.series = ['Bandwidth'];
		$scope.dsChart.data = $scope.finalData.chart;
		$scope.dsChart.options = {
			responsive: true,
			maintainAspectRatio: false,
			animation: {
				duration: 0
			},
			elements: {
				point: {
					radius: 0
				},
				line: {
					fill: false,
					tension: 0,
					borderColor: '#3498DB',
					borderWidth: 2
				},
				rectangle: {
					borderWidth: 2
				}
			},
			tooltips: {
				mode: 'nearest',
				intersect: false,
				position: 'nearest',
				backgroundColor: '#FFFFFF',
				xPadding: 6,
				yPadding: 6,
				caretPadding: 6,
				titleFontColor: '#73879C',
				bodyFontColor: '#73879C',
				borderColor: '#7d7d7d',
				borderWidth: 1,
				displayColors: false,
				callbacks: {
					label: function (tooltipItems, val) {
						return 'Bandwidth: ' + tooltipItems.yLabel + ' Gbps';
					}
				}
			},
			scales: {
				xAxes: [{
					type: 'time',
					time: {
						parser: function (utcMoment) {
							return moment(utcMoment).utcOffset('+0000');
						},
						unit: 'hour',
						displayFormats: {
							hour: 'HH:mm'
						},
						tooltipFormat: 'ddd MMM D HH:mm:ss (UTC)'
					},
					ticks: {
						callback: function (v) {
							return '  ' + moment(v, 'HH:mm').utcOffset('+0000').format('HH:mm') + '  ';
						},
						autoSkip: true,
					}
				}, {
					position: 'top',
					ticks: {
						display: false
					},
					gridLines: {
						display: false,
						drawTicks: false
					}
				}],
				yAxes: [
					{
						id: 'Bandwidth',
						type: 'linear',
						display: true,
						position: 'left',
						scaleLabel: {
							display: true,
							labelString: 'Bandwidth (Gbps)'
						}
					}, {
						position: 'right',
						ticks: {
							display: false
						},
						gridLines: {
							display: false,
							drawTicks: false
						}
					}]
			}
		};
	};

	$scope.resetChart = function () {
		$scope.finalData = {
			chart: [],
			labels: []
		};
		$scope.dsChart = {
			data: null,
			labels: null,
			series: null,
			options: null
		};
		$scope.chartData = [];
		$scope.selectedDeliveryService = {
			xmlId: ''
		};
	};

	var getDeliveryServices = function () {
		deliveryServiceService.getDeliveryServices()
			.then(function (result) {
				$scope.deliveryServices = result;
				$scope.dsCount = $scope.deliveryServices.length;
				for (var i = 0; i < $scope.deliveryServices.length; i++) {
					$scope.deliveryServices[i].idx = i;
				}
				$scope.getChartData($scope.deliveryServices[0], 0);
			});
	};

	$scope.navigateToCharts = function () {
		locationUtils.navigateToPath('/delivery-services/' + $scope.selectedDeliveryService.id + '/charts?type=' + $scope.selectedDeliveryService.type);
	};

	// pagination
	$scope.currentDeliveryServicesPage = 1;
	$scope.deliveryServicesPerPage = 10;

	$scope.navigateToPath = locationUtils.navigateToPath;

	$scope.$on("$destroy", function() {
		killInterval();
	});

	var init = function () {
		$scope.resetChart();
		getDeliveryServices();
		if (autoRefresh) {
			createInterval();
		}
	};
	init();
};

WidgetDeliveryServicesController.$inject = ['$scope', '$timeout', '$filter', '$q', '$interval', 'deliveryServiceService', 'deliveryServiceStatsService', 'locationUtils', 'dateUtils', 'numberUtils', 'propertiesModel'];
module.exports = WidgetDeliveryServicesController;
