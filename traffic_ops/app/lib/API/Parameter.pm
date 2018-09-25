package API::Parameter;
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
use UI::Parameter;
use Mojo::Base 'Mojolicious::Controller';
use Data::Dumper;
use POSIX qw(strftime);
use Time::Local;
use LWP;
use MojoPlugins::Response;
use MojoPlugins::Job;
use Utils::Helper::ResponseHelper;

sub index {
    my $self        = shift;
    my $name        = $self->param('name');
    my $config_file = $self->param('configFile');

    my %criteria;
    if ( defined $name ) {
        $criteria{'me.name'} = $name;
    }
    if ( defined $config_file ) {
        $criteria{'me.config_file'} = $config_file;
    }

    my $rs_data = $self->db->resultset("Parameter")->search(\%criteria);
    my @data = ();
	while ( my $row = $rs_data->next ) {
		my $value = $row->value;
		&UI::Parameter::conceal_secure_parameter_value( $self, $row->secure, \$value );
		push(
			@data, {
				"name"        => $row->name,
				"id"          => $row->id,
				"configFile"  => $row->config_file,
				"value"       => $value,
				"secure"      => \$row->secure,
				"lastUpdated" => $row->last_updated
			}
		);
	}
	$self->success( \@data );
}

sub show {
    my $self = shift;
    my $id     = $self->param('id');

    my $find = $self->db->resultset('Parameter')->find({ id => $id } );
    if ( !defined($find) ) {
        return $self->not_found("parameter [id:".$id."] does not exist.");
    }

    my @data = ();
    push(
        @data, {
            "id"          => $find->id,
            "name"        => $find->name,
            "configFile"  => $find->config_file,
            "value"       => $find->value,
            "secure"      => \$find->secure,
            "lastUpdated" => $find->last_updated
        }
    );
    $self->success( \@data );
}

sub get_profile_params {
	my $self         = shift;
	my $profile_id   = $self->param('id');
	my $profile_name = $self->param('name');

	my %criteria;
	if ( defined $profile_id ) {
		$criteria{'profile.id'} = $profile_id;
	} elsif ( defined $profile_name ) {
		$criteria{'profile.name'} = $profile_name;
	} else {
        return $self->alert("Profile ID or Name is required");
    }

	my $rs_data = $self->db->resultset("ProfileParameter")->search( \%criteria, { prefetch => [ 'parameter', 'profile' ] } );
	my @data = ();
	while ( my $row = $rs_data->next ) {
		my $value = $row->parameter->value;
		&UI::Parameter::conceal_secure_parameter_value( $self, $row->parameter->secure, \$value );
		push(
			@data, {
				"name"        => $row->parameter->name,
				"id"          => $row->parameter->id,
				"configFile"  => $row->parameter->config_file,
				"value"       => $value,
				"secure"      => \$row->parameter->secure,
				"lastUpdated" => $row->parameter->last_updated
			}
		);
	}
	$self->success( \@data );
}

sub get_profile_params_unassigned {
    my $self         = shift;
    my $profile_id   = $self->param('id');
    my $profile_name = $self->param('name');

    my %criteria;
    if ( defined $profile_id ) {
        $criteria{'profile.id'} = $profile_id;
    } elsif ( defined $profile_name ) {
        $criteria{'profile.name'} = $profile_name;
    } else {
        return $self->alert("Profile ID or Name is required");
    }

    my @assigned_params =
        $self->db->resultset('ProfileParameter')->search( \%criteria, { prefetch => [ 'parameter', 'profile' ] } )->get_column('parameter')->all();

    my $rs_data = $self->db->resultset("Parameter")->search( 'me.id' => { 'not in' => \@assigned_params } );
    my @data = ();
    while ( my $row = $rs_data->next ) {
        my $value = $row->value;
        &UI::Parameter::conceal_secure_parameter_value( $self, $row->secure, \$value );
        push(
            @data, {
                "name"        => $row->name,
                "id"          => $row->id,
                "configFile"  => $row->config_file,
                "value"       => $value,
                "secure"      => \$row->secure,
                "lastUpdated" => $row->last_updated
            }
        );
    }
    $self->success( \@data );
}

sub get_cachegroup_params {
	my $self         = shift;
	my $cg_id   = $self->param('id');

	my %criteria;
	if ( defined $cg_id ) {
		$criteria{'cachegroup.id'} = $cg_id;
	} else {
        return $self->alert("Cache Group ID is required");
    }

	my $rs_data = $self->db->resultset("CachegroupParameter")->search( \%criteria, { prefetch => [ 'cachegroup', 'parameter' ] } );
	my @data = ();
	while ( my $row = $rs_data->next ) {
		my $value = $row->parameter->value;
		&UI::Parameter::conceal_secure_parameter_value( $self, $row->parameter->secure, \$value );
		push(
			@data, {
				"name"        => $row->parameter->name,
				"id"          => $row->parameter->id,
				"configFile"  => $row->parameter->config_file,
				"value"       => $value,
				"secure"      => \$row->parameter->secure,
				"lastUpdated" => $row->parameter->last_updated
			}
		);
	}
	$self->success( \@data );
}

sub get_cachegroup_params_unassigned {
	my $self        = shift;
	my $cg_id       = $self->param('id');

	my %criteria;
	if ( defined $cg_id ) {
		$criteria{'cachegroup.id'} = $cg_id;
	} else {
        return $self->alert("Cache Group ID is required");
    }

    my @assigned_params =
        $self->db->resultset('CachegroupParameter')->search( \%criteria, { prefetch => [ 'parameter', 'cachegroup' ] } )->get_column('parameter')->all();

    my $rs_data = $self->db->resultset("Parameter")->search( 'me.id' => { 'not in' => \@assigned_params } );
    my @data = ();
    while ( my $row = $rs_data->next ) {
        my $value = $row->value;
        &UI::Parameter::conceal_secure_parameter_value( $self, $row->secure, \$value );
        push(
            @data, {
                "name"        => $row->name,
                "id"          => $row->id,
                "configFile"  => $row->config_file,
                "value"       => $value,
                "secure"      => \$row->secure,
                "lastUpdated" => $row->last_updated
            }
        );
    }
    $self->success( \@data );
}

sub create {
    my $self = shift;
    my $params = $self->req->json;

    if ( !defined($params) ) {
        return $self->alert("parameters must be in JSON format,  please check!");
    }

    if ( ref($params) ne 'ARRAY' ) {
        #not a array, create single parameter
        my @temparry;
        push(@temparry, $params);
        $params = \@temparry;
    }
    if ( scalar($params) == 0 ) {
        return $self->alert("parameters array length is 0.");
    }


    my @new_parameters = ();
    $self->db->txn_begin();
    my $param;
    foreach $param (@{ $params }) {
        if ( !defined($param->{name}) ) {
            $self->db->txn_rollback();
            return $self->alert("there is parameter name does not provide , configFile:".$param->{configFile}." , value:".$param->{value});
        }
        if ( !defined($param->{configFile}) ) {
            $self->db->txn_rollback();
            return $self->alert("there is parameter configFile does not provide , name:".$param->{name}." , value:".$param->{value});
        }
        if ( !defined($param->{value}) ) {
            $self->db->txn_rollback();
            return $self->alert("there is parameter value does not provide , name:".$param->{name}." , configFile:".$param->{configFile});
        }
        if ( !defined($param->{secure}) ) {
            $param->{secure} = 0
        } else {
            if (($param->{secure} ne '0') && ($param->{secure} ne '1')) {
                $self->db->txn_rollback();
                return $self->alert("secure must 0 or 1, parameter [name:".$param->{name}." , configFile:".$param->{configFile}." , value:".$param->{value}." , secure:".$param->{secure}."]");
            }
            $param->{secure} = 0 if ($param->{secure} eq '0' );
            $param->{secure} = 1 if ($param->{secure} eq '1' );
        }

        my $find = $self->db->resultset('Parameter')->find(
            {
                name            => $param->{name},
                config_file     => $param->{configFile},
                value           => $param->{value}
            }
        );
        if ( defined($find)) {
            $self->db->txn_rollback();
            return $self->alert("parameter [name:".$param->{name}." , configFile:".$param->{configFile}." , value:".$param->{value}."] already exists.");
        }

        my $insert = $self->db->resultset('Parameter')->create(
            {
                name            => $param->{name},
                config_file     => $param->{configFile},
                value           => $param->{value},
                secure          => $param->{secure}
            }
        );
        $insert->insert();
        push(@new_parameters, {
                'id'            => $insert->id,
                'name'          => $insert->name,
                'configFile'    => $insert->config_file,
                'value'         => $insert->value,
                'secure'        => $insert->secure
            })
    }
    $self->db->txn_commit();

    my $msg = scalar(@new_parameters) . " parameters created";
    &log( $self, $msg, "APICHANGE" );

    my $response  = \@new_parameters;
    return $self->success($response, $msg);
}

sub update {
    my $self = shift;
    my $id     = $self->param('id');
    my $params = $self->req->json;

    if ( !defined($params) ) {
        return $self->alert("parameters must be in JSON format,  please check!");
    }

    my $find = $self->db->resultset('Parameter')->find({ id => $id } );
    if ( !defined($find) ) {
        return $self->not_found("parameter [id:".$id."] does not exist.");
    }

    my $name = $params->{name} || $find->name;
    my $configFile = $params->{configFile} || $find->config_file;
    my $value = exists($params->{value}) ?  $params->{value} : $find->value;
    my $secure = $find->secure;
    if ( defined($params->{secure}) ) {
         $secure = $params->{secure};
    }

    $find->update(
        {
            name        => $name,
            config_file => $configFile,
            value       => $value,
            secure      => $secure
        }
    );

    my $msg = "Parameter [ $name ] updated";
    &log( $self, $msg, "APICHANGE" );

    my $response;
    $response->{id}     = $find->id;
    $response->{name}   = $find->name;
    $response->{configFile} = $find->config_file;
    $response->{value}  = $find->value;
    $response->{secure} = $find->secure;

    return $self->success($response, $msg);
}

sub delete {
    my $self = shift;
    my $id     = $self->param('id');
    my $params = $self->req->json;

    my $find = $self->db->resultset('Parameter')->find({ id => $id } );
	$self->app->log->debug("defined find #-> " . defined($find));
    if ( !defined($find) ) {
        return $self->not_found("parameter [id:".$id."] does not exist.");
    }

    my $find_profile = $self->db->resultset('ProfileParameter')->find( { parameter => $id } );
    if ( defined($find_profile) ) {
        return $self->alert("fail to delete parameter, parameter [id:".$id."] has profile associated.");
    }
 
    $find->delete();

    my $msg = "Parameter [ " . $find->name . " ] deleted";
    &log( $self, $msg, "APICHANGE" );

    return $self->success_message($msg);
}

sub validate {
    my $self = shift;
    my $params = $self->req->json;

    if ( !defined($params) ) {
        return $self->alert("parameters must be in JSON format,  please check!");
    }

    if ( !defined($params->{name}) ) {
        return $self->alert("Parameter name is required.");
    }
    if ( !defined($params->{configFile}) ) {
        return $self->alert("Parameter configFile is required.");
    }
    if ( !defined($params->{value}) ) {
        return $self->alert("Parameter value is required.");
    }

    my $find = $self->db->resultset('Parameter')->find({ 
            name => $params->{name},
            config_file => $params->{configFile},
            value => $params->{value},
        } ) ;
    if ( !defined($find) ) {
        return $self->alert("parameter [name:".$params->{name}.", config_file:".$params->{configFile}.", value:".$params->{value}."] does not exist.");
    }

    my $response;
    $response->{id}     = $find->id;
    $response->{name}   = $find->name;
    $response->{configFile} = $find->config_file;
    $response->{value}  = $find->value;
    $response->{secure} = $find->secure;

    return $self->success($response, "Parameter exists.");
}

1;
