package main

import (
	"log"
	"time"

	"github.com/nicklaw5/helix"
	"github.com/pajbot/pajbot2-discord/internal/config"
)

var (
	helixClient *helix.Client

	rdyHelixClient = make(chan struct{})
)

func init() {
	var err error
	helixClient, err = helix.NewClient(&helix.Options{
		ClientID:     config.TwitchClientID,
		ClientSecret: config.TwitchClientSecret,
	})

	if err != nil {
		log.Fatalf("Error initializing helix client:", err)
	}

	go initAppAccessToken(helixClient, rdyHelixClient)
}

// initAppAccessToken requests and sets app access token to the provided helix.Client
// and initializes a ticker running every 24 Hours which re-requests and sets app access token
func initAppAccessToken(helixAPI *helix.Client, tokenFetched chan struct{}) {
	response, err := helixAPI.RequestAppAccessToken([]string{})

	if err != nil {
		log.Fatalf("[Helix] Error requesting app access token: %s , \n %s", err.Error(), response.Error)
	}

	log.Printf("[Helix] Requested access token, status: %d, expires in: %d", response.StatusCode, response.Data.ExpiresIn)
	helixAPI.SetAppAccessToken(response.Data.AccessToken)
	close(tokenFetched)

	// initialize the ticker
	ticker := time.NewTicker(24 * time.Hour)

	for range ticker.C {
		response, err := helixAPI.RequestAppAccessToken([]string{})
		if err != nil {
			log.Printf("[Helix] Failed to re-request app access token from ticker, status: %d", response.StatusCode)
			continue
		}
		log.Printf("[Helix] Re-requested access token from ticker, status: %d, expires in: %d", response.StatusCode, response.Data.ExpiresIn)

		helixAPI.SetAppAccessToken(response.Data.AccessToken)
	}
}
