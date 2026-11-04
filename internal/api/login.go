package api

import (
	"encoding/json"
	"fmt"
	"github.com/pajbot/pajbot2-discord/internal/appctx"
	"github.com/pajbot/pajbot2-discord/internal/config"
	"github.com/pajbot/pajbot2-discord/internal/connection"
	"github.com/pajbot/pajbot2-discord/pkg"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
	"net/http"
)

var oauth2Config *oauth2.Config = nil
var approvalForceTwitch = oauth2.SetAuthURLParam("force_verify", "true")

func loginInit() {
	scopes := []string{}

	oauth2Config = &oauth2.Config{
		ClientID:     config.TwitchClientID,
		ClientSecret: config.TwitchClientSecret,
		Scopes:       scopes,
		Endpoint:     twitch.Endpoint,
		RedirectURL:  config.TwitchRedirectURI,
	}
}

// Redirect user to twitch page
func login(w http.ResponseWriter, r *http.Request) {
	connectionID := r.FormValue("code")
	if connectionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "missing connection ID")
		return
	}

	// Validate code
	pendingConnection := connection.GetPendingConnection(connectionID)
	if pendingConnection == nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "bad connection ID")
		return
	}

	fmt.Println("Redirecting login for", pendingConnection)

	http.Redirect(w, r, oauth2Config.AuthCodeURL(pendingConnection.State, approvalForceTwitch), http.StatusTemporaryRedirect)

}

func loginAuthorized(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "missing state")
		return
	}

	// Validate state
	connectionID, pendingConnection := connection.GetPendingConnectionByState(state)
	if pendingConnection == nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "bad state")
		return
	}

	token, err := oauth2Config.Exchange(r.Context(), r.FormValue("code"))
	if err != nil {
		fmt.Println("Error exchanging code", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
		connection.UnregisterPendingConnection(connectionID)
		return
	}

	go func() {
		// Fetch user ID
		req, err := http.NewRequest("GET", "https://id.twitch.tv/oauth2/validate", nil)
		if err != nil {
			fmt.Println("error making request to validate twitch token", err)
			connection.UnregisterPendingConnection(connectionID)
			return
		}

		req.Header.Add("Authorization", fmt.Sprintf("OAuth %s", token.AccessToken))

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("error in request to validate twitch token", err)
			connection.UnregisterPendingConnection(connectionID)
			return
		}

		structuredResponse := connection.TwitchValidateResponse{}

		if err := json.NewDecoder(res.Body).Decode(&structuredResponse); err != nil {
			fmt.Println("error decoding validate twitch token response:", err)
			connection.UnregisterPendingConnection(connectionID)
			return
		}

		probApp := r.Context().Value(appctx.Application)
		a := probApp.(pkg.App)

		go a.RegisterUser(pendingConnection, structuredResponse)
	}()
}

func loginAttach(handler *http.ServeMux) {
	handler.HandleFunc("/login", login)
	handler.HandleFunc("/login/authorized", loginAuthorized)
}
