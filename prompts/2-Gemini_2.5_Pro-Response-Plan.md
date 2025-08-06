Of course. As an expert Go engineering manager, I'll break down the development plan into a granular, sequential set of tasks. This approach ensures that each step is a small, verifiable, and single-purpose commit, moving the project from an empty repository to a feature-complete and robust application.

The plan is divided into distinct phases, each building upon the last.

---

### **Phase 0: Project Scaffolding & Foundation**

*Goal: Establish the project structure, CLI framework, and configuration loading. No core logic yet.*

* **Task 1: `Initial commit`**
    * **Action:** Initialize the Go module and create the basic repository structure.
    * **Details:** Run `go mod init github.com/your-org/smog`. Create an initial `.gitignore` file for Go projects. Create the initial directory structure: `/cmd/smog`.
    * **Commit Message:** `chore: Initial project setup and go mod init`

* **Task 2: `feat(cli): Add Cobra for command structure`**
    * **Action:** Integrate the Cobra library to define the command hierarchy (`serve`, `auth`, `config`).
    * **Details:** Add `github.com/spf13/cobra` as a dependency. In `cmd/smog/main.go`, create the root command and add stubs for the `serve`, `auth`, and `config` subcommands. They should only print a placeholder message for now (e.g., `"config command called"`).
    * **Commit Message:** `feat(cli): Add Cobra and define basic command structure`

* **Task 3: `feat(config): Define configuration struct and loader`**
    * **Action:** Create the configuration struct and a function to load it using Viper.
    * **Details:** Create `/internal/config/config.go`. Define the `Config` struct matching the `smog.conf` specification. Add `github.com/spf13/viper` dependency. Implement a `LoadConfig()` function that sets up Viper to search default paths (`/etc/`, `~/`, executable dir).
    * **Commit Message:** `feat(config): Define config struct and implement loading with Viper`

* **Task 4: `feat(config): Implement 'config create' command`**
    * **Action:** Make the `config create` subcommand functional.
    * **Details:** Implement the logic for `config create` to write a default, commented `smog.conf` file to the platform-specific default location. Handle potential file permission errors gracefully.
    * **Commit Message:** `feat(config): Implement 'config create' to generate default config file`

* **Task 5: `feat(log): Add basic logging framework`**
    * **Action:** Set up a centralized logging package.
    * **Details:** Create `/internal/log/logger.go`. Use Go's standard `slog` library (available since 1.21) or a library like `zerolog` to create a basic logger instance. This will be wired into the config later. For now, it can just log to `stdout`.
    * **Commit Message:** `feat(log): Add slog for structured logging`

### **Phase 1: Core Functionality - The Happy Path**

*Goal: Get a single email successfully from an SMTP client to the Gmail API. Focus on core logic, not edge cases.*

* **Task 6: `feat(auth): Add Google OAuth2 dependencies and module skeleton`**
    * **Action:** Add required Google libraries and create the auth module.
    * **Details:** `go get` `golang.org/x/oauth2/google` and `google.golang.org/api/gmail/v1`. Create `/internal/auth/google.go` with placeholder functions for the auth flow.
    * **Commit Message:** `feat(auth): Add Google API dependencies and auth module skeleton`

* **Task 7: `feat(auth): Implement headless/remote authorization flow`**
    * **Action:** Implement the `smog auth login` logic for remote machines.
    * **Details:** Implement the flow that prints a URL to the console, waits for the user to paste the authorization code, and exchanges it for a token. This is the simplest flow to implement first and proves the core token exchange works.
    * **Commit Message:** `feat(auth): Implement remote authorization flow via CLI prompt`

* **Task 8: `feat(auth): Implement token storage and retrieval`**
    * **Action:** Save the retrieved token to a file and load it back.
    * **Details:** After a successful token exchange, serialize the token to JSON and save it to the path specified in `GoogleTokenPath`. Implement a helper function to load the token from this file.
    * **Commit Message:** `feat(auth): Implement OAuth2 token storage and retrieval`

* **Task 9: `feat(smtp): Add basic go-smtp server`**
    * **Action:** Integrate a library to listen for SMTP connections.
    * **Details:** Add `github.com/emersion/go-smtp` dependency. In a new `/internal/smtp/relay.go`, create a `Backend` struct and a `RunServer()` function that starts the server listening on the configured port. Connect this to the `smog serve` command. At this stage, it can just accept and immediately close connections.
    * **Commit Message:** `feat(smtp): Integrate go-smtp and start basic SMTP listener`

* **Task 10: `feat(smtp): Implement SMTP AUTH backend`**
    * **Action:** Validate client credentials against the configuration.
    * **Details:** Implement the `Login` function for the `go-smtp` backend. It should compare the username and password provided by the client with the `SMTPUser` and `SMTPPassword` from the loaded configuration.
    * **Commit Message:** `feat(smtp): Implement AUTH PLAIN/LOGIN backend`

* **Task 11: `feat(smtp): Implement message reception`**
    * **Action:** Implement the `go-smtp` session logic to accept an email message into a buffer.
    * **Details:** Implement the `Mail`, `Rcpt`, and `Data` functions for the session. The `Data` function should read the entire raw email content into a `bytes.Buffer`.
    * **Commit Message:** `feat(smtp): Implement backend to receive full email message data`

* **Task 12: `feat(gmail): Implement email sending via API`**
    * **Action:** Create the function that sends a raw email buffer to the Gmail API.
    * **Details:** In a new `/internal/gmail/client.go`, create a `Send` function. It should take an OAuth2 token and a byte slice (the raw email). It must base64url-encode the bytes and send them using the `users.messages.send` method.
    * **Commit Message:** `feat(gmail): Implement raw email sending via Gmail API`

* **Task 13: `feat(core): Connect SMTP reception to Gmail sending`**
    * **Action:** Wire the two halves together. This is the MVP commit.
    * **Details:** After the SMTP backend successfully receives a message (`Data` stream ends), it should call the `gmail.Send` function, passing the buffered email data. Log success or failure.
    * **Commit Message:** `feat(core): Connect SMTP DATA handler to Gmail API sender`

### **Phase 2: Hardening & Features**

*Goal: Add security, reliability, and usability features to make the application robust.*

* **Task 14: `fix(serve): Implement graceful shutdown`**
    * **Action:** Ensure the server shuts down cleanly on `SIGINT`/`SIGTERM`.
    * **Details:** Listen for OS signals. On receipt, call the SMTP server's `Close()` method, which should stop accepting new connections, and wait for a timeout before exiting the application.
    * **Commit Message:** `fix(serve): Implement graceful shutdown on interrupt signal`

* **Task 15: `feat(security): Add startup check for default password`**
    * **Action:** Refuse to start if the default SMTP password is still in use.
    * **Details:** In the `serve` command's logic, before starting the server, check if `SMTPPassword` equals `"smoggmos"`. If so, log a fatal error and exit.
    * **Commit Message:** `feat(security): Add startup check to prevent use of default password`

* **Task 16: `feat(security): Implement IP/subnet filtering`**
    * **Action:** Allow connections only from whitelisted IP addresses or subnets.
    * **Details:** When a new connection is accepted, before the SMTP handshake, parse the `AllowedSubnets` array from the config. Check if the client's remote IP address falls within any of the configured subnets. If not, close the connection immediately.
    * **Commit Message:** `feat(security): Implement IP/subnet based connection filtering`

* **Task 17: `feat(smtp): Implement message size limit`**
    * **Action:** Reject incoming messages that are too large.
    * **Details:** Modify the `go-smtp` backend's `Data` handler. As data is streamed in, check its size against `MessageSizeLimitMB`. If it exceeds the limit, return an appropriate SMTP error code (`552`) and discard the message.
    * **Commit Message:** `feat(smtp): Implement server-side message size limit`

* **Task 18: `refactor(log): Implement configured log levels and file output`**
    * **Action:** Fully implement the logging configuration.
    * **Details:** In the `main` function, configure the logger based on `LogLevel` and `LogPath` from the config. Use structured logging for key events (e.g., `log.Info("email relayed", "client_ip", ip, "msg_id", id)`).
    * **Commit Message:** `refactor(log): Implement file and level-based logging from config`

* **Task 19: `feat(auth): Implement local browser-based authorization flow`**
    * **Action:** Add the more user-friendly auth flow for local machines.
    * **Details:** Enhance the `auth login` command to start a local web server to catch the OAuth2 redirect, automatically open the browser, and retrieve the code. This makes the process seamless for GUI users.
    * **Commit Message:** `feat(auth): Implement local web server for browser-based auth flow`

### **Phase 3: Testing, Polish & Production Readiness**

*Goal: Ensure quality through testing, and provide tools for easy deployment and use.*

* **Task 20: `test: Add unit tests for config and utils`**
    * **Action:** Write unit tests for non-network-dependent logic.
    * **Details:** Create `_test.go` files for the `config` package (testing loading/defaults) and any utility functions (like the IP subnet checker).
    * **Commit Message:** `test: Add unit tests for configuration loading and utilities`

* **Task 21: `test: Add integration test for SMTP-to-Gmail flow`**
    * **Action:** Create an end-to-end test for the core logic.
    * **Details:** Write a test that:
        1.  Starts a `smog` server on a random port.
        2.  Uses a mock `httptest.NewServer` to act as the Google API endpoint.
        3.  Uses a simple SMTP client library to connect to the `smog` server, authenticate, and send a message.
        4.  Asserts that the mock Google API server received the correct, base64url-encoded message.
    * **Commit Message:** `test: Add integration test for end-to-end message relay`

* **Task 22: `ci: Add GitHub Actions workflow for build and test`**
    * **Action:** Automate testing.
    * **Details:** Create a `.github/workflows/ci.yml` file that triggers on push. The workflow should set up Go, run `go build ./...`, and `go test ./...`.
    * **Commit Message:** `ci: Add GitHub Actions workflow for automated builds and testing`

* **Task 23: `build: Create cross-platform build script`**
    * **Action:** Simplify the process of creating binaries for release.
    * **Details:** Create a `scripts/build.sh` that uses `go build` with `GOOS` and `GOARCH` environment variables to compile binaries for Linux, Windows, and macOS (amd64/arm64).
    * **Commit Message:** `build: Add script for cross-platform compilation`

* **Task 24: `docs: Create service installation scripts`**
    * **Action:** Provide scripts to install `smog` as a system service.
    * **Details:** Create `scripts/install-service.sh` to generate a `systemd` unit file and `scripts/install-service.ps1` to register `smog` as a Windows Service.
    * **Commit Message:** `docs: Add scripts for installing as a systemd/Windows service`

* **Task 25: `docs: Finalize README, man page, and release v1.0.0`**
    * **Action:** Polish all user-facing documentation and tag the first official release.
    * **Details:** Thoroughly review and update the `README.md` and the `man` page content. Ensure all examples are correct. Once complete, tag the commit with `v1.0.0`.
    * **Commit Message:** `docs: Finalize documentation for v1.0.0 release`

This granular, commit-by-commit plan ensures a methodical development process, facilitates code reviews, and results in a well-structured and maintainable application.
