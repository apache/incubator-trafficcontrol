package API::Configs::ApacheTrafficServer;

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
use Date::Manip;
use NetAddr::IP;
use Data::Dumper;
use UI::DeliveryService;
use JSON;
use API::DeliveryService::KeysUrlSig qw(URL_SIG_KEYS_BUCKET);
use URI;
use File::Basename;
use File::Path;

use constant {
	HTTP					=> 0,
	HTTPS					=> 1,
	HTTP_AND_HTTPS  		=> 2,
	HTTP_TO_HTTPS  			=> 3,
	CONSISTENT_HASH			=> 0,
	PRIMARY_BACKUP			=> 1,
	STRICT_ROUND_ROBIN 		=> 2,
	IP_ROUND_ROBIN			=> 3,
	LATCH_ON_FAILOVER		=> 4,
	RRH_DONT_CACHE			=> 0,
	RRH_BACKGROUND_FETCH	=> 1,
	RRH_CACHE_RANGE_REQUEST	=> 2,
};

#Sub to generate config metadata
sub get_config_metadata {
	my $self     = shift;
	my $id       = $self->param('id');

	##verify that a valid server ID has been used
	my $server_obj = $self->server_data($id);
	if ( !defined($server_obj) ) {
		return $self->not_found();
	}
	my $data_obj;
	my $host_name = $server_obj->host_name;

	my %condition = ( 'me.host_name' => $host_name );
	my $rs_server = $self->db->resultset('Server')->search( \%condition, { prefetch => [ 'cdn', 'profile' ] } );
	my $tm_url = $self->db->resultset('Parameter')->search( { -and => [ name => 'tm.url', config_file => 'global' ] } )->get_column('value')->first();
	my $tm_rev_proxy_url = $self->db->resultset('Parameter')->search( { -and => [ name => 'tm.rev_proxy.url', config_file => 'global' ] } )->get_column('value')->first();
	my $cdn_name = $server_obj->cdn->name;
	my $server = $rs_server->next;
	my $config_file_obj;
	my @config_files;
	if ($server) {

		$data_obj->{'info'}->{'serverName'}	 = $server_obj->host_name;
		$data_obj->{'info'}->{'serverIpv4'}	 = $server_obj->ip_address;
		$data_obj->{'info'}->{'serverId'}		 = $server_obj->id;
		$data_obj->{'info'}->{'profileName'}	 = $server->profile->name;
		$data_obj->{'info'}->{'profileId'}		 = $server->profile->id;
		$data_obj->{'info'}->{'cdnName'}		 = $cdn_name;
		$data_obj->{'info'}->{'cdnId'}			 = $server->cdn->id;
		$data_obj->{'info'}->{'toUrl'}			 = $tm_url;
		$data_obj->{'info'}->{'serverTcpPort'} = $server_obj->tcp_port;
		if ( $tm_rev_proxy_url ) {
			$data_obj->{'info'}->{'toRevProxyUrl'}	= $tm_rev_proxy_url;
		}

		%condition = (
			'profile_parameters.profile' => $server->profile->id,
			-or                          => [ 'name' => 'location' , 'name' => 'URL' ]
		);
		my $rs_param = $self->db->resultset('Parameter')->search( \%condition, { join => 'profile_parameters' } );
		while ( my $param = $rs_param->next ) {
			if ( $param->name eq 'location' ) {
				$config_file_obj->{ $param->config_file }->{'fnameOnDisk'} = $param->config_file;
				$config_file_obj->{ $param->config_file }->{'location'} = $param->value;
			}
			if ( $param->name eq 'URL' ) {
				$config_file_obj->{ $param->config_file }->{'url'} = $param->value;
			}
		}

		# Synthesize file entries for uri_signing delivery services.
		if ( defined $config_file_obj->{'remap.config'} && defined $config_file_obj->{'remap.config'}->{'location'} ) {
			# Use the location of the remap.config for these files.
			my $config_location = $config_file_obj->{'remap.config'}->{'location'};

			# Query the xml_ids of all the delivery services to which this server is assigned
			# that use uri_signing.
			my $rs_uri_signing = $self->db->resultset('Deliveryservice')->search(
				{
					'server.host_name' => $host_name,
					'me.signing_algorithm' => 'uri_signing'
				},
				{
					join => {'deliveryservice_servers' => 'server'},
					columns => ['me.xml_id']
				}
			);

			while (my $us = $rs_uri_signing->next) {
				# If there's already a parameter for it, don't clobber it. The user
				# may wish to override the location.

				my $cfg_name = 'uri_signing_'.$us->xml_id.'.config';
				if (! defined $config_file_obj->{$cfg_name}) {
					$config_file_obj->{$cfg_name}->{'fnameOnDisk'} = $cfg_name;
					$config_file_obj->{$cfg_name}->{'location'} = $config_location;
				}
			}
		}

	}



	foreach my $config_file ( keys %{ $config_file_obj } ) {
		my $scope = $self->get_scope($config_file, $server_obj);
		my $scope_id;
		if (defined ( $config_file_obj->{$config_file}->{'url'} ) ) {
			$scope = 'cdns';
		}
		if ( $scope eq 'cdns' ) {
			$scope_id = $server->cdn->name;
		}
		elsif ( $scope eq 'profiles' ) {
			$scope_id = $server->profile->name;
		}
		else {
			$scope_id = $server_obj->host_name;
		}
		if (defined ( $config_file_obj->{$config_file}->{'url'} ) ) {
			#delete the url element so we don't see it in the metadata:
			delete $config_file_obj->{$config_file}->{'apiUri'};
		}
		else {
			$config_file_obj->{$config_file}->{'apiUri'} = "/api/1.2/" . $scope . "/" . $scope_id . "/configfiles/ats/" . $config_file;
		}
		$config_file_obj->{$config_file}->{'scope'} = $scope;
	}

	foreach my $config_file ( sort keys %{ $config_file_obj } ) {
		push ( @config_files, $config_file_obj->{$config_file} );
	}

	$data_obj->{'configFiles'} = \@config_files;
	my $file_contents = encode_json($data_obj);

	return $self->render( text => $file_contents, format => 'txt' );
}

#entry point for server scope api route.
sub get_server_config {
	my $self     = shift;
	my $filename = $self->param("filename");
	my $id       = $self->param('id');

	##verify that a valid server ID has been used
	my $server_obj = $self->server_data($id);
	if ( !defined($server_obj) ) {
		return $self->not_found();
	}

	my $scope    = $self->get_scope($filename, $server_obj);

	##check the scope - is this the correct route?
	if ( $scope ne 'servers' ) {
		return $self->alert( "Error - incorrect file scope for route used.  Please use the " . $scope . " route." );
	}

	#generate the config file using the appropriate function
	my $file_contents;
	if ( $filename =~ /to_ext_.*\.config/ ) { $file_contents = $self->to_ext_dot_config( $server_obj, $filename ); }
	elsif ( $filename eq "cache.config" ) { $file_contents = $self->server_cache_dot_config( $server_obj, $filename ); }
	elsif ( $filename eq "ip_allow.config" ) { $file_contents = $self->ip_allow_dot_config( $server_obj, $filename ); }
	elsif ( $filename eq "parent.config" ) { $file_contents = $self->parent_dot_config( $server_obj, $filename ); }
	elsif ( $filename eq "hosting.config" ) { $file_contents = $self->hosting_dot_config( $server_obj, $filename ); }
	elsif ( $filename eq "remap.config" ) { $file_contents = $self->remap_dot_config( $server_obj, $filename ); }
	elsif ( $filename eq "packages" ) {
		$file_contents = $self->get_package_versions( $server_obj, $filename );
		$file_contents = encode_json($file_contents);
	}
	elsif ( $filename eq "chkconfig" ) {
		$file_contents = $self->get_chkconfig( $server_obj, $filename );
		$file_contents = encode_json($file_contents);
	}
	else {
		my $file_param = $self->db->resultset('Parameter')->search( [ config_file => $filename ] )->first;
		if ( !defined($file_param) ) {
			return $self->not_found();
		}
		$file_contents = $self->take_and_bake_server( $server_obj, $filename );
	}

	#if we get an empty file, just send back an error.
	if ( !defined($file_contents) ) {
		return $self->not_found();
	}

	#return the file contents for fetch and db actions.
	return $self->render( text => $file_contents, format => 'txt' );
}

#entry point for cdn scope api route.
sub get_cdn_config {
	my $self     = shift;
	my $filename = $self->param('filename');
	my $id       = $self->param('id');
	my $scope    = $self->get_scope($filename);
	my $ext_url  = $self->param('url');

	##check the scope - is this the correct route?
	if ( $scope ne 'cdns' ) {
		return $self->alert( "Error - incorrect file scope for route used.  Please use the " . $scope . " route." );
	}

	##verify that a valid cdn ID has been used
	my $cdn_obj = $self->cdn_data($id);
	if ( !defined($cdn_obj) ) {
		return $self->not_found();
	}

	#generate the config file using the appropriate function
	my $file_contents;
	if ( $filename eq "bg_fetch.config" ) { $file_contents = $self->bg_fetch_dot_config( $cdn_obj, $filename ); }
	elsif ( $filename =~ /cacheurl.*\.config/ ) { $file_contents = $self->cacheurl_dot_config( $cdn_obj, $filename ); }
	elsif ( $filename =~ /hdr_rw_.*\.config/ ) { $file_contents = $self->header_rewrite_dot_config( $cdn_obj, $filename ); }
	elsif ( $filename =~ /regex_remap_.*\.config/ ) { $file_contents = $self->regex_remap_dot_config( $cdn_obj, $filename ); }
	elsif ( $filename eq "regex_revalidate.config" ) { $file_contents = $self->regex_revalidate_dot_config( $cdn_obj, $filename ); }
	elsif ( $filename =~ /set_dscp_.*\.config/ ) { $file_contents = $self->set_dscp_dot_config( $cdn_obj, $filename ); }
	elsif ( $filename eq "ssl_multicert.config" ) { $file_contents = $self->ssl_multicert_dot_config( $cdn_obj, $filename ); }
	else { return $self->not_found(); }

	if ( !defined($file_contents) ) {
		return $self->not_found();
	}

	return $self->render( text => $file_contents, format => 'txt' );
}

#entry point for profile scope api route.
sub get_profile_config {
	my $self     = shift;
	my $filename = $self->param("filename");
	my $id       = $self->param('id');
	my $scope    = $self->get_scope($filename);

	##check the scope - is this the correct route?
	if ( $scope ne 'profiles' ) {
		return $self->alert( "Error - incorrect file scope for route used.  Please use the " . $scope . " route." );
	}

	##verify that a valid profile ID has been used
	my $profile_obj = $self->profile_data($id);
	if ( !defined($profile_obj) ) {
		return $self->not_found();
	}

	#generate the config file using the appropriate function
	my $file_contents;
	if ( $filename eq "50-ats.rules" ) { $file_contents = $self->ats_dot_rules( $profile_obj, $filename ); }
	elsif ( $filename eq "12M_facts" ) { $file_contents = $self->facts( $profile_obj, $filename ); }
	elsif ( $filename eq "astats.config" ) { $file_contents = $self->generic_profile_config( $profile_obj, $filename ); }
	elsif ( $filename eq "cache.config" ) { $file_contents = $self->profile_cache_dot_config( $profile_obj, $filename ); }
	elsif ( $filename eq "drop_qstring.config" ) { $file_contents = $self->drop_qstring_dot_config( $profile_obj, $filename ); }
	elsif ( $filename eq "logs_xml.config" ) { $file_contents = $self->logs_xml_dot_config( $profile_obj, $filename ); }
	elsif ( $filename eq "logging.config" ) { $file_contents = $self->logging_dot_config( $profile_obj, $filename ); }
	elsif ( $filename eq "plugin.config" ) { $file_contents = $self->generic_profile_config( $profile_obj, $filename ); }
	elsif ( $filename eq "records.config" ) { $file_contents = $self->generic_profile_config( $profile_obj, $filename ); }
	elsif ( $filename eq "storage.config" ) { $file_contents = $self->storage_dot_config( $profile_obj, $filename ); }
	elsif ( $filename eq "sysctl.conf" ) { $file_contents = $self->generic_profile_config( $profile_obj, $filename ); }
	elsif ( $filename =~ /url_sig_.*\.config/ ) { $file_contents = $self->url_sig_dot_config( $profile_obj, $filename ); }
	elsif ( $filename =~ /uri_signing_.*\.config/ ) { $file_contents = $self->uri_signing_dot_config( $filename ); }
	elsif ( $filename eq "volume.config" ) { $file_contents = $self->volume_dot_config( $profile_obj, $filename ); }
	else {
		my $file_param = $self->db->resultset('Parameter')->search( [ config_file => $filename ] )->first;
		if ( !defined($file_param) ) {
			return $self->not_found();
		}
		$file_contents = $self->take_and_bake_profile( $profile_obj, $filename );
	}

	if ( !defined($file_contents) ) {
		return $self->not_found();
	}

	return $self->render( text => $file_contents, format => 'txt' );
}

my $separator ||= {
	"records.config"  => " ",
	"plugin.config"   => " ",
	"sysctl.conf"     => " = ",
	"url_sig_.config" => " = ",
	"astats.config"   => "=",
};

#identify the correct scope for each filename.  if not found, returns server scope as any
#undefined parameter based configs are designed with server scope using the take-and-bake sub.
sub get_scope {
	my $self  = shift;
	my $fname = shift;
	my $server_obj = shift;
	my $scope;

	my $type;
	if ( defined $server_obj ) { $type = $server_obj->type->name; }
	else { $type = ""; }

	if ( $fname eq "ip_allow.config" )                         { $scope = 'servers' }
	elsif ( $fname eq "parent.config" )                        { $scope = 'servers' }
	elsif ( $fname =~ /to_ext_.*\.config/ )                    { $scope = 'servers' }
	elsif ( $fname eq "hosting.config" )                       { $scope = 'servers' }
	elsif ( $fname eq "packages" )                             { $scope = 'servers' }
	elsif ( $fname eq "chkconfig" )                            { $scope = 'servers' }
	elsif ( $fname eq "remap.config" )                         { $scope = 'servers' }
	elsif ( $fname eq "12M_facts" )                            { $scope = 'profiles' }
	elsif ( $fname eq "50-ats.rules" )                         { $scope = 'profiles' }
	elsif ( $fname eq "astats.config" )                        { $scope = 'profiles' }
	elsif ( $fname eq "cache.config" && $type =~ m/^MID/ )     { $scope = 'servers' }
	elsif ( $fname eq "cache.config" )                         { $scope = 'profiles' }
	elsif ( $fname eq "drop_qstring.config" )                  { $scope = 'profiles' }
	elsif ( $fname eq "logs_xml.config" )                      { $scope = 'profiles' }
	elsif ( $fname eq "logging.config" )                       { $scope = 'profiles' }
	elsif ( $fname eq "plugin.config" )                        { $scope = 'profiles' }
	elsif ( $fname eq "records.config" )                       { $scope = 'profiles' }
	elsif ( $fname eq "storage.config" )                       { $scope = 'profiles' }
	elsif ( $fname eq "sysctl.conf" )                          { $scope = 'profiles' }
	elsif ( $fname =~ /url_sig_.*\.config/ )                   { $scope = 'profiles' }
	elsif ( $fname =~ /uri_signing_.*\.config/ )               { $scope = 'profiles' }
	elsif ( $fname eq "volume.config" )                        { $scope = 'profiles' }
	elsif ( $fname eq "bg_fetch.config" )                      { $scope = 'cdns' }
	elsif ( $fname =~ /cacheurl.*\.config/ )                   { $scope = 'cdns' }
	elsif ( $fname =~ /hdr_rw_.*\.config/ )                    { $scope = 'cdns' }
	elsif ( $fname =~ /regex_remap_.*\.config/ )               { $scope = 'cdns' }
	elsif ( $fname eq "regex_revalidate.config" )              { $scope = 'cdns' }
	elsif ( $fname =~ /set_dscp_.*\.config/ )                  { $scope = 'cdns' }
	elsif ( $fname eq "ssl_multicert.config" )                 { $scope = 'cdns' }
	else {
		$scope = $self->db->resultset('Parameter')->search( { -and => [ name => 'scope', config_file => $fname ] } )->get_column('value')->first();
		if ( !defined($scope) ) {
			$scope = 'servers';
		}
	}

	return $scope;
}

#takes the server name or ID and turns it into a server object that can reference either, making either work for the request.
sub server_data {
	my $self = shift;
	my $id   = shift;

	my $server_obj;

	#if an ID is passed, look up by ID.  Otherwise, look up by hostname.
	if ( $id =~ /^\d+$/ ) {
		$server_obj = $self->db->resultset('Server')->search( { 'me.id' => $id }, { prefetch => [ 'profile', 'type', 'cachegroup', 'cdn' ] } )->single;
	}
	else {
		$server_obj = $self->db->resultset('Server')->search( { host_name => $id }, { prefetch => [ 'profile', 'type', 'cachegroup', 'cdn' ] } )->single;
	}

	return $server_obj;
}

#takes the profile name or ID and turns it into a server object that can reference either, making either work for the request.
sub profile_data {
	my $self = shift;
	my $id   = shift;

	#if an ID is passed, look up by ID.  Otherwise, look up by profile name.
	my $profile_obj;
	if ( $id =~ /^\d+$/ ) {
		$profile_obj = $self->db->resultset('Profile')->search( { id => $id } )->first;
	}
	else {
		$profile_obj = $self->db->resultset('Profile')->search( { name => $id } )->first
	}

	return $profile_obj;
}

#takes the server name or ID and turns it into a server object that can reference either, making either work for the request.
sub cdn_data {
	my $self = shift;
	my $id   = shift;
	my $cdn_obj;

	if ( $id =~ /^\d+$/ ) {
		$cdn_obj = $self->db->resultset('Cdn')->search( { id => $id } )->first;
	}
	else {
		$cdn_obj = $self->db->resultset('Cdn')->search( { name => $id } )->first;
	}

	return $cdn_obj;
}

#generates the comment at the top of config files.
sub header_comment {
	my $self = shift;
	my $name = shift;

	my $text = "# DO NOT EDIT - Generated for " . $name . " by " . &name_version_string($self) . " on " . `date`;
	return $text;
}

#retrieves parameter data for a specific server by searching by the server's assigned profile.
sub param_data {
	my $self       = shift;
	my $server_obj = shift;
	my $filename   = shift;
	my $data;

	my $rs = $self->db->resultset('ProfileParameter')->search( { -and => [ profile => $server_obj->profile->id, 'parameter.config_file' => $filename ] },
		{ prefetch => [ { parameter => undef }, { profile => undef } ] } );
	while ( my $row = $rs->next ) {
		if ( $row->parameter->name eq "location" ) {
			next;
		}
		my $value = $row->parameter->value;

		# some files have multiple lines with the same key... handle that with param id.
		my $key = $row->parameter->name;
		if ( defined( $data->{$key} ) ) {
			$key .= "__" . $row->parameter->id;
		}
		if ( $value =~ /^STRING __HOSTNAME__$/ ) {
			$value = "STRING " . $server_obj->host_name . "." . $server_obj->domain_name;
		}
		$data->{$key} = $value;
	}
	return $data;
}

#retrieves parameter data for a specific profile by searching by the profile id.
sub profile_param_data {
	my $self     = shift;
	my $profile  = shift;
	my $filename = shift;
	my $data;

	my $rs = $self->db->resultset('ProfileParameter')->search( { -and => [ profile => $profile, 'parameter.config_file' => $filename ] },
		{ prefetch => [ { parameter => undef }, { profile => undef } ] } );
	while ( my $row = $rs->next ) {
		if ( $row->parameter->name eq "location" ) {
			next;
		}
		my $value = $row->parameter->value;

		# some files have multiple lines with the same key... handle that with param id.
		my $key = $row->parameter->name;
		if ( defined( $data->{$key} ) ) {
			$key .= "__" . $row->parameter->id;
		}
		if ( $value =~ /^STRING __HOSTNAME__$/ ) {
			$value = "STRING " . "__FULL_HOSTNAME__";
		}
		$data->{$key} = $value;
	}
	return $data;
}

#searches for the a specific parameter by name in a specific profile by profile ID and returns the data.
sub profile_param_value {
	my $self       = shift;
	my $pid        = shift;
	my $file       = shift;
	my $param_name = shift;
	my $default    = shift;

	# assign $ds_domain, $weight and $port, and cache the results %profile_cache
	my $param =
		$self->db->resultset('ProfileParameter')
		->search( { -and => [ profile => $pid, 'parameter.config_file' => $file, 'parameter.name' => $param_name ] },
		{ prefetch => [ 'parameter', 'profile' ] } )->first();

	return ( defined $param ? $param->parameter->value : $default );
}

sub delivery_service_data_by_profile {
	my $self = shift;
	my $profile_id = shift;
	my @data;

	my $dbh = $self->db->storage->dbh;
	my $qry = 'select
		deliveryservice.id,
		deliveryservice.xml_id,
		deliveryservice.dscp,
		deliveryservice.routing_name,
		deliveryservice.signing_algorithm,
		deliveryservice.qstring_ignore,
		(SELECT o.protocol::text || \'://\' || o.fqdn || rtrim(concat(\':\', o.port::text), \':\')
			FROM origin o
			WHERE o.deliveryservice = deliveryservice.id
			AND o.is_primary) as org_server_fqdn,
		deliveryservice.origin_shield,
		regex.pattern AS pattern,
    	retype.name AS re_type,
    	dstype.name AS ds_type,
    	cdn.domain_name AS domain_name,
		deliveryservice.profile,
		deliveryservice.protocol,
		deliveryservice.ssl_key_version,
		deliveryservice.range_request_handling,
		deliveryservice.fq_pacing_rate,
		deliveryservice.edge_header_rewrite,
		deliveryservice.mid_header_rewrite,
		deliveryservice.regex_remap,
		deliveryservice.cacheurl,
		deliveryservice.remap_text,
		deliveryservice.multi_site_origin,
		deliveryservice.multi_site_origin_algorithm
		from deliveryservice
		JOIN deliveryservice_regex ON deliveryservice_regex.deliveryservice = deliveryservice.id
        JOIN regex ON deliveryservice_regex.regex = regex.id
        JOIN type as retype ON regex.type = retype.id
        JOIN type as dstype ON deliveryservice.type = dstype.id
        JOIN cdn ON cdn.id = deliveryservice.cdn_id
		where deliveryservice.id = ANY (
			select DISTINCT
			deliveryservice
			from deliveryservice_server
			where server = ANY (
				select
				id
				from server
				where profile = $1
			)
		);
	';

	my $stmt = $dbh->prepare($qry);
	$stmt->execute($profile_id);

	my $deliveryservice_id;
	my $deliveryservice_xml_id;
	my $deliveryservice_dscp;
	my $deliveryservice_routing_name;
	my $deliveryservice_signing_algorithm;
	my $deliveryservice_qstring_ignore;
	my $deliveryservice_org_server_fqdn;
	my $deliveryservice_origin_shield;
	my $pattern;
	my $re_type;
	my $ds_type;
	my $domain_name;
	my $deliveryservice_profile;
	my $deliveryservice_protocol;
	my $deliveryservice_ssl_key_version;
	my $deliveryservice_range_request_handling;
	my $deliveryservice_fq_pacing_rate;
	my $deliveryservice_edge_header_rewrite;
	my $deliveryservice_mid_header_rewrite;
	my $deliveryservice_regex_remap;
	my $deliveryservice_cacheurl;
	my $deliveryservice_remap_text;
	my $deliveryservice_multi_site_origin;
	my $deliveryservice_multi_site_origin_algorithm;

	$stmt->bind_columns(
		\$deliveryservice_id,
		\$deliveryservice_xml_id,
		\$deliveryservice_dscp,
		\$deliveryservice_routing_name,
		\$deliveryservice_signing_algorithm,
		\$deliveryservice_qstring_ignore,
		\$deliveryservice_org_server_fqdn,
		\$deliveryservice_origin_shield,
		\$pattern,
		\$re_type,
		\$ds_type,
		\$domain_name,
		\$deliveryservice_profile,
		\$deliveryservice_protocol,
		\$deliveryservice_ssl_key_version,
		\$deliveryservice_range_request_handling,
		\$deliveryservice_fq_pacing_rate,
		\$deliveryservice_edge_header_rewrite,
		\$deliveryservice_mid_header_rewrite,
		\$deliveryservice_regex_remap,
		\$deliveryservice_cacheurl,
		\$deliveryservice_remap_text,
		\$deliveryservice_multi_site_origin,
		\$deliveryservice_multi_site_origin_algorithm
		);

	while ( my $row = $stmt->fetch() ) {
		push(
				@data, {
					"id" => $deliveryservice_id,
					"xml_id" => $deliveryservice_xml_id,
					"dscp" => $deliveryservice_dscp,
					"routing_name" => $deliveryservice_routing_name,
					"signed" => ( $deliveryservice_signing_algorithm eq "url_sig" ),
					"signing_algorithm" => $deliveryservice_signing_algorithm,
					"qstring_ignore" => $deliveryservice_qstring_ignore,
					"org_server_fqdn" => $deliveryservice_org_server_fqdn,
					"origin_shield" => $deliveryservice_origin_shield,
					"pattern" => $pattern,
					"re_type" => $re_type,
					"ds_type" => $ds_type,
					"domain_name" => $domain_name,
					"profile" => $deliveryservice_profile,
					"protocol" => $deliveryservice_protocol,
					"ssl_key_version" => $deliveryservice_ssl_key_version,
					"range_request_handling" => $deliveryservice_range_request_handling,
					"fq_pacing_rate" => $deliveryservice_fq_pacing_rate,
					"edge_header_rewrite" => $deliveryservice_edge_header_rewrite,
					"mid_header_rewrite" => $deliveryservice_mid_header_rewrite,
					"regex_remap" => $deliveryservice_regex_remap,
					"cacheurl" => $deliveryservice_cacheurl,
					"remap_text" => $deliveryservice_remap_text,
					"multi_site_origin" => $deliveryservice_multi_site_origin,
					"multi_site_origin_algorithm" => $deliveryservice_multi_site_origin_algorithm
				}
			);
	}

	return @data;
}

#gets the delivery service data for all servers assigned to a profile.
sub profile_ds_data {
	my $self = shift;
	my $profile_obj   = shift;
	my $dsinfo;

	my @rs = $self->delivery_service_data_by_profile( $profile_obj->id );

	my $j = 0;
	while ( scalar(@rs) !=0 ) {
		my $row = shift(@rs);

		my $org_server                  = $row->{'org_server_fqdn'};
		my $dscp                        = $row->{'dscp'};
		my $re_type                     = $row->{'re_type'};
		my $ds_type                     = $row->{'ds_type'};
		my $signed                      = $row->{'signed'};
		my $signing_algorithm           = $row->{'signing_algorithm'};
		my $qstring_ignore              = $row->{'qstring_ignore'};
		my $ds_xml_id                   = $row->{'xml_id'};
		my $ds_domain                   = $row->{'domain_name'};
		my $edge_header_rewrite         = $row->{'edge_header_rewrite'};
		my $mid_header_rewrite          = $row->{'mid_header_rewrite'};
		my $regex_remap                 = $row->{'regex_remap'};
		my $protocol                    = $row->{'protocol'};
		my $range_request_handling      = $row->{'range_request_handling'};
		my $fq_pacing_rate              = $row->{'fq_pacing_rate'};
		my $origin_shield               = $row->{'origin_shield'};
		my $cacheurl                    = $row->{'cacheurl'};
		my $remap_text                  = $row->{'remap_text'};
		my $multi_site_origin           = $row->{'multi_site_origin'};

		if ( $re_type eq 'HOST_REGEXP' ) {
			my $host_re = $row->{'pattern'};
			#my $map_to  = $org_server . "/";
			my $map_to;
			if ( defined($org_server) ) {
				$map_to  = $org_server . "/";
			}
			if ( $host_re =~ /\.\*$/ ) {
				my $re = $host_re;
				$re =~ s/\\//g;
				$re =~ s/\.\*//g;
				my $hname    = $ds_type =~ /^DNS/ ? $row->{'routing_name'} : "__http__";
				my $portstr  = ":" . "__SERVER_TCP_PORT__";
				my $map_from = "http://" . $hname . $re . $ds_domain . $portstr . "/";
				if ( $protocol == HTTP ) {
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTPS || $protocol == HTTP_TO_HTTPS ) {
					$map_from = "https://" . $hname . $re . $ds_domain . "/";
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTP_AND_HTTPS ) {

					#add the first one with http
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;

					#add the second one for https
					my $map_from2 = "https://" . $hname . $re . $ds_domain . "/";
					$dsinfo->{dslist}->[$j]->{"remap_line2"}->{$map_from2} = $map_to;
				}
			}
			else {
				my $map_from = "http://" . $host_re . "/";
				if ( $protocol == HTTP ) {
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTPS || $protocol == HTTP_TO_HTTPS ) {
					$map_from = "https://" . $host_re . "/";
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTP_AND_HTTPS ) {

					#add the first with http
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;

					#add the second with https
					my $map_from2 = "https://" . $host_re . "/";
					$dsinfo->{dslist}->[$j]->{"remap_line2"}->{$map_from2} = $map_to;
				}
			}
		}

		$dsinfo->{dslist}->[$j]->{"dscp"}                        = $dscp;
		$dsinfo->{dslist}->[$j]->{"org"}                         = $org_server;
		$dsinfo->{dslist}->[$j]->{"type"}                        = $ds_type;
		$dsinfo->{dslist}->[$j]->{"domain"}                      = $ds_domain;
		$dsinfo->{dslist}->[$j]->{"signed"}                      = $signed;
		$dsinfo->{dslist}->[$j]->{"signing_algorithm"}           = $signing_algorithm;
		$dsinfo->{dslist}->[$j]->{"qstring_ignore"}              = $qstring_ignore;
		$dsinfo->{dslist}->[$j]->{"ds_xml_id"}                   = $ds_xml_id;
		$dsinfo->{dslist}->[$j]->{"edge_header_rewrite"}         = $edge_header_rewrite;
		$dsinfo->{dslist}->[$j]->{"mid_header_rewrite"}          = $mid_header_rewrite;
		$dsinfo->{dslist}->[$j]->{"regex_remap"}                 = $regex_remap;
		$dsinfo->{dslist}->[$j]->{"range_request_handling"}      = $range_request_handling;
		$dsinfo->{dslist}->[$j]->{"fq_pacing_rate"}              = $fq_pacing_rate;
		$dsinfo->{dslist}->[$j]->{"origin_shield"}               = $origin_shield;
		$dsinfo->{dslist}->[$j]->{"cacheurl"}                    = $cacheurl;
		$dsinfo->{dslist}->[$j]->{"remap_text"}                  = $remap_text;
		$dsinfo->{dslist}->[$j]->{"multi_site_origin"}           = $multi_site_origin;

		if ( defined($edge_header_rewrite) ) {
			my $fname = "hdr_rw_" . $ds_xml_id . ".config";
			$dsinfo->{dslist}->[$j]->{"hdr_rw_file"} = $fname;
		}
		if ( defined($mid_header_rewrite) ) {
			my $fname = "hdr_rw_mid_" . $ds_xml_id . ".config";
			$dsinfo->{dslist}->[$j]->{"mid_hdr_rw_file"} = $fname;
		}
		if ( defined($cacheurl) ) {
			my $fname = "cacheurl_" . $ds_xml_id . ".config";
			$dsinfo->{dslist}->[$j]->{"cacheurl_file"} = $fname;
		}
		if ( defined( $row->{'profile'} ) ) {
			my $dsparamrs = $self->db->resultset('ProfileParameter')->search( { profile => $row->{'profile'} }, { prefetch => [ 'profile', 'parameter' ] } );
			while ( my $prow = $dsparamrs->next ) {
				$dsinfo->{dslist}->[$j]->{'param'}->{ $prow->parameter->config_file }->{ $prow->parameter->name } = $prow->parameter->value;
			}
		}

		$j++;
	}

	#	$self->app->session->{dsinfo} = $dsinfo;
	return $dsinfo;
}

#gets the delivery service data for an entire CDN.
sub cdn_ds_data {
	my $self = shift;
	my $cdn_obj   = shift;
	my $dsinfo;

	my $rs = $self->db->resultset('DeliveryServiceInfoForDomainList')->search( {}, { bind => [$cdn_obj->name] } );

	my $j = 0;
	while ( my $row = $rs->next ) {
		my $org_server                  = $row->org_server_fqdn;
		my $dscp                        = $row->dscp;
		my $re_type                     = $row->re_type;
		my $ds_type                     = $row->ds_type;
		my $signed                      = defined( $row->signing_algorithm ) ? ( $row->signing_algorithm eq "url_sig" ? \1 : \0 ) : \0;
		my $signing_algorithm           = $row->signing_algorithm;
		my $qstring_ignore              = $row->qstring_ignore;
		my $ds_xml_id                   = $row->xml_id;
		my $ds_domain                   = $row->domain_name;
		my $edge_header_rewrite         = $row->edge_header_rewrite;
		my $mid_header_rewrite          = $row->mid_header_rewrite;
		my $regex_remap                 = $row->regex_remap;
		my $protocol                    = $row->protocol;
		my $range_request_handling      = $row->range_request_handling;
		my $fq_pacing_rate              = $row->fq_pacing_rate;
		my $origin_shield               = $row->origin_shield;
		my $cacheurl                    = $row->cacheurl;
		my $remap_text                  = $row->remap_text;
		my $multi_site_origin           = $row->multi_site_origin;

		if ( $re_type eq 'HOST_REGEXP' ) {
			my $host_re = $row->pattern;
			#my $map_to  = $org_server . "/";
			my $map_to;
			if ( defined($org_server) ) {
				$map_to  = $org_server . "/";
			}
			if ( $host_re =~ /\.\*$/ ) {
				my $re = $host_re;
				$re =~ s/\\//g;
				$re =~ s/\.\*//g;
				my $hname    = $ds_type =~ /^DNS/ ? $row->routing_name : "__http__";
				my $portstr  = ":" . "SERVER_TCP_PORT";
				my $map_from = "http://" . $hname . $re . $ds_domain . $portstr . "/";
				if ( $protocol == HTTP ) {
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTPS || $protocol == HTTP_TO_HTTPS ) {
					$map_from = "https://" . $hname . $re . $ds_domain . "/";
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTP_AND_HTTPS ) {

					#add the first one with http
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;

					#add the second one for https
					my $map_from2 = "https://" . $hname . $re . $ds_domain . "/";
					$dsinfo->{dslist}->[$j]->{"remap_line2"}->{$map_from2} = $map_to;
				}
			}
			else {
				my $map_from = "http://" . $host_re . "/";
				if ( $protocol == HTTP ) {
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTPS || $protocol == HTTP_TO_HTTPS ) {
					$map_from = "https://" . $host_re . "/";
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTP_AND_HTTPS ) {

					#add the first with http
					$dsinfo->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;

					#add the second with https
					my $map_from2 = "https://" . $host_re . "/";
					$dsinfo->{dslist}->[$j]->{"remap_line2"}->{$map_from2} = $map_to;
				}
			}
		}

		$dsinfo->{dslist}->[$j]->{"dscp"}                        = $dscp;
		$dsinfo->{dslist}->[$j]->{"org"}                         = $org_server;
		$dsinfo->{dslist}->[$j]->{"type"}                        = $ds_type;
		$dsinfo->{dslist}->[$j]->{"domain"}                      = $ds_domain;
		$dsinfo->{dslist}->[$j]->{"signed"}                      = $signed;
		$dsinfo->{dslist}->[$j]->{"signing_algorithm"}           = $signing_algorithm;
		$dsinfo->{dslist}->[$j]->{"qstring_ignore"}              = $qstring_ignore;
		$dsinfo->{dslist}->[$j]->{"ds_xml_id"}                   = $ds_xml_id;
		$dsinfo->{dslist}->[$j]->{"edge_header_rewrite"}         = $edge_header_rewrite;
		$dsinfo->{dslist}->[$j]->{"mid_header_rewrite"}          = $mid_header_rewrite;
		$dsinfo->{dslist}->[$j]->{"regex_remap"}                 = $regex_remap;
		$dsinfo->{dslist}->[$j]->{"range_request_handling"}      = $range_request_handling;
		$dsinfo->{dslist}->[$j]->{"fq_pacing_rate"}              = $fq_pacing_rate;
		$dsinfo->{dslist}->[$j]->{"origin_shield"}               = $origin_shield;
		$dsinfo->{dslist}->[$j]->{"cacheurl"}                    = $cacheurl;
		$dsinfo->{dslist}->[$j]->{"remap_text"}                  = $remap_text;
		$dsinfo->{dslist}->[$j]->{"multi_site_origin"}           = $multi_site_origin;

		if ( defined($edge_header_rewrite) ) {
			my $fname = "hdr_rw_" . $ds_xml_id . ".config";
			$dsinfo->{dslist}->[$j]->{"hdr_rw_file"} = $fname;
		}
		if ( defined($mid_header_rewrite) ) {
			my $fname = "hdr_rw_mid_" . $ds_xml_id . ".config";
			$dsinfo->{dslist}->[$j]->{"mid_hdr_rw_file"} = $fname;
		}
		if ( defined($cacheurl) ) {
			my $fname = "cacheurl_" . $ds_xml_id . ".config";
			$dsinfo->{dslist}->[$j]->{"cacheurl_file"} = $fname;
		}
		if ( defined( $row->profile ) ) {
			my $dsparamrs = $self->db->resultset('ProfileParameter')->search( { profile => $row->profile }, { prefetch => [ 'profile', 'parameter' ] } );
			while ( my $prow = $dsparamrs->next ) {
				$dsinfo->{dslist}->[$j]->{'param'}->{ $prow->parameter->config_file }->{ $prow->parameter->name } = $prow->parameter->value;
			}
		}

		$j++;
	}

	#	$self->app->session->{dsinfo} = $dsinfo;
	return $dsinfo;
}

#gets the delivery service data for a specific server.
sub ds_data {
	my $self       = shift;
	my $server_obj = shift;
	my $response_obj;

	$response_obj->{host_name}   = $server_obj->host_name;
	$response_obj->{domain_name} = $server_obj->domain_name;

	my $rs_dsinfo;
	if ( $server_obj->type->name =~ m/^MID/ ) {
		# the mids will do all deliveryservices in this CDN
		$rs_dsinfo = $self->db->resultset('DeliveryServiceInfoForDomainList')->search( {}, { bind => [ $server_obj->cdn->name ] } );
	}
	else {
		$rs_dsinfo = $self->db->resultset('DeliveryServiceInfoForServerList')->search( {}, { bind => [ $server_obj->id ] } );
	}

	my $j = 0;
	while ( my $dsinfo = $rs_dsinfo->next ) {
		my $org_fqdn                  	= $dsinfo->org_server_fqdn;
		my $dscp                        = $dsinfo->dscp;
		my $re_type                     = $dsinfo->re_type;
		my $ds_type                     = $dsinfo->ds_type;
		my $signed                      = defined( $dsinfo->signing_algorithm ) ? ( $dsinfo->signing_algorithm eq "url_sig" ? \1 : \0 ) : \0;
		my $signing_algorithm           = $dsinfo->signing_algorithm;
		my $qstring_ignore              = $dsinfo->qstring_ignore;
		my $ds_xml_id                   = $dsinfo->xml_id;
		my $ds_domain                   = $dsinfo->domain_name;
		my $edge_header_rewrite         = $dsinfo->edge_header_rewrite;
		my $mid_header_rewrite          = $dsinfo->mid_header_rewrite;
		my $regex_remap                 = $dsinfo->regex_remap;
		my $protocol                    = $dsinfo->protocol;
		my $range_request_handling      = $dsinfo->range_request_handling;
		my $fq_pacing_rate              = $dsinfo->fq_pacing_rate;
		my $origin_shield               = $dsinfo->origin_shield;
		my $cacheurl                    = $dsinfo->cacheurl;
		my $remap_text                  = $dsinfo->remap_text;
		my $multi_site_origin           = $dsinfo->multi_site_origin;
		my $multi_site_origin_algorithm = 0;

		if ( $re_type eq 'HOST_REGEXP' && defined($org_fqdn) ) {
			my $host_re = $dsinfo->pattern;
			my $map_to  = $org_fqdn . "/";
			if ( $host_re =~ /\.\*$/ ) {
				my $re = $host_re;
				$re =~ s/\\//g;
				$re =~ s/\.\*//g;
				my $hname = $ds_type =~ /^DNS/ ? $dsinfo->routing_name : "__http__";
				my $portstr = "";
				if ( $hname eq "__http__" && $server_obj->tcp_port > 0 && $server_obj->tcp_port != 80 ) {
					$portstr = ":" . $server_obj->tcp_port;
				}
				my $map_from = "http://" . $hname . $re . $ds_domain . $portstr . "/";
				if ( $protocol == HTTP ) {
					$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTPS || $protocol == HTTP_TO_HTTPS ) {
					$map_from = "https://" . $hname . $re . $ds_domain . "/";
					$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTP_AND_HTTPS ) {

					#add the first one with http
					$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;

					#add the second one for https
					my $map_from2 = "https://" . $hname . $re . $ds_domain . "/";
					$response_obj->{dslist}->[$j]->{"remap_line2"}->{$map_from2} = $map_to;
				}
			}
			else {
				my $map_from = "http://" . $host_re . "/";
				if ( $protocol == HTTP ) {
					$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTPS || $protocol == HTTP_TO_HTTPS ) {
					$map_from = "https://" . $host_re . "/";
					$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
				}
				elsif ( $protocol == HTTP_AND_HTTPS ) {

					#add the first with http
					$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;

					#add the second with https
					my $map_from2 = "https://" . $host_re . "/";
					$response_obj->{dslist}->[$j]->{"remap_line2"}->{$map_from2} = $map_to;
				}
			}
		}
		$response_obj->{dslist}->[$j]->{"dscp"}                        = $dscp;
		$response_obj->{dslist}->[$j]->{"org"}                         = $org_fqdn;
		$response_obj->{dslist}->[$j]->{"type"}                        = $ds_type;
		$response_obj->{dslist}->[$j]->{"domain"}                      = $ds_domain;
		$response_obj->{dslist}->[$j]->{"signed"}                      = $signed;
		$response_obj->{dslist}->[$j]->{"signing_algorithm"}           = $signing_algorithm;
		$response_obj->{dslist}->[$j]->{"qstring_ignore"}              = $qstring_ignore;
		$response_obj->{dslist}->[$j]->{"ds_xml_id"}                   = $ds_xml_id;
		$response_obj->{dslist}->[$j]->{"edge_header_rewrite"}         = $edge_header_rewrite;
		$response_obj->{dslist}->[$j]->{"mid_header_rewrite"}          = $mid_header_rewrite;
		$response_obj->{dslist}->[$j]->{"regex_remap"}                 = $regex_remap;
		$response_obj->{dslist}->[$j]->{"range_request_handling"}      = $range_request_handling;
		$response_obj->{dslist}->[$j]->{"fq_pacing_rate"}              = $fq_pacing_rate;
		$response_obj->{dslist}->[$j]->{"origin_shield"}               = $origin_shield;
		$response_obj->{dslist}->[$j]->{"cacheurl"}                    = $cacheurl;
		$response_obj->{dslist}->[$j]->{"remap_text"}                  = $remap_text;
		$response_obj->{dslist}->[$j]->{"multi_site_origin"}           = $multi_site_origin;
		$response_obj->{dslist}->[$j]->{"multi_site_origin_algorithm"} = $multi_site_origin_algorithm;

		if ( defined($edge_header_rewrite) ) {
			my $fname = "hdr_rw_" . $ds_xml_id . ".config";
			$response_obj->{dslist}->[$j]->{"hdr_rw_file"} = $fname;
		}
		if ( defined($mid_header_rewrite) ) {
			my $fname = "hdr_rw_mid_" . $ds_xml_id . ".config";
			$response_obj->{dslist}->[$j]->{"mid_hdr_rw_file"} = $fname;
		}
		if ( defined($cacheurl) ) {
			my $fname = "cacheurl_" . $ds_xml_id . ".config";
			$response_obj->{dslist}->[$j]->{"cacheurl_file"} = $fname;
		}
		if ( defined( $dsinfo->profile ) ) {
			my $dsparamrs = $self->db->resultset('ProfileParameter')->search( { profile => $dsinfo->profile }, { prefetch => [ 'profile', 'parameter' ] } );
			while ( my $prow = $dsparamrs->next ) {
				$response_obj->{dslist}->[$j]->{'param'}->{ $prow->parameter->config_file }->{ $prow->parameter->name } = $prow->parameter->value;
			}
		}

		$j++;
	}

	return $response_obj;
}

sub parent_ds_data {
	my $self = shift;
	my $server_obj = shift;
	my $response_obj;

	$response_obj->{host_name}   = $server_obj->host_name;
	$response_obj->{domain_name} = $server_obj->domain_name;


	if ( $server_obj->type->name =~ m/^MID/ ) {
		# the mids will do all deliveryservices in this CDN
		my $rs_dsinfo = $self->db->resultset('DeliveryServiceInfoForDomainList')->search( {}, { bind => [ $server_obj->cdn->name ] } );

		my $j = 0;
		while ( my $dsinfo = $rs_dsinfo->next ) {
			my $org_fqdn                  	= $dsinfo->org_server_fqdn;
			my $qstring_ignore              = $dsinfo->qstring_ignore;
			my $ds_xml_id                   = $dsinfo->xml_id;
			my $origin_shield               = $dsinfo->origin_shield;
			my $multi_site_origin           = $dsinfo->multi_site_origin;

			$response_obj->{dslist}->[$j]->{"org"}                         = $org_fqdn;
			$response_obj->{dslist}->[$j]->{"qstring_ignore"}              = $qstring_ignore;
			$response_obj->{dslist}->[$j]->{"ds_xml_id"}                   = $ds_xml_id;
			$response_obj->{dslist}->[$j]->{"origin_shield"}               = $origin_shield;
			$response_obj->{dslist}->[$j]->{"multi_site_origin"}           = $multi_site_origin;


			if ( defined( $dsinfo->profile ) ) {
				my $dsparamrs = $self->db->resultset('ProfileParameter')->search( { profile => $dsinfo->profile }, { prefetch => [ 'profile', 'parameter' ] } );
				while ( my $prow = $dsparamrs->next ) {
					$response_obj->{dslist}->[$j]->{'param'}->{ $prow->parameter->config_file }->{ $prow->parameter->name } = $prow->parameter->value;
				}
			}

			$j++;
		}
		return $response_obj;
	}
	else {
		my $rs_dsinfo = $self->db->resultset('DeliveryServiceInfoForServerList')->search( {}, { bind => [ $server_obj->id ] } );

		my $j = 0;
		while ( my $dsinfo = $rs_dsinfo->next ) {
			my $org_fqdn                  	= $dsinfo->org_server_fqdn;
			my $ds_type                     = $dsinfo->ds_type;
			my $qstring_ignore              = $dsinfo->qstring_ignore;
			my $ds_xml_id                   = $dsinfo->xml_id;
			my $origin_shield               = $dsinfo->origin_shield;
			my $multi_site_origin           = $dsinfo->multi_site_origin;

			$response_obj->{dslist}->[$j]->{"org"}                         = $org_fqdn;
			$response_obj->{dslist}->[$j]->{"type"}                        = $ds_type;
			$response_obj->{dslist}->[$j]->{"qstring_ignore"}              = $qstring_ignore;

			if ( defined( $dsinfo->profile ) ) {
				my $dsqstring = $self->db->resultset('ProfileParameter')
				                ->search( { -and => [ 'parameter.config_file' => 'parent.config', 'parameter.name' => 'psel.qstring_handling', profile => $dsinfo->profile ] },
				                { prefetch => [ 'parameter' ] } )->get_column('parameter.value')->single();
				if ( defined $dsqstring ) {
					$response_obj->{dslist}->[$j]->{'param'}->{'parent.config'}->{'psel.qstring_handling'} = $dsqstring;
				}
			}
			$j++;
		}
		return $response_obj;
	}
}

sub remap_ds_data {
	my $self = shift;
	my $server_obj = shift;
	my $response_obj;

	$response_obj->{host_name}   = $server_obj->host_name;
	$response_obj->{domain_name} = $server_obj->domain_name;

	if ( $server_obj->type->name =~ m/^MID/ ) {
		# the mids will do all deliveryservices in this CDN
		my $rs_dsinfo = $self->db->resultset('DeliveryServiceInfoForDomainList')->search( {}, { bind => [ $server_obj->cdn->name ] } );

		my $j = 0;
		while ( my $dsinfo = $rs_dsinfo->next ) {
			my $org_fqdn                  	= $dsinfo->org_server_fqdn;
			my $ds_type                     = $dsinfo->ds_type;
			my $qstring_ignore              = $dsinfo->qstring_ignore;
			my $ds_xml_id                   = $dsinfo->xml_id;
			my $mid_header_rewrite          = $dsinfo->mid_header_rewrite;
			my $range_request_handling      = $dsinfo->range_request_handling;
			my $cacheurl                    = $dsinfo->cacheurl;



			$response_obj->{dslist}->[$j]->{"org"}                         = $org_fqdn;
			$response_obj->{dslist}->[$j]->{"type"}                        = $ds_type;
			$response_obj->{dslist}->[$j]->{"mid_header_rewrite"}          = $mid_header_rewrite;
			$response_obj->{dslist}->[$j]->{"cacheurl"}                    = $cacheurl;
			$response_obj->{dslist}->[$j]->{"qstring_ignore"}              = $qstring_ignore;
			$response_obj->{dslist}->[$j]->{"range_request_handling"}      = $range_request_handling;

			if ( defined($mid_header_rewrite) ) {
				my $fname = "hdr_rw_mid_" . $ds_xml_id . ".config";
				$response_obj->{dslist}->[$j]->{"mid_hdr_rw_file"} = $fname;
			}
			if ( defined($cacheurl) ) {
				my $fname = "cacheurl_" . $ds_xml_id . ".config";
				$response_obj->{dslist}->[$j]->{"cacheurl_file"} = $fname;
			}
			if ( defined( $dsinfo->profile ) ) {
				my $dsparamrs = $self->db->resultset('ProfileParameter')->search( { profile => $dsinfo->profile }, { prefetch => [ 'profile', 'parameter' ] } );
				while ( my $prow = $dsparamrs->next ) {
					$response_obj->{dslist}->[$j]->{'param'}->{ $prow->parameter->config_file }->{ $prow->parameter->name } = $prow->parameter->value;
				}
			}

			$j++;
		}
		return $response_obj;
	}
	else {
		my $rs_dsinfo = $self->db->resultset('DeliveryServiceInfoForServerList')->search( {}, { bind => [ $server_obj->id ] } );

		my $j = 0;
		while ( my $dsinfo = $rs_dsinfo->next ) {
			my $org_fqdn                  	= $dsinfo->org_server_fqdn;
			my $dscp                        = $dsinfo->dscp;
			my $re_type                     = $dsinfo->re_type;
			my $ds_type                     = $dsinfo->ds_type;
			my $signed                      = defined( $dsinfo->signing_algorithm ) ? ( $dsinfo->signing_algorithm eq "url_sig" ? \1 : \0 ) : \0;
			my $signing_algorithm           = $dsinfo->signing_algorithm;
			my $qstring_ignore              = $dsinfo->qstring_ignore;
			my $ds_xml_id                   = $dsinfo->xml_id;
			my $ds_domain                   = $dsinfo->domain_name;
			my $edge_header_rewrite         = $dsinfo->edge_header_rewrite;
			my $regex_remap                 = $dsinfo->regex_remap;
			my $protocol                    = $dsinfo->protocol;
			my $range_request_handling      = $dsinfo->range_request_handling;
			my $fq_pacing_rate              = $dsinfo->fq_pacing_rate;
			my $cacheurl                    = $dsinfo->cacheurl;
			my $remap_text                  = $dsinfo->remap_text;

			if ( $re_type eq 'HOST_REGEXP' && defined($org_fqdn) ) {
				my $host_re = $dsinfo->pattern;
				my $map_to  = $org_fqdn . "/";
				if ( $host_re =~ /\.\*$/ ) {
					my $re = $host_re;
					$re =~ s/\\//g;
					$re =~ s/\.\*//g;
					my $hname = $ds_type =~ /^DNS/ ? $dsinfo->routing_name : "__http__";
					my $portstr = "";
					if ( $hname eq "__http__" && $server_obj->tcp_port > 0 && $server_obj->tcp_port != 80 ) {
						$portstr = ":" . $server_obj->tcp_port;
					}
					my $map_from = "http://" . $hname . $re . $ds_domain . $portstr . "/";
					if ( $protocol == HTTP ) {
						$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
					}
					elsif ( $protocol == HTTPS || $protocol == HTTP_TO_HTTPS ) {
						$map_from = "https://" . $hname . $re . $ds_domain . "/";
						$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
					}
					elsif ( $protocol == HTTP_AND_HTTPS ) {

						#add the first one with http
						$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;

						#add the second one for https
						my $map_from2 = "https://" . $hname . $re . $ds_domain . "/";
						$response_obj->{dslist}->[$j]->{"remap_line2"}->{$map_from2} = $map_to;
					}
				}
				else {
					my $map_from = "http://" . $host_re . "/";
					if ( $protocol == HTTP ) {
						$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
					}
					elsif ( $protocol == HTTPS || $protocol == HTTP_TO_HTTPS ) {
						$map_from = "https://" . $host_re . "/";
						$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;
					}
					elsif ( $protocol == HTTP_AND_HTTPS ) {

						#add the first with http
						$response_obj->{dslist}->[$j]->{"remap_line"}->{$map_from} = $map_to;

						#add the second with https
						my $map_from2 = "https://" . $host_re . "/";
						$response_obj->{dslist}->[$j]->{"remap_line2"}->{$map_from2} = $map_to;
					}
				}
			}
			$response_obj->{dslist}->[$j]->{"dscp"}                        = $dscp;
			$response_obj->{dslist}->[$j]->{"org"}                         = $org_fqdn;
			$response_obj->{dslist}->[$j]->{"type"}                        = $ds_type;
			$response_obj->{dslist}->[$j]->{"signed"}                      = $signed;
			$response_obj->{dslist}->[$j]->{"signing_algorithm"}           = $signing_algorithm;
			$response_obj->{dslist}->[$j]->{"qstring_ignore"}              = $qstring_ignore;
			$response_obj->{dslist}->[$j]->{"ds_xml_id"}                   = $ds_xml_id;
			$response_obj->{dslist}->[$j]->{"edge_header_rewrite"}         = $edge_header_rewrite;
			$response_obj->{dslist}->[$j]->{"regex_remap"}                 = $regex_remap;
			$response_obj->{dslist}->[$j]->{"range_request_handling"}      = $range_request_handling;
			$response_obj->{dslist}->[$j]->{"fq_pacing_rate"}              = $fq_pacing_rate;
			$response_obj->{dslist}->[$j]->{"cacheurl"}                    = $cacheurl;
			$response_obj->{dslist}->[$j]->{"remap_text"}                  = $remap_text;

			if ( defined($edge_header_rewrite) ) {
				my $fname = "hdr_rw_" . $ds_xml_id . ".config";
				$response_obj->{dslist}->[$j]->{"hdr_rw_file"} = $fname;
			}
			if ( defined($cacheurl) ) {
				my $fname = "cacheurl_" . $ds_xml_id . ".config";
				$response_obj->{dslist}->[$j]->{"cacheurl_file"} = $fname;
			}
			if ( defined( $dsinfo->profile ) ) {
				my $dsparamrs = $self->db->resultset('ProfileParameter')->search( { profile => $dsinfo->profile }, { prefetch => [ 'profile', 'parameter' ] } );
				while ( my $prow = $dsparamrs->next ) {
					$response_obj->{dslist}->[$j]->{'param'}->{ $prow->parameter->config_file }->{ $prow->parameter->name } = $prow->parameter->value;
				}
			}

			$j++;
		}

		return $response_obj;

	}
}


#generates the 12m_facts file
sub facts {
	my $self       = shift;
	my $profile_obj = shift;
	my $filename   = shift;
	my $text       = $self->header_comment( $profile_obj->name );
	$text .= "profile:" . $profile_obj->name . "\n";

	return $text;
}


#generates a generic config file based on a server and parameters which match the supplied filename.
sub take_and_bake_server {
	my $self       = shift;
	my $server_obj = shift;
	my $filename   = shift;

	my $data = $self->param_data( $server_obj, $filename );
	my $header = $self->header_comment( $server_obj->host_name );
	my $text;
	foreach my $parameter ( sort keys %{$data} ) {
		if ( $parameter eq "header" && $data->{$parameter} eq "none" ) {
			$header = "";
		}
		elsif ( $parameter eq "header" ) {
			$header = $data->{$parameter} . "\n";
		}
		else {
			$text .= $data->{$parameter} . "\n";
		}
	}
	$text =~ s/\s*__RETURN__\s*/\n/g;
	my $response = $header . $text;
	return $response;
}

#generates a generic config file based on a profile and parameters which match the supplied filename.
sub take_and_bake_profile {
	my $self        = shift;
	my $profile_obj = shift;
	my $filename    = shift;

	my $data = $self->profile_param_data( $profile_obj->id, $filename );
	my $header = $self->header_comment( $profile_obj->name );
	my $text;
	foreach my $parameter ( sort keys %{$data} ) {
		if ( $parameter eq "header" && $data->{$parameter} eq "none" ) {
			$header = "";
		}
		elsif ( $parameter eq "header" ) {
			$header = $data->{$parameter} . "\n";
		}
		else {
			$text .= $data->{$parameter} . "\n";
		}
	}
	$text =~ s/\s*__RETURN__\s*/\n/g;
	my $response = $header . $text;
	return $response;
}

#generates a generic config file based on a profile and parameters which match the supplied filename.
#differs from take and bake in that it uses predefined separators.
sub generic_profile_config {
	my $self        = shift;
	my $profile_obj = shift;
	my $filename    = shift;

	my $sep = defined( $separator->{$filename} ) ? $separator->{$filename} : " = ";

	my $data = $self->profile_param_data( $profile_obj->id, $filename );
	my $text = $self->header_comment( $profile_obj->name );
	foreach my $parameter ( sort keys %{$data} ) {
		my $p_name = $parameter;
		$p_name =~ s/__\d+$//;
		$text .= $p_name . $sep . $data->{$parameter} . "\n";
	}

	return $text;
}

#generates a generic config file based on a server and parameters which match the supplied filename.
#differs from take and bake in that it uses predefined separators.
sub generic_server_config {
	my $self       = shift;
	my $server_obj = shift;
	my $filename   = shift;

	my $sep = defined( $separator->{$filename} ) ? $separator->{$filename} : " = ";

	my $data = $self->param_data( $server_obj, $filename );
	my $text = $self->header_comment( $server_obj->host_name );
	foreach my $parameter ( sort keys %{$data} ) {
		my $p_name = $parameter;
		$p_name =~ s/__\d+$//;
		$text .= $p_name . $sep . $data->{$parameter} . "\n";
	}

	return $text;
}

sub ats_dot_rules {
	my $self        = shift;
	my $profile_obj = shift;
	my $filename    = shift;

	my $text = $self->header_comment( $profile_obj->name );
	my $data = $self->profile_param_data( $profile_obj->id, "storage.config" );    # ats.rules is based on the storage.config params

	my $drive_prefix = $data->{Drive_Prefix};
	my @drive_postfix = split( /,/, $data->{Drive_Letters} );
	foreach my $l ( sort @drive_postfix ) {
		$drive_prefix =~ s/\/dev\///;
		$text .= "KERNEL==\"" . $drive_prefix . $l . "\", OWNER=\"ats\"\n";
	}
	if ( defined( $data->{RAM_Drive_Prefix} ) ) {
		$drive_prefix = $data->{RAM_Drive_Prefix};
		@drive_postfix = split( /,/, $data->{RAM_Drive_Letters} );
		foreach my $l ( sort @drive_postfix ) {
			$drive_prefix =~ s/\/dev\///;
			$text .= "KERNEL==\"" . $drive_prefix . $l . "\", OWNER=\"ats\"\n";
		}
	}

	return $text;
}

sub drop_qstring_dot_config {
	my $self        = shift;
	my $profile_obj = shift;
	my $filename    = shift;

	my $text = $self->header_comment( $profile_obj->name );

	my $drop_qstring = $self->profile_param_value( $profile_obj->id, 'drop_qstring.config', 'content', undef );

	if ($drop_qstring) {
		$text .= $drop_qstring . "\n";
	}
	else {
		$text .= "/([^?]+) \$s://\$t/\$1\n";
	}

	return $text;
}

sub logging_dot_config {
	my $self        = shift;
	my $profile_obj = shift;
	my $filename    = shift;

	my $data = $self->profile_param_data( $profile_obj->id, "logging.config" );

	# This is an LUA file, so we need to massage the header a bit for LUA commenting.
	my $text = "-- " . $self->header_comment( $profile_obj->name );
	$text =~ s/# //;
	$text =~ s/\n//;
	$text .= " --\n";

	my $max_log_objects = 10;
	for ( my $i = 0; $i < $max_log_objects; $i = $i + 1 ) {
		my $log_format_field = "LogFormat";
		my $log_object_field = "LogObject";
		if ( $i > 0 ) {
			$log_format_field = $log_format_field . "$i";
			$log_object_field = $log_object_field . "$i";
		}

		my $log_format_name = $data->{$log_format_field . ".Name"} || "";
		if ( length($log_format_name) > 0 ) {
			my $format = $data->{$log_format_field . ".Format"};
			$format =~ s/"/\\\"/g;
			$text .= $log_format_name . " = format {\n";
			$text .= "	Format = '" . $format . " '\n";
			$text .= "}\n";
		}

		my $log_object_filename = $data->{$log_object_field . ".Filename"} || "";
		if ( length($log_object_filename) > 0 ) {
			my $log_object_format               = $data->{$log_object_field . ".Format"}             || "";
			my $log_object_rolling_enabled      = $data->{$log_object_field . ".RollingEnabled"}     || "";
			my $log_object_rolling_interval_sec = $data->{$log_object_field . ".RollingIntervalSec"} || "";
			my $log_object_rolling_offset_hr    = $data->{$log_object_field . ".RollingOffsetHr"}    || "";
			my $log_object_rolling_size_mb      = $data->{$log_object_field . ".RollingSizeMb"}      || "";
			my $log_object_header               = $data->{$log_object_field . ".Header"}             || "";

			$text .= "\nlog.ascii {\n";
			$text .= "  Format = " . $log_format_name . ",\n";
			$text .= "  Filename = '" . $log_object_filename . "',\n";
			$text .= "  RollingEnabled = " . $log_object_rolling_enabled . ",\n" unless defined();
			$text .= "  RollingIntervalSec = " . $log_object_rolling_interval_sec . ",\n";
			$text .= "  RollingOffsetHr = " . $log_object_rolling_offset_hr . ",\n";
			$text .= "  RollingSizeMb = " . $log_object_rolling_size_mb . "\n";
			$text .= "}\n";
		}
	}

	return $text;
}

sub logs_xml_dot_config {
	my $self        = shift;
	my $profile_obj = shift;
	my $filename    = shift;

	my $data = $self->profile_param_data( $profile_obj->id, "logs_xml.config" );

	# This is an XML file, so we need to massage the header a bit for XML commenting.
	my $text = "<!-- " . $self->header_comment( $profile_obj->name );
	$text =~ s/# //;
	$text =~ s/\n//;
	$text .= " -->\n";

	my $max_log_objects = 10;
	for ( my $i = 0; $i < $max_log_objects; $i = $i + 1 ) {
		my $log_format_field = "LogFormat";
		my $log_object_field = "LogObject";
		if ( $i > 0 ) {
			$log_format_field = $log_format_field . "$i";
			$log_object_field = $log_object_field . "$i";
		}

		my $log_format_name = $data->{$log_format_field . ".Name"} || "";
		if ( length($log_format_name) > 0 ) {
			my $format = $data->{$log_format_field . ".Format"};
			$format =~ s/"/\\\"/g;
			$text .= "<LogFormat>\n";
			$text .= "  <Name = \"" . $log_format_name . "\"/>\n";
			$text .= "  <Format = \"" . $format . "\"/>\n";
			$text .= "</LogFormat>\n";
		}

		my $log_object_filename = $data->{$log_object_field . ".Filename"} || "";
		if ( length($log_object_filename) > 0 ) {
			my $log_object_format               = $data->{$log_object_field . ".Format"}             || "";
			my $log_object_rolling_enabled      = $data->{$log_object_field . ".RollingEnabled"}     || "";
			my $log_object_rolling_interval_sec = $data->{$log_object_field . ".RollingIntervalSec"} || "";
			my $log_object_rolling_offset_hr    = $data->{$log_object_field . ".RollingOffsetHr"}    || "";
			my $log_object_rolling_size_mb      = $data->{$log_object_field . ".RollingSizeMb"}      || "";
			my $log_object_header               = $data->{$log_object_field . ".Header"}             || "";

			$text .= "<LogObject>\n";
			$text .= "  <Format = \"" . $log_object_format . "\"/>\n";
			$text .= "  <Filename = \"" . $log_object_filename . "\"/>\n";
			$text .= "  <RollingEnabled = " . $log_object_rolling_enabled . "/>\n" unless defined();
			$text .= "  <RollingIntervalSec = " . $log_object_rolling_interval_sec . "/>\n";
			$text .= "  <RollingOffsetHr = " . $log_object_rolling_offset_hr . "/>\n";
			$text .= "  <RollingSizeMb = " . $log_object_rolling_size_mb . "/>\n";
			$text .= "  <Header = \"" . $log_object_header . "\"/>\n" if ( length($log_object_header) > 0 );
			$text .= "</LogObject>\n";
		}
	}

	return $text;
}

sub storage_dot_config_volume_text {
	my $prefix  = shift;
	my $letters = shift;
	my $volume  = shift;

	my $text = "";
	my @postfix = split( /,/, $letters );
	foreach my $l ( sort @postfix ) {
		$text .= $prefix . $l;
		$text .= " volume=" . $volume;
		$text .= "\n";
	}
	return $text;
}

sub storage_dot_config {
	my $self        = shift;
	my $profile_obj = shift;
	my $filename    = shift;

	my $text = $self->header_comment( $profile_obj->name );
	my $data = $self->profile_param_data( $profile_obj->id, "storage.config" );

	my $next_volume = 1;
	if ( defined( $data->{Drive_Prefix} ) ) {
		$text .= storage_dot_config_volume_text( $data->{Drive_Prefix}, $data->{Drive_Letters}, $next_volume );
		$next_volume++;
	}

	if ( defined( $data->{RAM_Drive_Prefix} ) ) {
		$text .= storage_dot_config_volume_text( $data->{RAM_Drive_Prefix}, $data->{RAM_Drive_Letters}, $next_volume );
		$next_volume++;
	}

	if ( defined( $data->{SSD_Drive_Prefix} ) ) {
		$text .= storage_dot_config_volume_text( $data->{SSD_Drive_Prefix}, $data->{SSD_Drive_Letters}, $next_volume );
		$next_volume++;
	}

	return $text;
}

sub to_ext_dot_config {
	my $self       = shift;
	my $server_obj = shift;
	my $filename   = shift;

	my $text = $self->header_comment( $server_obj->host_name );

	# get the subroutine name for this file from the parameter
	my $subroutine = $self->profile_param_value( $server_obj->profile->id, $filename, 'SubRoutine', undef );
	$self->app->log->error( "ToExtDotConfigFile == " . $subroutine );

	if ( defined $subroutine ) {
		my $package;
		( $package = $subroutine ) =~ s/(.*)(::)(.*)/$1/;
		eval "use $package;";

		# And call it - the below calls the subroutine in the var $subroutine.
		no strict 'refs';
		$text .= $subroutine->( $self, $server_obj->host_name, $filename );
		return $text;
	}
	else {
		return;
	}
}

sub get_num_volumes {
	my $data = shift;

	my $num            = 0;
	my @drive_prefixes = qw( Drive_Prefix SSD_Drive_Prefix RAM_Drive_Prefix);
	foreach my $pre (@drive_prefixes) {
		if ( exists $data->{$pre} ) {
			$num++;
		}
	}
	return $num;
}

sub volume_dot_config_volume_text {
	my $volume      = shift;
	my $num_volumes = shift;
	my $size        = int( 100 / $num_volumes );
	return "volume=$volume scheme=http size=$size%\n";
}

sub volume_dot_config {
	my $self        = shift;
	my $profile_obj = shift;

	my $data = $self->profile_param_data( $profile_obj->id, "storage.config" );
	my $text = $self->header_comment( $profile_obj->name );

	my $num_volumes = get_num_volumes($data);

	my $next_volume = 1;
	$text .= "# TRAFFIC OPS NOTE: This is running with forced volumes - the size is irrelevant\n";
	if ( defined( $data->{Drive_Prefix} ) ) {
		$text .= volume_dot_config_volume_text( $next_volume, $num_volumes );
		$next_volume++;
	}
	if ( defined( $data->{RAM_Drive_Prefix} ) ) {
		$text .= volume_dot_config_volume_text( $next_volume, $num_volumes );
		$next_volume++;
	}
	if ( defined( $data->{SSD_Drive_Prefix} ) ) {
		$text .= volume_dot_config_volume_text( $next_volume, $num_volumes );
		$next_volume++;
	}

	return $text;
}

# This is a temporary workaround until we have real partial object caching support in ATS, so hardcoding for now
sub bg_fetch_dot_config {
	my $self     = shift;
	my $cdn_obj  = shift;
	my $filename = shift;

	my $text = $self->header_comment( $cdn_obj->name );
	$text .= "include User-Agent *\n";

	return $text;
}

sub header_rewrite_dot_config {
	my $self     = shift;
	my $cdn_obj  = shift;
	my $filename = shift;

	my $text      = $self->header_comment( $cdn_obj->name );
	my $ds_xml_id = undef;
	if ( $filename =~ /^hdr_rw_mid_(.*)\.config$/ ) {
		$ds_xml_id = $1;
		my $ds = $self->db->resultset('Deliveryservice')->search( { xml_id => $ds_xml_id } )->single();
		my $actions = $ds->mid_header_rewrite;
		$text .= $actions . "\n";
	}
	elsif ( $filename =~ /^hdr_rw_(.*)\.config$/ ) {
		$ds_xml_id = $1;
		my $ds = $self->db->resultset('Deliveryservice')->search( { xml_id => $ds_xml_id } )->single();
		my $actions = $ds->edge_header_rewrite;
		$text .= $actions . "\n";
	}

	$text =~ s/\s*__RETURN__\s*/\n/g;

	#do the following text replacement at the ORT level for cacheability
	# my $ipv4 = $server_obj->ip_address;
	# $text =~ s/__CACHE_IPV4__/$ipv4/g;

	return $text;
}

sub cacheurl_dot_config {
	my $self     = shift;
	my $cdn_obj  = shift;
	my $filename = shift;

	my $text = $self->header_comment( $cdn_obj->name );

	if ( $filename eq "cacheurl_qstring.config" ) {    # This is the per remap drop qstring w cacheurl use case, the file is the same for all remaps
		$text .= "http://([^?]+)(?:\\?|\$)  http://\$1\n";
		$text .= "https://([^?]+)(?:\\?|\$)  https://\$1\n";
	}
	elsif ( $filename =~ /cacheurl_(.*).config/ )
	{    # Yes, it's possibe to have the same plugin invoked multiple times on the same remap line, this is from the remap entry
		my $ds_xml_id = $1;
		my $ds = $self->db->resultset('Deliveryservice')->search( { xml_id => $ds_xml_id }, { prefetch => [ 'type', 'profile' ] } )->first();
		if ($ds) {
			$text .= $ds->cacheurl . "\n";
		}
	}
	elsif ( $filename eq "cacheurl.config" ) {    # this is the global drop qstring w cacheurl use case
		my $data = $self->cdn_ds_data( $cdn_obj );
		my $last_org = '';
		foreach my $remap ( @{ $data->{dslist} } ) {
			if ( $remap->{qstring_ignore} == 1 ) {
				my $org = $remap->{org};
				if ( $last_org ne $org ){
					$org =~ /(https?:\/\/)(.*)/;
					$text .= "$1(" . $2 . "/[^?]+)(?:\\?|\$)  $1\$1\n";
				}
				$last_org = $org;
			}
		}

	}

	$text =~ s/\s*__RETURN__\s*/\n/g;

	return $text;
}

sub regex_remap_dot_config {
	my $self     = shift;
	my $cdn_obj  = shift;
	my $filename = shift;

	my $text = $self->header_comment( $cdn_obj->name );

	if ( $filename =~ /^regex_remap_(.*)\.config$/ ) {
		my $ds_xml_id = $1;
		my $ds = $self->db->resultset('Deliveryservice')->search( { xml_id => $ds_xml_id } )->single();
		$text .= $ds->regex_remap . "\n";
	}

	$text =~ s/\s*__RETURN__\s*/\n/g;

	return $text;
}

sub regex_revalidate_dot_config {
	my $self     = shift;
	my $cdn_obj  = shift;
	my $filename = shift;

	my $text = "# DO NOT EDIT - Generated for CDN " . $cdn_obj->name . " by " . &name_version_string($self) . " on " . `date`;

	my $max_days =
		$self->db->resultset('Parameter')->search( { name => "maxRevalDurationDays" }, { config_file => "regex_revalidate.config" } )->get_column('value')
		->first;
	my $interval = "> now() - interval '$max_days day'";

	my %regex_time;
	$max_days =
		$self->db->resultset('Parameter')->search( { name => "maxRevalDurationDays" }, { config_file => "regex_revalidate.config" } )->get_column('value')
		->first;
	my $max_hours = $max_days * 24;
	my $min_hours = 1;

	my $rs = $self->db->resultset('Job')->search( { start_time => \$interval }, { prefetch => 'job_deliveryservice' } );
	while ( my $row = $rs->next ) {
		next unless defined( $row->job_deliveryservice );

		# Purges are CDN - wide, and the job entry has the ds id in it.
		my $parameters = $row->parameters;
		my $ttl;
		if ( $row->keyword eq "PURGE" && ( defined($parameters) && $parameters =~ /TTL:(\d+)h/ ) ) {
			$ttl = $1;
			if ( $ttl < $min_hours ) {
				$ttl = $min_hours;
			}
			elsif ( $ttl > $max_hours ) {
				$ttl = $max_hours;
			}
		}
		else {
			next;
		}

		my $date       = new Date::Manip::Date();
		my $start_time = $row->start_time;
		my $start_date = ParseDate($start_time);
		my $end_date   = DateCalc( $start_date, ParseDateDelta( $ttl . ':00:00' ) );
		my $err        = $date->parse($end_date);
		if ($err) {
			print "ERROR ON DATE CONVERSION:" . $err;
			next;
		}
		my $purge_end = $date->printf("%s");    # this is in secs since the unix epoch

		if ( $purge_end < time() ) {            # skip purges that have an end_time in the past
			next;
		}
		my $asset_url = $row->asset_url;

		my $job_cdn_id = $row->job_deliveryservice->cdn_id;
		if ( $cdn_obj->id == $job_cdn_id ) {

			# if there are multipe with same re, pick the longes lasting.
			if ( !defined( $regex_time{ $row->asset_url } )
				|| ( defined( $regex_time{ $row->asset_url } ) && $purge_end > $regex_time{ $row->asset_url } ) )
			{
				$regex_time{ $row->asset_url } = $purge_end;
			}
		}
	}

	foreach my $re ( sort keys %regex_time ) {
		$text .= $re . " " . $regex_time{$re} . "\n";
	}

	return $text;
}

sub set_dscp_dot_config {
	my $self     = shift;
	my $cdn_obj  = shift;
	my $filename = shift;

	my $text = $self->header_comment( $cdn_obj->name );
	my $dscp_decimal;
	if ( $filename =~ /^set_dscp_(\d+)\.config$/ ) {
		$dscp_decimal = $1;
	}
	else {
		$text = "An error occured generating the DSCP header rewrite file.";
	}
	$text .= "cond %{REMAP_PSEUDO_HOOK}\n" . "set-conn-dscp " . $dscp_decimal . " [L]\n";

	return $text;
}

sub ssl_multicert_dot_config {
	my $self     = shift;
	my $cdn_obj  = shift;
	my $filename = shift;

	my $text = $self->header_comment( $cdn_obj->name );

	## We should break this search out into a separate sub later
	my $protocol_search = '> 0';
	my @ds_list = $self->db->resultset('Deliveryservice')->search( { -and => [ cdn_id => $cdn_obj->id, 'me.protocol' => \$protocol_search ] } )->all();
	foreach my $ds (@ds_list) {
		if ( $ds->type->name =~ /STEERING/ ) {
				next;    # Steering delivery service SSLs should not be on the edges.
		}
		my $ds_id        = $ds->id;
		my $xml_id       = $ds->xml_id;
		my $rs_ds        = $self->db->resultset('Deliveryservice')->search( { 'me.id' => $ds_id }, { prefetch => ['type'] } );
		my $data         = $rs_ds->first;
		my $domain_name  = $ds->cdn->domain_name,
		my $ds_regexes   = UI::DeliveryService::get_regexp_set( $self, $ds_id );
		my @example_urls = UI::DeliveryService::get_example_urls( $self, $ds_id, $ds_regexes, $data, $domain_name, $data->protocol );

		#first one is the one we want
		my $hostname = $example_urls[0];
		$hostname =~ /(https?:\/\/)(.*)/;
		my $new_host = $2;
		my $key_name = "$new_host.key";
		$new_host =~ tr/./_/;
		my $cer_name = $new_host . "_cert.cer";

		$text .= "ssl_cert_name=$cer_name\t ssl_key_name=$key_name\n";
	}

	return $text;
}

sub url_sig_dot_config {
	my $self        = shift;
	my $profile_obj = shift;
	my $filename    = shift;

	my $sep = defined( $separator->{$filename} ) ? $separator->{$filename} : " = ";
	my $data = $self->profile_param_data( $profile_obj->id, $filename );
	my $text = $self->header_comment( $profile_obj->name );

	my $response_container = $self->riak_get( URL_SIG_KEYS_BUCKET, $filename );
	my $response = $response_container->{response};
	if ( $response->is_success() ) {
		my $response_json = decode_json( $response->content );
		my $keys          = $response_json;
		foreach my $parameter ( sort keys %{$data} ) {
			if ( !defined($keys) || $parameter !~ /^key\d+/ ) {    # only use key parameters as a fallback (temp, remove me later)
				$text .= $parameter . $sep . $data->{$parameter} . "\n";
			}
		}

		foreach my $parameter ( sort keys %{$keys} ) {
			$text .= $parameter . $sep . $keys->{$parameter} . "\n";
		}

		return $text;
	}
	else {
		return;
	}
}

sub uri_signing_dot_config {
	my $self        = shift;
	my $filename    = shift;

	my $bucket = "cdn_uri_sig_keys";

	my ($key) = $filename =~ /uri_signing_(.*)\.config/;

	my $response_container = $self->riak_get( $bucket, $key );
	my $response = $response_container->{response};

	if ( $response->is_success() ) {
		return $response->content;
	}
	else {
		return;
	}
}

sub server_cache_dot_config {
	my $self       = shift;
	my $server_obj = shift;
	my $filename   = shift;


	my $text = $self->header_comment( $server_obj->host_name );
	my $data = $self->ds_data($server_obj);

	foreach my $ds ( @{ $data->{dslist} } ) {
		if ( $ds->{type} eq "HTTP_NO_CACHE" ) {
			my $org_fqdn = $ds->{org};
			$org_fqdn =~ s/https?:\/\///;
			$org_fqdn =~ m/(.*?):(\d+).*/;
			my $org_port = $2;

			if (defined($org_port)) {
				$org_fqdn = $1;
				$text .= "dest_domain=" . $org_fqdn . " port=" . $org_port . " scheme=http action=never-cache\n";
			} else {
				$text .= "dest_domain=" . $org_fqdn . " scheme=http action=never-cache\n";
			}
		}
	}

	#remove duplicate lines, since we're looking up by profile
	my @lines = split(/\n/,$text);

	my %seen;
	@lines = grep { !$seen{$_}++ } @lines;

	$text = join("\n",@lines);

	return $text;
}

sub profile_cache_dot_config {
	my $self       = shift;
	my $profile_obj = shift;
	my $filename   = shift;


	my $text = $self->header_comment( $profile_obj->name );
	my $data = $self->profile_ds_data($profile_obj);

	foreach my $ds ( @{ $data->{dslist} } ) {
		if ( $ds->{type} eq "HTTP_NO_CACHE" ) {
			my $org_fqdn = $ds->{org};
			$org_fqdn =~ s/https?:\/\///;
			$org_fqdn =~ m/(.*?):(\d+).*/;
			my $org_port = $2;
			
			if (defined($org_port)) {
				$org_fqdn = $1;
				$text .= "dest_domain=" . $org_fqdn . " port=" . $org_port . " scheme=http action=never-cache\n";
			} else {
				$text .= "dest_domain=" . $org_fqdn . " scheme=http action=never-cache\n";
			}
		}
	}

	#remove duplicate lines, since we're looking up by profile
	my @lines = split(/\n/,$text);

	my %seen;
	@lines = grep { !$seen{$_}++ } @lines;

	$text = join("\n",@lines);

	return $text;
}

sub hosting_dot_config {
	my $self       = shift;
	my $server_obj = shift;

	my $storage_data = $self->param_data( $server_obj, "storage.config" );
	my $text = $self->header_comment( $server_obj->host_name );

	my $data = $self->ds_data($server_obj);

	if ( defined( $storage_data->{RAM_Drive_Prefix} ) ) {
		my $next_volume = 1;
		if ( defined( $storage_data->{Drive_Prefix} ) ) {
			my $disk_volume = $next_volume;
			$text .= "# TRAFFIC OPS NOTE: volume " . $disk_volume . " is the Disk volume\n";
			$next_volume++;
		}
		my $ram_volume = $next_volume;
		$text .= "# TRAFFIC OPS NOTE: volume " . $ram_volume . " is the RAM volume\n";

		my %listed = ();
		foreach my $ds ( @{ $data->{dslist} } ) {
			if (   ( ( $ds->{type} =~ /_LIVE$/ || $ds->{type} =~ /_LIVE_NATNL$/ ) && $server_obj->type->name =~ m/^EDGE/ )
				|| ( $ds->{type} =~ /_LIVE_NATNL$/ && $server_obj->type->name =~ m/^MID/ ) )
			{
				if ( defined( $listed{ $ds->{org} } ) ) { next; }
				my $org_fqdn = $ds->{org};
				$org_fqdn =~ s/https?:\/\///;
				$text .= "hostname=" . $org_fqdn . " volume=" . $ram_volume . "\n";
				$listed{ $ds->{org} } = 1;
			}
		}
	}
	my $disk_volume = 1;    # note this will actually be the RAM (RAM_Drive_Prefix) volume if there is no Drive_Prefix parameter.
	$text .= "hostname=*   volume=" . $disk_volume . "\n";

	return $text;
}

sub ip_allow_data {
	my $self       = shift;
	my $server_obj = shift;

	my $ipallow;
	$ipallow = ();

	my $i = 0;

	# localhost is trusted.
	$ipallow->[$i]->{src_ip} = '127.0.0.1';
	$ipallow->[$i]->{action} = 'ip_allow';
	$ipallow->[$i]->{method} = "ALL";
	$i++;
	$ipallow->[$i]->{src_ip} = '::1';
	$ipallow->[$i]->{action} = 'ip_allow';
	$ipallow->[$i]->{method} = "ALL";
	$i++;

	# default for coalesce_ipv4 = 24, 5 and for ipv6 48, 5; override with the parameters in the server profile.
	my $coalesce_masklen_v4 = 24;
	my $coalesce_number_v4  = 5;
	my $coalesce_masklen_v6 = 48;
	my $coalesce_number_v6  = 5;
	my $rs_parameter =
		$self->db->resultset('ProfileParameter')->search( { profile => $server_obj->profile->id }, { prefetch => [ "parameter", "profile" ] } );

	while ( my $row = $rs_parameter->next ) {
		if ( $row->parameter->name eq 'purge_allow_ip' && $row->parameter->config_file eq 'ip_allow.config' ) {
			$ipallow->[$i]->{src_ip} = $row->parameter->value;
			$ipallow->[$i]->{action} = "ip_allow";
			$ipallow->[$i]->{method} = "ALL";
			$i++;
		}
		elsif ( $row->parameter->name eq 'coalesce_masklen_v4' && $row->parameter->config_file eq 'ip_allow.config' ) {
			$coalesce_masklen_v4 = $row->parameter->value;
		}
		elsif ( $row->parameter->name eq 'coalesce_number_v4' && $row->parameter->config_file eq 'ip_allow.config' ) {
			$coalesce_number_v4 = $row->parameter->value;
		}
		elsif ( $row->parameter->name eq 'coalesce_masklen_v6' && $row->parameter->config_file eq 'ip_allow.config' ) {
			$coalesce_masklen_v6 = $row->parameter->value;
		}
		elsif ( $row->parameter->name eq 'coalesce_number_v6' && $row->parameter->config_file eq 'ip_allow.config' ) {
			$coalesce_number_v6 = $row->parameter->value;
		}
	}


	if ( $server_obj->type->name =~ m/^MID/ ) {
		my @edge_locs = $self->db->resultset('Cachegroup')->search( { -or => [ parent_cachegroup_id => $server_obj->cachegroup->id, secondary_parent_cachegroup_id => $server_obj->cachegroup->id ] } )->get_column('id')->all();
		my %allow_locs;
		foreach my $loc (@edge_locs) {
			$allow_locs{$loc} = 1;
		}

		# get all the EDGE and RASCAL nets
		my @allowed_netaddrips;
		my @allowed_ipv6_netaddrips;
		my @types;
		push( @types, &type_ids( $self, 'EDGE%', 'server' ) );
		my $rtype = &type_id( $self, 'RASCAL' );
		push( @types, $rtype );
		my $rs_allowed = $self->db->resultset('Server')->search( { 'me.type' => { -in => \@types } }, { prefetch => [ 'type', 'cachegroup' ] } );

		while ( my $allow_row = $rs_allowed->next ) {
			if ( $allow_row->type->id == $rtype
				|| ( defined( $allow_locs{ $allow_row->cachegroup->id } ) && $allow_locs{ $allow_row->cachegroup->id } == 1 ) )
			{
				my $ipv4 = NetAddr::IP->new( $allow_row->ip_address, $allow_row->ip_netmask );

				if ( defined($ipv4) ) {
					push( @allowed_netaddrips, $ipv4 );
				}
				else {
					$self->app->log->error(
						$allow_row->host_name . " has an invalid IPv4 address; excluding from ip_allow data for " . $server_obj->host_name );
				}

				if ( defined $allow_row->ip6_address ) {
					my $ipv6 = NetAddr::IP->new( $allow_row->ip6_address );

					if ( defined($ipv6) ) {
						push( @allowed_ipv6_netaddrips, NetAddr::IP->new( $allow_row->ip6_address ) );
					}
					else {
						$self->app->log->error(
							$allow_row->host_name . " has an invalid IPv6 address; excluding from ip_allow data for " . $server_obj->host_name );
					}
				}
			}
		}

		# compact, coalesce and compact combined list again
		my @compacted_list = NetAddr::IP::Compact(@allowed_netaddrips);
		my $coalesced_list = NetAddr::IP::Coalesce( $coalesce_masklen_v4, $coalesce_number_v4, @allowed_netaddrips );
		my @combined_list  = NetAddr::IP::Compact( @allowed_netaddrips, @{$coalesced_list} );
		foreach my $net (@combined_list) {
			my $range = $net->range();
			$range =~ s/\s+//g;
			$ipallow->[$i]->{src_ip} = $range;
			$ipallow->[$i]->{action} = "ip_allow";
			$ipallow->[$i]->{method} = "ALL";
			$i++;
		}

		# now add IPv6. TODO JvD: paremeterize support enabled on/ofd and /48 and number 5
		my @compacted__ipv6_list = NetAddr::IP::Compact(@allowed_ipv6_netaddrips);
		my $coalesced_ipv6_list  = NetAddr::IP::Coalesce( $coalesce_masklen_v6, $coalesce_number_v6, @allowed_ipv6_netaddrips );
		my @combined_ipv6_list   = NetAddr::IP::Compact( @allowed_ipv6_netaddrips, @{$coalesced_ipv6_list} );
		foreach my $net (@combined_ipv6_list) {
			my $range = $net->range();
			$range =~ s/\s+//g;
			$ipallow->[$i]->{src_ip} = $range;
			$ipallow->[$i]->{action} = "ip_allow";
			$ipallow->[$i]->{method} = "ALL";
			$i++;
		}

		# allow RFC 1918 server space - TODO JvD: parameterize
		$ipallow->[$i]->{src_ip} = '10.0.0.0-10.255.255.255';
		$ipallow->[$i]->{action} = 'ip_allow';
		$ipallow->[$i]->{method} = "ALL";
		$i++;

		$ipallow->[$i]->{src_ip} = '172.16.0.0-172.31.255.255';
		$ipallow->[$i]->{action} = 'ip_allow';
		$ipallow->[$i]->{method} = "ALL";
		$i++;

		$ipallow->[$i]->{src_ip} = '192.168.0.0-192.168.255.255';
		$ipallow->[$i]->{action} = 'ip_allow';
		$ipallow->[$i]->{method} = "ALL";
		$i++;

		# end with a deny
		$ipallow->[$i]->{src_ip} = '0.0.0.0-255.255.255.255';
		$ipallow->[$i]->{action} = 'ip_deny';
		$ipallow->[$i]->{method} = "ALL";
		$i++;
		$ipallow->[$i]->{src_ip} = '::-ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff';
		$ipallow->[$i]->{action} = 'ip_deny';
		$ipallow->[$i]->{method} = "ALL";
		$i++;
	}
	else {

		# for edges deny "PUSH|PURGE|DELETE", allow everything else to everyone.
		$ipallow->[$i]->{src_ip} = '0.0.0.0-255.255.255.255';
		$ipallow->[$i]->{action} = 'ip_deny';
		$ipallow->[$i]->{method} = "PUSH|PURGE|DELETE";
		$i++;
		$ipallow->[$i]->{src_ip} = '::-ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff';
		$ipallow->[$i]->{action} = 'ip_deny';
		$ipallow->[$i]->{method} = "PUSH|PURGE|DELETE";
		$i++;
	}

	return $ipallow;
}

sub ip_allow_dot_config {
	my $self       = shift;
	my $server_obj = shift;
	my $filename   = shift;

	my $text = $self->header_comment( $server_obj->host_name );
	my $data = $self->ip_allow_data( $server_obj, $filename );

	foreach my $access ( @{$data} ) {
		$text .= sprintf( "src_ip=%-70s action=%-10s method=%-20s\n", $access->{src_ip}, $access->{action}, $access->{method} );
	}

	return $text;
}

sub by_parent_rank {
	my ($arank) = $a->{"rank"};
	my ($brank) = $b->{"rank"};
	( $arank || 1 ) <=> ( $brank || 1 );
}

sub cachegroup_profiles {
	my $self             = shift;
	my $ids              = shift;
	my $profile_cache    = shift;
	my $deliveryservices = shift;

	if ( !@$ids ) {
		return;    # nothing to see here..
	}
	my $online   = &admin_status_id( $self, "ONLINE" );
	my $reported = &admin_status_id( $self, "REPORTED" );

	my %condition = (
		status     => { -in => [ $online, $reported ] },
		cachegroup => { -in => $ids }
	);

	my $rs_parent = $self->db->resultset('Server')->search( \%condition, { prefetch => [ 'cachegroup', 'status', 'type', 'profile', 'cdn' ] } );

	while ( my $row = $rs_parent->next ) {

		next unless ( $row->type->name eq 'ORG' || $row->type->name =~ m/^EDGE/ || $row->type->name =~ m/^MID/ );
		if ( $row->type->name eq 'ORG' ) {
			my $rs_ds = $self->db->resultset('DeliveryserviceServer')->search( { server => $row->id }, { prefetch => ['deliveryservice'] } );
			while ( my $ds_row = $rs_ds->next ) {
				my $org_uri = URI->new( UI::DeliveryService::compute_org_server_fqdn($self, $ds_row->deliveryservice->id) );
				push( @{ $deliveryservices->{ $org_uri->host } }, $row );
			}
		}
		else {
			push( @{ $deliveryservices->{all_parents} }, $row );
		}

		# get the profile info, and cache it in %profile_cache
		my $pid = $row->profile->id;
		if ( !defined( $profile_cache->{$pid} ) ) {

			# assign $ds_domain, $weight and $port, and cache the results %profile_cache
			$profile_cache->{$pid} = {
				domain_name    => $row->cdn->domain_name,
				weight         => $self->profile_param_value( $pid, 'parent.config', 'weight', '0.999' ),
				port           => $self->profile_param_value( $pid, 'parent.config', 'port', undef ),
				use_ip_address => $self->profile_param_value( $pid, 'parent.config', 'use_ip_address', 0 ),
				rank           => $self->profile_param_value( $pid, 'parent.config', 'rank', 1 ),
				not_a_parent   => $self->profile_param_value( $pid, 'parent.config', 'not_a_parent', 'false' ),
			};
		}
	}
}

sub parent_data {
	my $self       = shift;
	my $server_obj = shift;

	my @parent_cachegroup_ids;
	my @secondary_parent_cachegroup_ids;
	my $org_cachegroup_type_id = &type_id( $self, "ORG_LOC" );
	if ( $server_obj->type->name =~ m/^MID/ ) {

		# multisite origins take all the org groups in to account
		@parent_cachegroup_ids = $self->db->resultset('Cachegroup')->search( { type => $org_cachegroup_type_id } )->get_column('id')->all();
	}
	else {
		@parent_cachegroup_ids =
			grep {defined} $self->db->resultset('Cachegroup')->search( { id => $server_obj->cachegroup->id } )->get_column('parent_cachegroup_id')->all();
		@secondary_parent_cachegroup_ids =
			grep {defined}
			$self->db->resultset('Cachegroup')->search( { id => $server_obj->cachegroup->id } )->get_column('secondary_parent_cachegroup_id')->all();
	}

	# get the server's cdn domain
	my $server_domain = $self->get_cdn_domain_by_profile_id( $server_obj->profile->id );

	my %profile_cache;
	my %deliveryservices;
	my %parent_info;

	$self->cachegroup_profiles( \@parent_cachegroup_ids,           \%profile_cache, \%deliveryservices );
	$self->cachegroup_profiles( \@secondary_parent_cachegroup_ids, \%profile_cache, \%deliveryservices );
	foreach my $prefix ( keys %deliveryservices ) {
		foreach my $row ( @{ $deliveryservices{$prefix} } ) {
			my $pid              = $row->profile->id;
			if ( $profile_cache{$pid}->{not_a_parent} ne 'false' ) {
				next;
			}
			my $ds_domain        = $profile_cache{$pid}->{domain_name};
			my $weight           = $profile_cache{$pid}->{weight};
			my $port             = $profile_cache{$pid}->{port};
			my $use_ip_address   = $profile_cache{$pid}->{use_ip_address};
			my $rank             = $profile_cache{$pid}->{rank};
			my $primary_parent   = $server_obj->cachegroup->parent_cachegroup_id // -1;
			my $secondary_parent = $server_obj->cachegroup->secondary_parent_cachegroup_id // -1;
			if ( defined($ds_domain) && defined($server_domain) && $ds_domain eq $server_domain ) {
				my %p = (
					host_name        => $row->host_name,
					port             => defined($port) ? $port : $row->tcp_port,
					domain_name      => $row->domain_name,
					weight           => $weight,
					use_ip_address   => $use_ip_address,
					rank             => $rank,
					ip_address       => $row->ip_address,
					primary_parent   => ( $primary_parent == $row->cachegroup->id ) ? 1 : 0,
					secondary_parent => ( $secondary_parent == $row->cachegroup->id ) ? 1 : 0,
				);
				push @{ $parent_info{$prefix} }, \%p;
			}
		}
	}
	return \%parent_info;
}

sub format_parent_info {
	my $parent = shift;
	if ( !defined $parent ) {
		return "";    # should never happen..
	}
	my $host =
		( $parent->{use_ip_address} == 1 )
		? $parent->{ip_address}
		: $parent->{host_name} . '.' . $parent->{domain_name};

	my $port   = $parent->{port};
	my $weight = $parent->{weight};
	my $text   = "$host:$port|$weight;";
	return $text;
}

sub parent_dot_config { #fix qstring - should be ignore for quika
	my $self       = shift;
	my $server_obj = shift;
	my $data;
	my $server_type = $server_obj->type->name;

	my $ats_ver =
		$self->db->resultset('ProfileParameter')
		->search( { 'parameter.name' => 'trafficserver', 'parameter.config_file' => 'package', 'profile.id' => $server_obj->profile->id },
		{ prefetch => [ 'profile', 'parameter' ] } )->get_column('parameter.value')->single();
	my $ats_major_version = substr( $ats_ver, 0, 1 );
	my $parent_info;
	my $header = $self->header_comment( $server_obj->host_name );
	my @text_array;


	if ( $server_type =~ m/^MID/ ) {
		my @unique_origins;
		if ( !defined($data) ) {
			$data = $self->parent_ds_data($server_obj);
		}
		foreach my $ds ( @{ $data->{dslist} } ) {
			my $text;
			my $xml_id                             = $ds->{ds_xml_id};
			my $origin_shield                      = $ds->{origin_shield};
			my $multi_site_origin                  = $ds->{multi_site_origin} || 0;
			my $mso_algorithm                      = $ds->{'param'}->{'parent.config'}->{'mso.algorithm'} || 'consistent_hash';
			my $parent_retry                       = $ds->{'param'}->{'parent.config'}->{'mso.parent_retry'} || 'both';
			my $unavailable_server_retry_responses = $ds->{'param'}->{'parent.config'}->{'mso.unavailable_server_retry_responses'} || '';
			my $max_simple_retries                 = $ds->{'param'}->{'parent.config'}->{'mso.max_simple_retries'} || 1;
			my $max_unavailable_server_retries     = $ds->{'param'}->{'parent.config'}->{'mso.max_unavailable_server_retries'} || 1;

			my $qsh            = $ds->{'param'}->{'parent.config'}->{'psel.qstring_handling'};
			my $parent_qstring = "ignore";                                                      # default is ignore, unless for alg consistent_hash
			if ( !defined($qsh) && $mso_algorithm eq 'consistent_hash' && $ds->{qstring_ignore} == 0 ) {
				$parent_qstring = 'consider';
			}

			my $org_uri = URI->new( $ds->{org} );

			# Don't duplicate origin line if multiple seen
			next if ( grep( /^$org_uri$/, @unique_origins ) );
			push @unique_origins, $org_uri;

			if ( defined($origin_shield) ) {
				my $parent_select_alg = $self->profile_param_value( $server_obj->profile->id, 'parent.config', 'algorithm', undef );
				my $algorithm = "";
				if ( defined($parent_select_alg) ) {
					$algorithm = "round_robin=$parent_select_alg";
				}
				$text .= "dest_domain=" . $org_uri->host . " port=" . $org_uri->port . " parent=$origin_shield $algorithm go_direct=true\n";
			}
			elsif ($multi_site_origin) {
				$text .= "dest_domain=" . $org_uri->host . " port=" . $org_uri->port . " ";

				# If we have multi-site origin, get parent_data once
				if ( not defined($parent_info) ) {
					$parent_info = $self->parent_data($server_obj);
				}

				my @ranked_parents = ();
				if ( exists( $parent_info->{ $org_uri->host } ) ) {
					@ranked_parents = sort by_parent_rank @{ $parent_info->{ $org_uri->host } };
				}
				else {
					$self->app->log->warn( "BUG: Did not match a multi-site origin: " . $org_uri );
				}

				my @parent_info;
				my @secondary_parent_info;
				my @null_parent_info;
				foreach my $parent (@ranked_parents) {
					if ( $parent->{primary_parent} ) {
						push @parent_info, format_parent_info($parent);
					}
					elsif ( $parent->{secondary_parent} ) {
						push @secondary_parent_info, format_parent_info($parent);
					}
					else {
						push @null_parent_info, format_parent_info($parent);
					}
				}

				# If we don't find any parents in the primary parent, then use the secondary parents list as primary and clear the secondary list.
				# Once we do that, null parent will be used in the secondary_parent category due to the join of the two arrays.
				# This prevents blank parent entries.
				if ( scalar @parent_info == 0  ) {
					# If no parents are found in the secondary parent either, then set the null parent list (parents in neither secondary or primary)
					# as the secondary parent list and clear the null parent list.
					if ( scalar @secondary_parent_info == 0  ) {
						@secondary_parent_info = @null_parent_info;
						@null_parent_info = ();
					}
					@parent_info = @secondary_parent_info;
					@secondary_parent_info = ();
				}

				my %seen;
				@parent_info = grep { !$seen{$_}++ } @parent_info;

				if ( scalar @secondary_parent_info > 0 ) {
					my %seen;
					@secondary_parent_info = grep { !$seen{$_}++ } @secondary_parent_info;
				}
				if ( scalar @null_parent_info > 0 ) {
					my %seen;
					@null_parent_info = grep { !$seen{$_}++ } @null_parent_info;
				}
				#If the ats version supports it and the algorithm is consistent hash, put secondary and non-primary parents into secondary parent group.
				#This will ensure that secondary and tertiary parents will be unused unless all hosts in the primary group are unavailable.
				my $parents= '';
				if ( $ats_major_version >= 6 && $mso_algorithm eq "consistent_hash" && ( scalar @secondary_parent_info > 0 || scalar @null_parent_info > 0 ) ) {
					$parents = 'parent="' . join( '', @parent_info ) . '" secondary_parent="' . join( '', @secondary_parent_info ) . '' . join( '', @null_parent_info ) . '"';
				}
				else {
					$parents = 'parent="' . join( '', @parent_info ) . '' . join( '', @secondary_parent_info ) . '' . join( '', @null_parent_info ) . '"';
				}
				$text .= "$parents round_robin=$mso_algorithm qstring=$parent_qstring go_direct=false parent_is_proxy=false";

				if ( $ats_major_version >= 6 && $parent_retry ne "" ) {
					if ( $unavailable_server_retry_responses ne "" && $unavailable_server_retry_responses =~ /^"(?:\d{3},)+\d{3}"\s*$/) {
						$text .= " parent_retry=$parent_retry unavailable_server_retry_responses=$unavailable_server_retry_responses";
					} else {
						$text .= " parent_retry=$parent_retry";
					}
					$text .= " max_simple_retries=$max_simple_retries max_unavailable_server_retries=$max_unavailable_server_retries";
				}
				$text .= "\n";
				push @text_array, $text;
			}
		}

		#$text .= "dest_domain=. go_direct=true\n"; # this is implicit.
		#$self->app->log->debug( "MID PARENT.CONFIG:\n" . $text . "\n" );
		my $text = $header;
		foreach my $line (sort @text_array ){
			$text .= $line;
		}

		return $text;
	}
	else {    #"True" Parent - we are genning a EDGE config that points to a parent proxy.
		$parent_info = $self->parent_data($server_obj);
		my %done = ();
		my $qsh = $self->profile_param_value( $server_obj->profile->id, 'parent.config', 'psel.qstring_handling');
		my @parent_info;
		my @secondary_parent_info;
		foreach my $parent ( @{ $parent_info->{all_parents} } ) {
			my $ptxt = format_parent_info($parent);
			if ( $parent->{primary_parent} ) {
				push @parent_info, $ptxt;
			}
			elsif ( $parent->{secondary_parent} ) {
				push @secondary_parent_info, $ptxt;
			}
		}
		if ( scalar @parent_info == 0  ) {
			@parent_info = @secondary_parent_info;
			@secondary_parent_info = ();
		}
		my %seen;
		@parent_info = grep { !$seen{$_}++ } @parent_info;
		if ( scalar @secondary_parent_info > 0 ) {
			my %seen;
			@secondary_parent_info = grep { !$seen{$_}++ } @secondary_parent_info;
		}
		#If the ats version supports it, put secondary parents into secondary parent group.
		#This will ensure that secondary parents will be unused unless all hosts in the primary group are unavailable.
		my $parents = '';
		my $secparents = '';
		if ( $ats_major_version >= 6 && @secondary_parent_info > 0 ) {
			$parents = 'parent="' . join( '', sort @parent_info ) . '"';
			$secparents = ' secondary_parent="' . join( '', sort @secondary_parent_info ) . '"';
		}
		else {
			$parents = 'parent="' . join( '', sort @parent_info ) . join( '', sort @secondary_parent_info ) . '"';
		}

		my $round_robin = 'round_robin=consistent_hash';
		my $go_direct   = 'go_direct=false';

		if ( !defined($data) ) {
			$data = $self->parent_ds_data($server_obj);
		}

		if (defined($data->{dslist})) {
			foreach my $ds ( sort @{ $data->{dslist} } ) {
				my $text;
				my $org = $ds->{org};
				next if !defined $org || $org eq "";
				next if $done{$org};
				my $org_uri = URI->new($org);
				if ( $ds->{type} eq "HTTP_NO_CACHE" || $ds->{type} eq "HTTP_LIVE" || $ds->{type} eq "DNS_LIVE" ) {
					$text .= "dest_domain=" . $org_uri->host . " port=" . $org_uri->port . " go_direct=true\n";
				}
				else {
					# check for profile psel.qstring_handling.  If this parameter is assigned to the server profile,
					# then edges will use the qstring handling value specified in the parameter for all profiles.
	
					# If there is no defined parameter in the profile, then check the delivery service profile.
					# If psel.qstring_handling exists in the DS profile, then we use that value for the specified DS only.
					# This is used only if not overridden by a server profile qstring handling parameter.
					my $ds_qsh = $qsh;
					if (!defined($qsh)) {
						$ds_qsh = $ds->{'param'}->{'parent.config'}->{'psel.qstring_handling'};
					}
					my $parent_qstring = defined($ds_qsh) ? $ds_qsh : "ignore";
	
					if ( $ds->{qstring_ignore} == 0 && !defined($ds_qsh) ) {
						$parent_qstring = "consider";
					}
					$text
						.= "dest_domain="
						. $org_uri->host
						. " port="
						. $org_uri->port
						. " $parents $secparents $round_robin $go_direct qstring=$parent_qstring\n";
				}
				push @text_array, $text;
				$done{$org} = 1;
			}
		}

		my $default_dest_text;
		my $parent_select_alg = $self->profile_param_value( $server_obj->profile->id, 'parent.config', 'algorithm', undef );
		if ( defined($parent_select_alg) && $parent_select_alg eq 'consistent_hash' ) {
			$default_dest_text .= "dest_domain=. ";
			$default_dest_text .= $parents . $secparents;
			$default_dest_text .= " round_robin=consistent_hash go_direct=false";
		}
		else {    # default to old situation.
			$default_dest_text .= "dest_domain=. ";
			$default_dest_text .= $parents;
			$default_dest_text .= " round_robin=urlhash go_direct=false";
		}

		my $qstring = $self->profile_param_value( $server_obj->profile->id, 'parent.config', 'qstring', undef );
		if ( defined($qstring) ) {
			$default_dest_text .= " qstring=" . $qstring;
		}

		$default_dest_text .= "\n";

		my $text = $header;
		foreach my $line (sort @text_array ){
			$text .= $line;
		}

		$text .= $default_dest_text;
		return $text;
	}
}

sub remap_dot_config {
	my $self       = shift;
	my $server_obj = shift;
	my $data;
	my @text_array;


	my $pdata = $self->param_data( $server_obj, 'package' );
	my $header = $self->header_comment( $server_obj->host_name );
	if ( !defined($data) ) {
		$data = $self->remap_ds_data($server_obj);
	}
	if ( $server_obj->type->name =~ m/^MID/ ) {
		my %mid_remap;
		foreach my $ds ( @{ $data->{dslist} } ) {
			if ( $ds->{type} =~ /LIVE/ && $ds->{type} !~ /NATNL/ ) {
				next;    # Live local delivery services skip mids
			}
			if ( defined( $ds->{org} ) && defined( $mid_remap{ $ds->{org} } ) ) {
				next;    # skip remap rules from extra HOST_REGEXP entries
			}

			if ( defined( $ds->{mid_header_rewrite} ) && $ds->{mid_header_rewrite} ne "" ) {
				$mid_remap{ $ds->{org} } .= " \@plugin=header_rewrite.so \@pparam=" . $ds->{mid_hdr_rw_file};
			}
			if ( $ds->{qstring_ignore} == 1 ) {
				$mid_remap{ $ds->{org} } .= UI::DeliveryService::get_qstring_ignore_remap(UI::DeliveryService::get_ats_major_version( $self, $server_obj ));
			}
			if ( defined( $ds->{cacheurl} ) && $ds->{cacheurl} ne "" ) {
				$mid_remap{ $ds->{org} } .= " \@plugin=cacheurl.so \@pparam=" . $ds->{cacheurl_file};
			}
			if ( defined( $ds->{'param'}->{'cachekey.config'} ) ) {
				$mid_remap{ $ds->{org} } .= " \@plugin=cachekey.so";
				foreach my $ck_entry ( keys %{ $ds->{'param'}->{'cachekey.config'} } ) {
					$mid_remap{ $ds->{org} } .= " \@pparam=--" . $ck_entry . "=" . $ds->{'param'}->{'cachekey.config'}->{$ck_entry};
				}
			}
			if ( $ds->{range_request_handling} == RRH_CACHE_RANGE_REQUEST ) {
				$mid_remap{ $ds->{org} } .= " \@plugin=cache_range_requests.so";
			}
		}
		foreach my $key ( keys %mid_remap ) {
			my $remap_text .= "map " . $key . " " . $key . $mid_remap{$key} . "\n";
			push @text_array, $remap_text;
		}
		my $text = $header;
		foreach my $line (sort @text_array ){
			$text .= $line;
		}
		return $text;
	}

	# mids don't get here.
	foreach my $ds ( @{ $data->{dslist} } ) {
		my $remap_text;
		if ( $ds->{type} eq 'ANY_MAP' ) {
			$remap_text .= $ds->{remap_text} . "\n";
		}
		else {
			foreach my $map_from ( keys %{ $ds->{remap_line} } ) {
				my $map_to = $ds->{remap_line}->{$map_from};
				$remap_text = $self->build_remap_line( $server_obj, $pdata, $remap_text, $ds, $map_from, $map_to );
			}
			foreach my $map_from ( keys %{ $ds->{remap_line2} } ) {
				my $map_to = $ds->{remap_line2}->{$map_from};
				$remap_text = $self->build_remap_line( $server_obj, $pdata, $remap_text, $ds, $map_from, $map_to );
			}
		}

		push @text_array, $remap_text;
	}
	my $text = $header;
	foreach my $line (sort @text_array ){
		$text .= $line;
	}
	return $text;
}

sub build_remap_line {
	my $self        = shift;
	my $server_obj 	= shift;
	my $pdata       = shift;
	my $text        = shift;
	my $remap       = shift;
	my $map_from    = shift;
	my $map_to      = shift;



	my $dscp      = $remap->{dscp};
	my $hostname  = $server_obj->host_name;

	$map_from =~ s/__http__/$hostname/;

	if ( defined( $pdata->{'dscp_remap'} ) ) {
		$text .= "map	" . $map_from . "     " . $map_to . " \@plugin=dscp_remap.so \@pparam=" . $dscp;
	}
	else {
		$text .= "map	" . $map_from . "     " . $map_to . " \@plugin=header_rewrite.so \@pparam=dscp/set_dscp_" . $dscp . ".config";
	}
	if ( defined( $remap->{edge_header_rewrite} ) ) {
		$text .= " \@plugin=header_rewrite.so \@pparam=" . $remap->{hdr_rw_file};
	}
	if ( defined( $remap->{signing_algorithm})) {
		if ( $remap->{signing_algorithm} eq "url_sig" ) {
			$text .= " \@plugin=url_sig.so \@pparam=url_sig_" . $remap->{ds_xml_id} . ".config";
		}
		elsif ( $remap->{signing_algorithm} eq "uri_signing" ) {
			$text .= " \@plugin=uri_signing.so \@pparam=uri_signing_" . $remap->{ds_xml_id} . ".config";
		}
	}
	if ( $remap->{qstring_ignore} == 2 ) {
		my $dqs_file = "drop_qstring.config";
		$text .= " \@plugin=regex_remap.so \@pparam=" . $dqs_file;
	}
	elsif ( $remap->{qstring_ignore} == 1 ) {
		my $global_exists = $self->profile_param_value( $server_obj->profile->id, 'cacheurl.config', 'location', undef );
		if ($global_exists) {
			$self->app->log->debug(
				"qstring_ignore == 1, but global cacheurl.config param exists, so skipping remap rename config_file=cacheurl.config parameter if you want to change"
			);
		}
		else {
			#If we are on ats 6 and later we want to use the cachekey plugin, otherwise we have to use cacheurl
			$text .= UI::DeliveryService::get_qstring_ignore_remap(UI::DeliveryService::get_ats_major_version($self, $server_obj ));
		}
	}
	if ( defined( $remap->{cacheurl} ) && $remap->{cacheurl} ne "" ) {
		$text .= " \@plugin=cacheurl.so \@pparam=" . $remap->{cacheurl_file};
	}

	if ( defined( $remap->{'param'}->{'cachekey.config'} ) ) {
		$text .= " \@plugin=cachekey.so";
		foreach my $ck_entry ( keys %{ $remap->{'param'}->{'cachekey.config'} } ) {
			$text .= " \@pparam=--" . $ck_entry . "=" . $remap->{'param'}->{'cachekey.config'}->{$ck_entry};
		}
	}

	# Note: should use full path here?
	if ( defined( $remap->{regex_remap} ) && $remap->{regex_remap} ne "" ) {
		$text .= " \@plugin=regex_remap.so \@pparam=regex_remap_" . $remap->{ds_xml_id} . ".config";
	}
	if ( $remap->{range_request_handling} == RRH_BACKGROUND_FETCH ) {
		$text .= " \@plugin=background_fetch.so \@pparam=bg_fetch.config";
	}
	elsif ( $remap->{range_request_handling} == RRH_CACHE_RANGE_REQUEST ) {
		$text .= " \@plugin=cache_range_requests.so ";
	}
	if ( defined( $remap->{remap_text} ) ) {
		$text .= " " . $remap->{remap_text};
	}
	if ( defined( $remap->{fq_pacing_rate} ) && $remap->{fq_pacing_rate} > 0 ) {
		$text .= " \@plugin=fq_pacing.so \@pparam=--rate=" . $remap->{fq_pacing_rate}; 
	}	
	$text .= "\n";
	return $text;
}

sub __get_json_parameter_list_by_host {
	my $self      = shift;
	my $host      = shift;
	my $value     = shift;
	my $key_name  = shift || "name";
	my $key_value = shift || "value";
	my $data_obj  = [];

	my $profile_id = $self->db->resultset('Server')->search( { host_name => $host } )->get_column('profile')->first();

	my %condition = ( 'profile_parameters.profile' => $profile_id, config_file => $value );
	my $rs_config = $self->db->resultset('Parameter')->search( \%condition, { join => 'profile_parameters' } );

	while ( my $row = $rs_config->next ) {
		push( @{$data_obj}, { $key_name => $row->name, $key_value => $row->value } );
	}

	return ($data_obj);
}

sub __get_json_parameter_by_host {
	my $self      = shift;
	my $host      = shift;
	my $parameter = shift;
	my $value     = shift;
	my $key_name  = shift || "name";
	my $key_value = shift || "value";
	my $data_obj;

	my $rs_profile = $self->db->resultset('Server')->search( { 'me.host_name' => $host }, { prefectch => [ 'cdn', 'profile' ] } );

	my $row = $rs_profile->next;
	my $id  = $row->id;
	if ( defined($row) && defined( $row->cdn->name ) ) {
		push( @{$data_obj}, { $key_name => "CDN_Name", $key_value => $row->cdn->name } );
	}

	my %condition = (
		'profile_parameters.profile' => $id,
		'config_file'                => $value,
		name                         => $parameter
	);
	my $rs_config = $self->db->resultset('Parameter')->search( \%condition, { join => 'profile_parameters' } );
	$row = $rs_config->next;

	if ( defined($row) && defined( $row->name ) && defined( $row->value ) ) {
		push( @{$data_obj}, { $key_name => $row->name, $key_value => $row->value } );
	}

	return ($data_obj);
}

sub get_package_versions {
	my $self       = shift;
	my $server_obj = shift;
	my $filename   = shift;

	my $host_name = $server_obj->host_name;
	my $data_obj = __get_json_parameter_list_by_host( $self, $host_name, "package", "name", "version" );

	return ($data_obj);
}

sub get_chkconfig {
	my $self       = shift;
	my $server_obj = shift;
	my $filename   = shift;

	my $host_name = $server_obj->host_name;
	my $data_obj = __get_json_parameter_list_by_host( $self, $host_name, "chkconfig" );

	return ($data_obj);
}

1;
