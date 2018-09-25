package API::Profile;
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
use UI::Parameter;
use Mojo::Base 'Mojolicious::Controller';
use Data::Dumper;
use JSON;
use Validate::Tiny ':all';

sub index {
	my $self = shift;
	my $parameter_id = $self->param('param');
	my $cdn_id = $self->param('cdn');

	my @data;
	my %criteria;

	if ( defined $parameter_id ) {
		my $rs = $self->db->resultset('ProfileParameter')->search( { parameter => $parameter_id },  { prefetch => [ 'profile' ], order_by => 'profile.name' }  );
		while ( my $row = $rs->next ) {
			push(
				@data, {
					"id" 			=> $row->profile->id,
					"name" 			=> $row->profile->name,
					"description" 	=> $row->profile->description,
					"cdn" 			=> defined($row->profile->cdn) ? $row->profile->cdn->id : undef,
					"cdnName" 		=> defined($row->profile->cdn) ? $row->profile->cdn->name : undef,
					"type" 			=> $row->profile->type,
					"routingDisabled"	=> \$row->profile->routing_disabled,
					"lastUpdated" 	=> $row->profile->last_updated
				}
			);
		}
	} else {
		if ( defined $cdn_id ) {
			$criteria{'cdn'} = $cdn_id;
		}
		my $rs_data = $self->db->resultset("Profile")->search( \%criteria, { prefetch => [ 'cdn' ], order_by => 'me.name' } );
		while ( my $row = $rs_data->next ) {
			push(
				@data, {
					"id"          => $row->id,
					"name"        => $row->name,
					"description" => $row->description,
					"cdn"         => defined($row->cdn) ? $row->cdn->id : undef,
					"cdnName"     => defined($row->cdn) ? $row->cdn->name : undef,
					"type"        => $row->type,
					"routingDisabled"	=> \$row->routing_disabled,
					"lastUpdated" => $row->last_updated
				}
			);
		}
	}
	$self->success( \@data );
}

sub index_trimmed {
	my $self = shift;
	my @data;
	my $orderby = $self->param('orderby') || "name";
	my $rs_data = $self->db->resultset("Profile")->search( undef, { order_by => $orderby } );
	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"name" => $row->name,
			}
		);
	}
	$self->render( json => \@data );
}

sub get_profiles_by_paramId {
	my $self    	= shift;
	my $param_id	= $self->param('id');

	my $param_profiles = $self->db->resultset('ProfileParameter')->search( { parameter => $param_id }, { prefetch => [ 'profile' ] } );

	my @data;
	while ( my $row = $param_profiles->next ) {
		push(
			@data, {
				"id"				=> $row->profile->id,
				"name"				=> $row->profile->name,
				"description"		=> $row->profile->description,
				"type"				=> $row->profile->type,
				"routingDisabled"	=> $row->profile->routing_disabled,
				"lastUpdated"		=> $row->profile->last_updated
			}
		);
	}

	return $self->success( \@data );
}

sub get_unassigned_profiles_by_paramId {
	my $self    	= shift;
	my $param_id	= $self->param('id');

	my %criteria;
	if ( defined $param_id ) {
		$criteria{'parameter.id'} = $param_id;
	} else {
		return $self->alert("Parameter ID is required");
	}

	my @assigned_profiles =
		$self->db->resultset('ProfileParameter')->search( \%criteria, { prefetch => [ 'parameter', 'profile' ] } )->get_column('profile')->all();

	my $rs_data = $self->db->resultset("Profile")->search( { 'me.id' => { 'not in' => \@assigned_profiles } }, { prefetch => [ 'cdn' ] } );
	my @data = ();
	while ( my $row = $rs_data->next ) {
		push(
			@data, {
				"id"          => $row->id,
				"name"        => $row->name,
				"description" => $row->description,
				"cdn"         => defined($row->cdn) ? $row->cdn->id : undef,
				"cdnName"     => defined($row->cdn) ? $row->cdn->name : undef,
				"type"        => $row->type,
				"routingDisabled"	=> \$row->routing_disabled,
				"lastUpdated" => $row->last_updated
			}
		);
	}
	$self->success( \@data );
}

sub show {
	my $self 			= shift;
	my $id   			= $self->param('id');
	my $include_params	= $self->param('includeParams') ? 1 : 0;
	my @params 			= ();

	my $profile = $self->db->resultset("Profile")->search( { 'me.id' => $id }, { prefetch => [ 'cdn' ] } );

	if ($include_params) {
		my %criteria;
		$criteria{'profile.id'} = $id;

		my $rs_profile_params = $self->db->resultset("ProfileParameter")->search( \%criteria, { prefetch => [ 'parameter', 'profile' ], order_by => 'parameter.name, parameter.config_file, parameter.value' } );

		while ( my $pp = $rs_profile_params->next ) {
			my $value = $pp->parameter->value;
			&UI::Parameter::conceal_secure_parameter_value( $self, $pp->parameter->secure, \$value );
			push(
				@params, {
					"name"        => $pp->parameter->name,
					"configFile"  => $pp->parameter->config_file,
					"value"       => $value,
				}
			);
		}
	}

	my @profiles = ();
	while ( my $row = $profile->next ) {
		my $profile = {
			"id"          => $row->id,
			"name"        => $row->name,
			"description" => $row->description,
			"cdn"         => defined($row->cdn) ? $row->cdn->id : undef,
			"cdnName"     => defined($row->cdn) ? $row->cdn->name : undef,
			"type"        => $row->type,
			"routingDisabled"	=> \$row->routing_disabled,
			"lastUpdated" => $row->last_updated
		};
		if ($include_params) {
			$profile->{params} = \@params;
		}
		push(@profiles, $profile);
	}
	$self->success( \@profiles );
}

sub create {
	my $self   = shift;
	my $params = $self->req->json;
	if ( !defined($params) ) {
		return $self->alert("parameters must be in JSON format,  please check!");
	}

	if ( !&is_oper($self) ) {
		return $self->alert( { Error => " - You must be an admin or oper to perform this operation!" } );
	}

	my ( $is_valid, $result ) = $self->is_profile_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $name = $params->{name};
	my $description = $params->{description};

	my $existing_profile = $self->db->resultset('Profile')->search( { name => $name } )->get_column('name')->single();
	if ( $existing_profile && $name eq $existing_profile ) {
		return $self->alert("profile with name $name already exists.");
	}

	my $existing_desc = $self->db->resultset('Profile')->find( { description => $description } );
	if ($existing_desc) {
		return $self->alert("a profile with the exact same description already exists.");
	}
	
	my $cdn = $params->{cdn};
	my $type = $params->{type};
	my $routing_disabled = defined($params->{routingDisabled}) ? $params->{routingDisabled} : 0;
	# Boolean values don't always show properly, so we're going to evaluate these then convert them to standard integers.
	# This allows the response output to always show true/false correctly.
	if ($routing_disabled == 1) {
		$routing_disabled = 1;
	}
	else { 
		$routing_disabled = 0;
	}
	my $insert = $self->db->resultset('Profile')->create(
		{
			name        => $name,
			description => $description,
			cdn         => $cdn,
			type        => $type,
			routing_disabled => $routing_disabled,
		}
	);
	$insert->insert();
	my $new_id = $insert->id;

	&log( $self, "Created profile with id: " . $new_id . " and name: " . $name, "APICHANGE" );

	my $response;
	$response->{id}          = $new_id;
	$response->{name}        = $name;
	$response->{description} = $description;
	$response->{cdn}         = $cdn;
	$response->{type}        = $type;
	$response->{routingDisabled} = \$routing_disabled;
	return $self->success($response);
}

sub copy {
	my $self = shift;

	my $name                   = $self->param('profile_name');
	my $profile_copy_from_name = $self->param('profile_copy_from');
	if ( !defined($name) || $name eq "" || $name =~ /\s/ ) {
		return $self->alert("profile 'name' is required and cannot contain spaces.");
	}
	if ( defined($profile_copy_from_name) and ( $profile_copy_from_name eq "" ) ) {
		return $self->alert("profile name 'profile_copy_from' can't be null.");
	}

	my $existing_profile = $self->db->resultset('Profile')->search( { name => $name } )->get_column('name')->single();
	if ( $existing_profile && $name eq $existing_profile ) {
		return $self->alert("profile with name $name already exists.");
	}

	my $rs = $self->db->resultset('Profile')->search( { name => $profile_copy_from_name } );
	my $row1 = $rs->next;
	if ( !$row1 ) {
		return $self->alert("profile_copy_from $profile_copy_from_name doesn't exist.");
	}
	my $profile_copy_from_id = $row1->id;
	my $description          = $row1->description;

	my $cdn = $row1->cdn;
	my $type = $row1->type;
	my $insert = $self->db->resultset('Profile')->create(
		{
			name        => $name,
			description => $description,
			cdn         => $cdn,
			type        => $type,
		}
	);
	$insert->insert();
	my $new_id = $insert->id;

	if ( defined($profile_copy_from_name) ) {
		my $rs_param =
			$self->db->resultset('ProfileParameter')
			->search( { profile => $profile_copy_from_id }, { prefetch => [ { profile => undef }, { parameter => undef } ] } );
		while ( my $row = $rs_param->next ) {
			my $insert = $self->db->resultset('ProfileParameter')->create(
				{
					profile   => $new_id,
					parameter => $row->parameter->id,
				}
			);
			$insert->insert();
		}
	}
	my $msg = "Created new profile [ $name ] from existing profile [ $profile_copy_from_name ]";

	&log( $self, $msg, "APICHANGE" );

	my $response;
	$response->{id}              = $new_id;
	$response->{name}            = $name;
	$response->{description}     = $description;
	$response->{profileCopyFrom} = $profile_copy_from_name;
	$response->{idCopyFrom}      = $profile_copy_from_id;
	return $self->success($response, $msg);
}

sub update {
	my $self   = shift;
	my $id     = $self->param('id');
	my $params = $self->req->json;

	if ( !&is_oper($self) ) {
		return $self->forbidden();
	}

	my $profile = $self->db->resultset('Profile')->find( { id => $id } );
	if ( !defined($profile) ) {
		return $self->not_found();
	}

	my ( $is_valid, $result ) = $self->is_profile_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $name = $params->{name};
	if ( $profile->name ne $name ) {
		my $existing = $self->db->resultset('Profile')->find( { name => $name } );
		if ($existing) {
			return $self->alert( "a profile with name " . $name . " already exists." );
		}
	}

	my $description = $params->{description};
	if ( $profile->description ne $description ) {
		my $existing = $self->db->resultset('Profile')->find( { description => $description } );
		if ($existing) {
			return $self->alert("a profile with the exact same description already exists.");
		}
	}

	my $routing_disabled = defined($params->{routingDisabled}) ? $params->{routingDisabled} : 0;
	# Boolean values don't always show properly, so we're going to evaluate these then convert them to standard integers.
	# This allows the response output to always show true/false correctly.
	if ($routing_disabled == 1) {
		$routing_disabled = 1;
	}
	else { 
		$routing_disabled = 0;
	}

	my $cdn = $params->{cdn};

	my $ex_server = $profile->servers->first;
	if ( defined $ex_server ) {
		if ( $cdn != $ex_server->cdn_id ) {
			return $self->alert("the assigned CDN does not match the CDN assigned to servers with this profile.");
		}
	}

	my $type = $params->{type};
	my $values = {
		name        => $name,
		description => $description,
		cdn         => $cdn,
		type        => $type,
		routing_disabled => $routing_disabled,
	};

	my $rs = $profile->update($values);
	if ($rs) {
		my $response;
		$response->{id}          = $id;
		$response->{name}        = $name;
		$response->{description} = $description;
		$response->{cdn}         = $cdn;
		$response->{type}        = $type;
		$response->{routingDisabled} = \$routing_disabled;
		&log( $self, "Update profile with id: " . $id . " and name: " . $name, "APICHANGE" );
		return $self->success( $response, "Profile was updated: " . $id );
	}
	else {
		return $self->alert("Profile update failed.");
	}
}

sub delete {
	my $self = shift;
	my $id   = $self->param('id');

	if ( !&is_oper($self) ) {
		return $self->forbidden();
	}

	my $profile = $self->db->resultset('Profile')->find( { id => $id } );
	if ( !defined($profile) ) {
		return $self->not_found();
	}

	if ($profile->name eq 'GLOBAL') {
		return $self->alert("Cannot delete the GLOBAL profile.");
	}

	my $server = $self->db->resultset('Server')->find( { profile => $profile->id } );
	if ( defined($server) ) {
		return $self->alert("the profile is used by some server(s).");
	}
	my $ds = $self->db->resultset('Deliveryservice')->find( { profile => $profile->id } );
	if ( defined($ds) ) {
		return $self->alert("the profile is used by some deliveryservice(s).");
	}

	my $profile_name = $profile->name;
	$profile->delete();

	&log( $self, "Delete profile with id: " . $id . " and name: " . $profile_name, "APICHANGE" );

	return $self->success_message("Profile was deleted.");
}

sub export {
	my $self	= shift;
	my $id		= $self->param('id');
	my $export	= {};

	my $rs = $self->db->resultset('ProfileParameter')->search( { profile => $id }, { prefetch => [ { parameter => undef }, { profile => undef } ] } );

	my $i = 0;
	while ( my $row = $rs->next ) {
		if ( !defined($export->{profile}) ) {
			$export->{profile}->{name}        	= $row->profile->name;
			$export->{profile}->{description} 	= $row->profile->description;
			$export->{profile}->{type}        	= $row->profile->type;
			$export->{profile}->{cdn}        	= defined($row->profile->cdn) ? $row->profile->cdn->name : undef,
		}
		$export->{parameters}->[$i] = {
			name        => $row->parameter->name,
			config_file => $row->parameter->config_file,
			value       => $row->parameter->value
		};
		$i++;
	}
	$self->render( json => $export );
}

sub import {
	my $self             = shift;
	my $new_id           = -1;
	my $data 			= $self->req->json;
	my $p_name           = $data->{profile}->{name};
	my $p_desc           = $data->{profile}->{description};
	my $p_type           = $data->{profile}->{type};
	my $p_cdn_id         = $self->db->resultset('Cdn')->search( { name => $data->{profile}->{cdn} } )->get_column('id')->single();
	my $existing_profile = $self->db->resultset('Profile')->search( { name => $p_name } )->get_column('name')->single();
	my @valid_types      = @{$self->db->source('ProfileTypeValue')->column_info('value')->{extra}->{list}};

	if (!defined($p_cdn_id)) {
		return $self->alert($data->{profile}->{cdn} . " CDN does not exist");
	}

	if ($existing_profile) {
		return $self->alert("A profile with the name \"$p_name\" already exists");
	}

	if (! grep(/^$p_type$/, @valid_types )) {
		my $vtypes = join(', ', @valid_types);
		return $self->alert("Profile contains type \"$p_type\" which is not a valid profile type. Valid types are: $vtypes");
	}

	my $insert = $self->db->resultset('Profile')->create(
		{
			name        => $p_name,
			description => $p_desc,
			type        => $p_type,
			cdn         => $p_cdn_id,
		}
	);
	$insert->insert();
	$new_id = $insert->id;

	my $new_count      = 0;
	my $existing_count = 0;
	my %done;
	foreach my $param ( @{ $data->{parameters} } ) {
		my $param_name        = $param->{name};
		my $param_config_file = $param->{config_file};
		my $param_value       = $param->{value};
		my $param_id =
			$self->db->resultset('Parameter')
				->search( { -and => [ name => $param_name, value => $param_value, config_file => $param_config_file ] }, { rows => 1 } )->get_column('id')
				->single();
		if ( !defined($param_id) ) {
			my $insert = $self->db->resultset('Parameter')->create(
				{
					name        => $param_name,
					config_file => $param_config_file,
					value       => $param_value,
				}
			);
			$insert->insert();
			$param_id = $insert->id();
			$new_count++;
		}
		else {
			next if defined( $done{$param_id} ); # just in case duplicate parameters were sent
			$existing_count++;
		}

		my $link_insert = $self->db->resultset('ProfileParameter')->create(
			{
				parameter => $param_id,
				profile   => $new_id,
			}
		);
		$link_insert->insert();
		$done{$param_id} = $new_id;
	}

	my $response;
	$response->{id}            	= $insert->id;
	$response->{name}          	= $insert->name;
	$response->{description}    = $insert->description;
	$response->{type} 			= $insert->type;
	$response->{cdn} 			= $insert->cdn->name;

	my $msg = "Profile imported [ " . $p_name . " ] with " . $new_count . " new and " . $existing_count . " existing parameters";
	&log( $self, $msg, "APICHANGE" );

	return $self->success( $response, $msg );
}

sub availableprofile {
	my $self = shift;
	my @data;
	my $paramid = $self->param('paramid');
	my %dsids;
	my %in_use;

	# Get a list of all profile id's associated with this param id
	my $rs_in_use = $self->db->resultset("ProfileParameter")->search( { 'parameter' => $paramid } );
	while ( my $row = $rs_in_use->next ) {
		$in_use{ $row->profile->id } = undef;
	}

	# Add remaining profile ids to @data
	my $rs_links = $self->db->resultset("Profile")->search( undef, { order_by => "description" } );
	while ( my $row = $rs_links->next ) {
		if ( !exists( $in_use{ $row->id } ) ) {
			push( @data, { "id" => $row->id, "description" => $row->description } );
		}
	}

	$self->success( \@data );
}

sub is_profile_valid {
	my $self   	= shift;
	my $params 	= shift;

	my $rules = {
		fields => [
			qw/name description cdn type routingDisabled/
		],

		# Validation checks to perform
		checks => [
			name			=> [ is_required("is required"), is_like( qr/^\S*$/, "must not contain spaces" ) ],
			description		=> [ is_required("is required") ],
			cdn				=> [ is_required("is required"), is_like( qr/^\d+$/, "must be a positive integer") ],
			type			=> [ is_required("is required") ],
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
