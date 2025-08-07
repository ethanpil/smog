package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/ethanpil/smog/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

func Login(logger *slog.Logger, cfg *config.Config) error {
	b, err := ioutil.ReadFile(cfg.GoogleCredentialsPath)
	if err != nil {
		return fmt.Errorf("unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailSendScope)
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %v", err)
	}
	client, err := getClient(logger, config, cfg.GoogleTokenPath)
	if err != nil {
		return fmt.Errorf("unable to retrieve client: %v", err)
	}

	srv, err := gmail.New(client)
	if err != nil {
		return fmt.Errorf("unable to retrieve Gmail client: %v", err)
	}

	logger.Info("checking gmail connection")
	_, err = srv.Users.Labels.List("me").Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve labels: %v", err)
	}
	logger.Info("gmail connection successful")
	return nil
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(logger *slog.Logger, config *oauth2.Config, tokenFile string) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	logger.Info("checking for existing token")
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		logger.Info("no token found, requesting one from the web")
		tok, err = getTokenFromWeb(logger, config)
		if err != nil {
			return nil, err
		}
		logger.Info("saving token to file")
		if err := saveToken(tokenFile, tok); err != nil {
			return nil, err
		}
	} else {
		logger.Info("token found in file")
	}
	return config.Client(context.Background(), tok), nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(logger *slog.Logger, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	logger.Info("Go to the following link in your browser then type the authorization code: \n\n%v\n", authURL)
	logger.Info("Enter authorization code: ")

	reader := bufio.NewReader(os.Stdin)
	authCode, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), strings.TrimSpace(authCode))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %v", err)
	}
	return tok, nil
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %v", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
