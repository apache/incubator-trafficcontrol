package API::PhysLocation;
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
use JSON;
use MojoPlugins::Response;
use Validate::Tiny ':all';

my $finfo = __FILE__ . ":";

sub index {
	my $self 		= shift;
	my $region		= $self->param('region');

	my %criteria;
	if ( defined $region ) {
		$criteria{'region.id'} = $region;
	}

	my @data;
	my $orderby = $self->param('orderby') || "name";
	my $rs_data = $self->db->resultset("PhysLocation")->search( \%criteria, { prefetch => ['region'], order_by => 'me.' . $orderby } );
	while ( my $row = $rs_data->next ) {

		next if $row->short_name eq 'UNDEF';

		push(
			@data, {
				"address"     => $row->address,
				"city"        => $row->city,
				"comments"    => $row->comments,
				"email"       => $row->email,
				"id"          => $row->id,
				"lastUpdated" => $row->last_updated,
				"name"        => $row->name,
				"phone"       => $row->phone,
				"poc"         => $row->poc,
				"region"      => $row->region->name,
				"regionId"    => $row->region->id,
				"shortName"   => $row->short_name,
				"state"       => $row->state,
				"zip"         => $row->zip
			}
		);
	}
	$self->success( \@data );
}

sub show {
	my $self = shift;
	my $id   = $self->param('id');

	my $rs_data = $self->db->resultset("PhysLocation")->search( { 'me.id' => $id }, { prefetch => ['region'] } );
	my @data = ();
	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"address"     => $row->address,
				"city"        => $row->city,
				"comments"    => $row->comments,
				"email"       => $row->email,
				"id"          => $row->id,
				"lastUpdated" => $row->last_updated,
				"name"        => $row->name,
				"phone"       => $row->phone,
				"poc"         => $row->poc,
				"region"      => $row->region->name,
				"regionId"    => $row->region->id,
				"shortName"   => $row->short_name,
				"state"       => $row->state,
				"zip"         => $row->zip
			}
		);
	}
	$self->success( \@data );
}

sub index_trimmed {
	my $self = shift;
	my @data;
	my $orderby = $self->param('orderby') || "name";
	my $rs_data = $self->db->resultset("PhysLocation")->search( undef, { order_by => 'me.' . $orderby } );
	while ( my $row = $rs_data->next ) {

		next if $row->short_name eq 'UNDEF';

		push(
			@data, {
				"name" => $row->name,
			}
		);
	}
	$self->success( \@data );
}

sub update {
	my $self   = shift;
	my $id     = $self->param('id');
	my $params = $self->req->json;

	my ( $is_valid, $result ) = $self->is_phys_location_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $phys_location = $self->db->resultset('PhysLocation')->find( { id => $id } );
	if ( !defined($phys_location) ) {
		return $self->not_found();
	}

	my $name = $params->{name};
	if ( $phys_location->name ne $name ) {
		my $existing = $self->db->resultset('PhysLocation')->find( { name => $name } );
		if ($existing) {
			return $self->alert( "A physical location with name " . $name . " already exists." );
		}
	}

	my $short_name = $params->{shortName};
	if ( $phys_location->short_name ne $short_name ) {
		my $existing = $self->db->resultset('PhysLocation')->find( { short_name => $short_name } );
		if ($existing) {
			return $self->alert( "A physical location with short_name " . $short_name . " already exists." );
		}
	}

	my $values = {
		address    => $params->{address},
		city       => $params->{city},
		comments   => $params->{comments},
		email      => $params->{email},
		name       => $name,
		phone      => $params->{phone},
		poc        => $params->{poc},
		region     => $params->{regionId},
		short_name => $short_name,
		state      => $params->{state},
		zip        => $params->{zip}
	};

	my $rs = $phys_location->update($values);
	if ($rs) {
		my $response;
		$response->{address}     = $rs->address;
		$response->{city}        = $rs->city;
		$response->{comments}    = $rs->comments;
		$response->{email}       = $rs->email;
		$response->{id}          = $rs->id;
		$response->{lastUpdated} = $rs->last_updated;
		$response->{name}        = $rs->name;
		$response->{phone}       = $rs->phone;
		$response->{poc}         = $rs->poc;
		$response->{region}      = $rs->region->name;
		$response->{regionId}    = $rs->region->id;
		$response->{shortName}   = $rs->short_name;
		$response->{state}       = $rs->state;
		$response->{zip}         = $rs->zip;

		&log( $self, "Updated Physical location name '" . $rs->name . "' for id: " . $rs->id, "APICHANGE" );

		return $self->success( $response, "Physical location update was successful." );
	}
	else {
		return $self->alert("Physical location update failed.");
	}

}

sub create {
	my $self        = shift;
	my $params      = $self->req->json;

	my ( $is_valid, $result ) = $self->is_phys_location_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $name = $params->{name};
	my $existing = $self->db->resultset('PhysLocation')->search( { name => $name } )->get_column('name')->single();
	if ( defined($existing) ) {
		return $self->alert( "Physical location [" . $params->{name} . "] already exists." );
	}
	my $short_name = $params->{shortName};
	$existing = $self->db->resultset('PhysLocation')->search( { short_name => $short_name } )->get_column('name')->single();
	if ( defined($existing) ) {
		return $self->alert( "Physical location with shortName [" . $params->{shortName} . "] already exists." );
	}

	my $values = {
		address    => $params->{address},
		city       => $params->{city},
		comments   => $params->{comments},
		email      => $params->{email},
		name       => $name,
		phone      => $params->{phone},
		poc        => $params->{poc},
		region     => $params->{regionId},
		short_name => $short_name,
		state      => $params->{state},
		zip        => $params->{zip}
	};

	my $insert = $self->db->resultset('PhysLocation')->create($values);
	my $rs = $insert->insert();
	if ($rs) {
		my $response;
		$response->{address}     = $rs->address;
		$response->{city}        = $rs->city;
		$response->{comments}    = $rs->comments;
		$response->{email}       = $rs->email;
		$response->{id}          = $rs->id;
		$response->{lastUpdated} = $rs->last_updated;
		$response->{name}        = $rs->name;
		$response->{phone}       = $rs->phone;
		$response->{poc}         = $rs->poc;
		$response->{region}      = $rs->region->name;
		$response->{regionId}    = $rs->region->id;
		$response->{shortName}   = $rs->short_name;
		$response->{state}       = $rs->state;
		$response->{zip}         = $rs->zip;

		&log( $self, "Created Phys location name '" . $rs->name . "' for id: " . $rs->id, "APICHANGE" );

		return $self->success( $response, "Phys location creation was successful." );
	} else {
		return $self->alert("Phys location creation failed.");
	}
}

sub create_for_region {
	my $self        = shift;
	my $region_name = $self->param('region_name');
	my $params      = $self->req->json;
	if ( !defined($params) ) {
		return $self->alert("parameters must be in JSON format,  please check!");
	}

	my $existing_physlocation = $self->db->resultset('PhysLocation')->search( { name => $params->{name} } )->get_column('name')->single();
	if ( defined($existing_physlocation) ) {
		return $self->alert( "physical location[" . $params->{name} . "] already exists." );
	}
	$existing_physlocation = $self->db->resultset('PhysLocation')->search( { name => $params->{shortName} } )->get_column('name')->single();
	if ( defined($existing_physlocation) ) {
		return $self->alert( "physical location with shortName[" . $params->{shortName} . "] already exists." );
	}
	my $region_id = $self->db->resultset('Region')->search( { name => $region_name } )->get_column('id')->single();
	if ( !defined($region_id) ) {
		return $self->alert( "region[" . $region_name . "] does not exist." );
	}

	my $insert = $self->db->resultset('PhysLocation')->create(
		{
			name       => $params->{name},
			short_name => $params->{shortName},
			region     => $region_id,
			address    => $self->undef_to_default( $params->{address}, "" ),
			city       => $self->undef_to_default( $params->{city}, "" ),
			state      => $self->undef_to_default( $params->{state}, "" ),
			zip        => $self->undef_to_default( $params->{zip}, "" ),
			phone      => $self->undef_to_default( $params->{phone}, "" ),
			poc        => $self->undef_to_default( $params->{poc}, "" ),
			email      => $self->undef_to_default( $params->{email}, "" ),
			comments   => $self->undef_to_default( $params->{comments}, "" ),
		}
	);
	$insert->insert();

	my $response;
	my $rs = $self->db->resultset('PhysLocation')->find( { id => $insert->id } );
	if ( defined($rs) ) {
		$response->{id}         = $rs->id;
		$response->{name}       = $rs->name;
		$response->{shortName}  = $rs->short_name;
		$response->{regionName} = $region_name;
		$response->{regionId}   = $rs->region->id;
		$response->{address}    = $rs->address;
		$response->{city}       = $rs->city;
		$response->{state}      = $rs->state;
		$response->{zip}        = $rs->zip;
		$response->{phone}      = $rs->phone;
		$response->{poc}        = $rs->poc;
		$response->{email}      = $rs->email;
		$response->{comments}   = $rs->comments;
		return $self->success($response);
	}
	return $self->alert( "create region " . $params->{name} . " failed." );
}

sub delete {
	my $self = shift;
	my $id     = $self->param('id');

	my $phys_location = $self->db->resultset('PhysLocation')->find( { id => $id } );
	if ( !defined($phys_location) ) {
		return $self->not_found();
	}

	my $rs = $phys_location->delete();
	if ($rs) {
		return $self->success_message("Physical location deleted.");
	} else {
		return $self->alert( "Physical delete failed." );
	}
}


sub is_phys_location_valid {
	my $self   = shift;
	my $params = shift;

	my $rules = {
		fields => [ qw/address city comments email name phone poc regionId shortName state zip/ ],

		# Validation checks to perform
		checks => [
			address		=> [ is_required("is required") ],
			city		=> [ is_required("is required") ],
			name		=> [ is_required("is required") ],
			regionId	=> [ is_required("is required"), is_like( qr/^\d+$/, "must be a positive integer" ) ],
			shortName	=> [ is_required("is required") ],
			state		=> [ is_required("is required") ],
			zip			=> [ is_required("is required") ],
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


sub undef_to_default {
	my $self    = shift;
	my $v       = shift;
	my $default = shift;

	return $v || $default;
}

1;
