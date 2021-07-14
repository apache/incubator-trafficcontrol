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

var FormEditDeliveryServiceRequestController = function(deliveryServiceRequest, dsCurrent, origin, topologies, type, types, $scope, $state, $stateParams, $controller, $uibModal, $anchorScroll, $q, $location, locationUtils, deliveryServiceService, deliveryServiceRequestService, messageModel, userModel) {

	$scope.dsRequest = deliveryServiceRequest[0];

	// extends the FormDeliveryServiceController to inherit common methods
	angular.extend(this, $controller('FormDeliveryServiceController', { deliveryService: $scope.dsRequest.requested || $scope.dsRequest.original, dsCurrent: dsCurrent, origin: origin, topologies: topologies, type: type, types: types, $scope: $scope }));

	$scope.changeType = $scope.dsRequest.changeType;

	$scope.requestStatus = $scope.dsRequest.status;

	$scope.deliveryServiceName = angular.copy(($scope.dsRequest.requested) ? $scope.dsRequest.requested.xmlId : $scope.dsRequest.original.xmlId);

	$scope.advancedShowing = true;

	$scope.settings = {
		isNew: false,
		isRequest: true,
		saveLabel: 'Update Request',
		deleteLabel: 'Delete Request'
	};

	$scope.saveable = function() {
		return $scope.dsRequest.changeType != 'delete' && ($scope.dsRequest.status == 'draft' || $scope.dsRequest.status == 'submitted');
	};

	$scope.deletable = function() {
		return ($scope.dsRequest.status == 'draft' || $scope.dsRequest.status == 'submitted');
	};

	$scope.fulfillable = function() {
		return $scope.dsRequest.status == 'submitted';
	};

	$scope.open = function() {
		return ($scope.dsRequest.status == 'draft' || $scope.dsRequest.status == 'submitted');
	};

	$scope.magicNumberLabel = function(collection, magicNumber) {
		const item = collection.find(function(x) {return x.value === magicNumber;});
		return item.label;
	};

	$scope.viewComments = function() {
		$location.path($location.path() + '/comments');
	};

	$scope.editStatus = function(status) {
		var params = {
			title: 'Change Delivery Service Request Status',
			message: "Are you sure you want to change the status of the delivery service request to '" + status + "'?"
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
		modalInstance.result.then(function() {
			deliveryServiceRequestService.updateDeliveryServiceRequestStatus($scope.dsRequest.id, status).
				then(function() {
					$state.reload();
					messageModel.setMessages([ { level: 'success', text: 'Delivery service request status was updated' } ], false);
			});
		}, function () {
			// do nothing
		});
	};

	var updateDeliveryServiceRequest = function(status) {
		var promises = [];
		// make sure the ds request is assigned to the user that is fulfilling the request
		promises.push(deliveryServiceRequestService.assignDeliveryServiceRequest($scope.dsRequest.id, userModel.user.username));
		// set the status if specified
		if (status) {
			promises.push(deliveryServiceRequestService.updateDeliveryServiceRequestStatus($scope.dsRequest.id, status));
		}
		return promises;
	};

	$scope.fulfillRequest = function(ds) {
		var params = {
			title: 'Delivery Service ' + $scope.changeType + ': ' + ds.xmlId,
			message: 'Are you sure you want to fulfill this delivery service request and ' + $scope.changeType + ' the ' + ds.xmlId + ' delivery service'
		};
		params['message'] += ($scope.changeType == 'create' || $scope.changeType == 'update') ? ' with these configuration settings?' : '?';
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
		modalInstance.result.then(function() {
			// create, update or delete the ds per the ds request
			if ($scope.changeType == 'create') {
				deliveryServiceService.createDeliveryService(ds).
					then(
						function(result) {
							updateDeliveryServiceRequest('pending'); // after a successful create, update the ds request, assignee and status
							messageModel.setMessages([ { level: 'success', text: 'Delivery Service [ ' + ds.xmlId + ' ] created' } ], true);
							locationUtils.navigateToPath('/delivery-services/' + result.data.response[0].id + '?type=' + result.data.response[0].type);
						},
						function(fault) {
							$anchorScroll(); // scrolls window to top
							messageModel.setMessages(fault.data.alerts, false);
						}
					);
			} else if ($scope.changeType == 'update') {
				deliveryServiceRequestService.updateDeliveryServiceRequestStatus($scope.dsRequest.id, 'pending').
					then(
						function(result) {
							deliveryServiceService.updateDeliveryService(ds).then(
								function (result) {
									updateDeliveryServiceRequest(); // after a successful ds update, update the assignee
									messageModel.setMessages([{
										level: 'success',
										text: 'Delivery Service [ ' + ds.xmlId + ' ] updated'
									}], true);
									locationUtils.navigateToPath('/delivery-services/' + result.data.response[0].id + '?type=' + result.data.response[0].type);
								},
								function (fault) {
									$anchorScroll(); // scrolls window to top
									messageModel.setMessages(fault.data.alerts, false);
								}
							)
						},
						function (fault) {
							$anchorScroll(); // scrolls window to top
							messageModel.setMessages(fault.data.alerts, false);
						}
				);
			} else if ($scope.changeType == 'delete') {
				// and we're going to ask even again if they really want to delete but this time they need to enter the ds name to confirm the delete
				params = {
					title: 'Delete Delivery Service: ' + ds.xmlId,
					key: ds.xmlId
				};
				modalInstance = $uibModal.open({
					templateUrl: 'common/modules/dialog/delete/dialog.delete.tpl.html',
					controller: 'DialogDeleteController',
					size: 'md',
					resolve: {
						params: function () {
							return params;
						}
					}
				});
				modalInstance.result.then(function() {
					deliveryServiceService.deleteDeliveryService(ds).
						then(
							function() {
								const promises = updateDeliveryServiceRequest('pending'); // after a successful delete, update the ds request, assignee and status and navigate to ds requests page
								$q.all(promises)
									.then(
										function() {
											messageModel.setMessages([ { level: 'success', text: 'Delivery service [ ' + ds.xmlId + ' ] deleted' } ], true);
											locationUtils.navigateToPath('/delivery-service-requests');
										});
							},
							function(fault) {
								$anchorScroll(); // scrolls window to top
								messageModel.setMessages(fault.data.alerts, false);
							}
						);
				}, function () {
					// do nothing
				});
			}
		}, function () {
			// do nothing
		});
	};

	$scope.save = function(deliveryService) {
		var params = {
			title: 'Update Delivery Service Request',
			statusMessage: 'Please select the status of your delivery service request.',
			commentMessage: 'Why is this request being changed?'
		};
		var modalInstance = $uibModal.open({
			templateUrl: 'common/modules/dialog/deliveryServiceRequest/dialog.deliveryServiceRequest.tpl.html',
			controller: 'DialogDeliveryServiceRequestController',
			size: 'md',
			resolve: {
				params: function () {
					return params;
				},
				statuses: function() {
					return [
						{ id: $scope.DRAFT, name: 'Save as Draft' },
						{ id: $scope.SUBMITTED, name: 'Submit for Review / Deployment' }
					];
				}
			}
		});
		modalInstance.result.then(function(options) {
			$scope.dsRequest.status = (options.status.id == $scope.SUBMITTED) ? 'submitted' : 'draft';
			$scope.dsRequest.requested = deliveryService;

			deliveryServiceRequestService.updateDeliveryServiceRequest($scope.dsRequest.id, $scope.dsRequest).
				then(
					function() {
						var comment = {
							deliveryServiceRequestId: $scope.dsRequest.id,
							value: options.comment
						};
						deliveryServiceRequestService.createDeliveryServiceRequestComment(comment).
							then(
								function() {
									const xmlId = ($scope.dsRequest.requested) ? $scope.dsRequest.requested.xmlId : $scope.dsRequest.original.xmlId;
									messageModel.setMessages([ { level: 'success', text: 'Updated delivery service request for ' + xmlId + ' and set status to ' + $scope.dsRequest.status } ], false);
									$anchorScroll(); // scrolls window to top
									$state.reload();
								}
							);
					}
				);
		}, function () {
			// do nothing
		});
	};

	$scope.confirmDelete = function(deliveryService) {
		var params = {
			title: 'Delete ' + deliveryService.xmlId + ' ' + $scope.dsRequest.changeType + ' request?',
			key: deliveryService.xmlId + ' request'
		};
		var modalInstance = $uibModal.open({
			templateUrl: 'common/modules/dialog/delete/dialog.delete.tpl.html',
			controller: 'DialogDeleteController',
			size: 'md',
			resolve: {
				params: function () {
					return params;
				}
			}
		});
		modalInstance.result.then(function() {
			deliveryServiceRequestService.deleteDeliveryServiceRequest($stateParams.deliveryServiceRequestId).
				then(function() {
					messageModel.setMessages([ { level: 'success', text: 'Delivery service request was deleted' } ], true);
					locationUtils.navigateToPath('/delivery-service-requests');
				});
		}, function () {
			// do nothing
		});
	};

};

FormEditDeliveryServiceRequestController.$inject = ['deliveryServiceRequest', 'dsCurrent', 'origin', 'topologies', 'type', 'types', '$scope', '$state', '$stateParams', '$controller', '$uibModal', '$anchorScroll', '$q', '$location', 'locationUtils', 'deliveryServiceService', 'deliveryServiceRequestService', 'messageModel', 'userModel'];
module.exports = FormEditDeliveryServiceRequestController;
