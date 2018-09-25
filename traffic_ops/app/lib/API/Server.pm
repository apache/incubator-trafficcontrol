package API::Server;
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
use POSIX qw(strftime);
use Time::Local;
use LWP;
use UI::ConfigFiles;
use UI::Tools;
use MojoPlugins::Response;
use MojoPlugins::Job;
use Utils::Helper::ResponseHelper;
use String::CamelCase qw(decamelize);
use Validate::Tiny ':all';

sub index {
	my $self         = shift;
	my $current_user = $self->current_user()->{username};
	my $ds_id        = $self->param('dsId');
	my $type         = $self->param('type');
	my $status       = $self->param('status');
	my $profile_id   = $self->param('profileId');
	my $cdn_id       = $self->param('cdn');
	my $cg_id        = $self->param('cachegroup');
	my $phys_loc_id	 = $self->param('physLocation');

	my $servers;
	my $forbidden;
	if ( defined $ds_id ) {
		( $forbidden, $servers ) = $self->get_servers_by_dsid( $current_user, $ds_id, $status );
	}
	elsif ( defined $type ) {
		( $forbidden, $servers ) = $self->get_servers_by_type( $current_user, $type, $status );
	}
	elsif ( defined $profile_id ) {
		( $forbidden, $servers ) = $self->get_servers_by_profile_id($profile_id);
	}
	elsif ( defined $cdn_id ) {
		( $forbidden, $servers ) = $self->get_servers_by_cdn($cdn_id);
	}
	elsif ( defined $cg_id ) {
		( $forbidden, $servers ) = $self->get_servers_by_cachegroup($cg_id);
	}
	elsif ( defined $phys_loc_id ) {
		( $forbidden, $servers ) = $self->get_servers_by_phys_loc($phys_loc_id);
	}
	else {
		( $forbidden, $servers ) = $self->get_servers_by_status( $current_user, $status );
	}

	if ( defined($forbidden) ) {
		return $self->forbidden($forbidden);
	}

	my @data;
	if ( defined($servers) ) {
		my $is_admin = &is_admin($self);
		while ( my $row = $servers->next ) {
			push(
				@data, {
					"cachegroup"     => $row->cachegroup->name,
					"cachegroupId"   => $row->cachegroup->id,
					"cdnId"          => $row->cdn->id,
					"cdnName"        => $row->cdn->name,
					"domainName"     => $row->domain_name,
					"guid"           => $row->guid,
					"hostName"       => $row->host_name,
					"httpsPort"      => $row->https_port,
					"id"             => $row->id,
					"iloIpAddress"   => $row->ilo_ip_address,
					"iloIpNetmask"   => $row->ilo_ip_netmask,
					"iloIpGateway"   => $row->ilo_ip_gateway,
					"iloUsername"    => $row->ilo_username,
					"iloPassword"    => $is_admin ? $row->ilo_password : "",
					"interfaceMtu"   => $row->interface_mtu,
					"interfaceName"  => $row->interface_name,
					"ip6Address"     => $row->ip6_address,
					"ip6Gateway"     => $row->ip6_gateway,
					"ipAddress"      => $row->ip_address,
					"ipNetmask"      => $row->ip_netmask,
					"ipGateway"      => $row->ip_gateway,
					"lastUpdated"    => $row->last_updated,
					"mgmtIpAddress"  => $row->mgmt_ip_address,
					"mgmtIpNetmask"  => $row->mgmt_ip_netmask,
					"mgmtIpGateway"  => $row->mgmt_ip_gateway,
					"offlineReason"  => $row->offline_reason,
					"physLocation"   => $row->phys_location->name,
					"physLocationId" => $row->phys_location->id,
					"profile"        => $row->profile->name,
					"profileId"      => $row->profile->id,
					"profileDesc"    => $row->profile->description,
					"rack"           => $row->rack,
					"routerHostName" => $row->router_host_name,
					"routerPortName" => $row->router_port_name,
					"status"         => $row->status->name,
					"statusId"       => $row->status->id,
					"tcpPort"        => $row->tcp_port,
					"type"           => $row->type->name,
					"typeId"         => $row->type->id,
					"updPending"     => \$row->upd_pending
				}
			);
		}
	}

	return $self->success( \@data );
}

sub show {
	my $self = shift;
	my $id   = $self->param('id');

	my $rs_data  = $self->db->resultset("Server")->search( { 'me.id' => $id }, { prefetch => [ 'cachegroup', 'cdn', 'phys_location', 'profile', 'status', 'type' ]} );
	my @data     = ();
	my $is_admin = &is_admin($self);
	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"cachegroup"     => $row->cachegroup->name,
				"cachegroupId"   => $row->cachegroup->id,
				"cdnId"          => $row->cdn->id,
				"cdnName"        => $row->cdn->name,
				"domainName"     => $row->domain_name,
				"guid"           => $row->guid,
				"hostName"       => $row->host_name,
				"httpsPort"      => $row->https_port,
				"id"             => $row->id,
				"iloIpAddress"   => $row->ilo_ip_address,
				"iloIpNetmask"   => $row->ilo_ip_netmask,
				"iloIpGateway"   => $row->ilo_ip_gateway,
				"iloUsername"    => $row->ilo_username,
				"iloPassword"    => $is_admin ? $row->ilo_password : "********",
				"interfaceMtu"   => $row->interface_mtu,
				"interfaceName"  => $row->interface_name,
				"ip6Address"     => $row->ip6_address,
				"ip6Gateway"     => $row->ip6_gateway,
				"ipAddress"      => $row->ip_address,
				"ipNetmask"      => $row->ip_netmask,
				"ipGateway"      => $row->ip_gateway,
				"lastUpdated"    => $row->last_updated,
				"mgmtIpAddress"  => $row->mgmt_ip_address,
				"mgmtIpNetmask"  => $row->mgmt_ip_netmask,
				"mgmtIpGateway"  => $row->mgmt_ip_gateway,
				"offlineReason"  => $row->offline_reason,
				"physLocation"   => $row->phys_location->name,
				"physLocationId" => $row->phys_location->id,
				"profile"        => $row->profile->name,
				"profileId"      => $row->profile->id,
				"profileDesc"    => $row->profile->description,
				"rack"           => $row->rack,
				"routerHostName" => $row->router_host_name,
				"routerPortName" => $row->router_port_name,
				"status"         => $row->status->name,
				"statusId"       => $row->status->id,
				"tcpPort"        => $row->tcp_port,
				"type"           => $row->type->name,
				"typeId"         => $row->type->id,
				"updPending"     => \$row->upd_pending
			}
		);
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

	my ( $is_valid, $result ) = $self->is_server_valid($params, $id);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $server = $self->db->resultset('Server')->find( { id => $id } );
	if ( !defined($server) ) {
		return $self->not_found();
	}

	my $values = {
		cachegroup			=> $params->{cachegroupId},
		cdn_id				=> $params->{cdnId},
		domain_name			=> $params->{domainName},
		host_name			=> $params->{hostName},
		https_port			=> $params->{httpsPort},
		ilo_ip_address		=> $params->{iloIpAddress},
		ilo_ip_netmask		=> $params->{iloIpNetmask},
		ilo_ip_gateway		=> $params->{iloIpGateway},
		ilo_username		=> $params->{iloUsername},
		ilo_password		=> $params->{iloPassword},
		interface_mtu		=> $params->{interfaceMtu},
		interface_name		=> $params->{interfaceName},
		ip6_address			=> ($params->{ip6Address}) ? $params->{ip6Address} : undef, # non empty string or null
		ip6_gateway			=> $params->{ip6Gateway},
		ip_address			=> $params->{ipAddress},
		ip_netmask			=> $params->{ipNetmask},
		ip_gateway			=> $params->{ipGateway},
		mgmt_ip_address		=> $params->{mgmtIpAddress},
		mgmt_ip_netmask		=> $params->{mgmtIpNetmask},
		mgmt_ip_gateway		=> $params->{mgmtIpGateway},
		offline_reason		=> $params->{offlineReason},
		phys_location		=> $params->{physLocationId},
		profile				=> $params->{profileId},
		rack				=> $params->{rack},
		router_host_name	=> $params->{routerHostName},
		router_port_name	=> $params->{routerPortName},
		status				=> $params->{statusId},
		tcp_port			=> $params->{tcpPort},
		type				=> $params->{typeId},
		upd_pending			=> $params->{updPending},
	};

	my $rs = $server->update($values);
	if ($rs) {
		my @response;
		push(
			@response, {
				"cachegroup"     => $rs->cachegroup->name,
				"cachegroupId"   => $rs->cachegroup->id,
				"cdnId"          => $rs->cdn->id,
				"cdnName"        => $rs->cdn->name,
				"domainName"     => $rs->domain_name,
				"guid"           => $rs->guid,
				"hostName"       => $rs->host_name,
				"httpsPort"      => $rs->https_port,
				"id"             => $rs->id,
				"iloIpAddress"   => $rs->ilo_ip_address,
				"iloIpNetmask"   => $rs->ilo_ip_netmask,
				"iloIpGateway"   => $rs->ilo_ip_gateway,
				"iloUsername"    => $rs->ilo_username,
				"iloPassword"    => $rs->ilo_password,
				"interfaceMtu"   => $rs->interface_mtu,
				"interfaceName"  => $rs->interface_name,
				"ip6Address"     => $rs->ip6_address,
				"ip6Gateway"     => $rs->ip6_gateway,
				"ipAddress"      => $rs->ip_address,
				"ipNetmask"      => $rs->ip_netmask,
				"ipGateway"      => $rs->ip_gateway,
				"lastUpdated"    => $rs->last_updated,
				"mgmtIpAddress"  => $rs->mgmt_ip_address,
				"mgmtIpNetmask"  => $rs->mgmt_ip_netmask,
				"mgmtIpGateway"  => $rs->mgmt_ip_gateway,
				"offlineReason"  => $rs->offline_reason,
				"physLocation"   => $rs->phys_location->name,
				"physLocationId" => $rs->phys_location->id,
				"profile"        => $rs->profile->name,
				"profileId"      => $rs->profile->id,
				"profileDesc"    => $rs->profile->description,
				"rack"           => $rs->rack,
				"routerHostName" => $rs->router_host_name,
				"routerPortName" => $rs->router_port_name,
				"status"         => $rs->status->name,
				"statusId"       => $rs->status->id,
				"tcpPort"        => $rs->tcp_port,
				"type"           => $rs->type->name,
				"typeId"         => $rs->type->id,
				"updPending"     => \$rs->upd_pending
			}
		);

		&log( $self, "Updated server [ '" . $rs->host_name . "' ] with id: " . $rs->id, "APICHANGE" );

		return $self->success( \@response, "Cachegroup update was successful." );
	}
	else {
		return $self->alert("Cachegroup update failed.");
	}
}

sub create {
	my $self   = shift;
	my $params = $self->req->json;

	if ( !&is_oper($self) ) {
		return $self->forbidden();
	}

	my ( $is_valid, $result ) = $self->is_server_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $values = {
		cachegroup			=> $params->{cachegroupId},
		cdn_id				=> $params->{cdnId},
		domain_name			=> $params->{domainName},
		host_name			=> $params->{hostName},
		https_port			=> $params->{httpsPort},
		ilo_ip_address		=> $params->{iloIpAddress},
		ilo_ip_netmask		=> $params->{iloIpNetmask},
		ilo_ip_gateway		=> $params->{iloIpGateway},
		ilo_username		=> $params->{iloUsername},
		ilo_password		=> $params->{iloPassword},
		interface_mtu		=> $params->{interfaceMtu},
		interface_name		=> $params->{interfaceName},
		ip6_address			=> ($params->{ip6Address}) ? $params->{ip6Address} : undef, # non empty string or null
		ip6_gateway			=> $params->{ip6Gateway},
		ip_address			=> $params->{ipAddress},
		ip_netmask			=> $params->{ipNetmask},
		ip_gateway			=> $params->{ipGateway},
		mgmt_ip_address		=> $params->{mgmtIpAddress},
		mgmt_ip_netmask		=> $params->{mgmtIpNetmask},
		mgmt_ip_gateway		=> $params->{mgmtIpGateway},
		offline_reason		=> $params->{offlineReason},
		phys_location		=> $params->{physLocationId},
		profile				=> $params->{profileId},
		rack				=> $params->{rack},
		router_host_name	=> $params->{routerHostName},
		router_port_name	=> $params->{routerPortName},
		status				=> $params->{statusId},
		tcp_port			=> $params->{tcpPort},
		type				=> $params->{typeId},
		upd_pending			=> $params->{updPending},
	};

	my $insert = $self->db->resultset('Server')->create($values);
	my $rs     = $insert->insert();
	if ($rs) {
		my @response;
		push(
			@response, {
				"cachegroup"     => $rs->cachegroup->name,
				"cachegroupId"   => $rs->cachegroup->id,
				"cdnId"          => $rs->cdn->id,
				"cdnName"        => $rs->cdn->name,
				"domainName"     => $rs->domain_name,
				"guid"           => $rs->guid,
				"hostName"       => $rs->host_name,
				"httpsPort"      => $rs->https_port,
				"id"             => $rs->id,
				"iloIpAddress"   => $rs->ilo_ip_address,
				"iloIpNetmask"   => $rs->ilo_ip_netmask,
				"iloIpGateway"   => $rs->ilo_ip_gateway,
				"iloUsername"    => $rs->ilo_username,
				"iloPassword"    => $rs->ilo_password,
				"interfaceMtu"   => $rs->interface_mtu,
				"interfaceName"  => $rs->interface_name,
				"ip6Address"     => $rs->ip6_address,
				"ip6Gateway"     => $rs->ip6_gateway,
				"ipAddress"      => $rs->ip_address,
				"ipNetmask"      => $rs->ip_netmask,
				"ipGateway"      => $rs->ip_gateway,
				"lastUpdated"    => $rs->last_updated,
				"mgmtIpAddress"  => $rs->mgmt_ip_address,
				"mgmtIpNetmask"  => $rs->mgmt_ip_netmask,
				"mgmtIpGateway"  => $rs->mgmt_ip_gateway,
				"offlineReason"  => $rs->offline_reason,
				"physLocation"   => $rs->phys_location->name,
				"physLocationId" => $rs->phys_location->id,
				"profile"        => $rs->profile->name,
				"profileId"      => $rs->profile->id,
				"profileDesc"    => $rs->profile->description,
				"rack"           => $rs->rack,
				"routerHostName" => $rs->router_host_name,
				"routerPortName" => $rs->router_port_name,
				"status"         => $rs->status->name,
				"statusId"       => $rs->status->id,
				"tcpPort"        => $rs->tcp_port,
				"type"           => $rs->type->name,
				"typeId"         => $rs->type->id,
				"updPending"     => \$rs->upd_pending
			}
		);

		&log( $self, "Created server [ '" . $rs->host_name . "' ] with id: " . $rs->id, "APICHANGE" );

		return $self->success( \@response, "Server creation was successful." );
	}
	else {
		return $self->alert("Server creation failed.");
	}
}

sub get_servers_by_status {
	my $self              = shift;
	my $current_user      = shift;
	my $status            = shift;
	my $orderby           = $self->param('orderby') || "hostName";
	my $orderby_snakecase = lcfirst( decamelize($orderby) );

	my $forbidden;
	my $servers;
	if ( !&is_oper($self) ) {
		$forbidden = "Forbidden. You must have the operations role to perform this operation.";
		return ( $forbidden, $servers );
	}

	my %criteria;
	if ( defined $status ) {
		$criteria{'status.name'} = $status;
	}
	$servers = $self->db->resultset('Server')->search(
		\%criteria, {
			prefetch => [ 'cdn', 'cachegroup', 'type', 'profile', 'status', 'phys_location' ],
			order_by => 'me.' . $orderby_snakecase,
		}
	);

	return ($forbidden, $servers);
}

sub get_servers_by_dsid {
	my $self              = shift;
	my $current_user      = shift;
	my $ds_id             = shift;
	my $status            = shift;
	my $orderby           = $self->param('orderby') || "hostName";
	my $orderby_snakecase = lcfirst( decamelize($orderby) );
	my $helper            = new Utils::Helper( { mojo => $self } );

	my $ds = $self->db->resultset('Deliveryservice')->search( { 'me.id' => $ds_id }, { prefetch => ['type'] } )->single();
	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();

	my @ds_servers;
	my $forbidden;
	my $servers;

	if (defined($ds) && !$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		$forbidden = "Forbidden. Delivery service not available for user's tenant.";
		return ($forbidden, $servers);
	}
	elsif ( &is_privileged($self) || $tenant_utils->use_tenancy() || $self->is_delivery_service_assigned($ds_id) ) {
		@ds_servers = $self->db->resultset('DeliveryserviceServer')->search( { deliveryservice => $ds_id } )->get_column('server')->all();
	}
	else {
		$forbidden = "Forbidden. Delivery service not assigned to user.";
		return ($forbidden, $servers);
	}

	if ( scalar(@ds_servers) ) {
		my %criteria = ( -or => [ 'me.id' => { -in => \@ds_servers } ] );

		# currently these are the ds types that bypass the mids
		my @types_no_mid = qw( HTTP_NO_CACHE HTTP_LIVE DNS_LIVE );
		if ( !grep { $_ eq $ds->type->name } @types_no_mid ) {

# if the delivery service employs mids, we're gonna pull mid servers too by pulling the cachegroups of the edges and finding those cachegroups parent cachegroup...
# then we see which servers have cachegroup in parent cachegroup list...that's how we find mids for the ds :)
			my @parent_cachegroup_ids = $self->db->resultset('ServersParentCachegroupList')->search( { 'me.server_id' => { -in => \@ds_servers } } )
				->get_column('parent_cachegroup_id')->all();
			push @{ $criteria{-or} }, { 'me.cachegroup' => { -in => \@parent_cachegroup_ids } };
		}

		if ( defined $status ) {
			$criteria{'status.name'} = $status;
		}

		$servers = $self->db->resultset('Server')->search(
			\%criteria, {
				prefetch => [ 'cdn', 'cachegroup', 'type', 'profile', 'status', 'phys_location' ],
				order_by => 'me.' . $orderby_snakecase,
			}
		);
	}

	return ( $forbidden, $servers );
}

sub get_edge_servers_by_dsid {
	my $self    = shift;
	my $ds_id   = $self->param('id');

	my $ds = $self->db->resultset('Deliveryservice')->search( { 'me.id' => $ds_id } )->single();
	if ( !defined($ds) ) {
		return $self->not_found();
	}

	my $ds_servers;
	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Delivery-service tenant is not available to the user.");
	}
	elsif ( &is_privileged($self) || $tenant_utils->use_tenancy() || $self->is_delivery_service_assigned($ds_id) ) {
		$ds_servers = $self->db->resultset('DeliveryserviceServer')->search( { deliveryservice => $ds_id } );
	}
	else {
		#for the reviewer - I believe it should turn into forbidden as well
		return $self->alert("Forbidden. Delivery service not assigned to user.");
	}

	my $servers = $self->db->resultset('Server')->search(
		{ 'me.id' => { -in => $ds_servers->get_column('server')->as_query } },
		{ prefetch => [ 'cdn', 'cachegroup', 'type', 'profile', 'status', 'phys_location' ] }
	);

	my @data;
	if ( defined($servers) ) {
		my $is_admin = &is_admin($self);
		while ( my $row = $servers->next ) {
			push(
				@data, {
					"cachegroup"     => $row->cachegroup->name,
					"cachegroupId"   => $row->cachegroup->id,
					"cdnId"          => $row->cdn->id,
					"cdnName"        => $row->cdn->name,
					"domainName"     => $row->domain_name,
					"guid"           => $row->guid,
					"hostName"       => $row->host_name,
					"httpsPort"      => $row->https_port,
					"id"             => $row->id,
					"iloIpAddress"   => $row->ilo_ip_address,
					"iloIpNetmask"   => $row->ilo_ip_netmask,
					"iloIpGateway"   => $row->ilo_ip_gateway,
					"iloUsername"    => $row->ilo_username,
					"iloPassword"    => $is_admin ? $row->ilo_password : "",
					"interfaceMtu"   => $row->interface_mtu,
					"interfaceName"  => $row->interface_name,
					"ip6Address"     => $row->ip6_address,
					"ip6Gateway"     => $row->ip6_gateway,
					"ipAddress"      => $row->ip_address,
					"ipNetmask"      => $row->ip_netmask,
					"ipGateway"      => $row->ip_gateway,
					"lastUpdated"    => $row->last_updated,
					"mgmtIpAddress"  => $row->mgmt_ip_address,
					"mgmtIpNetmask"  => $row->mgmt_ip_netmask,
					"mgmtIpGateway"  => $row->mgmt_ip_gateway,
					"offlineReason"  => $row->offline_reason,
					"physLocation"   => $row->phys_location->name,
					"physLocationId" => $row->phys_location->id,
					"profile"        => $row->profile->name,
					"profileId"      => $row->profile->id,
					"profileDesc"    => $row->profile->description,
					"rack"           => $row->rack,
					"routerHostName" => $row->router_host_name,
					"routerPortName" => $row->router_port_name,
					"status"         => $row->status->name,
					"statusId"       => $row->status->id,
					"tcpPort"        => $row->tcp_port,
					"type"           => $row->type->name,
					"typeId"         => $row->type->id,
					"updPending"     => \$row->upd_pending
				}
			);
		}
	}

	return $self->success( \@data );
}

sub get_unassigned_servers_by_dsid {
	my $self    = shift;
	my $ds_id   = $self->param('id');

	my $ds = $self->db->resultset('Deliveryservice')->search( { id => $ds_id } )->single();
	if ( !defined($ds) ) {
		return $self->not_found();
	}

	my %ds_server_criteria;
	$ds_server_criteria{'deliveryservice.id'} = $ds_id;

	my @assigned_servers;
	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();

	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Delivery-service tenant is not available to the user.");
	}
	elsif ( &is_privileged($self) || $tenant_utils->use_tenancy() || $self->is_delivery_service_assigned($ds_id) ) {
		@assigned_servers = $self->db->resultset('DeliveryserviceServer')->search( \%ds_server_criteria, { prefetch => [ 'deliveryservice', 'server' ] } )->get_column('server')->all();
	}
	else {
		#for the reviewer - I believe it should turn into forbidden as well
		return $self->Forbidden("Forbidden. Delivery service not assigned to user.");
	}

	my %server_criteria; # please fetch the following...
	$server_criteria{'me.id'} = { 'not in' => \@assigned_servers }; # ...unassigned servers...
	$server_criteria{'type.name'} = [ { -like => 'EDGE%' }, { -like => 'ORG' } ]; # ...of type EDGE% or ORG...
	$server_criteria{'cdn.id'} = $ds->cdn_id; # ...that belongs to the same cdn as the ds...

	my $servers = $self->db->resultset('Server')->search(
		\%server_criteria,
		{ prefetch => [ 'cdn', 'cachegroup', 'type', 'profile', 'status', 'phys_location' ] }
	);

	my @data;
	if ( defined($servers) ) {
		my $is_admin = &is_admin($self);
		while ( my $row = $servers->next ) {
			push(
				@data, {
					"cachegroup"     => $row->cachegroup->name,
					"cachegroupId"   => $row->cachegroup->id,
					"cdnId"          => $row->cdn->id,
					"cdnName"        => $row->cdn->name,
					"domainName"     => $row->domain_name,
					"guid"           => $row->guid,
					"hostName"       => $row->host_name,
					"httpsPort"      => $row->https_port,
					"id"             => $row->id,
					"iloIpAddress"   => $row->ilo_ip_address,
					"iloIpNetmask"   => $row->ilo_ip_netmask,
					"iloIpGateway"   => $row->ilo_ip_gateway,
					"iloUsername"    => $row->ilo_username,
					"iloPassword"    => $is_admin ? $row->ilo_password : "",
					"interfaceMtu"   => $row->interface_mtu,
					"interfaceName"  => $row->interface_name,
					"ip6Address"     => $row->ip6_address,
					"ip6Gateway"     => $row->ip6_gateway,
					"ipAddress"      => $row->ip_address,
					"ipNetmask"      => $row->ip_netmask,
					"ipGateway"      => $row->ip_gateway,
					"lastUpdated"    => $row->last_updated,
					"mgmtIpAddress"  => $row->mgmt_ip_address,
					"mgmtIpNetmask"  => $row->mgmt_ip_netmask,
					"mgmtIpGateway"  => $row->mgmt_ip_gateway,
					"offlineReason"  => $row->offline_reason,
					"physLocation"   => $row->phys_location->name,
					"physLocationId" => $row->phys_location->id,
					"profile"        => $row->profile->name,
					"profileId"      => $row->profile->id,
					"profileDesc"    => $row->profile->description,
					"rack"           => $row->rack,
					"routerHostName" => $row->router_host_name,
					"routerPortName" => $row->router_port_name,
					"status"         => $row->status->name,
					"statusId"       => $row->status->id,
					"tcpPort"        => $row->tcp_port,
					"type"           => $row->type->name,
					"typeId"         => $row->type->id,
					"updPending"     => \$row->upd_pending
				}
			);
		}
	}

	return $self->success( \@data );
}
sub get_eligible_servers_by_dsid {
	my $self    = shift;
	my $ds_id   = $self->param('id');

	my $ds = $self->db->resultset('Deliveryservice')->search( { id => $ds_id } )->single();
	if ( !defined($ds) ) {
		return $self->not_found();
	}

	my %ds_server_criteria;
	$ds_server_criteria{'deliveryservice.id'} = $ds_id;

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();

	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Delivery-service tenant is not available to the user.");
	}
	elsif ( !&is_privileged($self) && !$tenant_utils->use_tenancy() && !$self->is_delivery_service_assigned($ds_id) ) {
		#for the reviewer - I believe it should turn into forbidden as well
		return $self->Forbidden("Forbidden. Delivery service not assigned to user.");
	}

	my %server_criteria; # please fetch the following...
	$server_criteria{'type.name'} = [ { -like => 'EDGE%' }, { -like => 'ORG' } ]; # ...of type EDGE% or ORG...
	$server_criteria{'cdn.id'} = $ds->cdn_id; # ...that belongs to the same cdn as the ds...

	my $servers = $self->db->resultset('Server')->search(
		\%server_criteria,
		{ prefetch => [ 'cdn', 'cachegroup', 'type', 'profile', 'status', 'phys_location' ] }
	);

	my @data;
	if ( defined($servers) ) {
		my $is_admin = &is_admin($self);
		while ( my $row = $servers->next ) {
			push(
				@data, {
					"cachegroup"     => $row->cachegroup->name,
					"cachegroupId"   => $row->cachegroup->id,
					"cdnId"          => $row->cdn->id,
					"cdnName"        => $row->cdn->name,
					"domainName"     => $row->domain_name,
					"guid"           => $row->guid,
					"hostName"       => $row->host_name,
					"httpsPort"      => $row->https_port,
					"id"             => $row->id,
					"iloIpAddress"   => $row->ilo_ip_address,
					"iloIpNetmask"   => $row->ilo_ip_netmask,
					"iloIpGateway"   => $row->ilo_ip_gateway,
					"iloUsername"    => $row->ilo_username,
					"iloPassword"    => $is_admin ? $row->ilo_password : "",
					"interfaceMtu"   => $row->interface_mtu,
					"interfaceName"  => $row->interface_name,
					"ip6Address"     => $row->ip6_address,
					"ip6Gateway"     => $row->ip6_gateway,
					"ipAddress"      => $row->ip_address,
					"ipNetmask"      => $row->ip_netmask,
					"ipGateway"      => $row->ip_gateway,
					"lastUpdated"    => $row->last_updated,
					"mgmtIpAddress"  => $row->mgmt_ip_address,
					"mgmtIpNetmask"  => $row->mgmt_ip_netmask,
					"mgmtIpGateway"  => $row->mgmt_ip_gateway,
					"offlineReason"  => $row->offline_reason,
					"physLocation"   => $row->phys_location->name,
					"physLocationId" => $row->phys_location->id,
					"profile"        => $row->profile->name,
					"profileId"      => $row->profile->id,
					"profileDesc"    => $row->profile->description,
					"rack"           => $row->rack,
					"routerHostName" => $row->router_host_name,
					"routerPortName" => $row->router_port_name,
					"status"         => $row->status->name,
					"statusId"       => $row->status->id,
					"tcpPort"        => $row->tcp_port,
					"type"           => $row->type->name,
					"typeId"         => $row->type->id,
					"updPending"     => \$row->upd_pending
				}
			);
		}
	}

	return $self->success( \@data );
}

sub get_servers_by_type {
	my $self              = shift;
	my $current_user      = shift;
	my $type              = shift;
	my $status            = shift;
	my $orderby           = $self->param('orderby') || "hostName";
	my $orderby_snakecase = lcfirst( decamelize($orderby) );

	my $forbidden;
	my $servers;
	if ( !&is_oper($self) ) {
		$forbidden = "Forbidden. You must have the operations role to perform this operation.";
		return ( $forbidden, $servers );
	}

	my %criteria = ( 'type.name' => $type );
	if (defined $status) {
		$criteria{'status.name'} = $status;
	}

	$servers = $self->db->resultset('Server')->search(
		\%criteria, {
			prefetch => [ 'cdn', 'cachegroup', 'type', 'profile', 'status', 'phys_location' ],
			order_by => 'me.' . $orderby_snakecase,
		}
	);

	return ($forbidden, $servers);
}

sub totals {
	my $self = shift;

	my @data;
	my @rs = $self->db->resultset('ServerTypes')->search();
	foreach my $rs (@rs) {
		my $type_name = $rs->name;
		my $count     = $self->get_count_by_type($type_name);
		push(
			@data, {
				"type"  => $rs->name,
				"count" => $count,
			}
		);
	}

	return $self->success( \@data );

}

sub status_count {
	my $self		= shift;
	my $type		= $self->param('type');
	my $response		= {};

	my $server_count = $self->db->resultset('Server')->search()->count();
	if ($server_count == 0) {
		# if there are no servers, just return 0 for all statuses
		my $statuses = $self->db->resultset('Status')->search();
		while ( my $status = $statuses->next ) {
			$response->{ $status->name } = 0;
		}
	} else {
		my %criteria;
		if ( defined $type ) {
			%criteria = ( 'type.name' => { '~' => $type } );
		}
		my $rs = $self->db->resultset('Server')->search(
			\%criteria,
			{
				join     => [qw/ status type /],
				select   => [ 'status.name', { count => 'me.id' } ],
				as       => [qw/ status_name server_count /],
				group_by => [qw/ status.id /]
			}
		);

		while ( my $row = $rs->next ) {
			$response->{ $row->{'_column_data'}->{'status_name'} } = $row->{'_column_data'}->{'server_count'};
		}
	}

	return $self->success( $response );
}

sub update_status {
	my $self 	= shift;
	my $id     	= $self->param('id');
	my $params 	= $self->req->json;

	my $server = $self->db->resultset('Server')->find( { id => $id }, { prefetch => [ 'type' ] } );
	if ( !defined($server) ) {
		return $self->not_found();
	}

	if ( !defined( $params->{status} ) ) {
		return $self->alert("Status is required.");
	}

	my $server_status;
	if  ( $params->{status} =~ /\d+/ ) {
		$server_status = $self->db->resultset('Status')->search( { id => $params->{status} }, { columns => [qw/id name/] } )->single();
	} else {
		$server_status = $self->db->resultset('Status')->search( { name => $params->{status} }, { columns => [qw/id name/] } )->single();
	}

	if ( !defined($server_status) ) {
		return $self->alert("Invalid status.");
	}

	my $offline_reason = $params->{offlineReason};
	if ( $server_status->name eq 'ADMIN_DOWN' || $server_status->name eq 'OFFLINE' ) {
		if ( !defined( $offline_reason ) ) {
			return $self->alert("Offline reason is required for ADMIN_DOWN or OFFLINE status.");
		} else {
			# prepend current user to offline message
			my $current_username = $self->current_user()->{username};
			$offline_reason = "$current_username: $offline_reason";

		}
	} else {
		$offline_reason = undef;
	}

	my $values = {
		status           => $server_status->id,
		offline_reason   => $offline_reason,
	};

	my $update = $server->update($values);
	if ($update) {
		my $fqdn = $update->host_name . "." . $update->domain_name;
		my $msg = "Updated status [ " . $server_status->name . " ] for $fqdn [ $offline_reason ]";

		# queue updates on child servers if server is ^EDGE or ^MID
		if ( $server->type->name =~ m/^EDGE/ || $server->type->name =~ m/^MID/ ) {
			my @cg_ids = $self->get_child_cachegroup_ids($server);
			my $servers = $self->db->resultset('Server')->search( { cachegroup => { -in => \@cg_ids }, cdn_id => $server->cdn_id } );
			$servers->update( { upd_pending => 1 } );
			$msg .= " and queued updates on all child caches";
		}

        &log( $self, $msg, "APICHANGE" );
		return $self->success_message( $msg );
	}
	else {
		return $self->alert( "Server status update failed." );
	}

}

sub get_child_cachegroup_ids {
    my $self    = shift;
    my $server    = shift;

    my @edge_cache_groups = $self->db->resultset('Cachegroup')->search( { parent_cachegroup_id => $server->cachegroup->id } )->all();
    return map { $_->id } @edge_cache_groups;
}

sub get_count_by_type {
	my $self      = shift;
	my $type_name = shift;
	return $self->db->resultset('Server')->search( { 'type.name' => $type_name }, { join => 'type' } )->count();
}

sub details_v11 {
	my $self = shift;
	my @data;
	my $isadmin   = &is_admin($self);
	my $host_name = $self->param('name');
	my $rs_data   = $self->db->resultset('Server')->search( { host_name => $host_name },
		{ prefetch => [ 'cdn', 'cachegroup', 'type', 'profile', 'status', 'phys_location', 'hwinfos', { 'deliveryservice_servers' => 'deliveryservice' } ], } );
	while ( my $row = $rs_data->next ) {
		my $serv = {
			"id"             => $row->id,
			"hostName"       => $row->host_name,
			"domainName"     => $row->domain_name,
			"tcpPort"        => $row->tcp_port,
			"httpsPort"      => $row->https_port,
			"xmppId"         => $row->xmpp_id,
			"xmppPasswd"     => $isadmin ? $row->xmpp_passwd : "********",
			"interfaceName"  => $row->interface_name,
			"ipAddress"      => $row->ip_address,
			"ipNetmask"      => $row->ip_netmask,
			"ipGateway"      => $row->ip_gateway,
			"ip6Address"     => $row->ip6_address,
			"ip6Gateway"     => $row->ip6_gateway,
			"interfaceMtu"   => $row->interface_mtu,
			"cachegroup"     => $row->cachegroup->name,
			"physLocation"   => $row->phys_location->name,
			"guid"           => $row->guid,
			"rack"           => $row->rack,
			"type"           => $row->type->name,
			"status"         => $row->status->name,
			"offlineReason"  => $row->offline_reason,
			"profile"        => $row->profile->name,
			"profileDesc"    => $row->profile->description,
			"mgmtIpAddress"  => $row->mgmt_ip_address,
			"mgmtIpNetmask"  => $row->mgmt_ip_netmask,
			"mgmtIpGateway"  => $row->mgmt_ip_gateway,
			"iloIpAddress"   => $row->ilo_ip_address,
			"iloIpNetmask"   => $row->ilo_ip_netmask,
			"iloIpGateway"   => $row->ilo_ip_gateway,
			"iloUsername"    => $row->ilo_username,
			"iloPassword"    => $isadmin ? $row->ilo_password : "********",
			"routerHostName" => $row->router_host_name,
			"routerPortName" => $row->router_port_name,
			"cdnName"        => $row->cdn->name,
		};
		my $hw_rs = $row->hwinfos;
		while ( my $hwinfo_row = $hw_rs->next ) {
			$serv->{hardwareInfo}->{ $hwinfo_row->description } = $hwinfo_row->val;
		}

		my $rs_ds_data = $row->deliveryservice_servers;
		my $tenant_utils = Utils::Tenant->new($self);
		my $tenants_data = $tenant_utils->create_tenants_data_from_db();
		while ( my $dsrow = $rs_ds_data->next ) {
			if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $dsrow->deliveryservice->tenant_id)) {
				next;
			}
			push( @{ $serv->{deliveryservices} }, $dsrow->deliveryservice->id );
		}

		push( @data, $serv );
	}
	$self->success(@data);
}

sub details {
	my $self              = shift;
	my $orderby           = $self->param('orderby') || "hostName";
	my $orderby_snakecase = lcfirst( decamelize($orderby) );
	my $limit             = $self->param('limit') || 1000;
	my @data;
	my $isadmin          = &is_admin($self);
	my $phys_location_id = $self->param('physLocationID');
	my $host_name        = $self->param('hostName');

	if ( !defined($phys_location_id) && !defined($host_name) ) {
		return $self->alert("Missing required fields: 'hostName' or 'physLocationID'");
	}

	my $rs_data = $self->db->resultset('Server')->search(
		[ { host_name => $host_name }, { phys_location => $phys_location_id } ], {
			prefetch => [ 'cachegroup', 'type', 'profile', 'status', 'phys_location', 'hwinfos', { 'deliveryservice_servers' => 'deliveryservice' } ],
			order_by => 'me.' . $orderby_snakecase
		}
	);

	if ( $rs_data->count() > 0 ) {

		while ( my $row = $rs_data->next ) {

			my $serv = {
				"id"             => $row->id,
				"hostName"       => $row->host_name,
				"domainName"     => $row->domain_name,
				"tcpPort"        => $row->tcp_port,
				"httpsPort"      => $row->https_port,
				"xmppId"         => $row->xmpp_id,
				"xmppPasswd"     => $isadmin ? $row->xmpp_passwd : "********",
				"interfaceName"  => $row->interface_name,
				"ipAddress"      => $row->ip_address,
				"ipNetmask"      => $row->ip_netmask,
				"ipGateway"      => $row->ip_gateway,
				"ip6Address"     => $row->ip6_address,
				"ip6Gateway"     => $row->ip6_gateway,
				"interfaceMtu"   => $row->interface_mtu,
				"cachegroup"     => $row->cachegroup->name,
				"physLocation"   => $row->phys_location->name,
				"guid"           => $row->guid,
				"rack"           => $row->rack,
				"type"           => $row->type->name,
				"status"         => $row->status->name,
				"offlineReason"  => $row->offline_reason,
				"profile"        => $row->profile->name,
				"profileDesc"    => $row->profile->description,
				"mgmtIpAddress"  => $row->mgmt_ip_address,
				"mgmtIpNetmask"  => $row->mgmt_ip_netmask,
				"mgmtIpGateway"  => $row->mgmt_ip_gateway,
				"iloIpAddress"   => $row->ilo_ip_address,
				"iloIpNetmask"   => $row->ilo_ip_netmask,
				"iloIpGateway"   => $row->ilo_ip_gateway,
				"iloUsername"    => $row->ilo_username,
				"routerHostName" => $row->router_host_name,
				"routerPortName" => $row->router_port_name,
			};
			my $hw_rs = $row->hwinfos;
			while ( my $hwinfo_row = $hw_rs->next ) {
				$serv->{hardwareInfo}->{ $hwinfo_row->description } = $hwinfo_row->val;
			}

			my $rs_ds_data = $row->deliveryservice_servers;
			my $tenant_utils = Utils::Tenant->new($self);
			my $tenants_data = $tenant_utils->create_tenants_data_from_db();
            while ( my $dsrow = $rs_ds_data->next ) {
				if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $dsrow->deliveryservice->tenant_id)) {
					next;
				}
				push( @{ $serv->{deliveryservices} }, $dsrow->deliveryservice->id );
			}

			push( @data, $serv );
		}
		my $size = @data;
		$self->success( \@data, undef, $orderby, $limit, $size );
	}
	else {
		$self->success( [] );
	}
}

sub delete {
	my ( $params, $data, $err ) = ( undef, undef, undef );
	my $self = shift;

	if ( !&is_oper($self) ) {
		return $self->forbidden("Forbidden. You must have the operations role to perform this operation.");
	}

	my $id = $self->param('id');
	my $server = $self->db->resultset('Server')->find( { id => $id } );
	if ( !defined($server) ) {
		return $self->not_found();
	}
	my $delete = $self->db->resultset('Server')->search( { id => $id } );
	my $host_name = $delete->get_column('host_name')->single();
	$delete->delete();

	&log( $self, "Delete server with id:" . $id . " named " . $host_name, "APICHANGE" );

	return $self->success_message( "Server was deleted: " . $host_name );
}

sub postupdatequeue {
	my $self   = shift;
	my $params = $self->req->json;
	my $id     = $self->param('id');

	my $update = $self->db->resultset('Server')->find( { id => $id } );
	if ( !defined($update) ) {
		return $self->alert("Failed to find server id = $id");
	}

	my $setqueue = $params->{action};
	if ( !defined($setqueue) ) {
		return $self->alert("action needed, should be queue or dequeue.");
	}
	if ( $setqueue eq "queue" ) {
		$setqueue = 1;
	}
	elsif ( $setqueue eq "dequeue" ) {
		$setqueue = 0;
	}
	else {
		return $self->alert("action should be queue or dequeue.");
	}
	$update->update( { upd_pending => $setqueue } );

	my $response;
	$response->{serverId} = $id;
	$response->{action} = ( $setqueue == 1 ) ? "queue" : "dequeue";
	return $self->success($response);
}

sub get_servers_by_profile_id {
	my $self       = shift;
	my $profile_id = shift;

	my $forbidden;
	my $servers;
	if ( !&is_oper($self) ) {
		$forbidden = "Forbidden. You must have the operations role to perform this operation.";
		return ( $forbidden, $servers );
	}

	$servers = $self->db->resultset('Server')->search( { profile => $profile_id }, { prefetch => [ 'cdn', 'cachegroup', 'type', 'profile', 'status', 'phys_location' ] } );
	return ( $forbidden, $servers );
}

sub get_servers_by_cdn {
	my $self   = shift;
	my $cdn_id = shift;

	my $forbidden;
	my $servers;
	if ( !&is_oper($self) ) {
		$forbidden = "Forbidden. You must have the operations role to perform this operation.";
		return ( $forbidden, $servers );
	}

	$servers = $self->db->resultset('Server')->search( { cdn_id => $cdn_id }, { prefetch => [ 'cdn', 'cachegroup', 'type', 'profile', 'status', 'phys_location' ] } );
	return ( $forbidden, $servers );
}

sub get_servers_by_phys_loc {
	my $self   			= shift;
	my $phys_loc_id 	= shift;

	my $forbidden;
	my $servers;
	if ( !&is_oper($self) ) {
		$forbidden = "Forbidden. You must have the operations role to perform this operation.";
		return ( $forbidden, $servers );
	}

	$servers = $self->db->resultset('Server')->search( { phys_location => $phys_loc_id }, { prefetch => [ 'cdn', 'cachegroup', 'type', 'profile', 'status', 'phys_location' ] } );
	return ( $forbidden, $servers );
}

sub get_servers_by_cachegroup {
	my $self  = shift;
	my $cg_id = shift;

	my $forbidden;
	my $servers;
	if ( !&is_oper($self) ) {
		$forbidden = "Forbidden. You must have the operations role to perform this operation.";
		return ( $forbidden, $servers );
	}

	$servers = $self->db->resultset('Server')->search( { cachegroup => $cg_id }, { prefetch => [ 'cdn', 'cachegroup', 'type', 'profile', 'status', 'phys_location' ] } );
	return ( $forbidden, $servers );
}

sub is_server_valid {
	my $self   	= shift;
	my $params 	= shift;
	my $id 		= shift;

	if ( !$self->is_valid_server_type( $params->{typeId} ) ) {
		return ( 0, "Invalid server type" );
	}

	my $cdn_mismatch;
		my $profile = $self->db->resultset('Profile')->search( { 'me.id' => $params->{profileId}}, { prefetch => ['cdn'] } )->single();
		if ( !defined($profile->cdn) ) {
			$cdn_mismatch = 1;
		}
		elsif ( $params->{cdnId} != $profile->cdn->id ) {
			$cdn_mismatch = 1;
		}

	if ($cdn_mismatch) {
		return ( 0, "CDN of profile does not match Server CDN" );
	}

	my $ip_used_for_profile;
	if ($id) {
		$ip_used_for_profile = $self->db->resultset('Server')
			->search( { -and => [ 'ip_address' => $params->{ipAddress} , 'profile' => $params->{profileId}, 'id' => { '!=' => $id } ] })->single();
	} else {
		$ip_used_for_profile = $self->db->resultset('Server')
			->search( { -and => [ 'ip_address' => $params->{ipAddress} , 'profile' => $params->{profileId} ] })->single();
	}

	if ($ip_used_for_profile) {
		return ( 0, "IP Address already in use for that profile" );
	}

	my $ip6_used_for_profile;
	if (defined($params->{ip6Address}) && $params->{ip6Address} ne "") {
		if ($id) {
			$ip6_used_for_profile = $self->db->resultset('Server')
				->search( { -and => [ 'ip6_address' => $params->{ip6Address} , 'profile' => $params->{profileId}, 'id' => { '!=' => $id } ] })->single();
		} else {
			$ip6_used_for_profile = $self->db->resultset('Server')
				->search( { -and => [ 'ip6_address' => $params->{ip6Address} , 'profile' => $params->{profileId} ] })->single();
		}

	}

	if ($ip6_used_for_profile) {
		return ( 0, "IP6 Address already in use for that profile" );
	}

	my $rules = {
		fields => [
			qw/cachegroupId cdnId domainName hostName httpsPort iloIpAddress iloIpNetmask iloIpGateway iloUsername iloPassword interfaceMtu interfaceName ip6Address ip6Gateway ipAddress ipNetmask ipGateway mgmtIpAddress mgmtIpNetmask mgmtIpGateway offlineReason physLocationId profileId rack routerHostName routerPortName statusId tcpPort typeId updPending/
		],

		# Validation checks to perform
		checks => [
			cachegroupId	=> [ is_required("is required"), is_like( qr/^\d+$/, "must be an integer" ) ],
			cdnId			=> [ is_required("is required"), is_like( qr/^\d+$/, "must be an integer" ) ],
			domainName		=> [ is_required("is required"), is_like( qr/^\S*$/, "must not contain spaces" ) ],
			hostName		=> [ is_required("is required"), is_like( qr/^\S*$/, "must not contain spaces" ) ],
			httpsPort		=> [ \&is_valid_port ],
			interfaceMtu	=> [ is_required("is required"), is_like( qr/^\d+$/, "must be an integer" ) ],
			interfaceName	=> [ is_required("is required") ],
			ipAddress		=> [ is_required("is required") ],
			ipNetmask		=> [ is_required("is required") ],
			ipGateway		=> [ is_required("is required") ],
			physLocationId	=> [ is_required("is required"), is_like( qr/^\d+$/, "must be an integer" ) ],
			profileId		=> [ is_required("is required"), is_like( qr/^\d+$/, "must be an integer" ) ],
			statusId		=> [ is_required("is required"), is_like( qr/^\d+$/, "must be an integer" ) ],
			tcpPort			=> [ \&is_valid_port ],
			typeId			=> [ is_required("is required"), is_like( qr/^\d+$/, "must be an integer" ) ],
			updPending		=> [ is_required("is required") ]
		]
	};

	# Validate the input against the rules
	my $result = validate( $params, $rules );

	if ( $result->{success} ) {
		return ( 1, $result->{data} );
	}
	else {
		return ( 0, $result->{error} );
	}
}

sub is_valid_server_type {
	my $self    = shift;
	my $type_id = shift;

	my $rs = $self->db->resultset("Type")->find( { id => $type_id } );
	if ( defined($rs) && ( $rs->use_in_table eq "server" ) ) {
		return 1;
	}
	return 0;
}

sub is_valid_port {
	my ( $value, $params ) = @_;

	if ( !defined $value ) {
		return undef;
	}

	if ( !( $value =~ /^\d+$/ ) ) {
		return "must be an integer.";
	}

	return undef;
}


1;
