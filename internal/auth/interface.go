//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/puzpuzpuz/xsync/v3"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// Identity represents a particular user's identity in a particular trust domain
// (represented by an IdentityProvider).
type Identity struct {
	// UserID is a stable unique identifier for the user.  This may be a large
	// integer or a UUID, rather than something human-readable.
	//
	// For KeyCloak, this is `sub`.
	UserID string
	// HumanName is a human-readable name.  Because humans are fickle, these may
	// not be unique or stable over time, though they should be unique at any
	// particular time.  For example, Alex may change their handle from
	// "alexsmith" to "alexawesome" after a life change, and someone else might
	// enroll the "alexsmith" handle.  If you are storing data, you want UserID,
	// not HumanName.  If you are presenting data, you probably want HumanName.
	//
	// For KeyCloak, this is `preferred_username`.  For some other providers,
	// this might be an email address.
	HumanName string
	// Provider is the identity provider that vended this identity.  Note that
	// UserID and HumanName are only unique within the context of a single
	// identity provider.
	Provider IdentityProvider
	// FirstName and LastName are optional fields that may be provided by the
	// identity provider. These are not guaranteed to be present, and may be
	// empty.
	FirstName string
	LastName  string
}

// String implements strings.Stringer, and also provides a stable storage
// representation of the Identity.
func (i *Identity) String() string {
	if i == nil {
		return ""
	}
	if i.Provider == nil || i.Provider.String() == "" {
		// Special case for provider registered as "".
		return i.UserID
	}
	return fmt.Sprintf("%s/%s", i.Provider.String(), i.UserID)
}

// Human returns a human-readable representation of the identity, suitable for
// presentation to humans.
func (i *Identity) Human() string {
	if i == nil {
		return "<unknown>"
	}
	if i.Provider == nil || i.Provider.String() == "" {
		// Special case for provider registered as "".
		return i.HumanName
	}
	return fmt.Sprintf("%s/%s", i.Provider.String(), i.HumanName)
}

// Resolver is an interface for resolving human-readable or stable identifiers
// from either JWTs or stored strings
type Resolver interface {

	// Validate validates a token and returns an underlying identity representation
	// suitable for use in authz calls.  This _probably_ reads data from the token,
	// but could fetch from an external provider.
	Validate(ctx context.Context, token jwt.Token) (*Identity, error)

	// Resolve takes either a human-readable identifier or a stable identifier and
	// returns the underlying identity.  This may involve looking up or defining
	// the identity in the remote identity provider.
	//
	// For Keycloak + GitHub, this may define a new user in Keycloak based on
	// GitHub user data if the user is not already known to Keycloak.
	Resolve(ctx context.Context, id string) (*Identity, error)
}

// IdentityProvider provides an abstract interface for looking up identities
// in a remote identity provider.
type IdentityProvider interface {
	Resolver

	// String returns the name of the identity provider.  This should be a short
	// one-word string suitable for presentation.  As a special case, a _single_
	// provider may use the empty string as its name to act as a default / fallback
	// provider.
	String() string
	// URL returns the `iss` URL of the identity provider.
	URL() url.URL
}

// IdentityClient supports the ability to look up identities in one or more
// IdentityProviders.
type IdentityClient struct {
	// This map is a bit overloaded; it maps both short provider names and URLs to IdentityProviders.
	providers *xsync.MapOf[string, IdentityProvider]
}

var _ Resolver = (*IdentityClient)(nil)

// NewIdentityClient creates a new IdentityClient with the supplied providers.
func NewIdentityClient(providers ...IdentityProvider) (*IdentityClient, error) {
	c := &IdentityClient{
		providers: xsync.NewMapOf[string, IdentityProvider](xsync.WithPresize(len(providers) * 2)),
	}
	for _, p := range providers {
		u := p.URL() // URL's String has a pointer receiver

		prev, ok := c.providers.LoadOrStore(p.String(), p)
		if ok { // We had an existing value, this is a configuration error.
			prevUrl := prev.URL()
			return nil, fmt.Errorf("duplicate provider for %q, new %s, old %s", p.String(), u.String(), prevUrl.String())
		}

		prev, ok = c.providers.LoadOrStore(u.String(), p)
		if ok { // We had an existing value, this is a configuration error.
			return nil, fmt.Errorf("duplicate provider for URL %s, new %q, old %q", u.String(), p.String(), prev.String())
		}
	}
	return c, nil
}

// Register registers a new identity provider with the client.
func (c *IdentityClient) Register(p IdentityProvider) error {
	if c == nil {
		return errors.New("cannot register with a nil IdentityClient")
	}
	u := p.URL() // URL's String has a pointer receiver

	prev, ok := c.providers.LoadOrStore(p.String(), p)
	if ok { // We had an existing value, this is a configuration error.
		prevUrl := prev.URL()
		return fmt.Errorf("duplicate provider for %q, new %s, old %s", p.String(), u.String(), prevUrl.String())
	}

	prev, ok = c.providers.LoadOrStore(u.String(), p)
	if ok { // We had an existing value, this is a configuration error.
		return fmt.Errorf("duplicate provider for URL %s, new %q, old %q", u.String(), p.String(), prev.String())
	}

	return nil
}

// Resolve implements Resolver.
func (c *IdentityClient) Resolve(ctx context.Context, id string) (*Identity, error) {
	providerName := ""
	slash := strings.Index(id, "/")
	if slash >= 0 {
		providerName = id[:slash]
		id = id[slash+1:]
	}
	provider, ok := c.providers.Load(providerName)
	if !ok || provider == nil {
		return nil, fmt.Errorf("unknown provider %q", providerName)
	}

	return provider.Resolve(ctx, id)
}

// Validate implements Resolver.
func (c *IdentityClient) Validate(ctx context.Context, token jwt.Token) (*Identity, error) {
	iss := token.Issuer()
	if iss == "" {
		return nil, errors.New("token has no issuer")
	}
	provider, ok := c.providers.Load(iss)
	if !ok || provider == nil {
		return nil, fmt.Errorf("unknown provider %q", iss)
	}

	return provider.Validate(ctx, token)
}
