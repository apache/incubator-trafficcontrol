package API::Region;
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
	my $self 			= shift;
	my $division_id		= $self->param('division');

	my %criteria;
	if ( defined $division_id ) {
		$criteria{'division'} = $division_id;
	}

	my @data;
	my $orderby = $self->param('orderby') || "name";
	my $rs_data = $self->db->resultset("Region")->search( \%criteria, { prefetch => ['division'], order_by => 'me.' . $orderby } );
	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"id"           => $row->id,
				"name"         => $row->name,
				"division"     => $row->division->id,
				"divisionName" => $row->division->name
			}
		);
	}
	$self->success( \@data );
}

sub index_by_name {
	my $self = shift;
	my $name   = $self->param('name');

	my $rs_data = $self->db->resultset("Region")->search( { 'me.name' => $name }, { prefetch => ['division'] } );
	my @data = ();
	while ( my $row = $rs_data->next ) {
		my $division = { "id"     => $row->division->id,
			"name"   => $row->division->name
		};
		push(
			@data, {
				"id"           => $row->id,
				"name"         => $row->name,
				"division"     => $division,
			}
		);
	}
	$self->success( \@data );
}

sub show {
	my $self = shift;
	my $id   = $self->param('id');

	my $rs_data = $self->db->resultset("Region")->search( { 'me.id' => $id }, { prefetch => ['division'] } );
	my @data = ();
	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"id"           => $row->id,
				"name"         => $row->name,
				"division"     => $row->division->id,
				"divisionName" => $row->division->name
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

	my $region = $self->db->resultset('Region')->find( { id => $id } );
	if ( !defined($region) ) {
		return $self->not_found();
	}

	my ( $is_valid, $result ) = $self->is_region_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $values = {
		name     => $params->{name},
		division => $params->{division}
	};

	my $rs = $region->update($values);
	if ($rs) {
		my $response;
		$response->{id}          = $rs->id;
		$response->{name}        = $rs->name;
		$response->{division}    = $rs->division->id;
		$response->{divisionName}= $rs->division->name;
		$response->{lastUpdated} = $rs->last_updated;
		&log( $self, "Updated Region name '" . $rs->name . "' for id: " . $rs->id, "APICHANGE" );
		return $self->success( $response, "Region update was successful." );
	}
	else {
		return $self->alert("Region update failed.");
	}

}

sub create {
	my $self   = shift;
	my $params = $self->req->json;

	if ( !&is_oper($self) ) {
		return $self->forbidden();
	}

	my ( $is_valid, $result ) = $self->is_region_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $existing = $self->db->resultset('Region')->search( { name => $params->{name} } )->get_column('name')->single();
	if ($existing) {
		return $self->alert("A region with name " . $params->{name} . " already exists.");
	}

	my $values = {
		name 		=> $params->{name} ,
		division 	=> $params->{division}
	};

	my $insert = $self->db->resultset('Region')->create($values);
	my $rs = $insert->insert();
	if ($rs) {
		my $response;
		$response->{id}          	= $rs->id;
		$response->{name}        	= $rs->name;
		$response->{division}       = $rs->division->id;
		$response->{divisionName}   = $rs->division->name;
		$response->{lastUpdated} 	= $rs->last_updated;

		&log( $self, "Created Region name '" . $rs->name . "' for id: " . $rs->id, "APICHANGE" );

		return $self->success( $response, "Region create was successful." );
	}
	else {
		return $self->alert("Region create failed.");
	}

}

sub create_for_division {
	my $self          = shift;
	my $division_name = $self->param('division_name');
	my $params        = $self->req->json;
	if ( !defined($params) ) {
		return $self->alert("parameters must be in JSON format,  please check!");
	}

	my $existing_region = $self->db->resultset('Region')->search( { name => $params->{name} } )->get_column('name')->single();
	if ( defined($existing_region) ) {
		return $self->alert( "region[" . $params->{name} . "] already exists." );
	}

	my $divsion_id = $self->db->resultset('Division')->search( { name => $division_name } )->get_column('id')->single();
	if ( !defined($divsion_id) ) {
		return $self->alert( "division[" . $division_name . "] does not exist." );
	}

	my $insert = $self->db->resultset('Region')->create(
		{
			name     => $params->{name},
			division => $divsion_id
		}
	);
	$insert->insert();

	my $response;
	my $rs = $self->db->resultset('Region')->find( { id => $insert->id } );
	if ( defined($rs) ) {
		$response->{id}           = $rs->id;
		$response->{name}         = $rs->name;
		$response->{divisionName} = $division_name;
		$response->{divsionId}    = $rs->division->id;
		return $self->success($response);
	}
	return $self->alert( "create region " . $params->{name} . " failed." );
}

sub delete {
	my $self = shift;
	my $id     = $self->param('id');

	if ( !&is_oper($self) ) {
		return $self->forbidden();
	}

	my $region = $self->db->resultset('Region')->find( { id => $id } );
	if ( !defined($region) ) {
		return $self->not_found();
	}

	my $rs = $region->delete();
	if ($rs) {
		return $self->success_message("Region deleted.");
	} else {
		return $self->alert( "Region delete failed." );
	}
}

sub delete_by_name {
	my $self = shift;
	my $name     = $self->param('name');

	if ( !&is_oper($self) ) {
		return $self->forbidden();
	}

	my $region = $self->db->resultset('Region')->find( { name => $name } );
	if ( !defined($region) ) {
		return $self->not_found();
	}

	my $rs = $region->delete();
	if ($rs) {
		return $self->success_message("Region deleted.");
	} else {
		return $self->alert( "Region delete failed." );
	}
}

sub is_region_valid {
	my $self   	= shift;
	my $params 	= shift;

	my $rules = {
		fields => [
			qw/name division/
		],

		# Validation checks to perform
		checks => [
			name		=> [ is_required("is required") ],
			division	=> [ is_required("is required"), is_like( qr/^\d+$/, "must be a positive integer" ) ],
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
