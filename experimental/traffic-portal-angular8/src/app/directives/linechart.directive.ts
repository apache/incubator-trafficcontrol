/*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/
import { AfterViewInit, Directive, ElementRef, OnDestroy } from "@angular/core";

import { Observable, Subscription } from "rxjs";

import { DataSet } from "../models/data";

import { Chart } from "chart.js"; // TODO: use plotly instead for WebGL-capabale browsers?

/**
 * LineChartType enumerates the valid types of charts.
 */
export enum LineChartType {
	/**
	 * Plots category proportions.
	 */
	Category = "category",
	/**
	 * Scatter plots.
	 */
	Linear = "linear",
	/**
	 * Logarithmic-scale scatter plots.
	 */
	Logarithmic = "logarithmic",
	/**
	 * Time-series data.
	 */
	Time = "time"
}

/**
 * LinechartDirective decorates canvases by creating a rendering context for
 * ChartJS charts.
 */
@Directive({
	inputs: [
		"chartTitle",
		"chartLabels",
		"chartDataSets",
		"chartType",
		"chartXAxisLabel",
		"chartYAxisLabel",
		"chartLabelCallback",
		"chartDisplayLegend"
	],
	selector: "[linechart]",
})
export class LinechartDirective implements AfterViewInit, OnDestroy {

	private ctx: CanvasRenderingContext2D; // | WebGLRenderingContext;
	private chart: Chart;

	/** The title of the chart. */
	public chartTitle?: string;
	/** Labels for the datasets. */
	public chartLabels?: any[];
	/** Data to be plotted by the chart. */
	public chartDataSets: Observable<DataSet[]>;
	/** The type of the chart. */
	public chartType?: LineChartType;
	/** A label for the X-axis of the chart. */
	public chartXAxisLabel?: string;
	/** A label for the Y-axis of the chart. */
	public chartYAxisLabel?: string;
	/** A callback for the label of each data point, to be optionally provided. */
	public chartLabelCallback?: (v: any, i: number, va: any[]) => any;
	/** Whether or not to display the chart's legend. */
	public chartDisplayLegend?: boolean;

	private subscription: Subscription;
	private opts: any;

	constructor (private readonly element: ElementRef) { }

	/**
	 * Initializes the chart using the input data.
	 */
	public ngAfterViewInit (): void {
		if (this.element.nativeElement === null) {
			console.warn("Use of DOM directive in non-DOM context!");
			return;
		}

		if (!(this.element.nativeElement instanceof HTMLCanvasElement)) {
			throw new Error("[linechart] Directive can only be used on a canvas!");
		}

		const ctx = (this.element.nativeElement as HTMLCanvasElement).getContext("2d", {alpha: false});
		if (!ctx) {
			console.error("Failed to get 2D context for chart canvas");
			return;
		}
		this.ctx = ctx;

		if (!this.chartType) {
			this.chartType = LineChartType.Linear;
		}

		if (this.chartDisplayLegend === null || this.chartDisplayLegend === undefined) {
			this.chartDisplayLegend = false;
		}

		this.opts = {
			data: {
				datasets: [],
			},
			options: {
				legend: {
					display: true
				},
				scales: {
					xAxes: [{
						display: true,
						scaleLabel: {
							display: this.chartXAxisLabel ? true : false,
							labelString: this.chartXAxisLabel
						},
						type: this.chartType,
					}],
					yAxes: [{
						display: true,
						scaleLabel: {
							display: this.chartYAxisLabel ? true : false,
							labelString: this.chartYAxisLabel
						},
						ticks: {
							suggestedMin: 0
						}
					}]
				},
				title: {
					display: this.chartTitle ? true : false,
					text: this.chartTitle
				},
				tooltips: {
					intersect: false,
					mode: "x"
				}
			},
			type: "line"
		};

		this.subscription = this.chartDataSets.subscribe(
			(data: DataSet[]) => {
				this.dataLoad(data);
			},
			(e: Error) => {
				this.dataError(e);
			}
		);
	}

	private destroyChart(): void {
		if (this.chart) {
			this.chart.clear();
			this.chart.destroy();
			this.chart = null;
			this.ctx.clearRect(0, 0, this.element.nativeElement.width, this.element.nativeElement.height);
			this.opts.data = {datasets: [], labels: []};
		}
	}


	private dataLoad(data: DataSet[]): void {
		this.destroyChart();

		if (data === null || data === undefined || data.some(x => x === null)) {
			this.noData();
			return;
		}

		this.opts.data.datasets = data;

		this.chart = new Chart(this.ctx, this.opts);
	}

	private dataError(e: Error): void {
		this.destroyChart();
		this.ctx.font = "30px serif";
		this.ctx.fillStyle = "black";
		this.ctx.textAlign = "center";
		this.ctx.fillText("Error Fetching Data", this.element.nativeElement.width / 2, this.element.nativeElement.height / 2);
	}

	private noData(): void {
		this.destroyChart();
		this.ctx.font = "30px serif";
		this.ctx.fillStyle = "black";
		this.ctx.textAlign = "center";
		this.ctx.fillText("No Data", this.element.nativeElement.width / 2, this.element.nativeElement.height / 2);
	}

	/** Cleans up chart resources on element destruction. */
	public ngOnDestroy(): void {
		this.destroyChart();

		if (this.subscription) {
			this.subscription.unsubscribe();
		}
	}

}
