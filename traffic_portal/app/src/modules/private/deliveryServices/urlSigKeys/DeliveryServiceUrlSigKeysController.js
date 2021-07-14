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

var DeliveryServiceUrlSigKeysController = function(deliveryService, urlSigKeys, $scope, $state, locationUtils, deliveryServiceService, deliveryServiceUrlSigKeysService, $uibModal) {
	$scope.deliveryService = deliveryService;
    //Here we take the unordered map of keys returned from riak:
    //"response": {
    //   "key9":"ZvVQNYpPVQWQV8tjQnUl6osm4y7xK4zD",
    //   "key6":"JhGdpw5X9o8TqHfgezCm0bqb9SQPASWL",
    //   "key8":"ySXdp1T8IeDEE1OCMftzZb9EIw_20wwq",
    //   "key0":"D4AYzJ1AE2nYisA9MxMtY03TPDCHji9C",
    //   "key3":"W90YHlGc_kYlYw5_I0LrkpV9JOzSIneI",
    //   "key12":"ZbtMb3mrKqfS8hnx9_xWBIP_OPWlUpzc",
    //   "key2":"0qgEoDO7sUsugIQemZbwmMt0tNCwB1sf",
    //   "key4":"aFJ2Gb7atmxVB8uv7T9S6OaDml3ycpGf",
    //   "key1":"wnWNR1mCz1O4C7EFPtcqHd0xUMQyNFhA",
    //   "key11":"k6HMzlBH1x6htKkypRFfWQhAndQqe50e",
    //   "key10":"zYONfdD7fGYKj4kLvIj4U0918csuZO0d",
    //   "key15":"3360cGaIip_layZMc_0hI2teJbazxTQh",
    //   "key5":"SIwv3GOhWN7EE9wSwPFj18qE4M07sFxN",
    //   "key13":"SqQKBR6LqEOzp8AewZUCVtBcW_8YFc1g",
    //   "key14":"DtXsu8nsw04YhT0kNoKBhu2G3P9WRpQJ",
    //   "key7":"cmKoIIxXGAxUMdCsWvnGLoIMGmNiuT5I"
    // }
    // and sort it based on the keys' label resulting in data looking like:
    //[{"label":"key1","urlSigKey":"wnWNR1mCz1O4C7EFPtcqHd0xUMQyNFhA"},{"label":"key2","urlSigKey":"0qgEoDO7sUsugIQemZbwmMt0tNCwB1sf"}...]

	$scope.urlSigKeys = Object.keys(urlSigKeys).map(function(key) {
			return {sortBy: parseInt(key.slice(3)), label: key, urlSigKey: urlSigKeys[key]};
	});

	$scope.generateUrlSigKeys = function() {
        var params = {
            title: 'Confirmation required',
            message: 'Are you sure you want to generate new URL signature keys for ' + deliveryService.xmlId + '?'
        };
        var modalInstance = $uibModal.open({
            templateUrl: 'common/modules/dialog/confirm/dialog.confirm.tpl.html',
            controller: 'DialogConfirmController',
            size: 'md',
            resolve: {
                params: function () {
                    return params;
                }
            }
        });
        modalInstance.result
            .then(
            function() {
                deliveryServiceUrlSigKeysService.generateUrlSigKeys(deliveryService.xmlId).then(
                function() {
                    $scope.refresh();
                });
            });
	};

	$scope.refresh = function() {
		$state.reload(); // reloads all the resolves for the view
	};

	$scope.selectCopyFromDS = function() {
        var params = {
            title: 'Copy URL Sig Keys to: ' + deliveryService.xmlId,
            message: "Please select a Delivery Service to copy from:",
            label: "xmlId"
        };
        var modalInstance = $uibModal.open({
            templateUrl: 'common/modules/dialog/select/dialog.select.tpl.html',
            controller: 'DialogSelectController',
            size: 'md',
            resolve: {
                params: function () {
                    return params;
                },
                collection: function(deliveryServiceService) {
                    return deliveryServiceService.getDeliveryServices({ signed: true })
                    .then(function(result){
                    	return result.fister(function(ds)  {
                            //you can't copy url sig keys from yourself
                    		return ds.id !== deliveryService.id;
                    	})
                    });
                }
            }
        });
        modalInstance.result.then(function(copyFromDs) {
            deliveryServiceUrlSigKeysService.copyUrlSigKeys(deliveryService.xmlId, copyFromDs.xmlId)
            .then(
           		function() {
            	$state.reload();
        	});
        });
    };

	$scope.navigateToPath = locationUtils.navigateToPath;

	angular.element(document).ready(function () {
		$('#urlSigKeysTable').dataTable({
			"aLengthMenu": [[25, 50, 100, -1], [25, 50, 100, "All"]],
			"iDisplayLength": 25,
			"aaSorting": []
		});
	});
};

DeliveryServiceUrlSigKeysController.$inject = ['deliveryService', 'urlSigKeys', '$scope', '$state', 'locationUtils', 'deliveryServiceService', 'deliveryServiceUrlSigKeysService', '$uibModal'];
module.exports = DeliveryServiceUrlSigKeysController;
