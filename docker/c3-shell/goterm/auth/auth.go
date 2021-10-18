package auth

import (
	"context"
	"log"
	"os"

	oidc "github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

var ResolveGroup bool

type Authenticator struct {
	Provider *oidc.Provider
	Config   oauth2.Config
	Ctx      context.Context
	LoRedirectUrl string
}

func NewAuthenticator() (*Authenticator, error) {
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, "https://login.microsoftonline.com/"+os.Getenv("ISSUER")+"/v2.0")
	if err != nil {
		log.Printf("failed to get provider: %v", err)
		return nil, err
	}

	scopes := []string{oidc.ScopeOpenID, "profile"}
	if ResolveGroup {
		scopes = []string{oidc.ScopeOpenID, "profile", "User.Read.All", "Group.Read.All"}
	}

	conf := oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RedirectURL:  os.Getenv("AUTH_CALLBACK"),
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	log.Print(conf)

	return &Authenticator{
		Provider: provider,
		Config:   conf,
		Ctx:      ctx,
		LoRedirectUrl: os.Getenv("LO_REDIRECT_URL"),
	}, nil
}
