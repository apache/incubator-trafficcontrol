package API::Tenant;
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
use UI::TenantUtils;

use Mojo::Base 'Mojolicious::Controller';
use Data::Dumper;
use JSON;
use MojoPlugins::Response;

my $finfo = __FILE__ . ":";

sub getTenantName {
	my $self 		= shift;
	my $tenant_id		= shift;
	return defined($tenant_id) ? $self->db->resultset('Tenant')->search( { id => $tenant_id } )->get_column('name')->single() : "n/a";
}

sub index {
	my $self 	= shift;

	my %idnames;
	my $rs_data = $self->db->resultset("Tenant")->search();
	while ( my $row = $rs_data->next ) {
		$idnames{ $row->id } = $row->name;
	}

	my @data = ();
	my $tenantUtils = UI::TenantUtils->new($self);
	my @tenants_list = $tenantUtils->get_hierarchic_tenants_list();
	foreach my $row (@tenants_list) {
		if ($tenantUtils->is_tenant_resource_readable($row->id)) {
			push(
				@data, {
					"id"           => $row->id,
					"name"         => $row->name,
					"active"       => \$row->active,
					"parentId"     => $row->parent_id,
					"parentName"   => ( defined $row->parent_id ) ? $idnames{ $row->parent_id } : undef,
				}
			);
		}
	}
	$self->success( \@data );
}


sub show {
	my $self = shift;
	my $id   = $self->param('id');

	my @data = ();
	my %idnames;

	my $rs_idnames = $self->db->resultset("Tenant")->search( undef, { columns => [qw/id name/] } );
	while ( my $row = $rs_idnames->next ) {
		$idnames{ $row->id } = $row->name;
	}

	my $tenantUtils = UI::TenantUtils->new($self);
	my $rs_data = $self->db->resultset("Tenant")->search( { 'me.id' => $id });
	while ( my $row = $rs_data->next ) {
		if ($tenantUtils->is_tenant_resource_readable($row->id)) {
			push(
				@data, {
					"id"           => $row->id,
					"name"         => $row->name,
					"active"       => \$row->active,
					"parentId"     => $row->parent_id,
					"parentName"   => ( defined $row->parent_id ) ? $idnames{ $row->parent_id } : undef,
				}
			);
		}
	}
	$self->success( \@data );
}

sub update {
	my $self   = shift;
	my $id     = $self->param('id');
	my $params = $self->req->json;

	if ( !&is_oper($self) ) {
		return $self->forbidden();
	}

	my $tenant = $self->db->resultset('Tenant')->find( { id => $id } );
	if ( !defined($tenant) ) {
		return $self->not_found();
	}

	if ( !defined($params) ) {
		return $self->alert("Parameters must be in JSON format.");
	}

	if ( !defined( $params->{name} ) ) {
		return $self->alert("Tenant name is required.");
	}
	
	if ( $params->{name} ne $self->getTenantName($id) ) {
	        my $name = $params->{name};
		my $existing = $self->db->resultset('Tenant')->search( { name => $name } )->get_column('name')->single();
		if ($existing) {
			return $self->alert("A tenant with name \"$name\" already exists.");
		}	
	}	

	my $tenantUtils = UI::TenantUtils->new($self);

	if ( !defined( $params->{parentId}) && !$tenantUtils->is_root_tenant($id) ) {
		# Cannot turn a simple tenant to a root tenant.
		# Practically there is no problem with doing so, but it is to risky to be done by mistake. 
		return $self->alert("Parent Id is required.");
	}
	
	if ( !defined( $params->{active} ) ) {
		return $self->alert("Active field is required.");
	}

	my $is_active = $params->{active};
	
	if ( !$params->{active} && $tenantUtils->is_root_tenant($id)) {
		return $self->alert("Root tenant cannot be in-active.");
	}

	#this is a write operation, allowed only by parents of the tenant (which are the owners of the resource of type tenant)	
	my $current_resource_tenancy = $self->db->resultset('Tenant')->search( { id => $id } )->get_column('parent_id')->single();
	if (!defined($current_resource_tenancy)) {
		#no parent - the tenant is its-own owner
		$current_resource_tenancy = $id;
	}
	
	if (!$tenantUtils->is_tenant_resource_writeable($current_resource_tenancy)) {
		return $self->alert("Current owning tenant is not under user's tenancy.");
	}

	if (!$tenantUtils->is_tenant_resource_writeable($params->{parentId})) {
		return $self->alert("Parent tenant to be set is not under user's tenancy.");
	}


	#operation	
	my $values = {
		name      => $params->{name},
		active    => $params->{active},
		parent_id => $params->{parentId}
	};

	my $rs = $tenant->update($values);
	if ($rs) {
		my %idnames;
		my $response;

		my $rs_idnames = $self->db->resultset("Tenant")->search( undef, { columns => [qw/id name/] } );
		while ( my $row = $rs_idnames->next ) {
			$idnames{ $row->id } = $row->name;
		}

		$response->{id}          = $rs->id;
		$response->{name}        = $rs->name;
		$response->{active}      = $rs->active;
		$response->{parentId}    = $rs->parent_id;
		$response->{parentName}  = ( defined $rs->parent_id ) ? $idnames{ $rs->parent_id } : undef;
		$response->{lastUpdated} = $rs->last_updated;
		&log( $self, "Updated Tenant name '" . $rs->name . "' for id: " . $rs->id, "APICHANGE" );
		return $self->success( $response, "Tenant update was successful." );
	}
	else {
		return $self->alert("Tenant update failed.");
	}

}


sub create {
	my $self   = shift;
	my $params = $self->req->json;

	if ( !&is_oper($self) ) {
		return $self->forbidden();
	}

	my $name = $params->{name};
	if ( !defined($name) ) {
		return $self->alert("Tenant name is required.");
	}

	#not allowing to create additional root tenants.
	#there is no real problem with that, but no real use also
	my $parent_id = $params->{parentId};
	if ( !defined($parent_id) ) {
		return $self->alert("Parent Id is required.");
	}
	
	my $tenantUtils = UI::TenantUtils->new($self);
	if (!$tenantUtils->is_tenant_resource_writeable($params->{parentId})) {
		return $self->alert("Parent tenant to be set is not under user's tenancy.");
	}

	my $existing = $self->db->resultset('Tenant')->search( { name => $name } )->get_column('name')->single();
	if ($existing) {
		return $self->alert("A tenant with name \"$name\" already exists.");
	}

	my $is_active = exists($params->{active})? $params->{active} : 0; #optional, if not set use default
	
	if ( !$is_active && !defined($parent_id)) {
		return $self->alert("Root user cannot be in-active.");
	}
	
	my $values = {
		name 		=> $params->{name} ,
		active		=> $is_active,
		parent_id 	=> $params->{parentId}
	};

	my $insert = $self->db->resultset('Tenant')->create($values);
	my $rs = $insert->insert();
	if ($rs) {
		my %idnames;
		my $response;

		my $rs_idnames = $self->db->resultset("Tenant")->search( undef, { columns => [qw/id name/] } );
		while ( my $row = $rs_idnames->next ) {
			$idnames{ $row->id } = $row->name;
		}

		$response->{id}          	= $rs->id;
		$response->{name}        	= $rs->name;
		$response->{active}        	= $rs->active;
		$response->{parentId}		= $rs->parent_id;
		$response->{parentName}  	= ( defined $rs->parent_id ) ? $idnames{ $rs->parent_id } : undef;
		$response->{lastUpdated} 	= $rs->last_updated;

		&log( $self, "Created Tenant name '" . $rs->name . "' for id: " . $rs->id, "APICHANGE" );

		return $self->success( $response, "Tenant create was successful." );
	}
	else {
		return $self->alert("Tenant create failed.");
	}

}


sub delete {
	my $self = shift;
	my $id     = $self->param('id');

	if ( !&is_oper($self) ) {
		return $self->forbidden();
	}

	my $tenant = $self->db->resultset('Tenant')->find( { id => $id } );
	if ( !defined($tenant) ) {
		return $self->not_found();
	}	

	my $parent_tenant = $tenant->parent_id;
	
	my $tenantUtils = UI::TenantUtils->new($self);
	if (!$tenantUtils->is_tenant_resource_writeable($parent_tenant)) {
		return $self->alert("Parent tenant is not under user's tenancy.");
	}

	my $name = $tenant->name;
	
	my $existing_child = $self->db->resultset('Tenant')->search( { parent_id => $id }, {order_by => 'me.name' } )->get_column('name')->first();
	if ($existing_child) {
		return $self->alert("Tenant '$name' has children tenant(s): e.g '$existing_child'. Please update these tenants and retry.");
	}

	#The order of the below tests is intentional
	my $existing_ds = $self->db->resultset('Deliveryservice')->search( { tenant_id => $id }, {order_by => 'me.xml_id' })->get_column('xml_id')->first();
	if ($existing_ds) {
		return $self->alert("Tenant '$name' is assign with delivery-services(s): e.g. '$existing_ds'. Please update/delete these delivery-services and retry.");
	}

	my $existing_user = $self->db->resultset('TmUser')->search( { tenant_id => $id }, {order_by => 'me.username' })->get_column('username')->first();
	if ($existing_user) {
		return $self->alert("Tenant '$name' is assign with user(s): e.g. '$existing_user'. Please update these users and retry.");
	}

	my $rs = $tenant->delete();
	if ($rs) {
		return $self->success_message("Tenant deleted.");
	} else {
		return $self->alert( "Tenant delete failed." );
	}
}


