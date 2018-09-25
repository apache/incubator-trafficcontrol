package API::DeliveryService::SslKeys;
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
use UI::Utils;
use Utils::Tenant;
use Mojo::Base 'Mojolicious::Controller';
use MojoPlugins::Response;
use JSON;
use MIME::Base64;
use UI::DeliveryService;
use Data::Dumper;
use Validate::Tiny ':all';

sub add {
	my $self     = shift;
	my $key_type = "ssl";
	my $key      = $self->req->json->{key};
	my $version  = $self->req->json->{version};
	my $crt      = $self->req->json->{certificate}->{crt};
	my $csr      = $self->req->json->{certificate}->{csr};
	my $priv_key = $self->req->json->{certificate}->{key};
	my $hostname = $self->req->json->{hostname};
	my $cdn = $self->req->json->{cdn};
	my $deliveryservice = $self->req->json->{deliveryservice};

	my $ds = $self->db->resultset('Deliveryservice')->search( { xml_id => $deliveryservice })->single();
	if (!$ds) {
		return $self->not_found("Could not found delivery service with xml_id=$deliveryservice" );
	}
	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Delivery-service tenant is not available to the user.");
	}

	my $record = {
		key => $key,
		version => $version,
		hostname => $hostname,
		cdn => $cdn,
		deliveryservice => $deliveryservice,
		certificate => {
			crt => $crt,
			csr => $csr,
			key => $priv_key
		}
	};

	my $response_container = $self->add_ssl_keys_to_riak( $record );
	my $response = $response_container->{"response"};
	if ( $response->is_success() ) {
		&log( $self, "Added ssl keys for Delivery Service $key", "APICHANGE" );
		return $self->success("Successfully added ssl keys for $key");
	}
	else {
		return $self->alert( $response->{_content} );
	}
}

#named like this because there is a plugin called generate_ssl_keys in Mojoplugins/SslKeys.pm
sub generate {
	my $self         = shift;
	my $params = $self->req->json;

	my ( $is_valid, $result ) = $self->is_valid( $params );

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $key_type     = "ssl";
	my $key          = $params->{key};
	my $version      = $params->{version};        # int
	my $hostname     = $params->{hostname};
	my $country      = $params->{country};
	my $state        = $params->{state};
	my $city         = $params->{city};
	my $org          = $params->{organization};
	my $unit         = $params->{businessUnit};
	my $cdn          = $params->{cdn};
	my $deliveryservice = $params->{deliveryservice};
	my $tmp_location = "/var/tmp";

	if ( !&is_oper($self) ) {
		return $self->forbidden();
	}

	if (defined($deliveryservice)) {
		my $ds = $self->db->resultset('Deliveryservice')->search( { xml_id => $deliveryservice })->single();
		if (!$ds) {
			return $self->not_found("Could not found delivery service with xml_id=$deliveryservice" );
		}
		my $tenant_utils = Utils::Tenant->new($self);
		my $tenants_data = $tenant_utils->create_tenants_data_from_db();
		if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
			return $self->forbidden("Forbidden. Delivery-service tenant is not available to the user.");
		}
	}

	#generate the cert:
	my $record = {
		key => $key,
		version => $version,
		hostname => $hostname,
		country => $country,
		city => $city,
		state => $state,
		org => $org,
		unit => $unit,
		cdn => $cdn,
		deliveryservice => $deliveryservice,
	};

	my $response_container = $self->generate_ssl_keys( $record );
	my $response = $response_container->{"response"};
	if ( $response->is_success() ) {
		&log( $self, "Created ssl keys for Delivery Service $key", "APICHANGE" );
		return $self->success("Successfully created ssl keys for $key");
	}
	else {
		return $self->alert( $response->{_content} );
	}
}

sub is_valid {
	my $self   = shift;
	my $params = shift;

	my $rules = {
		fields => [ qw/cdn deliveryservice key version hostname country state city organization businessUnit/ ],

		# Validation checks to perform
		checks => [
			deliveryservice	=> [ is_required("is required") ],
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

sub view_by_xml_id {
	my $self    = shift;
	my $xml_id     = $self->param('xmlid');
	my $version = $self->param('version');
	my $decode  = $self->param('decode');

	if ( ! defined $decode ) {
		$decode = 0;
	}

	if ( !$version ) {
		$version = 'latest';
	}

	my $key = "$xml_id-$version";
	my $ds = $self->db->resultset('Deliveryservice')->search( { xml_id => $xml_id })->single();
	if (!$ds) {
		return $self->not_found();
	}
	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Delivery-service tenant is not available to the user.");
	}
	my $response_container = $self->riak_get( "ssl", $key );
	my $response = $response_container->{"response"};


	if ( $response->is_success() ){
		my $toSend = decode_json( $response->content );

		if ( $decode ){
			$toSend->{certificate}->{csr} = decode_base64($toSend->{certificate}->{csr});
			$toSend->{certificate}->{crt} = decode_base64($toSend->{certificate}->{crt});
			$toSend->{certificate}->{key} = decode_base64($toSend->{certificate}->{key});
		}


		$self->success( $toSend )

	} else {
		$self->success({}, " - A record for ssl key $key could not be found. ");
	}
}

sub view_by_hostname {
	my $self    = shift;
	my $key     = $self->param('hostname');
	my $version = $self->param('version');
	my $decode  = $self->param('decode');

	if ( ! defined $decode ) {
		$decode = 0;
	}

	#use hostname to get hostname regex
	my @split_url = split( /\./, $key );
	my $host_regex = $split_url[1];
	my $domain_name;

	for ( my $i = 2; $i < $#split_url; $i++ ) {
		$domain_name .= $split_url[$i] . ".";
	}
	$domain_name .= $split_url[$#split_url];

	$host_regex = '.*\.' . $host_regex . '\..*';

	if ( !$host_regex || !$domain_name ) {
		return $self->alert( { Error => " - $key does not contain a valid delivery service." } ) if !$host_regex;
		return $self->alert( { Error => " - $key does not contain a valid domain name." } )      if !$domain_name;
	}

	my $cdn_id = $self->db->resultset('Cdn')->search( { domain_name => $domain_name } )->get_column('id')->single();
	if (!$cdn_id || $cdn_id == "") {
		return $self->alert( {Error => " - a cdn does not exist for the domain: $domain_name parsed from hostname: $key" } );
	}

	my $ds = $self->db->resultset('Deliveryservice')->search( { 'regex.pattern' => "$host_regex", 'cdn_id' => "$cdn_id" }, { join => { deliveryservice_regexes => { regex => undef } } } )->single();
	if (!$ds) {
		return $self->alert( { Error => " - A delivery service does not exist for a host with hostname of $key" } );
	}
	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Delivery-service tenant is not available to the user.");
	}
	my $xml_id = $ds->xml_id;

	if ( !$version ) {
		$version = 'latest';
	}
	$key = "$xml_id-$version";
	my $response_container = $self->riak_get( "ssl", $key );
	my $response = $response_container->{"response"};


	if ( $response->is_success() ){
		my $toSend = decode_json( $response->content );

		if ( $decode ){
			$toSend->{certificate}->{csr} = decode_base64($toSend->{certificate}->{csr});
			$toSend->{certificate}->{crt} = decode_base64($toSend->{certificate}->{crt});
			$toSend->{certificate}->{key} = decode_base64($toSend->{certificate}->{key});
		}

	
		$self->success( $toSend )

	} else {
		$self->success({}, " - A record for ssl key $key could not be found. ");
	}
}

sub delete {
	my $self    = shift;
	my $xml_id     = $self->param('xmlid');
	my $version = $self->param('version');
	my $response_container;
	my $response;

	my $ds = $self->db->resultset('Deliveryservice')->search( { xml_id => $xml_id })->single();
	if (!$ds) {
		return $self->alert( { Error => " - Could not found delivery service with xml_id=$xml_id!" } );
	}
	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id)) {
		return $self->forbidden("Forbidden. Delivery-service tenant is not available to the user.");
	}
	my $key = $xml_id;
	if ($version) {
		$key = $key . "-" . $version;
		$self->app->log->info("deleting key_type = ssl, key = $key");
		$response_container = $self->riak_delete( "ssl", $key );
		$response = $response_container->{"response"};
	}
	else {
		#TODO figure out riak searching so we dont have to hardcode "latest"
		$key = "$key-latest";
		$self->app->log->info("deleting key_type = ssl, key = $key");
		$response_container = $self->riak_delete( "ssl", $key );
		$response = $response_container->{"response"};
	}

	# $self->app->log->info("delete rc = $rc");
	if ( $response->is_success() ) {
		&log( $self, "Deleted ssl keys for Delivery Service $xml_id", "APICHANGE" );
		return $self->success("Successfully deleted ssl keys for $key");
	}
	else {
		return $self->alert( $response->content );
	}
}

1;
