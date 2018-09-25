package API::DeliveryServiceServer;
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

# JvD Note: you always want to put Utils as the first use. Sh*t don't work if it's after the Mojo lines.
use UI::DeliveryService;
use UI::Utils;
use Utils::Tenant;
use Mojo::Base 'Mojolicious::Controller';
use Data::Dumper;
use Utils::Helper;

sub index {
	my $self = shift;
	my @data;
	my $orderby = $self->param('orderby') || "deliveryservice";

	#FOR THE REVIEWER - Currently I do not check DS tenancy here.
	#I assume the operation is of a CDN owner for debug and I would not like to hide data here.
	# Additionally I assume the operation is protected by "roles"
	#Also note that the ds/user table is note tested here originally

	# defaulted pagination and limits because there are 38129 rows in this table and counting...
	my $page  = $self->param('page')  || 1;
	my $limit = $self->param('limit') || 20;
	my $rs_data = $self->db->resultset("DeliveryserviceServer")->search( undef, { prefetch => [ 'deliveryservice', 'server' ], page => $page, rows => $limit, order_by => $orderby } );
	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	while ( my $row = $rs_data->next ) {
		if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $row->deliveryservice->tenant_id)) {
			next;
		}
		push(
			@data, {
				"deliveryService" => $row->deliveryservice->id,
				"server"          => $row->server->id,
				"lastUpdated"     => $row->last_updated,
			}
		);
	}
	#update to be ints
	$limit += 0;
	$page += 0;
	$self->success( \@data, undef, $orderby, $limit, $page );
}


# why is this here and in API/Cdn.pm?
sub domains {
	my $self = shift;
	my @data;

	my $rs = $self->db->resultset('Profile')->search( { 'me.name' => { -like => 'CCR%' } }, { prefetch => ['cdn'] } );
	while ( my $row = $rs->next ) {
		push(
			@data, {
				"domainName"         => $row->cdn->domain_name,
				"parameterId"        => -1,  # it's not a parameter anymore
				"profileId"          => $row->id,
				"profileName"        => $row->name,
				"profileDescription" => $row->description,
			}
		);

	}
	$self->success( \@data );
}

sub assign_servers_to_ds {
	my $self 		= shift;
	my $params 		= $self->req->json;
	my $ds_id 		= $params->{dsId};
	my $servers 	= $params->{servers};
	my $replace 	= $params->{replace};
	my $count		= 0;

	if ( !&is_oper($self) ) {
		return $self->forbidden();
	}

	my $ds = $self->db->resultset('Deliveryservice')->find( { id => $ds_id } );
	if ( !defined($ds) ) {
		return $self->not_found();
	}

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->alert("Invalid delivery-service assignment. The delivery-service is not avaialble for your tenant.");
	}

	if ( ref($servers) ne 'ARRAY' ) {
		return $self->alert("Servers must be an array");
	}

	if ( $replace ) {
		# start fresh and delete existing deliveryservice/server associations
		my $delete = $self->db->resultset('DeliveryserviceServer')->search( { deliveryservice => $ds_id } );
		$delete->delete();
	}

	my @values = ( [ qw( deliveryservice server ) ]); # column names are required for 'populate' function

	foreach my $server_id (@{ $servers }) {
		push(@values, [ $ds_id, $server_id ]);
		$count++;
	}

	$self->db->resultset("DeliveryserviceServer")->populate(\@values);

	# create location parameters for header_rewrite*, regex_remap* and cacheurl* config files if necessary
	&UI::DeliveryService::header_rewrite( $self, $ds->id, $ds->profile, $ds->xml_id, $ds->edge_header_rewrite, "edge" );
	&UI::DeliveryService::regex_remap( $self, $ds->id, $ds->profile, $ds->xml_id, $ds->regex_remap );
	&UI::DeliveryService::cacheurl( $self, $ds->id, $ds->profile, $ds->xml_id, $ds->cacheurl );

	&log( $self, $count . " servers were assigned to " . $ds->xml_id, "APICHANGE" );

	my $response = $params;
	return $self->success($response, "Server assignments complete.");
}

sub assign_ds_to_cachegroup {
	my $self   = shift;
	my $cg_id  = $self->param('id');
	my $params = $self->req->json;

	my $cachegroup = $self->db->resultset('Cachegroup')->search( { id => $cg_id } )->single();
	if ( !defined($cachegroup) ) {
		return $self->not_found();
	}

	if ( ($cachegroup->type->name ne "EDGE_LOC") and ($cachegroup->type->name ne "ORG_LOC") ) {
		return $self->alert("cachegroup should be type EDGE_LOC or ORG_LOC.");
	}

	if ( !defined($params) ) {
		return $self->alert("parameters should in json format.");
	}

	if ( !defined($params->{deliveryServices}) ) {
		return $self->alert("parameter deliveryServices is must.");
	}

	if ( ref($params->{deliveryServices}) ne 'ARRAY' ) {
		return $self->alert("parameter deliveryServices must be array.");
	}

	my $cdn = "";
	my $servers = $self->db->resultset('Server')->search(
		{
			cachegroup => $cg_id,
			'type.name' => { -in => ['EDGE', 'ORG'] }
		},
		{ prefetch => ['type'] }
	);
	while ( my $server = $servers->next ) {
		if ($cdn eq "") {
			$cdn = $server->cdn_id;
		} elsif ($cdn ne $server->cdn_id) {
			return $self->alert("servers do not belong to same cdn.");
		}
	}

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	my $deliveryservice_IDs = "";
	foreach my $ds_id (@{ $params->{deliveryServices} }) {
		my $ds = $self->db->resultset('Deliveryservice')->find( { id => $ds_id } );
		if ( !defined($ds) ) {
			return $self->alert("deliveryservice with id $ds_id does not existed");
		}
		if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
			return $self->alert("deliveryservice with id $ds_id is not available to your tenant");
		}
		if ($cdn eq "") {
			$cdn = $ds->cdn_id;
		} elsif ($cdn ne $ds->cdn_id) {
			return $self->alert("servers/deliveryservices do not belong to same cdn.");
		}
		$deliveryservice_IDs = $deliveryservice_IDs . " " .  $ds_id;
	}

	$servers = $self->db->resultset('Server')->search(
		{
			cachegroup => $cg_id,
			'type.name' => { -in => ['EDGE', 'ORG'] }
		},
		{ prefetch => ['type'] }
	);

	my @server_names = ();
	while ( my $server = $servers->next ) {
		push(@server_names, $server->host_name);
		foreach my $ds_id (@{ $params->{deliveryServices} }) {
			my $find = $self->db->resultset('DeliveryserviceServer')->find(
				{
					deliveryservice => $ds_id,
					server          => $server->id
				}
			);

			if (!defined($find)) {
				my $insert = $self->db->resultset('DeliveryserviceServer')->create(
					{
						deliveryservice => $ds_id,
						server          => $server->id
					}
				);
				$insert->insert();

				if ($server->type->name eq 'EDGE') {
					my $ds = $self->db->resultset('Deliveryservice')->search( { id => $ds_id } )->single();
					&UI::DeliveryService::header_rewrite( $self, $ds->id, $ds->profile, $ds->xml_id, $ds->edge_header_rewrite, "edge" );
					&UI::DeliveryService::regex_remap( $self, $ds->id, $ds->profile, $ds->xml_id, $ds->regex_remap );
					&UI::DeliveryService::cacheurl( $self, $ds->id, $ds->profile, $ds->xml_id, $ds->cacheurl );
				}
				$self->app->log->info("assign server " . $server->id . " to ds " . $ds_id);
			}
		}
	}

	&log( $self, "assign servers in cache group $cg_id to deliveryservices $deliveryservice_IDs", "APICHANGE" );

	my $response;
	$response->{id} = $cg_id;
	$response->{serverNames} = \@server_names;
	$response->{deliveryServices} = $params->{deliveryServices};
	$self->success( $response, "Delivery services successfully assigned to all the servers of cache group $cg_id" );
}

sub remove_server_from_ds {
	my $self     	= shift;
	my $ds_id  	 	= $self->param('dsId');
	my $server_id	= $self->param('serverId');

	my $tenant_utils = Utils::Tenant->new($self);
	if ( !&is_privileged($self) && !$tenant_utils->use_tenancy() && !$self->is_delivery_service_assigned($ds_id) ) {
		return $self->forbidden("Forbidden. Delivery service not assigned to user.");
	}

	my $ds = $self->db->resultset('Deliveryservice')->find( { id => $ds_id } );
	my $ds_tenant = undef;
	if ( defined($ds) ) {
		$ds_tenant = $ds->tenant_id;
	}
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds_tenant)) {
		return $self->forbidden("Forbidden. Delivery service not available on user tenancy.");
	}

	my $ds_server = $self->db->resultset('DeliveryserviceServer')->search( { deliveryservice => $ds_id, server => $server_id }, { prefetch => [ 'deliveryservice', 'server' ] } );
	if ( $ds_server->count == 0 ) {
		return $self->not_found();
	}

	my $row = $ds_server->next;
	my $rs = $ds_server->delete();
	if ($rs) {
		&log( $self, "Server [ " . $row->server->id . " | " . $row->server->host_name . " ] unlinked from deliveryservice [ " . $row->deliveryservice->id . " | " . $row->deliveryservice->xml_id . " ].", "APICHANGE" );
		return $self->success_message("Server unlinked from delivery service.");
	}

	return $self->alert( "Failed to unlink server from delivery service." );
}

1;
