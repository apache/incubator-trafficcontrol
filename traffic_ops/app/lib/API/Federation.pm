package API::Federation;
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
use Data::Dumper;
use Net::CIDR;
use JSON;
use Validate::Tiny ':all';
use Data::Validate::IP qw(is_ipv4 is_ipv6);

use constant SUCCESS => 0;
use constant ERROR   => 1;

sub get_all_federation_resolver_mappings {
	my $self     = shift;
	my $cdn_name = $self->param('cdnName');
	my $data     = [];

	my $rs_data;
	if ( defined $cdn_name ) {
		push(
			@{$data}, {
				"cdnName" => $cdn_name
			}
		);

		$rs_data = $self->find_federations_by_cdn($cdn_name);
	}
	else {
		$rs_data = $self->find_federations();
	}

	while ( my $row = $rs_data->next ) {
		my $mapping;
		$mapping->{'cname'} = $row->federation->cname;
		$mapping->{'ttl'}   = $row->federation->ttl;

		my $federation_id = $row->federation->id;
		my @resolvers     = $self->find_federation_resolvers($federation_id);
		for my $resolver (@resolvers) {
			my $type = lc $resolver->type->name;
			if ( !defined $mapping->{$type} ) {
				$mapping->{$type} = [];
			}
			push( @{ $mapping->{$type} }, $resolver->ip_address );
		}

		my $xml_id = $row->deliveryservice->xml_id;
		if ( defined $data ) {
			my $ds = $self->find_delivery_service( $xml_id, $data );
			if ( !defined $ds ) {
				$data = $self->add_delivery_service( $xml_id, $mapping, $data );
			}
			else {
				$self->update_delivery_service( $ds, $mapping );
			}
		}
		else {
			$data = $self->add_delivery_service( $xml_id, $mapping, $data );
		}
	}

	$self->success($data);
}

sub find_federations {
	my $self           = shift;
	my $federation_ids = shift;
	my $rs_data;

	if ($federation_ids) {
		$rs_data = $self->db->resultset('FederationDeliveryservice')->search(
			{ federation => { -in => $federation_ids } },
			{
				prefetch => [ 'federation', 'deliveryservice' ],
				order_by => "deliveryservice.xml_id"
			}
		);
	}
	else {
		$rs_data = $self->db->resultset('FederationDeliveryservice')->search(
			{},
			{
				prefetch => [ 'federation', 'deliveryservice' ],
				order_by => "deliveryservice.xml_id"
			}
		);
	}
	return $rs_data;
}

sub find_federations_by_cdn {
	my $self     = shift;
	my $cdn_name = shift;

	my @ds_ids = $self->db->resultset('Deliveryservice')->search( { 'cdn.name' => $cdn_name }, { prefetch => 'cdn' } )->get_column('id')->all();

	my $rs_data = $self->db->resultset('FederationDeliveryservice')->search(
		{ deliveryservice => { -in => \@ds_ids } },
		{
			prefetch => [ 'federation', 'deliveryservice' ],
			order_by => "deliveryservice.xml_id"
		}
	);
	return $rs_data;
}

sub find_federation_resolvers {
	my $self          = shift;
	my $federation_id = shift;

	my @resolvers = $self->db->resultset('FederationResolver')
		->search( { 'federation_federation_resolvers.federation' => $federation_id }, { prefetch => 'federation_federation_resolvers' } )->all();

	return @resolvers;
}

sub find_delivery_service {
	my $self   = shift;
	my $xml_id = shift;
	my $data   = shift;
	my $ds;

	foreach my $service ( @{$data} ) {
		if ( $service->{'deliveryService'} eq $xml_id ) {
			$ds = $service;
		}
	}
	return $ds;
}

sub add_delivery_service {
	my $self   = shift;
	my $xml_id = shift;
	my $m      = shift;
	my $data   = shift;

	push(
		@{$data}, {
			"deliveryService" => $xml_id,
			"mappings"        => [$m]
		}
	);
	return $data;
}

sub update_delivery_service {
	my $self = shift;
	my $ds   = shift;
	my $m    = shift;

	push( @{ $ds->{'mappings'} }, $m );
}

sub get_current_user_federation_resolver_mappings {
	my $self             = shift;
	my $data             = [];
	my $current_username = $self->current_user()->{username};

	if ( !&is_federation($self) ) {
		return $self->forbidden();
	}

	my $rs_data;
	my ( $rc, $response, @federation_ids ) = $self->find_federation_tmuser($current_username);
	if ( $rc == SUCCESS ) {
		$rs_data = $self->find_federations( \@federation_ids );
	}
	else {
		return $self->alert($response);
	}

	while ( my $row = $rs_data->next ) {
		my $mapping;
		$mapping->{'cname'} = $row->federation->cname;
		$mapping->{'ttl'}   = $row->federation->ttl;

		my $federation_id = $row->federation->id;
		my @resolvers     = $self->find_federation_resolvers($federation_id);
		for my $resolver (@resolvers) {
			my $type = lc $resolver->type->name;
			if ( !defined $mapping->{$type} ) {
				$mapping->{$type} = [];
			}
			push( @{ $mapping->{$type} }, $resolver->ip_address );
		}

		my $xml_id = $row->deliveryservice->xml_id;
		if ( defined $data ) {
			my $ds = $self->find_delivery_service( $xml_id, $data );
			if ( !defined $ds ) {
				$data = $self->add_delivery_service( $xml_id, $mapping, $data );
			}
			else {
				$self->update_delivery_service( $ds, $mapping );
			}
		}
		else {
			$data = $self->add_delivery_service( $xml_id, $mapping, $data );
		}
	}
	$self->success($data);
}

sub find_federation_tmuser {
	my $self             = shift;
	my $current_username = shift;
	my @federation_ids;

	my ( $rc, $response, $tm_user ) = $self->find_tmuser($current_username);
	if ( $rc == SUCCESS ) {
		@federation_ids = $self->db->resultset('FederationTmuser')->search(
			{
				tm_user => $tm_user->id,
				role    => $tm_user->role->id
			},
		)->get_column('federation')->all();

		return ( SUCCESS, $response, @federation_ids );
	}
	else {
		return ( ERROR, $response, @federation_ids );
	}
}

sub find_tmuser {
	my $self             = shift;
	my $current_username = shift;

	my $tm_user =
		$self->db->resultset('TmUser')->search( { username => $current_username } )->single();

	my $response;
	return ( SUCCESS, $response, $tm_user );
}

sub add_federation_resolver_mappings_for_current_user {
	my $self = shift;
	my $current_username = $self->current_user()->{username};

	if ( !&is_federation($self) ) {
		return $self->forbidden();
	}

	my ( $rc, $response, $user ) = $self->find_tmuser($current_username);
	if ( $rc == ERROR ) {
		return $self->alert($response);
	}

	my $federations = $self->req->json->{'federations'};
	foreach my $ds ( @{$federations} ) {
		my $xml_id = $ds->{'deliveryService'};
		my $map    = $ds->{'mappings'};

		my ( $is_valid, $result ) = $self->is_valid(
			{
				deliveryService => $xml_id,
				mappings        => $map
			}
		);
		if ( $is_valid == ERROR ) {
			return $self->alert($result);
		}

		my ( $rc, $response, $federation_id ) = $self->find_federation_deliveryservice( $user, $xml_id );
		if ( $rc == ERROR ) {
			return $self->alert($response);
		}

		my $resolve4 = $map->{'resolve4'};
		my $resolve6 = $map->{'resolve6'};
		( $rc, $response ) = $self->add_resolvers( $resolve4, $resolve6, $xml_id, $federation_id );

		if ( $rc == SUCCESS ) {
			$self->app->log->info($response);
			&log( $self, $response, "APICHANGE" );
		}
		else {
			return $self->alert($response);
		}
	}
	$self->success("$current_username successfully created federation resolvers.");
}

sub is_valid {
	my $self       = shift;
	my $federation = shift;

	my $rules = {
		fields => [qw/deliveryService mappings/],

		checks => [ [qw/deliveryService mappings/] => is_required("is required") ]
	};

	my $result = validate( $federation, $rules );
	if ( $result->{success} ) {
		return ( SUCCESS, $result->{data} );
	}
	else {
		return ( ERROR, $result->{error} );
	}
}

sub find_federation_deliveryservice {
	my $self             = shift;
	my $user             = shift;
	my $xml_id           = shift;
	my $current_username = $self->current_user()->{username};

	my @ids = $self->db->resultset('FederationTmuser')->search( { tm_user => $user->id } )->get_column('federation')->all();
	my $ds = $self->db->resultset('Deliveryservice')->search( { xml_id => $xml_id } )->get_column('id')->single();

	my @federation_ids;
	my $response;
	if ( scalar @ids ) {
		@federation_ids = $self->db->resultset('FederationDeliveryservice')->search(
			{
				deliveryservice => $ds,
				federation      => { -in => \@ids }
			},
			{ prefetch => 'federation' }
		)->get_column('federation.id')->all();

		if ( !scalar @federation_ids ) {
			$response = "No federation(s) found for user $current_username on delivery service '$xml_id'.";
			return ( ERROR, $response, @federation_ids );
		}
		if ( @federation_ids > 1 ) {
			$response = "Found more than one federation for Delivery Service '$xml_id'.  Please contact your administrator.";
			return ( ERROR, $response, @federation_ids );
		}
	}
	else {
		$response = "No federation(s) found for user $current_username.";
		return ( ERROR, $response, @federation_ids );
	}

	return ( SUCCESS, $response, $federation_ids[0] );
}

sub add_resolvers {
	my $self             = shift;
	my $resolve4         = shift;
	my $resolve6         = shift;
	my $xml_id           = shift;
	my $federation_id    = shift;
	my $current_username = $self->current_user()->{username};

	my @resolver_ips;
	if ( defined $resolve4 ) {
		my ( $rc, $response, @ip4 ) = $self->add_federation_resolver( $resolve4, $federation_id, "RESOLVE4" );
		if ( $rc == ERROR ) {
			return ( ERROR, $response );
		}
		push( @resolver_ips, @ip4 );
	}

	if ( defined $resolve6 ) {
		my ( $rc, $response, @ip6 ) = $self->add_federation_resolver( $resolve6, $federation_id, "RESOLVE6" );
		if ( $rc == ERROR ) {
			return ( ERROR, $response );
		}
		push( @resolver_ips, @ip6 );
	}

	my $response = "$current_username successfully added federation resolvers for '$xml_id': [ " . join( ', ', @resolver_ips ) . " ]";
	return ( SUCCESS, $response );
}

sub add_federation_resolver {
	my $self          = shift;
	my $resolvers     = shift;
	my $federation_id = shift;
	my $type_name     = shift;
	my @resolver_ips;

	my $response;
	foreach my $r ( @{$resolvers} ) {
		for my $ip ($r) {
			my $invalid_ip = $ip;
			my $cidr       = Net::CIDR::range2cidr($ip);
			if ( !defined $cidr ) {
				$response = "[ $invalid_ip ] is not a valid ip address.";
				return ( ERROR, $response, @resolver_ips );
			}

			my $resolver = $self->db->resultset('FederationResolver')->find_or_create(
				{
					ip_address => $cidr,
					type       => $self->db->resultset('Type')->search( { name => $type_name } )->get_column('id')->single()
				}
			);

			if ( defined $resolver ) {
				$self->db->resultset('FederationFederationResolver')->find_or_create(
					{
						federation          => $federation_id,
						federation_resolver => $resolver->id
					}
				);
				push( @resolver_ips, $resolver->ip_address );
			}
		}
	}
	return ( SUCCESS, $response, @resolver_ips );
}

sub delete_federation_resolver_mappings_for_current_user {
	my $self             = shift;
	my $current_username = $self->current_user()->{username};

	if ( !&is_federation($self) ) {
		return $self->forbidden();
	}

	my ( $rc, $response, $user ) = $self->find_tmuser($current_username);
	if ( $rc == ERROR ) {
		return $self->alert($response);
	}

	( $rc, $response ) = $self->delete_federation_resolver($user);
	if ( $rc == SUCCESS ) {
		$self->app->log->info($response);
		&log( $self, $response, "APICHANGE" );
		$self->success($response);
	}
	else {
		return $self->alert($response);
	}
}

sub delete_federation_resolver {
	my $self             = shift;
	my $user             = shift;
	my $current_username = $self->current_user()->{username};

	my @federation_ids = $self->db->resultset('FederationTmuser')->search( { tm_user => $user->id } )->get_column('federation')->all();

	my @resolvers;
	my @resolver_ips;
	if ( scalar @federation_ids ) {
		@resolvers = $self->db->resultset('FederationResolver')
			->search( { 'federation_federation_resolvers.federation' => { -in => \@federation_ids } }, { prefetch => 'federation_federation_resolvers' } );

		if ( scalar @resolvers ) {
			foreach my $federation (@resolvers) {
				push( @resolver_ips, $federation->ip_address );
				$federation->delete();
			}
		}
	}

	my $response;
	if ( scalar @resolver_ips ) {
		$response = "$current_username successfully deleted all federation resolvers: [ " . join( ', ', @resolver_ips ) . " ].";
		return ( SUCCESS, $response );
	}
	else {
		$response = "No federation resolvers to delete for user $current_username.";
		return ( ERROR, $response );
	}
}

sub update_federation_resolver_mappings_for_current_user {
	my $self             = shift;
	my $current_username = $self->current_user()->{username};

	if ( !&is_federation($self) ) {
		return $self->forbidden();
	}

	$self->delete_federation_resolver_mappings_for_current_user();
	$self->add_federation_resolver_mappings_for_current_user();
}

sub get_cdn_federations {
	my $self		= shift;
	my $cdn_name	= $self->param('name');

	my $cdn = $self->db->resultset('Cdn')->find( { name => $cdn_name } );
	if ( !defined($cdn) ) {
		return $self->not_found();
	}

	my @ds_ids = $self->db->resultset('Deliveryservice')->search( { 'cdn.name' => $cdn_name }, { prefetch => 'cdn' } )->get_column('id')->all();
	my $ds_ids_list = join(',', @ds_ids);

	my $dbh = $self->db->storage->dbh;
	my $sth = $dbh->prepare("
			SELECT f.id AS id,
				f.cname AS cname,
				f.description AS description,
				f.ttl AS ttl,
				ds.id AS deliveryservice_id,
				ds.xml_id AS deliveryservice_xml_id,
				ds.tenant_id AS ds_tenant_id
			FROM federation f
				LEFT JOIN federation_deliveryservice fd ON fd.federation = f.id
				LEFT JOIN deliveryservice ds ON fd.deliveryservice = ds.id
			WHERE fd.deliveryservice IN ($ds_ids_list)
			ORDER BY f.cname");
	$sth->execute();

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();

	my @data;
	while ( my $federation = $sth->fetchrow_hashref ) {
		if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $federation->{ds_tenant_id})) {
			next; # skip if delivery service is outside the user's tenancy
		}
		push(
			@data, {
				"id"                => $federation->{id},
				"cname"             => $federation->{cname},
				"description"       => $federation->{description},
				"ttl"               => $federation->{ttl},
				"deliveryService"   => {
					"id"        => $federation->{deliveryservice_id},
					"xmlId"     => $federation->{deliveryservice_xml_id}
				},
			}
		);
	}

	$self->success( \@data );
}

sub get_cdn_federation {
	my $self  		= shift;
	my $fed_id 		= $self->param('fedId');

	my $fed = $self->db->resultset('Federation')->find( { id => $fed_id } );
	if ( !defined($fed) ) {
		return $self->not_found();
	}

	my $dbh = $self->db->storage->dbh;
	my $sth = $dbh->prepare("
			SELECT f.id AS id,
				f.cname AS cname,
				f.description AS description,
				f.ttl AS ttl,
				ds.tenant_id AS ds_tenant_id
			FROM federation f
				LEFT JOIN federation_deliveryservice fd ON fd.federation = f.id
				LEFT JOIN deliveryservice ds ON fd.deliveryservice = ds.id
			WHERE f.id = $fed_id");
	$sth->execute();

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();

	my @data;
	while ( my $federation = $sth->fetchrow_hashref ) {
		if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $federation->{ds_tenant_id})) {
			return $self->forbidden(); # return 403 if federation's delivery service is outside the user's tenancy
		}
		push(
			@data, {
				"id"            => $federation->{id},
				"cname"         => $federation->{cname},
				"description"   => $federation->{description},
				"ttl"           => $federation->{ttl},
			}
		);
	}

	$self->success( \@data );
}

sub create_cdn_federation {
	my $self        = shift;
	my $params      = $self->req->json;

	my ( $is_valid, $result ) = $self->is_federation_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $values = {
		cname    	=> $params->{cname},
		ttl       	=> $params->{ttl},
		description	=> $params->{description},
	};

	my $insert = $self->db->resultset('Federation')->create($values);
	my $rs = $insert->insert();
	if ($rs) {
		my $response;
		$response->{id}             = $rs->id;
		$response->{cname}          = $rs->cname;
		$response->{ttl}            = $rs->ttl;
		$response->{description}    = $rs->description;

		my $msg = "Federation created [ cname = " . $rs->cname . " ] with id: " . $rs->id;
		&log( $self, $msg, "APICHANGE" );
		return $self->success( $response, $msg );
	} else {
		return $self->alert("Federation creation failed");
	}
}

sub update_cdn_federation {
	my $self	= shift;
	my $fed_id	= $self->param('fedId');
	my $params	= $self->req->json;

	my ( $is_valid, $result ) = $self->is_federation_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $fed = $self->db->resultset('Federation')->find( { id => $fed_id } );
	if ( !defined($fed) ) {
		return $self->not_found();
	}

	my $values = {
		cname    	=> $params->{cname},
		ttl       	=> $params->{ttl},
		description	=> $params->{description},
	};

	my $rs = $fed->update($values);
	if ($rs) {
		my $response;
		$response->{cname}          = $rs->cname;
		$response->{ttl}            = $rs->ttl;
		$response->{description}    = $rs->description;

		my $msg = "Federation updated [ cname = " . $rs->cname . " ] with id: " . $rs->id;
		&log( $self, $msg, "APICHANGE" );
		return $self->success( $response, $msg );
	}
	else {
		return $self->alert("Federation update failed.");
	}
}

sub delete_cdn_federation {
	my $self	= shift;
	my $fed_id	= $self->param('fedId');

	my $fed = $self->db->resultset('Federation')->find( { id => $fed_id } );
	if ( !defined($fed) ) {
		return $self->not_found();
	}

	my $rs = $fed->delete();
	if ($rs) {
		my $msg = "Federation deleted [ cname = " . $rs->cname . " ] with id: " . $rs->id;
		&log( $self, $msg, "APICHANGE" );
		return $self->success_message($msg);
	} else {
		return $self->alert( "Federation delete failed." );
	}
}

sub is_federation_valid {
    my $self   = shift;
    my $params = shift;

    my $rules = {
        fields => [
            qw/cname ttl description/
        ],

        # Validation checks to perform
        checks => [
            cname   => [ is_required("is required"), is_like( qr/^\S*\.$/, "must contain no spaces and end with a dot" ) ],
            ttl     => [ is_required("is required"), is_like( qr/^\d+$/, "must be a number" ) ],
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


1;
