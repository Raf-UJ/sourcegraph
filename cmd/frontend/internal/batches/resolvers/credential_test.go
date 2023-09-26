package resolvers

import (
	"net/url"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestUnmarshalBatchChangesCredentialID(t *testing.T) {
	siteCred := marshalBatchChangesCredentialID(123, true)
	userCred := marshalBatchChangesCredentialID(123, false)
	tcs := []struct {
		id               graphql.ID
		credentialID     int64
		isSiteCredential bool
		wantErr          bool
	}{
		{
			id:               siteCred,
			credentialID:     123,
			isSiteCredential: true,
		},
		{
			id:               userCred,
			credentialID:     123,
			isSiteCredential: false,
		},
		// Encoded value is not a string.
		{
			id:      relay.MarshalID("BatchChangesCredential", 123),
			wantErr: true,
		},
		// Encoded value does not conform to `<scope>:<int_id>` pattern.
		{
			id:      relay.MarshalID("BatchChangesCredential", "site123"),
			wantErr: true,
		},
		// Encoded value does not contain valid scope.
		{
			id:      relay.MarshalID("BatchChangesCredential", "invalidkind:1"),
			wantErr: true,
		},
		// Encoded value does not contain valid int id.
		{
			id:      relay.MarshalID("BatchChangesCredential", "user:invalidid"),
			wantErr: true,
		},
	}
	for _, tc := range tcs {
		haveCredentialID, haveIsSiteCredential, haveErr := unmarshalBatchChangesCredentialID(tc.id)
		if haveCredentialID != tc.credentialID {
			t.Errorf("invalid credential ID returned for %q: want=%d have=%d", tc.id, tc.credentialID, haveCredentialID)
		}
		if haveIsSiteCredential != tc.isSiteCredential {
			t.Errorf("invalid isSiteCredential returned for %q: want=%t have=%t", tc.id, tc.isSiteCredential, haveIsSiteCredential)
		}
		if (haveErr != nil) != tc.wantErr {
			t.Errorf("invalid error %+v", haveErr)
		}
	}
}

func TestCommentSSHKey(t *testing.T) {
	conf.Mock(&conf.Unified{
		SiteConfiguration: schema.SiteConfiguration{
			ExternalURL: "https://www.sourcegraph.com",
		},
	})
	defer conf.Mock(nil)

	externalURL, err := url.Parse(conf.Get().ExternalURL)
	if err != nil {
		t.Errorf("could not parse external url: %v", err)
	}

	publicKey := "public\n"
	sshKey := commentSSHKey(&auth.BasicAuthWithSSH{BasicAuth: auth.BasicAuth{Username: "foo", Password: "bar"}, PrivateKey: "private", PublicKey: publicKey, Passphrase: "pass"})
	expectedKey := "public Sourcegraph " + externalURL.Host

	if sshKey != expectedKey {
		t.Errorf("found wrong ssh key: want=%q, have=%q", expectedKey, sshKey)
	}
}
