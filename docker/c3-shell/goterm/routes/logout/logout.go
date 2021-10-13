package logout

import (
	"net/http"
	"net/url"
	"os"

	"app"
	"auth"
)

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
        session, err := app.Store.Get(r, "auth-session")
        if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
        }

	delete(session.Values, "id_token")
	delete(session.Values, "access_token")
	delete(session.Values, "profile")
        err = session.Save(r, w)

        if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
        }

        authenticator, err := auth.NewAuthenticator()
        if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
        }

	loRedirectUrl := authenticator.LoRedirectUrl

	logoutUrl, err := url.Parse("https://login.microsoftonline.com/" + os.Getenv("ISSUER") + "/oauth2/logout?client_id=" + os.Getenv("CLIENT_ID") + "&post_logout_redirect_uri=" + loRedirectUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, logoutUrl.String(), http.StatusTemporaryRedirect)
}
