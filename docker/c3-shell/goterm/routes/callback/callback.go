package callback

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/coreos/go-oidc"

	"app"
	"auth"
	jwt "github.com/dgrijalva/jwt-go"
)

var ResolveGroup bool

func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	session, err := app.Store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("state") != session.Values["state"] {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	authenticator, err := auth.NewAuthenticator()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := authenticator.Config.Exchange(context.TODO(), r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("no token found: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
		return
	}

	oidcConfig := &oidc.Config{
		ClientID: os.Getenv("CLIENT_ID"),
	}

	idToken, err := authenticator.Provider.Verifier(oidcConfig).Verify(context.TODO(), rawIDToken)

	if err != nil {
		http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Getting now the userInfo
	var profile map[string]interface{}
	if err := idToken.Claims(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(rawIDToken, claims, nil)
	if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	}

	groups := claims["groups"]
	if ResolveGroup && groups != nil {
		group := groups.([]interface{})[0].(string)

		req, _ := http.NewRequest("GET", "https://graph.microsoft.com/v1.0/groups/{" + group + "}", nil)
		req.Header.Set("Authorization", "Bearer " + token.AccessToken)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		var objmap map[string]interface{}
		if err := json.Unmarshal(body, &objmap); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println("group: ", objmap["displayName"])
	}

	session.Values["id_token"] = rawIDToken
	session.Values["access_token"] = token.AccessToken
	session.Values["profile"] = profile

	err = session.Save(r, w)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to logged in page
	// http.Redirect(w, r, "/user", http.StatusSeeOther)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
