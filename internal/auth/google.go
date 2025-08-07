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
	oauthConfig, err := google.ConfigFromJSON(b, gmail.GmailSendScope)
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %v", err)
	}
	client, err := getClient(logger, oauthConfig, cfg)
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
func getClient(logger *slog.Logger, oauthConfig *oauth2.Config, cfg *config.Config) (*http.Client, error) {
	tok, err := LoadToken(logger, cfg)
	if err != nil {
		return nil, err
	}

	if tok == nil {
		logger.Info("no token found, requesting one from the web")
		tok, err = getTokenFromWeb(logger, oauthConfig)
		if err != nil {
			return nil, err
		}
		logger.Info("saving token to file")
		if err := saveToken(logger, cfg.GoogleTokenPath, tok); err != nil {
			return nil, err
		}
	}

	return oauthConfig.Client(context.Background(), tok), nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(logger *slog.Logger, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	logger.Info("Go to the following link in your browser then type the authorization code:", "url", authURL)
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

// LoadToken retrieves a token from a file, if it doesn't exist it returns a nil token.
func LoadToken(logger *slog.Logger, cfg *config.Config) (*oauth2.Token, error) {
	logger.Debug("loading token from file")
	token, err := tokenFromFile(logger, cfg.GoogleTokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get token from file: %w", err)
	}

	if token == nil {
		logger.Warn("token file is missing or empty, run smog login to create one")
		return nil, nil
	}

	return token, nil
}

// Retrieves a token from a local file.
func tokenFromFile(logger *slog.Logger, file string) (*oauth2.Token, error) {
	logger.Info("checking for token", "path", file)
	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("token file not found")
			return nil, nil // Return nil token and nil error
		}
		return nil, fmt.Errorf("failed to open token file: %w", err)
	}
	defer f.Close()

	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}
	logger.Info("token file found")
	return tok, nil
}

// Saves a token to a file path.
func saveToken(logger *slog.Logger, path string, token *oauth2.Token) error {
	logger.Info("caching oauth token", "path", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %v", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
