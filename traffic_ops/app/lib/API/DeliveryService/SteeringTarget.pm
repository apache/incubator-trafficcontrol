package API::DeliveryService::SteeringTarget;
#
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
#
#

use UI::Utils;
use Utils::Tenant;

use Mojo::Base 'Mojolicious::Controller';
use Data::Dumper;
use JSON;
use MojoPlugins::Response;
use Validate::Tiny ':all';

sub index {
	my $self        = shift;
	my $steering_id = $self->param('id');
	my @data;

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	my $ds = $self->db->resultset('Deliveryservice')->find( { id => $steering_id } );
	if ( !defined($ds) ) {
		return $self->not_found();
	}
	elsif (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Steering delivery-service tenant is not available to the user.");
	}

	my %criteria;
	$criteria{'deliveryservice'} = $steering_id;

	my $steering_targets = $self->db->resultset('SteeringTarget')->search( \%criteria, { prefetch => [ 'deliveryservice', 'target', 'type' ] } );
	while ( my $row = $steering_targets->next ) {
		push(
			@data, {
				"deliveryServiceId" => $row->deliveryservice->id,
				"deliveryService"   => $row->deliveryservice->xml_id,
				"targetId"          => $row->target->id,
				"target"            => $row->target->xml_id,
				"value"             => $row->value,
				"typeId"            => $row->type->id,
				"type"              => $row->type->name,
			}
		);
	}
	$self->success( \@data );
}

sub show {
	my $self        = shift;
	my $steering_id = $self->param('id');
	my $target_id   = $self->param('target_id');

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	my $ds = $self->db->resultset('Deliveryservice')->find( { id => $steering_id } );

	if ( !defined($ds) ) {
		return $self->not_found();
	}
	elsif (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Steering delivery-service tenant is not available to the user.");
	}

	my %criteria;
	$criteria{'deliveryservice'} = $steering_id;
	$criteria{'target'}          = $target_id;

	my $steering_target = $self->db->resultset('SteeringTarget')->search( \%criteria, { prefetch => [ 'deliveryservice', 'target', 'type' ] } );
	my @data;
	while ( my $row = $steering_target->next ) {
		push(
			@data, {
				"deliveryServiceId" => $row->deliveryservice->id,
				"deliveryService"   => $row->deliveryservice->xml_id,
				"targetId"          => $row->target->id,
				"target"            => $row->target->xml_id,
				"value"             => $row->value,
				"typeId"            => $row->type->id,
				"type"              => $row->type->name,
			}
		);
	}
	$self->success( \@data );
}

sub update {
	my $self           = shift;
	my $steering_ds_id = $self->param('id');
	my $target_ds_id   = $self->param('target_id');
	my $params         = $self->req->json;

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	my $ds = $self->db->resultset('Deliveryservice')->find( { id => $steering_ds_id } );
	if ( !defined($ds) ) {
		return $self->not_found();
	}
	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Steering delivery-service tenant is not available to the user.");
	}
	my $target_ds = $self->db->resultset('Deliveryservice')->find( { id => $target_ds_id } );
	if ( !defined($target_ds) ) {
		return $self->not_found();
	}
	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $target_ds->tenant_id)) {
		return $self->forbidden("Forbidden. Steering target delivery-service tenant is not available to the user.");
	}

	$params->{targetId} = $target_ds_id; # to ensure that is_valid passes
	my ( $is_valid, $result ) = $self->is_target_valid($params, $ds);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my %criteria;
	$criteria{'deliveryservice'} = $steering_ds_id;
	$criteria{'target'}          = $target_ds_id;

	my $steering_target = $self->db->resultset('SteeringTarget')->search( \%criteria )->single();
	if ( !defined($steering_target) ) {
		return $self->not_found();
	}

	my $values = {
		value           => $params->{value},
		type            => $params->{typeId},
	};

	my $update = $steering_target->update($values);

	if ($update) {

		my $response;
		$response->{deliveryServiceId} = $update->deliveryservice->id;
		$response->{deliveryService}   = $update->deliveryservice->xml_id;
		$response->{targetId}          = $update->target->id;
		$response->{target}            = $update->target->xml_id;
		$response->{value}             = $update->value;
		$response->{typeId}            = $update->type->id;
		$response->{type}              = $update->type->name;

		&log( $self, "Updated steering target [ " . $target_ds_id . " ] for deliveryservice: " . $steering_ds_id, "APICHANGE" );

		return $self->success( $response, "Delivery service steering target update was successful." );
	}
	else {
		return $self->alert("Delivery service steering target update failed.");
	}

}

sub create {
	my $self   = shift;
	my $params = $self->req->json;
	my $steering_ds_id = $self->param('id');
	my $target_ds_id   = $params->{targetId};

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	my $ds = $self->db->resultset('Deliveryservice')->find( { id => $steering_ds_id } );
	if ( !defined($ds) ) {
		return $self->not_found();
	}
	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Steering delivery-service tenant is not available to the user.");
	}
	my $target_ds = $self->db->resultset('Deliveryservice')->find( { id => $target_ds_id } );
	if ( !defined($target_ds) ) {
		return $self->alert("Target delivery-service not found");
	}
	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $target_ds->tenant_id)) {
		return $self->alert("Steering target delivery-service tenant is not available to the user.");
	}

	my ( $is_valid, $result ) = $self->is_target_valid($params, $ds);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my %criteria;
	$criteria{'deliveryservice'} = $steering_ds_id;
	$criteria{'target'}          = $target_ds_id;

	my $existing = $self->db->resultset('SteeringTarget')->search( \%criteria )->single();
	if ( defined($existing) ) {
		return $self->alert('Steering target already exists');
	}

	my $values = {
		deliveryservice => $steering_ds_id,
		target          => $target_ds_id,
		value           => $params->{value},
		type            => $params->{typeId},
	};

	my $insert = $self->db->resultset('SteeringTarget')->create($values)->insert();
	if ($insert) {
		my @response;
		push( @response, {
				deliveryServiceId 	=> $insert->deliveryservice->id,
				deliveryService 	=> $insert->deliveryservice->xml_id,
				targetId          	=> $insert->target->id,
				target          	=> $insert->target->xml_id,
				value           	=> $insert->value,
				typeId            	=> $insert->type->id,
				type            	=> $insert->type->name,
			} );

		&log( $self, "Created steering target [ '" . $target_ds_id . "' ] for delivery service: " . $steering_ds_id, "APICHANGE" );

		return $self->success( \@response, "Delivery service target creation was successful." );
	}
	else {
		return $self->alert("Delivery service target creation failed.");
	}

}

sub delete {
	my $self           = shift;
	my $steering_ds_id = $self->param('id');
	my $target_ds_id   = $self->param('target_id');

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	my $ds = $self->db->resultset('Deliveryservice')->find( { id => $steering_ds_id } );
	if ( !defined($ds) ) {
		return $self->not_found();
	}
	elsif (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Steering delivery-service tenant is not available to the user.");
	}

	my $target_ds = $self->db->resultset('Deliveryservice')->find( { id => $target_ds_id } );
	if ( !defined($target_ds) ) {
		return $self->not_found();
	}
	elsif (!$tenant_utils->is_ds_resource_accessible($tenants_data, $target_ds->tenant_id)) {
		return $self->forbidden("Forbidden. Steering target delivery-service tenant is not available to the user.");
	}

	my $target = $self->db->resultset('SteeringTarget')->search( { deliveryservice => $steering_ds_id, target => $target_ds_id } )->single();
	if ( !defined($target) ) {
		return $self->not_found();
	}

	my $delete = $target->delete();
	if ($delete) {

		&log( $self, "Deleted steering target [ " . $target_ds_id . " ] for deliveryservice: " . $steering_ds_id, "APICHANGE" );

		return $self->success_message("Delivery service target delete was successful.");
	}
	else {
		return $self->alert("Delivery service target delete failed.");
	}

}

sub is_target_valid {
	my $self   = shift;
	my $params = shift;
	my $steering_ds = shift;

	my ( $is_valid, $target_type ) = $self->is_valid_target_type( $params->{typeId} );
	if ( !$is_valid ) {
		return ( 0, "Invalid target type" );
	}

	if ( $steering_ds->type->name ne "CLIENT_STEERING" && ($target_type eq "STEERING_GEO_WEIGHT" || $target_type eq "STEERING_GEO_ORDER") ) {
		return(0, "Invalid target type: STEERING_GEO_WEIGHT and STEERING_GEO_ORDER are only supported in CLIENT_STEERING delivery services");
	}

	my $rules = {
		fields => [qw/value typeId/],

		# Validation checks to perform
		checks => [
			value             => [ is_required("is required") ],
			typeId            => [ is_required("is required") ],
		]
	};

	# Validate the input against the rules
	my $result = validate( $params, $rules );

	if ($result->{success}) {
		if (($target_type eq "STEERING_WEIGHT") and ($params->{value} < 0)) {
			return(0, "Invalid value for target type STEERING_WEIGHT: cannot be negative");
		}
		elsif (($target_type eq "STEERING_GEO_WEIGHT") and ($params->{value} < 0)) {
			return(0, "Invalid value for target type STEERING_GEO_WEIGHT: cannot be negative");
		}
		return(1, $result->{data});
	}
	else {
		return(0, $result->{error});
	}
}

sub is_valid_target_type {
	my $self    = shift;
	my $type_id = shift;

	my $rs = $self->db->resultset("Type")->find( { id => $type_id } );
	if ( defined($rs) && ( $rs->use_in_table eq "steering_target" ) ) {
		return ( 1, $rs->name );
	}
	return ( 0, "" );
}

1;
