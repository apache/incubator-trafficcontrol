package API::FederationFederationResolver;
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

sub index {
    my $self    = shift;
    my $fed_id  = $self->param('fedId');

    my @data;
    my $rs_data = $self->db->resultset("FederationFederationResolver")->search( { 'federation' => $fed_id }, { prefetch => [ 'federation_resolver' ], order_by => 'federation_resolver.ip_address' } );
    while ( my $row = $rs_data->next ) {
        push(
            @data, {
                "id"        => $row->federation_resolver->id,
                "ipAddress" => $row->federation_resolver->ip_address,
                "type"      => $row->federation_resolver->type->name,
            }
        );
    }
    $self->success( \@data );
}

sub assign_fed_resolver_to_federation {
    my $self                = shift;
    my $fed_id              = $self->param('fedId');
    my $params              = $self->req->json;
    my $fed_resolver_ids    = $params->{fedResolverIds};
    my $replace             = $params->{replace};
    my $count               = 0;

    my $fed = $self->db->resultset('Federation')->find( { id => $fed_id } );
    if ( !defined($fed) ) {
        return $self->not_found();
    }

    if ( ref($fed_resolver_ids) ne 'ARRAY' ) {
        return $self->alert("Fed Resolver IDs must be an array");
    }

    if ( $replace ) {
        # start fresh and delete existing fed/fed resolver associations
        my $delete = $self->db->resultset('FederationFederationResolver')->search( { federation => $fed_id } );
        $delete->delete();
    }

    my @values = ( [ qw( federation federation_resolver ) ]); # column names are required for 'populate' function

    foreach my $fed_resolver_id (@{ $fed_resolver_ids }) {
        push(@values, [ $fed_id, $fed_resolver_id ]);
        $count++;
    }

    $self->db->resultset("FederationFederationResolver")->populate(\@values);

    my $msg = $count . " resolver(s) were assigned to the " . $fed->cname . " federation";
    &log( $self, $msg, "APICHANGE" );

    my $response = $params;
    return $self->success($response, $msg);
}

1;
