package iso

import (
	"fmt"
	"testing"
)

func TestCrypt(t *testing.T) {
	// The crypt function is expected to behave like Perl's builtin `crypt` function.
	// The following expected values were generated using the following Perl script
	// run on the Perl Traffic Ops CiaB Docker container.
	/*
		#! /usr/bin/perl

		my ($pw, $salt) = @ARGV;

		if ((not defined $pw) || (not defined $salt)) {
			die("usage: $0 <password> <salt>\n");
		}

		print crypt("$pw", "\$1\$$salt\$");
		print "\n";
	*/

	cases := []struct {
		password, salt string
		expected       string
	}{
		{"password", "salt", "$1$salt$qJH7.N4xYta3aEG/dfqo/0"},
		{"Traffic Ops", "pepper", "$1$pepper$AHauHHBeRPuBP0LCO0oBW0"},
		{"T0p S3cr3T", "N4xYta3a", "$1$N4xYta3a$E4g3CFzttHfxgEvY4PmrI/"},
		{"a", "b", "$1$b$J4vSIPg.1IiJxJ.JOHsOS1"},
		{"", "salt", "$1$salt$UsdFqFVB.FsuinRDK5eE.."},
		{"pw", "", "$1$$F0Fc2lbYpzr3KKdKkM0Wj."},
		{"", "", "$1$$qRPK7m23GJusamGpoGLby/"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("", func(t *testing.T) {
			got, err := crypt(tc.password, tc.salt)
			if err != nil {
				t.Fatalf("crypt(%q, %q) err = %v; expected no error", tc.password, tc.salt, err)
			}

			if got != tc.expected {
				t.Fatalf("crypt(%q, %q) = %q; expected %q", tc.password, tc.salt, got, tc.expected)
			}
			t.Logf("crypt(%q, %q) = %q", tc.password, tc.salt, got)
		})
	}
}

func TestRndSalt(t *testing.T) {
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			// Ensure salt is correct length and contains valid characters.

			got := rndSalt(i)
			if gotL := len(got); gotL != i {
				t.Fatalf("rndSalt(%d) = %q (length = %d); expected length = %d", i, got, gotL, i)
			}

			// Ensure proper characters
		MATCH:
			for _, c := range got {
				for _, r := range saltChars {
					if c == r {
						continue MATCH
					}
				}
				t.Fatalf("rndSalt(%d) = %q; unexpected character %q", i, got, c)
			}

			t.Logf("rndSalt(%d) = %q", i, got)
		})
	}
}
