/*


 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.

 */

var NumberUtils = function($filter) {

	var k = 1000,
		sizes = ['B', 'Kb', 'Mb', 'Gb', 'Tb', 'Pb'];

	this.addCommas = function(nStr)
	{
		nStr += '';
		x = nStr.split('.');
		x1 = x[0];
		x2 = x.length > 1 ? '.' + x[1] : '';
		var rgx = /(\d+)(\d{3})/;
		while (rgx.test(x1)) {
			x1 = x1.replace(rgx, '$1' + ',' + '$2');
		}
		return x1 + x2;
	};

	/**
	 * This function takes big scary kilobit numbers and 'shrinks' them to a friendly version
	 * i.e. 10,000 kilobits is easier read as 10 megabits...
	 */
	this.shrink = function(kilounits) {
		if (!angular.isNumber(kilounits) || kilounits == 0) return [ 0, 'Kb' ];
		var units = kilounits * 1000;
		var i = Math.floor(Math.log(units) / Math.log(k));
		if (i < 1) { i = 1 } // kilobits is the lowest we will go
		if (i > 5) { i = 5 } // petabits is the highest we will go
		return [ Math.round((units / Math.pow(k, i)) * 100) / 100, sizes[i] ];
	};

	this.convertTo = function(kilounits, size) {
		if (!angular.isNumber(kilounits)) return null;
		if (kilounits == 0) return 0;
		var units = kilounits * 1000;
		var i = sizes.indexOf(size);
		if (i == -1) {
			return 0;
		}
		return Math.round((units / Math.pow(k, i)) * 100) / 100;
	};

	this.average = function(arr)
	{
		if (!angular.isArray(arr) || arr.length == 0 ) return 0;
		return arr.reduce(function(memo, num)  {
				return memo + num;
			}) / arr.length;
	}

	this.ratio = function(numerator, denominator)
	{
		if (numerator === 0 || denominator === 0) {
			return 'N/A';
		} else {
			return $filter('number')(numerator/denominator, 2) + ':1';
		}
	}

};

NumberUtils.$inject = ['$filter'];
module.exports = NumberUtils;
