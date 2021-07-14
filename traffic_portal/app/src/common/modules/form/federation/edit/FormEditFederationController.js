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

var FormEditFederationController = function(cdn, federation, resolvers, deliveryServices, federationDeliveryServices, $scope, $state, $controller, $uibModal, $anchorScroll, locationUtils, federationService, federationResolverService, messageModel) {

	// extends the FormFederationController to inherit common methods
	angular.extend(this, $controller('FormFederationController', { cdn: cdn, federation: federation, deliveryServices: deliveryServices, $scope: $scope }));

	var deleteFederation = function(fed) {
		federationService.deleteFederation(cdn.id, fed.id)
			.then(function() {
				locationUtils.navigateToPath('/cdns/' + cdn.id + '/federations');
			});
	};

	var createFederationResolver = function(fedRes) {
		federationResolverService.createFederationResolver(fedRes)
			.then(
				function(result) {
					messageModel.setMessages(result.data.alerts, false);
				},
				function(fault) {
					messageModel.setMessages(fault.data.alerts, false);
				}
			);
	};

	var assignFederationResolvers = function(fedId, fedResIds) {
		federationResolverService.assignFederationResolvers(fedId, fedResIds, true)
			.then(function() {
				$state.reload(); // reloads all the resolves for the view
			});
	};

	// lots of hacking going on here due to poor data model. i.e. there is a federation_deliveryservice table but a federation can have only one DS. grrr.
	$scope.federation['dsId'] = (federationDeliveryServices[0]) ? federationDeliveryServices[0].id : null;

	$scope.resolvers = resolvers;

	$scope.cname = angular.copy(federation.cname);

	$scope.settings = {
		isNew: false,
		saveLabel: 'Update'
	};

	$scope.save = function(fed) {
		federationService.updateFederation(cdn.name, fed).
			then(function() {
				$scope.cname = angular.copy(fed.cname);
				$anchorScroll(); // scrolls window to top
			});
	};

	$scope.confirmDeleteFederation = function(fed) {
		var params = {
			title: 'Delete Federation: ' + fed.cname,
			key: fed.cname
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
			deleteFederation(fed);
		}, function () {
			// do nothing
		});
	};

	$scope.confirmRemoveResolver = function(resolverToRemove) {
		var params = {
			title: 'Remove Federation Resolver: ' + resolverToRemove.ipAddress,
			message: 'Are you sure you want to remove this federation resolver from the ' + federation.cname + ' federation?'
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
			const resolvers = $scope.resolvers.filter( function(res)  { return res.id != resolverToRemove.id; });
			const resolverIds = resolvers.map( function(r)  {return r.id;});
			assignFederationResolvers($scope.federation.id, resolverIds)
		}, function () {
			// do nothing
		});
	};

	$scope.createResolver = function() {
		var modalInstance = $uibModal.open({
			templateUrl: 'common/modules/dialog/federationResolver/dialog.federationResolver.tpl.html',
			controller: 'DialogFederationResolverController',
			size: 'md',
			resolve: {
				federation: function() {
					return federation;
				},
				resolver: function() {
					return {};
				},
				types: function(typeService) {
					return typeService.getTypes({ useInTable: 'federation' });
				}
			}
		});
		modalInstance.result.then(function(resolver) {
			createFederationResolver(resolver);
		}, function () {
			// do nothing
		});
	};

	$scope.selectResolvers = function() {
		var modalInstance = $uibModal.open({
			templateUrl: 'common/modules/table/federationResolvers/table.assignFedResolvers.tpl.html',
			controller: 'TableAssignFedResolversController',
			size: 'lg',
			resolve: {
				federation: function() {
					return federation;
				},
				resolvers: function(federationResolverService) {
					return federationResolverService.getFederationResolvers();
				},
				assignedResolvers: function() {
					return resolvers;
				}
			}
		});
		modalInstance.result.then(function(selectedResolverIds) {
			assignFederationResolvers($scope.federation.id, selectedResolverIds);
		}, function () {
			// do nothing
		});
	};

};

FormEditFederationController.$inject = ['cdn', 'federation', 'resolvers', 'deliveryServices', 'federationDeliveryServices', '$scope', '$state', '$controller', '$uibModal', '$anchorScroll', 'locationUtils', 'federationService', 'federationResolverService', 'messageModel'];
module.exports = FormEditFederationController;
