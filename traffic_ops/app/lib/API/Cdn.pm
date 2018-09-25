
package API::Cdn;
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
use Mojo::Base 'Mojolicious::Controller';
use Data::Dumper;
use Carp qw(cluck confess);
use JSON;
use MIME::Base64;
use UI::DeliveryService;
use MojoPlugins::Response;
use Common::ReturnCodes qw(SUCCESS ERROR);
use strict;

sub index {
	my $self = shift;
	my @data;
	my $orderby = $self->param('orderby') || "name";
	my $rs_data = $self->db->resultset("Cdn")->search( undef, { order_by => $orderby } );
	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"id"            => $row->id,
				"name"          => $row->name,
				"domainName"    => $row->domain_name,
				"dnssecEnabled" => \$row->dnssec_enabled,
				"lastUpdated" 	=> $row->last_updated,
			}
		);
	}
	$self->success( \@data );
}

sub show {
	my $self = shift;
	my $id   = $self->param('id');

	my $rs_data = $self->db->resultset("Cdn")->search( { id => $id } );
	my @data = ();
	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"id"            => $row->id,
				"name"          => $row->name,
				"domainName"    => $row->domain_name,
				"dnssecEnabled" => \$row->dnssec_enabled,
				"lastUpdated" 	=> $row->last_updated,
			}
		);
	}
	$self->success( \@data );
}

sub name {
	my $self = shift;
	my $cdn  = $self->param('name');

	my $rs_data = $self->db->resultset("Cdn")->search( { name => $cdn } );
	my @data = ();
	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"id"            => $row->id,
				"name"          => $row->name,
				"domainName"    => $row->domain_name,
				"dnssecEnabled" => \$row->dnssec_enabled,
				"lastUpdated"   => $row->last_updated,
			}
		);
	}
	$self->success( \@data );
}

sub create {
	my $self   = shift;
	my $params = $self->req->json;

	if ( !defined($params) ) {
		return $self->alert("parameters must be in JSON format.");
	}

	if ( !defined( $params->{name} ) ) {
		return $self->alert("CDN 'name' is required.");
	}

	if ( !defined( $params->{dnssecEnabled} ) ) {
		return $self->alert("dnssecEnabled is required.");
	}

	if ( !defined( $params->{domainName} ) ) {
		return $self->alert("Domain Name is required.");
	}

	my $existing = $self->db->resultset('Cdn')->search( { name => $params->{name} } )->single();
	if ($existing) {
		$self->app->log->error( "a cdn with name '" . $params->{name} . "' already exists." );
		return $self->alert( "a cdn with name " . $params->{name} . " already exists." );
	}

	$existing = $self->db->resultset('Cdn')->search( { domain_name => $params->{domainName} } )->single();
	if ($existing) {
		$self->app->log->error( "a cdn with domain name '" . $params->{domainName} . "' already exists." );
		return $self->alert( "a cdn with domain " . $params->{domainName} . " already exists." );
	}

	my $values = {
		name => $params->{name},
		dnssec_enabled => $params->{dnssecEnabled},
		domain_name => $params->{domainName},
	};

	my $insert = $self->db->resultset('Cdn')->create($values);
	$insert->insert();

	my $rs = $self->db->resultset('Cdn')->find( { id => $insert->id } );
	if ( defined($rs) ) {
		my $response;
		$response->{id}            = $rs->id;
		$response->{name}          = $rs->name;
		$response->{domainName}    = $rs->domain_name;
		$response->{dnssecEnabled} = \$rs->dnssec_enabled;
		&log( $self, "Created CDN with id: " . $rs->id . " and name: " . $rs->name, "APICHANGE" );
		return $self->success( $response, "cdn was created." );
	}
	return $self->alert("create cdn failed.");
}

sub update {
	my $self   = shift;
	my $id     = $self->param('id');
	my $params = $self->req->json;

	my $cdn = $self->db->resultset('Cdn')->find( { id => $id } );
	if ( !defined($cdn) ) {
		return $self->not_found();
	}

	if ( !defined( $params->{name} ) ) {
		return $self->alert("Name is required.");
	}

	if ( !defined( $params->{dnssecEnabled} ) ) {
		return $self->alert("dnssecEnabled is required.");
	}

	if ( !defined( $params->{domainName} ) ) {
		return $self->alert("Domain Name is required.");
	}

	my $existing = $self->db->resultset('Cdn')->search( { name => $params->{name} } )->single();
	if ( $existing && $existing->id != $cdn->id ) {
		return $self->alert( "a cdn with name " . $params->{name} . " already exists." );
	}

	$existing = $self->db->resultset('Cdn')->search( { domain_name => $params->{domainName} } )->single();
	if ( $existing && $existing->id != $cdn->id ) {
		return $self->alert( "a cdn with domain name " . $params->{domainName} . " already exists." );
	}

	my $values = {
		name => $params->{name},
		dnssec_enabled => $params->{dnssecEnabled},
		domain_name => $params->{domainName},
	};

	my $rs = $cdn->update($values);
	if ( $rs ) {
		my $response;
		$response->{id}            	= $rs->id;
		$response->{name}          	= $rs->name;
		$response->{domainName}		= $rs->domain_name;
		$response->{dnssecEnabled} = \$rs->dnssec_enabled;
		&log( $self, "Updated CDN name '" . $rs->name . "' for id: " . $rs->id, "APICHANGE" );
		return $self->success( $response, "CDN update was successful." );
	}
	return $self->alert("CDN update failed.");
}

sub delete {
	my $self = shift;
	my $id   = $self->param('id');

	my $cdn = $self->db->resultset('Cdn')->search( { id => $id } );
	if ( !defined($cdn) ) {
		return $self->not_found();
	}

	my $rs = $self->db->resultset('Server')->search( { cdn_id => $id } );
	if ( $rs->count() > 0 ) {
		$self->app->log->error("Failed to delete cdn id = $id has servers");
		return $self->alert("Failed to delete cdn id = $id has servers");
	}

	$rs = $self->db->resultset('Deliveryservice')->search( { cdn_id => $id } );
	if ( $rs->count() > 0 ) {
		$self->app->log->error("Failed to delete cdn id = $id has delivery services");
		return $self->alert("Failed to delete cdn id = $id has delivery services");
	}

	my $name = $cdn->get_column('name')->single();
	$cdn->delete();
	&log( $self, "Delete cdn " . $name, "APICHANGE" );
	return $self->success_message("cdn was deleted.");
}

sub delete_by_name {
	my $self = shift;
	my $cdn_name   = $self->param('name');

	my $cdn = $self->db->resultset('Cdn')->find( { name => $cdn_name } );
	if ( !defined($cdn) ) {
		return $self->not_found();
	}
	my $id = $cdn->id;

	my $rs = $self->db->resultset('Server')->search( { cdn_id => $id } );
	if ( $rs->count() > 0 ) {
		$self->app->log->error("Failed to delete cdn id = $id has servers");
		return $self->alert("Failed to delete cdn id = $id has servers");
	}

	$rs = $self->db->resultset('Deliveryservice')->search( { cdn_id => $id } );
	if ( $rs->count() > 0 ) {
		$self->app->log->error("Failed to delete cdn id = $id has delivery services");
		return $self->alert("Failed to delete cdn id = $id has delivery services");
	}

	$cdn->delete();
	&log( $self, "Delete cdn " . $cdn_name, "APICHANGE" );
	return $self->success_message("cdn was deleted.");
}

sub queue_updates {
	my $self		= shift;
	my $params		= $self->req->json;
	my $cdn_id		= $self->param('id');

	my $cdn = $self->db->resultset('Cdn')->find( { id => $cdn_id } );
	if ( !defined($cdn) ) {
		return $self->not_found();
	}

	my $cdn_servers = $self->db->resultset('Server')->search( { cdn_id => $cdn_id } );

	if ( $cdn_servers->count() < 1 ) {
		return $self->alert("No servers found for cdn_id = $cdn_id");
	}

	my $setqueue = $params->{action};

	if ( $setqueue eq "queue" ) {
		$setqueue = 1;
	}
	elsif ( $setqueue eq "dequeue" ) {
		$setqueue = 0;
	}
	else {
		return $self->alert("Action required, Should be queue or dequeue.");
	}

	$cdn_servers->update( { upd_pending => $setqueue } );

	my $msg = "Server updates " . $params->{action} . "d for " . $cdn->name;
	&log( $self, $msg, "APICHANGE" );

	my $response;
	$response->{cdnId} = $cdn_id;
	$response->{action} = $params->{action};

	return $self->success($response, $msg);
}


sub configs_monitoring {
	my $self      = shift;
	my $cdn_name  = $self->param('name');
	my $extension = $self->param('extension');

	my $data_obj = $self->get_traffic_monitor_config($cdn_name);
	$self->success($data_obj);
}

sub get_traffic_monitor_config {
	my $self = shift;
	my $cdn_name = shift || confess("Please supply a CDN name");
	my $rascal_profile;
	my @cache_profiles;
	my @ccr_profiles;
	my $ccr_profile_id;
	my $data_obj;
	my $profile_to_type;

	my $rs_pp = $self->db->resultset('Server')->search(
		{ 'cdn.name' => $cdn_name },
		{   prefetch => ['cdn', 'profile', 'type'],
			select   => 'me.profile',
			distinct => 1,
			group_by => [qw/cdn.id profile.id me.profile type.id/],
		}
	);

	while ( my $row = $rs_pp->next ) {
		if ( $row->type->name =~ m/^RASCAL/ ) {
			$rascal_profile = $row->profile->name;
		}
		elsif ( $row->type->name =~ m/^CCR/ ) {
			push( @ccr_profiles, $row->profile->name );

			# TODO MAT: support multiple CCR profiles
			$ccr_profile_id = $row->profile->id;
		}
		elsif ( $row->type->name =~ m/^EDGE/ || $row->type->name =~ m/^MID/ ) {
			push( @cache_profiles, $row->profile->name );
			$profile_to_type->{$row->profile->name}->{$row->type->name} = $row->type->name;
		}
	}

	my %condition = (
		'parameter.config_file' => 'rascal-config.txt',
		'profile.name'          => $rascal_profile
	);

	$rs_pp = $self->db->resultset('ProfileParameter')->search( \%condition, { prefetch => [ { 'parameter' => undef }, { 'profile' => undef } ] } );

	while ( my $row = $rs_pp->next ) {
		my $parameter;

		if ( $row->parameter->name =~ m/location/ ) { next; }

		if ( $row->parameter->value =~ m/^\d+$/ ) {
			$data_obj->{'config'}->{ $row->parameter->name } =
				int( $row->parameter->value );
		}
		else {
			$data_obj->{'config'}->{ $row->parameter->name } = $row->parameter->value;
		}
	}

	%condition = (
		'parameter.config_file' => 'rascal.properties',
		'profile.name'          => { -in => \@cache_profiles }
	);

	$rs_pp = $self->db->resultset('ProfileParameter')->search( \%condition, { prefetch => [ { 'parameter' => undef }, { 'profile' => undef } ] } );

	if ( !exists( $data_obj->{'profiles'} ) ) {
		$data_obj->{'profiles'} = undef;
	}

	my $profile_tracker;

	while ( my $row = $rs_pp->next ) {
		if ( exists($profile_to_type->{$row->profile->name}) ) {
			for my $profile_type ( keys(%{$profile_to_type->{$row->profile->name}}) ) {
				$profile_tracker->{ $profile_type }->{ $row->profile->name }->{'type'} = $profile_type;
				$profile_tracker->{ $profile_type }->{ $row->profile->name }->{'name'} = $row->profile->name;

				if ( $row->parameter->value =~ m/^\d+$/ ) {
					$profile_tracker->{ $profile_type }->{ $row->profile->name }->{'parameters'}->{ $row->parameter->name } = int( $row->parameter->value );
				}

				else {
					$profile_tracker->{ $profile_type }->{ $row->profile->name }->{'parameters'}->{ $row->parameter->name } = $row->parameter->value;
				}
			}
		}
	}

	foreach my $type ( keys %{$profile_tracker} ) {
		foreach my $profile ( keys %{$profile_tracker->{$type}} ) {
			push( @{ $data_obj->{'profiles'} }, $profile_tracker->{$type}->{$profile} );
		}
	}

	foreach my $ccr_profile (@ccr_profiles) {
		my $profile;
		$profile->{'name'}       = $ccr_profile;
		$profile->{'type'}       = "CCR";
		$profile->{'parameters'} = undef;
		push( @{ $data_obj->{'profiles'} }, $profile );
	}

	my $rs_ds = $self->db->resultset('Deliveryservice')->search( { 'me.profile' => $ccr_profile_id, 'active' => 1 }, {} );

	while ( my $row = $rs_ds->next ) {
		my $delivery_service;

		# MAT: Do we move this to the DB? Rascal needs to know if it should monitor a DS or not, and the status=REPORTED is what we do for caches.
		$delivery_service->{'xmlId'}  = $row->xml_id;
		$delivery_service->{'status'} = "REPORTED";
		$delivery_service->{'totalKbpsThreshold'} =
			( defined( $row->global_max_mbps ) && $row->global_max_mbps > 0 ) ? ( $row->global_max_mbps * 1000 ) : 0;
		$delivery_service->{'totalTpsThreshold'} = int( $row->global_max_tps || 0 );
		push( @{ $data_obj->{'deliveryServices'} }, $delivery_service );
	}

        my $caches_query = 'SELECT
                              me.host_name as hostName,
                              CONCAT(me.host_name, \'.\', me.domain_name) as fqdn,
                              status.name as status,
                              cachegroup.name as cachegroup,
                              me.tcp_port as port,
                              me.ip_address as ip,
                              me.ip6_address as ip6,
                              profile.name as profile,
                              me.interface_name as interfaceName,
                              type.name as type,
                              me.xmpp_id as hashId
                            FROM server me
                              JOIN type type ON type.id = me.type
                              JOIN status status ON status.id = me.status
                              JOIN cachegroup cachegroup ON cachegroup.id = me.cachegroup
                              JOIN profile profile ON profile.id = me.profile
                              JOIN cdn cdn ON cdn.id = me.cdn_id
                            WHERE cdn.name = ?;';
	my $dbh = $self->db->storage->dbh;
	my $caches_servers = $dbh->selectall_arrayref( $caches_query, {Columns=>{}}, ($cdn_name) );
	foreach (@{ $caches_servers }) {
			if ( $_->{'type'} eq "RASCAL" ) {
					push( @{ $data_obj->{'trafficMonitors'} }, $_ );
			}
			elsif ( $_->{'type'} =~ m/^EDGE/ || $_->{'type'} =~ m/^MID/ ) {
					push( @{ $data_obj->{'trafficServers'} }, $_ );
			}
	}

	my $rs_loc = $self->db->resultset('Server')->search(
		{ 'cdn.name' => $cdn_name },
		{
			join   => [ 'cdn',             'cachegroup' ],
			select => [ 'cachegroup.name', 'cachegroup.latitude', 'cachegroup.longitude' ],
			distinct => 1
		}
	);

	while ( my $row = $rs_loc->next ) {
		my $cache_group;
		my $latitude  = $row->cachegroup->latitude + 0;
		my $longitude = $row->cachegroup->longitude + 0;
		$cache_group->{'coordinates'}->{'latitude'}  = $latitude;
		$cache_group->{'coordinates'}->{'longitude'} = $longitude;
		$cache_group->{'name'}                       = $row->cachegroup->name;
		push( @{ $data_obj->{'cacheGroups'} }, $cache_group );
	}

	return ($data_obj);
}

sub capacity {
	my $self = shift;

	return $self->get_cache_capacity();
}

sub health {
	my $self = shift;
	my $args = {};

	my $cdn_name = $self->param('name');
	if (defined($cdn_name)) {
		$args->{'cdn_name'} = $cdn_name;
	}

	return $self->get_cache_health($args);
}

sub routing {
	my $self = shift;
	my $args = shift;

	if ( !exists( $args->{status} ) ) {
		$args->{status} = "ONLINE";
	}

	$args->{type} = "CCR";

	my $ccr_map = $self->get_host_map($args);
	my $data    = {};
	my $stats   = {
		totalCount => 0,
		raw        => {},
	};
	for my $cdn_name ( keys( %{$ccr_map} ) ) {
		for my $ccr ( keys( %{ $ccr_map->{$cdn_name} } ) ) {
			my $ccr_host = $ccr_map->{$cdn_name}->{$ccr}->{host_name} . "." . $ccr_map->{$cdn_name}->{$ccr}->{domain_name};

			# TODO: what happens when the request to CCR times out? -jse
			my $c = $self->get_traffic_router_connection( { hostname => $ccr_host } );
			my $s = $c->get_crs_stats();
			if ( !defined($s) ) {
				$self->app->log->error("Unable to contact $ccr_host for $cdn_name.");
				return $self->internal_server_error( { "Internal Server" => "Error: Unable to contact $ccr_host" } );
			}
			else {

				if ( exists( $s->{stats} ) ) {
					for my $type ( "httpMap", "dnsMap" ) {
						next
							if ( exists( $args->{stat_key} )
							&& $args->{stat_key} ne $type );

						if ( exists( $s->{stats}->{$type} ) ) {
							for my $fqdn ( keys( %{ $s->{stats}->{$type} } ) ) {
								my $count = 1;

								if ( exists( $args->{patterns} )
									&& ref( $args->{patterns} ) eq "ARRAY" )
								{
									$count = 0;

									for my $pattern ( @{ $args->{patterns} } ) {
										if ( $fqdn =~ /$pattern/ ) {
											$count = 1;
											last;
										}
									}
								}

								if ($count) {
									for my $counter ( keys( %{ $s->{stats}->{$type}->{$fqdn} } ) ) {
										if ( !exists( $stats->{raw}->{$counter} ) ) {
											$stats->{raw}->{$counter} = 0;
										}

										$stats->{raw}->{$counter} += $s->{stats}->{$type}->{$fqdn}->{$counter};
										$stats->{totalCount} += $s->{stats}->{$type}->{$fqdn}->{$counter};
									}
								}
								if ($count) {
									for my $counter ( keys( %{ $s->{stats}->{$type}->{$fqdn} } ) ) {
										if ( !exists( $stats->{raw}->{$counter} ) ) {
											$stats->{raw}->{$counter} = 0;
										}

										$stats->{raw}->{$counter} += $s->{stats}->{$type}->{$fqdn}->{$counter};
										$stats->{totalCount} += $s->{stats}->{$type}->{$fqdn}->{$counter};
									}
								}
							}
						}
					}
				}
			}
		}
	}

	for my $counter ( keys( %{ $stats->{raw} } ) ) {
		my $p = $counter;
		$p =~ s/Count//gi;

		if ( $stats->{totalCount} > 0 ) {
			$data->{$p} =
				( $stats->{raw}->{$counter} / $stats->{totalCount} ) * 100;
		}
		else {
			$data->{$p} = 0;
		}
	}

	$self->success($data);
}

sub configs_routing {
	my $self     = shift;
	my $cdn_name = $self->param('name');
	my $data_obj;
	my $json = $self->gen_traffic_router_config($cdn_name);
	$self->success($json);
}

sub gen_traffic_router_config {
	my $self     = shift;
	my $cdn_name = shift;
	my $data_obj;
	my $ccr_profile_id;
	my $ccr_domain_name = "";
	my $cdn_soa_minimum = 30;
	my $cdn_soa_expire  = 604800;
	my $cdn_soa_retry   = 7200;
	my $cdn_soa_refresh = 28800;
	my $cdn_soa_admin   = "traffic_ops";
	my $tld_ttls_soa    = 86400;
	my $tld_ttls_ns     = 3600;

	$SIG{__WARN__} = sub {
		warn $_[0]
			unless $_[0] =~ m/Prefetching multiple has_many rels deliveryservice_servers/;
	};

	$data_obj->{'stats'}->{'cdnName'}           = $cdn_name;
	$data_obj->{'stats'}->{'date'}              = time();
	$data_obj->{'stats'}->{'trafficOpsVersion'} = &tm_version();
	$data_obj->{'stats'}->{'trafficOpsPath'} =
		$self->req->url->path->{'path'};
	$data_obj->{'stats'}->{'trafficOpsHost'} = $self->req->headers->host;
	$data_obj->{'stats'}->{'trafficOpsUser'} =
		$self->current_user()->{username};

	my @cdn_profiles = $self->db->resultset('Server')->search( { 'cdn.name' => $cdn_name }, { prefetch => ['cdn'] } )->get_column('profile')->all();
	if ( scalar(@cdn_profiles) ) {
		$ccr_profile_id =
			$self->db->resultset('Profile')->search( { id => { -in => \@cdn_profiles }, name => { -like => 'CCR%' } } )->get_column('id')->single();
		if ( !defined($ccr_profile_id) ) {
			my $e = Mojo::Exception->throw("No CCR profile found in profile IDs: @cdn_profiles ");
		}
	}
	else {
		my $e = Mojo::Exception->throw( "No profiles found for CDN_name: " . $cdn_name );
	}

	my %condition = (
		'profile_parameters.profile' => $ccr_profile_id,
		'config_file'                => 'CRConfig.json'
	);
	$ccr_domain_name = $self->db->resultset('Cdn')->search({ 'name' => $cdn_name})->get_column('domain_name')->single();
	my $rs_config = $self->db->resultset('Parameter')->search( \%condition, { join => 'profile_parameters' } );
	while ( my $row = $rs_config->next ) {
		if ( $row->name eq 'tld.soa.admin' ) {
			$cdn_soa_admin = $row->value;
		}
		if ( $row->name eq 'tld.soa.expire' ) {
			$cdn_soa_expire = $row->value;
		}
		if ( $row->name eq 'tld.soa.minimum' ) {
			$cdn_soa_minimum = $row->value;
		}
		if ( $row->name eq 'tld.soa.refresh' ) {
			$cdn_soa_refresh = $row->value;
		}
		if ( $row->name eq 'tld.soa.retry' ) {
			$cdn_soa_retry = $row->value;
		}
		if ( $row->name eq 'tld.ttls.SOA' ) {
			$tld_ttls_soa = $row->value;
		}
		if ( $row->name eq 'tld.ttls.NS' ) {
			$tld_ttls_ns = $row->value;
		}

		my $parameter->{'type'} = "parameter";
		if ( $row->value =~ m/^\d+$/ ) {
			$data_obj->{'config'}->{ $row->name } = int( $row->value );
		}
		else {
			$data_obj->{'config'}->{ $row->name } = $row->value;
		}
	}

	my $rs_loc = $self->db->resultset('Server')->search(
		{ 'cdn.name' => $cdn_name },
		{
			join   => [ 'cdn',             'cachegroup' ],
			select => [ 'cachegroup.name', 'cachegroup.latitude', 'cachegroup.longitude' ],
			distinct => 1
		}
	);
	while ( my $row = $rs_loc->next ) {
		my $cache_group;
		my $latitude  = $row->cachegroup->latitude + 0;
		my $longitude = $row->cachegroup->longitude + 0;
		$cache_group->{'coordinates'}->{'latitude'}  = $latitude;
		$cache_group->{'coordinates'}->{'longitude'} = $longitude;
		$cache_group->{'name'}                       = $row->cachegroup->name;
		push( @{ $data_obj->{'cacheGroups'} }, $cache_group );
	}

	my $regex_tracker;
	my $rs_regexes = $self->db->resultset('Regex')->search( {}, { 'prefetch' => 'type' } );
	while ( my $row = $rs_regexes->next ) {
		$regex_tracker->{ $row->id }->{'type'}    = $row->type->name;
		$regex_tracker->{ $row->id }->{'pattern'} = $row->pattern;
	}
	my %cache_tracker;
	my $rs_caches = $self->db->resultset('Server')->search(
		{ 'profile' => { -in => \@cdn_profiles } },
		{
			prefetch => [ 'type',      'status',      'cachegroup', 'profile' ],
			columns  => [ 'host_name', 'domain_name', 'tcp_port',   'interface_name', 'ip_address', 'ip6_address', 'id', 'xmpp_id' ]
		}
	);
	while ( my $row = $rs_caches->next ) {
		if ( $row->type->name eq "RASCAL" ) {
			my $traffic_monitor;
			$traffic_monitor->{'hostName'} = $row->host_name;
			$traffic_monitor->{'fqdn'}     = $row->host_name . "." . $row->domain_name;
			$traffic_monitor->{'status'}   = $row->status->name;
			$traffic_monitor->{'location'} = $row->cachegroup->name;
			$traffic_monitor->{'port'}     = int( $row->tcp_port );
			$traffic_monitor->{'ip'}       = $row->ip_address;
			$traffic_monitor->{'ip6'}      = $row->ip6_address;
			$traffic_monitor->{'profile'}  = $row->profile->name;
			push( @{ $data_obj->{'trafficMonitors'} }, $traffic_monitor );

		}
		elsif ( $row->type->name eq "CCR" ) {
			my $rs_param = $self->db->resultset('Parameter')->search(
				{
					'profile_parameters.profile' => $row->profile->id,
					'name'                       => 'api.port'
				},
				{ join => 'profile_parameters' }
			);
			my $r = $rs_param->single;
			my $api_port =
				( defined($r) && defined( $r->value ) ) ? $r->value : 3333;

			my $traffic_router;

			$traffic_router->{'hostName'} = $row->host_name;
			$traffic_router->{'fqdn'}     = $row->host_name . "." . $row->domain_name;
			$traffic_router->{'status'}   = $row->status->name;
			$traffic_router->{'location'} = $row->cachegroup->name;
			$traffic_router->{'port'}     = int( $row->tcp_port );
			$traffic_router->{'apiPort'}  = int($api_port);
			$traffic_router->{'ip'}       = $row->ip_address;
			$traffic_router->{'ip6'}      = $row->ip6_address;
			$traffic_router->{'profile'}  = $row->profile->name;
			push( @{ $data_obj->{'trafficRouters'} }, $traffic_router );
		}
		elsif ( $row->type->name =~ m/^EDGE/ || $row->type->name =~ m/^MID/ ) {
			if ( !exists $cache_tracker{ $row->id } ) {
				$cache_tracker{ $row->id } = $row->host_name;
			}

			my $traffic_server;
			$traffic_server->{'cacheGroup'}    = $row->cachegroup->name;
			$traffic_server->{'hostName'}      = $row->host_name;
			$traffic_server->{'fqdn'}          = $row->host_name . "." . $row->domain_name;
			$traffic_server->{'port'}          = int( $row->tcp_port );
			$traffic_server->{'interfaceName'} = $row->interface_name;
			$traffic_server->{'status'}        = $row->status->name;
			$traffic_server->{'ip'}            = $row->ip_address;
			$traffic_server->{'ip6'}           = ( $row->ip6_address || "" );
			$traffic_server->{'profile'}       = $row->profile->name;
			$traffic_server->{'type'}          = $row->type->name;
			$traffic_server->{'hashId'}        = $row->xmpp_id;
			push( @{ $data_obj->{'trafficServers'} }, $traffic_server );
		}

	}

	my $ds_regex_tracker;
	my $regexps;
	my $rs_ds = $self->db->resultset('Deliveryservice')
		->search( { 'me.profile' => $ccr_profile_id, 'active' => 1 }, { prefetch => [ 'deliveryservice_servers', 'deliveryservice_regexes', 'type' ] } );
	while ( my $row = $rs_ds->next ) {
		my $delivery_service;
		$delivery_service->{'xmlId'} = $row->xml_id;
		my $protocol;
		if ( $row->type->name =~ m/DNS/ ) {
			$protocol = 'DNS';
		}
		else {
			$protocol = 'HTTP';
		}
		my @server_subrows = $row->deliveryservice_servers->all;
		my @regex_subrows  = $row->deliveryservice_regexes->all;
		my $regex_to_props;
		my %ds_to_remap;
		if ( scalar(@regex_subrows) ) {
			foreach my $subrow (@regex_subrows) {
				$delivery_service->{'matchSets'}->[ $subrow->set_number ]->{'protocol'} = $protocol;
				$regex_to_props->{ $subrow->{'_column_data'}->{'regex'} }->{'pattern'} =
					$regex_tracker->{ $subrow->{'_column_data'}->{'regex'} }->{'pattern'};
				$regex_to_props->{ $subrow->{'_column_data'}->{'regex'} }->{'setNumber'} = $subrow->set_number;
				$regex_to_props->{ $subrow->{'_column_data'}->{'regex'} }->{'type'} = $regex_tracker->{ $subrow->{'_column_data'}->{'regex'} }->{'type'};
				if ( $regex_to_props->{ $subrow->{'_column_data'}->{'regex'} }->{'type'} eq 'HOST_REGEXP' ) {
					$ds_to_remap{ $row->xml_id }->[ $subrow->set_number ] = $regex_to_props->{ $subrow->{'_column_data'}->{'regex'} }->{'pattern'};
				}
			}
		}
		my $domains;
		foreach my $regex ( sort keys %{$regex_to_props} ) {
			my $set_number = $regex_to_props->{$regex}->{'setNumber'};
			my $pattern    = $regex_to_props->{$regex}->{'pattern'};
			my $type       = $regex_to_props->{$regex}->{'type'};
			if ( $type eq 'HOST_REGEXP' ) {
				push( @{ $delivery_service->{'matchSets'}->[$set_number]->{'matchList'} }, { 'matchType' => 'HOST', 'regex' => $pattern } );
				my $host = $pattern;
				$host =~ s/\\//g;
				$host =~ s/\.\*//g;
				$host =~ s/\.//g;
				push @$domains, "$host.$ccr_domain_name";
			}
			elsif ( $type eq 'PATH_REGEXP' ) {
				push( @{ $delivery_service->{'matchSets'}->[$set_number]->{'matchList'} }, { 'matchType' => 'PATH', 'regex' => $pattern } );
			}
			elsif ( $type eq 'HEADER_REGEXP' ) {
				push( @{ $delivery_service->{'matchSets'}->[$set_number]->{'matchList'} }, { 'matchType' => 'HEADER', 'regex' => $pattern } );
			}
		}
		$delivery_service->{'domains'} = $domains;
		if ( scalar(@server_subrows) ) {

			#my $host_regex = qr/(^(\.)+\*\\\.)(.*)(\\\.(\.)+\*$)/;
			my $host_regex1 = qr/\\|\.\*/;

			#MAT: Have to do this dedup because @server_subrows contains duplicates (* the # of host regexes)
			my %server_subrow_dedup;
			foreach my $subrow (@server_subrows) {
				$server_subrow_dedup{ $subrow->{'_column_data'}->{'server'} } =
					$subrow->{'_column_data'}->{'deliveryservice'};
			}
			my $ds_regex->{'xmlId'} = $row->xml_id;
			foreach my $server ( keys %server_subrow_dedup ) {
				my @remaps;
				foreach my $host ( @{ $ds_to_remap{ $row->xml_id } } ) {
					my $remap;
					if ( $host =~ m/\.\*$/ ) {
						my $host_copy = $host;
						$host_copy =~ s/$host_regex1//g;
						if ( $protocol eq 'DNS' ) {
							$remap = 'edge' . $host_copy . $ccr_domain_name;
						}
						else {
							my $cache_tracker_server = $cache_tracker{$server} || "";
							my $host_copy            = $host_copy              || "";
							my $ccr_domain_name      = $ccr_domain_name        || "";
							$remap = $cache_tracker_server . $host_copy . $ccr_domain_name;
						}
					}
					else {
						$remap = $host;
					}
					push( @remaps, $remap );
				}
				my $cache_tracker_server = $cache_tracker{$server} || "";
				push( @{ $ds_regex_tracker->{$cache_tracker_server}->{ $row->xml_id }->{'remaps'} }, @remaps );
			}
		}

		$delivery_service->{'ttl'} = int( $row->ccr_dns_ttl );
		my $geo_limit = $row->geo_limit;
		if ( $geo_limit == 1 ) {

			# Ref to 0 or 1 makes JSON bool value
			$delivery_service->{'coverageZoneOnly'} = \1;
			$delivery_service->{'geoEnabled'}       = [];
		}
		elsif ( $geo_limit == 2 ) {

			# Ref to 0 or 1 makes JSON bool value
			$delivery_service->{'coverageZoneOnly'} = \0;
			$delivery_service->{'geoEnabled'} = [ { 'countryCode' => 'US' } ];
		}
		elsif ( $geo_limit == 3 ) {

			# Ref to 0 or 1 makes JSON bool value
			$delivery_service->{'coverageZoneOnly'} = \0;
			$delivery_service->{'geoEnabled'} = [ { 'countryCode' => 'CA' } ];
		}
		else {
			# Ref to 0 or 1 makes JSON bool value
			$delivery_service->{'coverageZoneOnly'} = \0;
			$delivery_service->{'geoEnabled'}       = [];
		}
		my $bypass_destination;
		if ( $protocol =~ m/DNS/ ) {
			$bypass_destination->{'type'} = 'DNS';
			if ( defined( $row->dns_bypass_ip ) && $row->dns_bypass_ip ne "" ) {
				$bypass_destination->{'ip'} = $row->dns_bypass_ip;
			}
			if ( defined( $row->dns_bypass_ip6 )
				&& $row->dns_bypass_ip6 ne "" )
			{
				$bypass_destination->{'ip6'} = $row->dns_bypass_ip6;
			}
			if ( defined( $row->dns_bypass_cname )
				&& $row->dns_bypass_cname ne "" )
			{
				$bypass_destination->{'cname'} = $row->dns_bypass_cname;
			}
			if ( defined( $row->dns_bypass_ttl )
				&& $row->dns_bypass_ttl ne "" )
			{
				$bypass_destination->{'ttl'} = int( $row->dns_bypass_ttl );
			}
			if ( defined( $row->max_dns_answers )
				&& $row->max_dns_answers ne "" )
			{
				$bypass_destination->{'maxDnsIpsForLocation'} = int( $row->max_dns_answers );
			}
		}
		elsif ( $protocol =~ m/HTTP/ ) {
			$bypass_destination->{'type'} = 'HTTP';
			if ( defined( $row->http_bypass_fqdn )
				&& $row->http_bypass_fqdn ne "" )
			{
				my $full = $row->http_bypass_fqdn;
				my $fqdn;
				if ( $full =~ m/\:/ ) {
					my $port;
					( $fqdn, $port ) = split( /\:/, $full );
					# Specify port number only if explicitly set by the DS 'Bypass FQDN' field - issue 1493
					$bypass_destination->{'port'} = int($port);
				}
				else {
					$fqdn = $full;
				}
				$bypass_destination->{'fqdn'} = $fqdn;
			}
		}
		$delivery_service->{'bypassDestination'} = $bypass_destination;

		if ( defined( $row->miss_lat ) && $row->miss_lat ne "" ) {
			$delivery_service->{'missCoordinates'}->{'latitude'} = $row->miss_lat + 0;
		}
		if ( defined( $row->miss_long ) && $row->miss_long ne "" ) {
			$delivery_service->{'missCoordinates'}->{'longitude'} = $row->miss_long + 0;
		}
		$delivery_service->{'ttls'} = {
			'A'    => int( $row->ccr_dns_ttl ),
			'AAAA' => int( $row->ccr_dns_ttl ),
			'NS'   => int($tld_ttls_ns),
			'SOA'  => int($tld_ttls_soa)
		};
		$delivery_service->{'soa'}->{'minimum'} = int($cdn_soa_minimum);
		$delivery_service->{'soa'}->{'expire'}  = int($cdn_soa_expire);
		$delivery_service->{'soa'}->{'retry'}   = int($cdn_soa_retry);
		$delivery_service->{'soa'}->{'refresh'} = int($cdn_soa_retry);
		$delivery_service->{'soa'}->{'admin'}   = $cdn_soa_admin;

		my $rs_dns = $self->db->resultset('Staticdnsentry')->search(
			{
				'deliveryservice.active'  => 1,
				'deliveryservice.profile' => $ccr_profile_id
			}, {
				prefetch => [ 'deliveryservice', 'type' ],
				columns  => [ 'host',            'type', 'ttl', 'address' ]
			}
		);

		while ( my $dns_row = $rs_dns->next ) {
			my $dns_obj;
			$dns_obj->{'name'}  = $dns_row->host;
			$dns_obj->{'ttl'}   = int( $dns_row->ttl );
			$dns_obj->{'value'} = $dns_row->address;

			my $type = $dns_row->type->name;
			$type =~ s/\_RECORD//g;
			$dns_obj->{'type'} = $type;
			push( @{ $delivery_service->{'staticDnsEntries'} }, $dns_obj );
		}

		push( @{ $data_obj->{'deliveryServices'} }, $delivery_service );
	}

	foreach my $cache_hostname ( sort keys %{$ds_regex_tracker} ) {
		my $i = 0;
		my $server_ref;
		foreach my $traffic_server ( @{ $data_obj->{'trafficServers'} } ) {
			$i++;
			my $traffic_server_hostname = $traffic_server->{'hostName'} || "";
			next if ( $traffic_server_hostname ne $cache_hostname );
			$server_ref = $data_obj->{'trafficServers'}->[ $i - 1 ];
		}

		foreach my $xml_id ( sort keys %{ $ds_regex_tracker->{$cache_hostname} } ) {
			my $ds;
			$ds->{'xmlId'} = $xml_id;
			$ds->{'remaps'} =
				$ds_regex_tracker->{$cache_hostname}->{$xml_id}->{'remaps'};
			push( @{ $server_ref->{'deliveryServices'} }, $ds );
			$data_obj->{'trafficServers'}->[$i] = $server_ref;
		}
	}

	my @empty_array;
	foreach my $traffic_server ( @{ $data_obj->{'trafficServers'} } ) {
		if ( !defined( $traffic_server->{'deliveryServices'} ) ) {
			push( @{ $traffic_server->{'deliveryServices'} }, @empty_array );
		}
	}
	return ($data_obj);
}

# Produces a list of Cdns for traversing child links
sub get_cdns {
	my $self = shift;

	my $rs_data =
		$self->db->resultset("Cdn")->search( {}, { order_by => "name" } );
	my $json_response = $self->build_cdns_json( $rs_data, "id,name" );

	#push( @{$json_response}, { "links" => [ { "rel" => "configs", "href" => "child" } ] } );
	$self->success($json_response);
}

sub build_cdns_json {
	my $self            = shift;
	my $rs_data         = shift;
	my $default_columns = shift;
	my $columns;

	if ( defined $self->param('columns') ) {
		$columns = $self->param('columns');
	}
	else {
		$columns = $default_columns;
	}

	my (@columns) = split( /,/, $columns );
	my %columns;
	foreach my $col (@columns) {
		$columns{$col} = defined;
	}

	my @data;
	my @cols = grep { exists $columns{$_} } $rs_data->result_source->columns;

	while ( my $row = $rs_data->next ) {
		my %parameter;
		foreach my $col (@cols) {
			$parameter{$col} = $row->$col;
		}
		push( @data, \%parameter );
	}
	return \@data;
}

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

sub dnssec_keys {
	my $self       = shift;
	my $is_updated = 0;

	my $cdn_name = $self->param('name');
	my $keys;
	my $response_container = $self->riak_get( "dnssec", $cdn_name );
	my $get_keys = $response_container->{'response'};
	if ( $get_keys->is_success() ) {
		$keys = decode_json( $get_keys->content );
		return $self->success($keys);
	}
	else {
		return $self->success({}, " - Dnssec keys for $cdn_name could not be found. ");
	}
	return $self->alert( { Error => " - You must be an ADMIN to perform this operation!" } );
}

#checks if keys are expired and re-generates them if they are.
sub dnssec_keys_refresh {
	my $self = shift;

	# fork and daemonize so we can avoid blocking
	my $rc = $self->fork_and_daemonize();
	if ( $rc < 0 ) {
		my $error = "Unable to fork_and_daemonize to check DNSSEC keys for refresh in the background";
		$self->app->log->fatal($error);
		return $self->alert( { Error => $error } );
	}
	elsif ( $rc > 0 ) {
    	# This is the parent, report success and return
		return $self->success("Checking DNSSEC keys for refresh in the background");
	}

	# we're in the fork()ed process now, do the work and exit
	$self->refresh_keys();
	exit(0);
}

sub refresh_keys {
	my $self       = shift;
	my $is_updated = 0;
	my $error_message;
	$self->app->log->debug("Starting refresh of DNSSEC keys");
	my $rs_data = $self->db->resultset("Cdn")->search( {}, { order_by => "name" } );

	while ( my $row = $rs_data->next ) {
		if ( $row->dnssec_enabled == 1 ) {
			my $cdn_name = $row->name;
			my $cdn_domain_name = $row->domain_name;
			my $keys;
			my $response_container = $self->riak_get( "dnssec", $cdn_name );
			my $get_keys = $response_container->{'response'};
			if ( !$get_keys->is_success() ) {
				$error_message = "Can't update dnssec keys for $cdn_name!  Response was: " . $get_keys->content;
				$self->app->log->warn($error_message);
				next;
			}

			$keys = decode_json( $get_keys->content );

			#get DNSKEY ttl, generation multiplier, and effective mutiplier for CDN TLD
			my $profile_id = $self->get_profile_id_by_cdn($cdn_name);
			my $dnskey_gen_multiplier;
			my $dnskey_ttl;
			my $dnskey_effective_multiplier;
			my %condition = (
				'parameter.name' => 'tld.ttls.DNSKEY',
				'profile.name'   => $profile_id
			);
			my $rs_pp =
				$self->db->resultset('ProfileParameter')->search( \%condition, { prefetch => [ { 'parameter' => undef }, { 'profile' => undef } ] } )
				->single;
			$rs_pp ? $dnskey_ttl = $rs_pp->parameter->value : $dnskey_ttl = '60';

			%condition = (
				'parameter.name' => 'DNSKEY.generation.multiplier',
				'profile.name'   => $profile_id
			);
			$rs_pp = $self->db->resultset('ProfileParameter')->search( \%condition, { prefetch => [ { 'parameter' => undef }, { 'profile' => undef } ] } )
				->single;
			$rs_pp
				? $dnskey_gen_multiplier = $rs_pp->parameter->value
				: $dnskey_gen_multiplier = '10';

			%condition = (
				'parameter.name' => 'DNSKEY.effective.multiplier',
				'profile.name'   => $profile_id
			);
			$rs_pp = $self->db->resultset('ProfileParameter')->search( \%condition, { prefetch => [ { 'parameter' => undef }, { 'profile' => undef } ] } )
				->single;
			$rs_pp
				? $dnskey_effective_multiplier = $rs_pp->parameter->value
				: $dnskey_effective_multiplier = '10';

			my $key_expiration = time() + ( $dnskey_ttl * $dnskey_gen_multiplier );

			#get default expiration days and ttl for DSs from CDN record
			my $default_k_exp_days = "365";
			my $default_z_exp_days = "30";
			my $cdn_ksk            = $keys->{$cdn_name}->{ksk};
			foreach my $cdn_krecord (@$cdn_ksk) {
				my $cdn_kstatus = $cdn_krecord->{status};
				if ( $cdn_kstatus eq 'new' ) {    #ignore anything other than the 'new' record
					my $cdn_k_exp   = $cdn_krecord->{expirationDate};
					my $cdn_k_incep = $cdn_krecord->{inceptionDate};
					$default_k_exp_days = ( $cdn_k_exp - $cdn_k_incep ) / 86400;
				}
			}
			my $cdn_zsk = $keys->{$cdn_name}->{zsk};
			foreach my $cdn_zrecord (@$cdn_zsk) {
				my $cdn_zstatus = $cdn_zrecord->{status};
				if ( $cdn_zstatus eq 'new' ) {    #ignore anything other than the 'new' record
					my $cdn_z_exp   = $cdn_zrecord->{expirationDate};
					my $cdn_z_incep = $cdn_zrecord->{inceptionDate};
					$default_z_exp_days = ( $cdn_z_exp - $cdn_z_incep ) / 86400;

					#check if zsk is expired, if so re-generate
					if ( $cdn_z_exp < $key_expiration ) {

						#if expired create new keys
						$self->app->log->info("The ZSK keys for $cdn_name are expired!");
						my $effective_date = $cdn_z_exp - ( $dnskey_ttl * $dnskey_effective_multiplier );
						my $new_dnssec_keys = $self->regen_expired_keys( "zsk", $cdn_name, $keys, $effective_date );
						$keys->{$cdn_name} = $new_dnssec_keys;
					}
				}
			}

			#get DeliveryServices for CDN
			my %search = ( cdn_id => $row->id );
			my @ds_rs = $self->db->resultset('Deliveryservice')->search( \%search, { prefetch => ['type'] });

			foreach my $ds (@ds_rs) {
				my $type = $ds->type->name;
				if (   $type !~ m/^HTTP/
					&& $type !~ m/^CLIENT_STEERING$/
					&& $type !~ m/^STEERING$/
					&& $type !~ m/^DNS/ )
				{
					next;
				}

				#check if keys exist for ds
				my $xml_id  = $ds->xml_id;
				my $ds_keys = $keys->{$xml_id};
				if ( !$ds_keys ) {

					#create keys
					$self->app->log->info("Keys do not exist for ds $xml_id");
					my $ds_id = $ds->id;

					#create the ds domain name for dnssec keys
					my $ds_name = UI::DeliveryService::get_ds_domain_name($self, $ds_id, $xml_id, $cdn_domain_name);

					my $inception    = time();
					my $z_expiration = $inception + ( 86400 * $default_z_exp_days );
					my $k_expiration = $inception + ( 86400 * $default_k_exp_days );

					my $zsk = $self->get_dnssec_keys( "zsk", $ds_name, $dnskey_ttl, $inception, $z_expiration, "new", $inception );
					my $ksk = $self->get_dnssec_keys( "ksk", $ds_name, $dnskey_ttl, $inception, $k_expiration, "new", $inception );

					#add to keys hash
					$keys->{$xml_id} = { zsk => [$zsk], ksk => [$ksk] };

					#update is_updated param
					$is_updated = 1;
				}
				#if keys do exist, check expiration
				else {
					my $ksk = $ds_keys->{ksk};
					foreach my $krecord (@$ksk) {
						my $kstatus = $krecord->{status};
						if ( $kstatus eq 'new' ) {
							if ( $krecord->{expirationDate} < $key_expiration ) {
								#if expired create new keys
								$self->app->log->info("The KSK keys for $xml_id are expired!");
								my $effective_date = $krecord->{expirationDate} - ( $dnskey_ttl * $dnskey_effective_multiplier );
								my $new_dnssec_keys = $self->regen_expired_keys( "ksk", $xml_id, $keys, $effective_date );
								$keys->{$xml_id} = $new_dnssec_keys;

								#update is_updated param
								$is_updated = 1;
							}
						}
					}
					my $zsk = $ds_keys->{zsk};
					foreach my $zrecord (@$zsk) {
						my $zstatus = $zrecord->{status};
						if ( $zstatus eq 'new' ) {
							if ( $zrecord->{expirationDate} < $key_expiration ) {

								#if expired create new keys
								$self->app->log->info("The ZSK keys for $xml_id are expired!");
								my $effective_date = $zrecord->{expirationDate} - ( $dnskey_ttl * $dnskey_effective_multiplier );
								my $new_dnssec_keys = $self->regen_expired_keys( "zsk", $xml_id, $keys, $effective_date );
								$keys->{$xml_id} = $new_dnssec_keys;

								#update is_updated param
								$is_updated = 1;
							}
						}
					}
				}
			}

			if ( $is_updated == 1 ) {
				# #convert hash to json and store in Riak
				my $json_data = encode_json($keys);
				$response_container = $self->riak_put( "dnssec", $cdn_name, $json_data );
			}

			my $response = $response_container->{"response"};
			if ( !$response->is_success() ) {
				$error_message = "DNSSEC keys could not be stored for $cdn_name!  Response was: " . $response->content;
				$self->app->log->warn($error_message);
				next;
			}
		}
	}
	$self->app->log->debug("Done refreshing DNSSEC keys");
}

sub regen_expired_keys {
	my $self           = shift;
	my $type           = shift;
	my $key            = shift;
	my $existing_keys  = shift;
	my $effective_date = shift;
	my $tld            = shift;
	my $reset_exp      = shift;
	my $regen_keys     = {};
	my $old_key;

	my $existing = $existing_keys->{$key}->{$type};
	foreach my $record (@$existing) {
		if ( $record->{status} eq 'new' ) {
			$old_key = $record;
		}
	}
	my $name            = $old_key->{name};
	my $ttl             = $old_key->{ttl};
	my $expiration      = $old_key->{expirationDate};
	my $inception       = $old_key->{inceptionDate};
	my $expiration_days = ( $expiration - $inception ) / 86400;

	#create new expiration and inception time
	my $new_inception = time();
	my $new_expiration = $new_inception + ( 86400 * $expiration_days );

	#generate new keys
	my $new_key = $self->get_dnssec_keys( $type, $name, $ttl, $new_inception, $new_expiration, "new", $effective_date, $tld );

	if ( $type eq "ksk" ) {

		#get existing zsk
		my @zsk = $existing_keys->{$key}->{zsk};

		#set existing ksk status to "expired"
		$old_key->{status} = "expired";
		if ($reset_exp) {
			$old_key->{expirationDate} = $effective_date;
		}
		$regen_keys = { zsk => @zsk, ksk => [ $new_key, $old_key ] };
	}
	elsif ( $type eq "zsk" ) {

		#get existing ksk
		my @ksk = $existing_keys->{$key}->{ksk};

		#set existing ksk status to "expired"
		$old_key->{status} = "expired";
		if ($reset_exp) {
			$old_key->{expirationDate} = $effective_date;
		}
		$regen_keys = { zsk => [ $new_key, $old_key ], ksk => @ksk };
	}
	return $regen_keys;
}

sub dnssec_keys_generate {
	my $self = shift;

	my $key_type      = "dnssec";
	my $key           = $self->req->json->{key};
	my $name          = $self->req->json->{name};
	my $ttl           = $self->req->json->{ttl};
	my $k_exp_days    = $self->req->json->{kskExpirationDays};
	my $z_exp_days    = $self->req->json->{zskExpirationDays};
	my $effectiveDate = $self->req->json->{effectiveDate};
	if ( !defined($effectiveDate) ) {
		$effectiveDate = time();
	}
	my $res      = $self->generate_store_dnssec_keys( $key, $name, $ttl, $k_exp_days, $z_exp_days, $effectiveDate );
	my $response = $res->{response};
	my $rc       = $response->{_rc};
	if ( $rc eq "204" ) {
		&log( $self, "Generated DNSSEC keys for CDN $key", "APICHANGE" );
		$self->success_message("Successfully created $key_type keys for $key");
	}
	else {
		$self->alert( { Error => " - DNSSEC keys for $key could not be created.  Response was" . $response->content } );
	}
}

sub delete_dnssec_keys {
	my $self     = shift;
	my $key      = $self->param('name');
	my $key_type = "dnssec";
	my $response;

	$self->app->log->info("deleting key_type = $key_type, key = $key");
	my $response_container = $self->riak_delete( $key_type, $key );
	$response = $response_container->{"response"};
	if ( $response->is_success() ) {
		&log( $self, "Deleted DNSSEC keys for CDN $key", "UICHANGE" );
		$self->success("Successfully deleted $key_type keys for $key");
	}
	else {
		$self->alert( { Error => " - SSL keys for key type $key_type and key $key could not be deleted.  Response was" . $response->content } );
	}
}

sub ssl_keys {
	my $self = shift;

	my $cdn_name = $self->param('name');
	my $keys;

	#get "latest" ssl records for all DSs in the CDN
	my $response_container = $self->riak_search( "sslkeys", "q=cdn:$cdn_name&fq=_yz_rk:*latest&start=0&rows=1000" );
	my $response = $response_container->{'response'};
	if ( $response->is_success() ) {
		my $content = decode_json( $response->content )->{response}->{docs};
		unless ( scalar(@$content) > 0 ) {
			return $self->render( json => { "message" => "No SSL certificates found for $cdn_name" }, status => 404 );
		}
		foreach my $record (@$content) {
			push(
				@$keys, {
					deliveryservice => $record->{deliveryservice},
					certificate     => {
						crt => $record->{'certificate.crt'},
						key => $record->{'certificate.key'},
					},
					hostname => $record->{hostname}
				}
			);
		}
		return $self->success($keys);
	}

	return $self->alert( { Error => " - Could not retrieve SSL records for $cdn_name!  Response was: " . $response->content } );
}

sub tool_logout {
	my $self = shift;

	$self->logout();
	$self->success_message("You are logged out.");
}

sub catch_all {
	my $self     = shift;
	my $mimetype = $self->req->headers->content_type;

	if ( defined( $self->current_user() ) ) {
		if ( &UI::Utils::is_ldap( $self ) ) {
			my $config = $self->app->config;
			return $self->forbidden( $config->{'to'}{'no_account_found_msg'} );
		} else {
			return $self->not_found();
		}
	}
	else {
		return $self->unauthorized();
	}
}

1;
