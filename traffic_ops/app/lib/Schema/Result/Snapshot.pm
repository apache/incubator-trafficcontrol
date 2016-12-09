use utf8;
package Schema::Result::Snapshot;

# Created by DBIx::Class::Schema::Loader
# DO NOT MODIFY THE FIRST PART OF THIS FILE

=head1 NAME

Schema::Result::Snapshot

=cut

use strict;
use warnings;

use base 'DBIx::Class::Core';

=head1 TABLE: C<snapshot>

=cut

__PACKAGE__->table("snapshot");

=head1 ACCESSORS

=head2 cdn

  data_type: 'text'
  is_foreign_key: 1
  is_nullable: 0

=head2 content

  data_type: 'json'
  is_nullable: 0

=head2 last_updated

  data_type: 'timestamp with time zone'
  default_value: current_timestamp
  is_nullable: 1
  original: {default_value => \"now()"}

=cut

__PACKAGE__->add_columns(
  "cdn",
  { data_type => "text", is_foreign_key => 1, is_nullable => 0 },
  "content",
  { data_type => "json", is_nullable => 0 },
  "last_updated",
  {
    data_type     => "timestamp with time zone",
    default_value => \"current_timestamp",
    is_nullable   => 1,
    original      => { default_value => \"now()" },
  },
);

=head1 PRIMARY KEY

=over 4

=item * L</cdn>

=back

=cut

__PACKAGE__->set_primary_key("cdn");

=head1 RELATIONS

=head2 cdn

Type: belongs_to

Related object: L<Schema::Result::Cdn>

=cut

__PACKAGE__->belongs_to(
  "cdn",
  "Schema::Result::Cdn",
  { name => "cdn" },
  { is_deferrable => 0, on_delete => "NO ACTION", on_update => "CASCADE" },
);


# Created by DBIx::Class::Schema::Loader v0.07042 @ 2016-12-09 12:04:19
# DO NOT MODIFY THIS OR ANYTHING ABOVE! md5sum:WQw7mw5c5UgACYTxspyMeg


# You can replace this text with custom code or comments, and it will be preserved on regeneration
1;
