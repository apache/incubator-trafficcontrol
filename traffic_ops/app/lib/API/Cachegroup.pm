package API::Cachegroup;
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
# a note about locations and cachegroups. This used to be "Location", before we had physical locations in 12M. Very confusing.
# What used to be called a location is now called a "cache group" and location is now a physical address, not a group of caches working together.
#

# JvD Note: you always want to put Utils as the first use. Sh*t don't work if it's after the Mojo lines.
use UI::Utils;

use Mojo::Base 'Mojolicious::Controller';
use Data::Dumper;
use JSON;
use MojoPlugins::Response;
use Validate::Tiny ':all';

sub index {
	my $self = shift;
	my @data;
	my %idnames;
	my $orderby = $self->param('orderby') || "name";
	my $type_id = $self->param('type');

	my $rs_idnames = $self->db->resultset("Cachegroup")->search( undef, { columns => [qw/id name/] } );
	while ( my $row = $rs_idnames->next ) {
		$idnames{ $row->id } = $row->name;
	}

	my %criteria;
	if ( defined $type_id ) {
		$criteria{'type'} = $type_id;
	}

	my $rs_data = $self->db->resultset("Cachegroup")->search( \%criteria, { prefetch => [ 'type', 'coordinate' ], order_by => 'me.' . $orderby } );
	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"id"                            => $row->id,
				"name"                          => $row->name,
				"shortName"                     => $row->short_name,
				"latitude"                      => defined($row->coordinate) ? 0.0 + $row->coordinate->latitude : undef,
				"longitude"                     => defined($row->coordinate) ? 0.0 + $row->coordinate->longitude : undef,
				"lastUpdated"                   => $row->last_updated,
				"parentCachegroupId"            => $row->parent_cachegroup_id,
				"parentCachegroupName"          => ( defined $row->parent_cachegroup_id ) ? $idnames{ $row->parent_cachegroup_id } : undef,
				"fallbackToClosest"           => \$row->fallback_to_closest,
				"secondaryParentCachegroupId"   => $row->secondary_parent_cachegroup_id,
				"secondaryParentCachegroupName" => ( defined $row->secondary_parent_cachegroup_id )
				? $idnames{ $row->secondary_parent_cachegroup_id }
				: undef,
				"typeId"   => $row->type->id,
				"typeName" => $row->type->name
			}
		);
	}
	$self->success( \@data );
}

sub index_trimmed {
	my $self = shift;
	my @data;
	my $orderby = $self->param('orderby') || "name";

	my $rs_data = $self->db->resultset("Cachegroup")->search( undef, { order_by => 'me.' . $orderby } );
	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"name" => $row->name,
			}
		);
	}
	$self->success( \@data );
}

sub show {
	my $self = shift;
	my $id   = $self->param('id');

	my $rs_data = $self->db->resultset("Cachegroup")->search( { 'me.id' => $id }, { prefetch => [ 'type', 'coordinate' ] } );

	my @data = ();
	my %idnames;

	my $rs_idnames = $self->db->resultset("Cachegroup")->search( undef, { columns => [qw/id name/] } );
	while ( my $row = $rs_idnames->next ) {
		$idnames{ $row->id } = $row->name;
	}

	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"id"                            => $row->id,
				"name"                          => $row->name,
				"shortName"                     => $row->short_name,
				"latitude"                      => defined($row->coordinate) ? 0.0 + $row->coordinate->latitude : undef,
				"longitude"                     => defined($row->coordinate) ? 0.0 + $row->coordinate->longitude : undef,
				"lastUpdated"                   => $row->last_updated,
				"parentCachegroupId"            => $row->parent_cachegroup_id,
				"parentCachegroupName"          => ( defined $row->parent_cachegroup_id ) ? $idnames{ $row->parent_cachegroup_id } : undef,
				"fallbackToClosest"           => \$row->fallback_to_closest,
				"secondaryParentCachegroupId"   => $row->secondary_parent_cachegroup_id,
				"secondaryParentCachegroupName" => ( defined $row->secondary_parent_cachegroup_id )
				? $idnames{ $row->secondary_parent_cachegroup_id }
				: undef,
				"typeId"   => $row->type->id,
				"typeName" => $row->type->name
			}
		);
	}
	$self->success( \@data );
}

sub update {
	my $self   = shift;
	my $id     = $self->param('id');
	my $params = $self->req->json;

	my ( $is_valid, $result ) = $self->is_cachegroup_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $cachegroup = $self->db->resultset('Cachegroup')->find( { id => $id }, { prefetch => 'coordinate' } );
	if ( !defined($cachegroup) ) {
		return $self->not_found();
	}

	my $coordinate = $cachegroup->coordinate;

	my $name = $params->{name};
	if ( $cachegroup->name ne $name ) {
		my $existing = $self->db->resultset('Cachegroup')->find( { name => $name } );
		if ($existing) {
			return $self->alert( "A cachegroup with name " . $name . " already exists." );
		}
	}

	my $short_name = $params->{shortName};
	if ( $cachegroup->short_name ne $short_name ) {
		my $existing = $self->db->resultset('Cachegroup')->find( { short_name => $short_name } );
		if ($existing) {
			return $self->alert( "A cachegroup with short_name " . $short_name . " already exists." );
		}
	}

	my $fallback_to_closest = $params->{fallbackToClosest};
	if ( !defined ($fallback_to_closest) ) {
		$fallback_to_closest = $cachegroup->fallback_to_closest;
	}

	# note: this Perl API has been updated just to keep the Perl API tests happy until they can be removed.
	# Hence, this assumes lat/long are always passed (even though they're technically optional)
	my $lat = $params->{latitude};
	my $long = $params->{longitude};
	my $coordinate_values = {
		latitude  => $lat,
		longitude => $long
	};

	my $coordinate_rs = $coordinate->update($coordinate_values);

	my $values = {
		name                           => $params->{name},
		short_name                     => $params->{shortName},
		parent_cachegroup_id           => $params->{parentCachegroupId},
		fallback_to_closest            => $fallback_to_closest,
		secondary_parent_cachegroup_id => $params->{secondaryParentCachegroupId},
		type                           => $params->{typeId}
	};

	my $rs = $cachegroup->update($values);
	if ($rs && $coordinate_rs) {
		my %idnames;
		my $response;

		my $rs_idnames = $self->db->resultset("Cachegroup")->search( undef, { columns => [qw/id name/] } );
		while ( my $row = $rs_idnames->next ) {
			$idnames{ $row->id } = $row->name;
		}

		$response->{id}                 = $rs->id;
		$response->{name}               = $rs->name;
		$response->{shortName}          = $rs->short_name;
		$response->{latitude}           = 0.0 + $coordinate_rs->latitude;
		$response->{longitude}          = 0.0 + $coordinate_rs->longitude;
		$response->{lastUpdated}        = $rs->last_updated;
		$response->{parentCachegroupId} = $rs->parent_cachegroup_id;
		$response->{parentCachegroupName} =
			( defined $rs->parent_cachegroup_id )
			? $idnames{ $rs->parent_cachegroup_id }
			: undef;
		$response->{fallbackToClosest} = $rs->fallback_to_closest;
		$response->{secondaryParentCachegroupId} = $rs->secondary_parent_cachegroup_id;
		$response->{secondaryParentCachegroupName} =
			( defined $rs->secondary_parent_cachegroup_id )
			? $idnames{ $rs->secondary_parent_cachegroup_id }
			: undef;
		$response->{typeId}   = $rs->type->id;
		$response->{typeName} = $rs->type->name;

		&log( $self, "Updated Cachegroup named '" . $rs->name . "' with id: " . $rs->id, "APICHANGE" );

		return $self->success( $response, "Cachegroup update was successful." );
	}
	else {
		return $self->alert("Cachegroup update failed.");
	}

}

sub create {
	my $self   = shift;
	my $params = $self->req->json;

	my ( $is_valid, $result ) = $self->is_cachegroup_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $name = $params->{name};
	my $existing = $self->db->resultset('Cachegroup')->find( { name => $name } );
	if ($existing) {
		return $self->alert( "A cachegroup with name " . $name . " already exists." );
	}

	my $short_name = $params->{shortName};
	$existing = $self->db->resultset('Cachegroup')->find( { short_name => $short_name } );
	if ($existing) {
		return $self->alert( "A cachegroup with short_name " . $short_name . " already exists." );
	}

	my $coordinate_values = {
		name      => "from_cachegroup_" . $params->{name},
		latitude  => $params->{latitude},
		longitude => $params->{longitude},
	};

	my $coordinate = $self->db->resultset('Coordinate')->create($coordinate_values);
	my $coordinate_rs = $coordinate->insert();

	my $values = {
		name                           => $params->{name},
		short_name                     => $params->{shortName},
		coordinate                     => $coordinate_rs->id,
		parent_cachegroup_id           => $params->{parentCachegroupId},
		fallback_to_closest            => exists ($params->{fallbackToClosest}) ? $params->{fallbackToClosest} : 1,# defaults to true
		secondary_parent_cachegroup_id => $params->{secondaryParentCachegroupId},
		type                           => $params->{typeId}
	};

	my $insert = $self->db->resultset('Cachegroup')->create($values);
	my $rs = $insert->insert();
	if ($rs && $coordinate_rs) {
		my %idnames;
		my $response;

		my $rs_idnames = $self->db->resultset("Cachegroup")->search( undef, { columns => [qw/id name/] } );
		while ( my $row = $rs_idnames->next ) {
			$idnames{ $row->id } = $row->name;
		}

		$response->{id}                 = $rs->id;
		$response->{name}               = $rs->name;
		$response->{shortName}          = $rs->short_name;
		$response->{latitude}           = 0.0 + $coordinate_rs->latitude;
		$response->{longitude}          = 0.0 + $coordinate_rs->longitude;
		$response->{lastUpdated}        = $rs->last_updated;
		$response->{parentCachegroupId} = $rs->parent_cachegroup_id;
		$response->{parentCachegroupName} =
			( defined $rs->parent_cachegroup_id )
			? $idnames{ $rs->parent_cachegroup_id }
			: undef;
		$response->{fallbackToClosest} = $rs->fallback_to_closest;
		$response->{secondaryParentCachegroupId} = $rs->secondary_parent_cachegroup_id;
		$response->{secondaryParentCachegroupName} =
			( defined $rs->secondary_parent_cachegroup_id )
			? $idnames{ $rs->secondary_parent_cachegroup_id }
			: undef;
		$response->{typeId}   = $rs->type->id;
		$response->{typeName} = $rs->type->name;

		&log( $self, "Created Cachegroup named '" . $rs->name . "' with id: " . $rs->id, "APICHANGE" );

		return $self->success( $response, "Cachegroup creation was successful." );
	}
	else {
		return $self->alert("Cachegroup creation failed.");
	}

}

sub delete {
	my $self = shift;
	my $id     = $self->param('id');

	my $cg = $self->db->resultset('Cachegroup')->find( { id => $id } );
	if ( !defined($cg) ) {
		return $self->not_found();
	}

	my $servers = $self->db->resultset('Server')->find( { cachegroup => $cg->id } );
	if ( defined($servers) ) {
		return $self->alert("This cachegroup is currently used by servers.");
	}

	my $parent_cgs = $self->db->resultset('Cachegroup')->find( { parent_cachegroup_id => $cg->id } );
	if ( defined($parent_cgs) ) {
		return $self->alert("This cachegroup is currently used as a parent cachegroup.");
	}

	my $secondary_parent_cgs = $self->db->resultset('Cachegroup')->find( { secondary_parent_cachegroup_id => $cg->id } );
	if ( defined($secondary_parent_cgs) ) {
		return $self->alert("This cachegroup is currently used as a secondary parent cachegroup.");
	}

	my $asns = $self->db->resultset('Asn')->find( { cachegroup => $cg->id } );
	if ( defined($asns) ) {
		return $self->alert("This cachegroup is currently used by one or more ASNs.");
	}

	my $coordinate = $cg->coordinate;
	my $rs = $cg->delete();
	my $rs_coordinate = $coordinate->delete();
	if ($rs && $rs_coordinate) {
		&log( $self, "Deleted Cachegroup named '" . $cg->name . "' with id: " . $cg->id, "APICHANGE" );
		return $self->success_message("Cachegroup deleted.");
	} else {
		return $self->alert( "Cachegroup delete failed." );
	}
}


sub by_parameter_id {
	my $self    = shift;
	my $paramid = $self->param('paramid');

	my @data;
	my %dsids;
	my %in_use;

	# Get a list of all cachegroup id's associated with this param id
	my $rs_in_use = $self->db->resultset("CachegroupParameter")->search( { 'parameter' => $paramid } );
	while ( my $row = $rs_in_use->next ) {
		$in_use{ $row->cachegroup->id } = 1;
	}

	# Add remaining cachegroup ids to @data
	my $rs_links = $self->db->resultset("Cachegroup")->search( undef, { order_by => "name" } );
	while ( my $row = $rs_links->next ) {
		if ( !defined( $in_use{ $row->id } ) ) {
			push( @data, { "id" => $row->id, "name" => $row->name } );
		}
	}

	$self->success( { cachegroups => \@data } );
}

sub available_for_parameter {
	my $self = shift;
	my @data;
	my $paramid = $self->param('paramid');
	my %dsids;
	my %in_use;

	# Get a list of all profile id's associated with this param id
	my $rs_in_use = $self->db->resultset("CachegroupParameter")->search( { 'parameter' => $paramid } );
	while ( my $row = $rs_in_use->next ) {
		$in_use{ $row->cachegroup->id } = 1;
	}

	# Add remaining cachegroup ids to @data
	my $rs_links = $self->db->resultset("Cachegroup")->search( undef, { order_by => "name" } );
	while ( my $row = $rs_links->next ) {
		if ( !defined( $in_use{ $row->id } ) ) {
			push( @data, { "id" => $row->id, "name" => $row->name } );
		}
	}

	$self->success( \@data );
}

sub postupdatequeue {
	my $self   = shift;
	my $params = $self->req->json;
	my $name;
	my $id = $self->param('id');
	$name = $self->db->resultset('Cachegroup')->search( { id => $id } )->get_column('name')->single();

	if ( !defined($name) ) {
		return $self->alert( "cachegroup id[" . $id . "] does not exist." );
	}

	my $cdn = $params->{cdn};
	my $cdn_id = $params->{cdnId} // $self->db->resultset('Cdn')->search( { name => $cdn } )->get_column('id')->single();
	if ( !defined($cdn_id) ) {
		return $self->alert( "cdn " . $cdn . " does not exist." );
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

	my $servers = $self->db->resultset('Server')->search(
		{
			-and => [
				cachegroup	=> $id,
				cdn_id		=> $cdn_id
			]
		}
	);

	my $response;
	my @svrs = ();
	if ( $servers->count() > 0 ) {
		$servers->update( { upd_pending => $setqueue } );
		my @row = $servers->get_column('host_name')->all();
		foreach my $svr (@row) {
			push( @svrs, $svr );
		}
	}

	$response->{serverNames}    = \@svrs;
	$response->{action}         = ( $setqueue == 1 ) ? "queue" : "dequeue";
	$response->{cdn}            = $cdn;
	$response->{cachegroupName} = $name;
	$response->{cachegroupId}   = $id;

	my $msg = "Server updates $params->{action}d for $name cache group";
	&log( $self, $msg, "APICHANGE" );

	return $self->success($response);
}

sub is_cachegroup_valid {
	my $self   = shift;
	my $params = shift;

	if (!$self->is_valid_cachegroup_type($params->{typeId})) {
		return ( 0, "Invalid cachegroup type" );
	}

	my $rules = {
		fields => [ qw/name shortName latitude longitude parentCachegroupId secondaryParentCachegroupId typeId/ ],

		# Validation checks to perform
		checks => [
			name						=> [ is_required("is required"), \&is_alphanumeric, is_like( qr/^\S*$/, "must not contain spaces" ) ],
			shortName					=> [ is_required("is required"), \&is_alphanumeric, is_like( qr/^\S*$/, "must not contain spaces" ) ],
			latitude					=> [ \&is_valid_lat ],
			longitude					=> [ \&is_valid_long ],
			parentCachegroupId			=> [ \&is_int_or_undef ],
			secondaryParentCachegroupId => [ \&is_int_or_undef ],
			typeId						=> [ is_required("is required"), \&is_int_or_undef ],
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

sub is_alphanumeric {
	my ( $value, $params ) = @_;

	if ( !defined $value or $value eq '' ) {
		return undef;
	}

	if ( !( $value =~ /^[0-9a-zA-Z_\.\-]+$/ ) ) {
		return "invalid. Use alphanumeric . or _ .";
	}

	return undef;
}

sub is_int_or_undef {
	my ( $value, $params ) = @_;

	if ( !defined $value ) {
		return undef;
	}

	if ( !( $value =~ /^\d+$/ ) ) {
		return "invalid. Must be a positive integer or null.";
	}

	return undef;
}

sub is_valid_lat {
	my ( $value, $params ) = @_;

	if ( !defined $value ) {
		return undef;
	}

	if ( !( $value =~ /^[-]*[0-9]+[.]*[0-9]*/ ) ) {
		return "invalid. Must be a float number.";
	}

	if ( abs $value > 90 ) {
		return "invalid. May not exceed +- 90.0.";
	}

	return undef;
}

sub is_valid_long {
	my ( $value, $params ) = @_;

	if ( !defined $value ) {
		return undef;
	}

	if ( !( $value =~ /^[-]*[0-9]+[.]*[0-9]*/ ) ) {
		return "invalid. Must be a float number.";
	}

	if ( abs $value > 180 ) {
		return "invalid. May not exceed +- 180.0.";
	}

	return undef;
}

sub is_valid_cachegroup_type {
	my $self     = shift;
	my $type_id = shift;

	my $rs = $self->db->resultset("Type")->find( { id => $type_id } );
	if ( defined($rs) && ( $rs->use_in_table eq "cachegroup" ) ) {
		return 1;
	}
	return 0;
}

1;
