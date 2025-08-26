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
	"path/filepath"
	"strings"
	"time"

	"github.com/ethanpil/smog/internal/config"
	"github.com/pkg/browser"
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
	client, err := getClientForLogin(logger, oauthConfig, cfg)
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

// getClientForLogin retrieves a token, saves the token, then returns the generated client.
// This is used for the interactive login flow.
func getClientForLogin(logger *slog.Logger, oauthConfig *oauth2.Config, cfg *config.Config) (*http.Client, error) {
	tok, err := LoadToken(logger, cfg)
	if err != nil {
		// If there's an error loading the token (e.g., corrupted file),
		// we can proceed to get a new one from the web.
		logger.Warn("could not load existing token, will request a new one", "err", err)
	}

	if tok == nil {
		logger.Info("no token found, attempting to get one")
		// Try browser-based flow first
		tok, err = getTokenFromBrowser(logger, oauthConfig)
		if err != nil {
			logger.Warn("could not get token from browser, falling back to manual mode", "err", err)
			// Fallback to manual copy-paste flow
			tok, err = getTokenFromWeb(logger, oauthConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to get token manually: %w", err)
			}
		}
		logger.Info("saving token to file")
		if err := saveToken(logger, cfg.GoogleTokenPath, tok); err != nil {
			return nil, err
		}
	}

	return oauthConfig.Client(context.Background(), tok), nil
}

// GetClient returns an authenticated http.Client and the corresponding token.
// It's used by the server at startup. It requires a valid, pre-existing token.
func GetClient(logger *slog.Logger, cfg *config.Config) (*http.Client, *oauth2.Token, error) {
	b, err := ioutil.ReadFile(cfg.GoogleCredentialsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	oauthConfig, err := google.ConfigFromJSON(b, gmail.GmailSendScope)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	tok, err := LoadToken(logger, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load token: %w", err)
	}
	if tok == nil {
		// This case is handled by the startup validation in main.go, but is here for safety.
		return nil, nil, fmt.Errorf("token not found, please run 'smog auth login'")
	}

	return oauthConfig.Client(context.Background(), tok), tok, nil
}

// getTokenFromBrowser attempts to automatically open a browser for OAuth authentication.
func getTokenFromBrowser(logger *slog.Logger, config *oauth2.Config) (*oauth2.Token, error) {
	// Use a fixed port as specified in AGENTS.md
	const callbackPort = 53682
	config.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d", callbackPort)

	// Create a channel to receive the authorization code
	codeChan := make(chan string)
	errChan := make(chan error)

	// Setup a temporary HTTP server to handle the OAuth redirect
	server := &http.Server{Addr: fmt.Sprintf(":%d", callbackPort)}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Get the authorization code from the query parameters
		code := r.URL.Query().Get("code")
		if code == "" {
			logger.Error("oauth callback received no code")
			fmt.Fprintln(w, "Error: Authorization code not found in request.")
			errChan <- fmt.Errorf("authorization code not found")
			return
		}
		// Send the code to the channel
		codeChan <- code
		// Respond to the browser
		fmt.Fprintln(w, "Authentication successful! You can close this window now.")
	})

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("failed to start local server: %w", err)
		}
	}()

	// Generate the authentication URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	// Try to open the URL in a browser
	err := browser.OpenURL(authURL)
	if err != nil {
		logger.Warn("failed to open browser automatically", "err", err)
		// If we can't open the browser, this flow has failed.
		// We shut down the server and return an error, so the caller can fall back.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return nil, fmt.Errorf("could not open browser: %w", err)
	}

	logger.Info("Your browser has been opened to visit the Google authentication page.")
	logger.Info("If your browser is on a different machine, please open the URL above.")
	logger.Info("Waiting for authorization...")

	// Wait for the authorization code or an error
	select {
	case code := <-codeChan:
		// Shutdown the server once the code is received
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)

		// Exchange the code for a token
		tok, err := config.Exchange(context.Background(), code)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
		}
		return tok, nil
	case err := <-errChan:
		// Shutdown the server on error
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return nil, err
	case <-time.After(5 * time.Minute): // Timeout after 5 minutes
		// Shutdown the server on timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return nil, fmt.Errorf("timed out waiting for authorization code")
	}
}

// getTokenFromWeb handles the manual, copy-paste based token retrieval.
func getTokenFromWeb(logger *slog.Logger, config *oauth2.Config) (*oauth2.Token, error) {
	config.RedirectURL = "urn:ietf:wg:oauth:2.0:oob"
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	logger.Info("You are running in a headless environment or the browser could not be opened.")
	logger.Info("Please follow these steps:")
	logger.Info("1. Open this URL on any machine with a web browser:", "url", authURL)
	logger.Info("2. Authenticate with Google and grant permissions.")
	logger.Info("3. Google will show you an authorization code. Copy it.")
	logger.Info("Enter the authorization code here:")

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

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("cannot create token directory: %w", err)
	}

	// Backup existing token
	if _, err := os.Stat(path); err == nil {
		backupPath := path + ".bak"
		logger.Info("backing up existing token", "path", backupPath)
		if err := os.Rename(path, backupPath); err != nil {
			return fmt.Errorf("failed to backup token: %w", err)
		}
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %v", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

// RevokeToken securely deletes the token file.
func RevokeToken(logger *slog.Logger, cfg *config.Config) error {
	tokenPath := cfg.GoogleTokenPath
	logger.Info("revoking token", "path", tokenPath)

	// Overwrite with zeros
	f, err := os.OpenFile(tokenPath, os.O_WRONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("token file not found, nothing to revoke")
			return nil
		}
		return fmt.Errorf("failed to open token file for writing: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to get token file info: %w", err)
	}

	if _, err := f.Write(make([]byte, stat.Size())); err != nil {
		return fmt.Errorf("failed to overwrite token file: %w", err)
	}

	// Delete the file
	if err := os.Remove(tokenPath); err != nil {
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	logger.Info("token revoked successfully")
	return nil
}
