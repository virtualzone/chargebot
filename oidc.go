package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type Claims map[string]interface{}

type OIDCProvider struct {
	OAuthConfig *oauth2.Config
	Provider    *oidc.Provider
	States      []string
	Context     context.Context
	Verifier    *oidc.IDTokenVerifier
}

// var OIDCTestingMode bool = false
// var OIDCTestingSecret string = "nQTJgFvQGwNacHG6"
var OIDCInstance *OIDCProvider = &OIDCProvider{}

func GetOIDCProvider() *OIDCProvider {
	return OIDCInstance
}

func (op *OIDCProvider) Init() {
	op.Context = context.Background()

	provider, err := oidc.NewProvider(op.Context, GetConfig().AuthURL)
	if err != nil {
		log.Fatal(err)
	}

	op.OAuthConfig = &oauth2.Config{
		ClientID:     GetConfig().AuthClientID,
		ClientSecret: GetConfig().AuthClientSecret,
		RedirectURL:  "https://" + GetConfig().Hostname + "/auth/callback",

		// Discovery returns the OAuth2 endpoints.
		Endpoint: provider.Endpoint(),

		// "openid" is a required scope for OpenID Connect flows.
		Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}
	op.Verifier = provider.Verifier(&oidc.Config{ClientID: op.OAuthConfig.ClientID, SkipClientIDCheck: true})
	op.Provider = provider
	op.States = []string{}
}

func (op *OIDCProvider) GetUserForSubject(subject string) *User {
	user := GetDB().GetUser(subject)
	return user
}

func (op *OIDCProvider) VerifyAuthHeader(jwt string) (*oidc.IDToken, *Claims, error) {
	idToken, err := op.Verifier.Verify(op.Context, jwt)
	if err != nil {
		return nil, nil, err
	}
	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		return nil, nil, err
	}
	return idToken, &claims, nil
}

func (op *OIDCProvider) GetRoles(claims *Claims) ([]string, error) {
	pathElements := strings.Split(GetConfig().AuthRolesPath, ".")
	var structure map[string]interface{} = (*claims)
	for i, pathElement := range pathElements {
		if structure != nil && structure[pathElement] != nil {
			curType := fmt.Sprintf("%v", reflect.TypeOf(structure[pathElement]))
			if curType == "map[string]interface {}" {
				structure = (structure[pathElement]).(map[string]interface{})
			} else if curType == "[]interface {}" {
				if i != len(pathElements)-1 {
					return nil, errors.New("json path resolved to array too soon")
				}
				list := (structure[pathElement]).([]interface{})
				res := []string{}
				for _, listItem := range list {
					if _, ok := listItem.(string); ok {
						res = append(res, listItem.(string))
					}
				}
				return res, nil
			} else {
				return nil, errors.New("unexpected element in json path")
			}
		}
	}

	return nil, errors.New("could not find roles array in json path")
}
