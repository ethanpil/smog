Here is a comprehensive development plan for `smog`, a Go-based SMTP to Gmail API relay tool. This plan covers the project's architecture, command-line interface, user flows, and configuration, designed to be robust, flexible, and secure.

-----

## Smog: Development Plan

### 1. Project Mission & Goals

**Mission:** To provide a simple, secure, and reliable cross-platform SMTP relay that allows legacy devices and applications to send emails through a modern, authenticated Gmail account using the Gmail API.

**Core Goals:**

  * **Simplicity:** Easy to configure and run, with a "set it and forget it" operational model.
  * **Security:** Enforce modern authentication (OAuth2) for Google and provide strong access controls for the local SMTP server.
  * **Transparency:** Ensure emails are relayed without modification, preserving all original headers, content, and attachments.
  * **Reliability:** Robust error handling, automatic token renewal, and clear logging for diagnostics.
  * **Portability:** A single, statically-linked binary that runs on Linux, macOS, and Windows.

-----

### 2. Command-Line Interface (CLI) Design

The CLI will be structured using subcommands for clarity and extensibility (e.g., `smog serve`, `smog auth`).

#### **Main Command**

`smog [global flags]`

#### **Subcommands**

  * `smog serve`: The main operational command. Starts the SMTP server and listens for connections. This is the default command if no other is specified.
  * `smog auth`: Manages Google API authorization.
      * `smog auth login`: Initiates the OAuth2 authorization flow.
      * `smog auth revoke`: Deletes the stored OAuth2 token, forcing re-authorization on the next run.
  * `smog config`: Manages the configuration file.
      * `smog config create`: Creates a default configuration file at the platform-specific location.
      * `smog config show`: Displays the current configuration that `smog` will use, showing which file is loaded.
  * `smog version`: Prints the application version.
  * `smog help`: Displays help information.

#### **Global Flags**

  * `-c, --config <path>`: Path to the configuration file. Overrides default search paths.
      * Example: `smog serve -c /opt/smog/smog.conf`
  * `-v, --verbose`: Enables verbose logging to the console, overriding the `LogLevel` in the config file for console output.
  * `-s, --silent`: Disables all console output except for fatal errors.
  * `--help`: Displays help for any command.

-----

### 3\. `man` Page (Linux/macOS)

```
SMOG(1)                   General Commands Manual                  SMOG(1)

NAME
     smog - A cross-platform SMTP to Gmail API relay tool.

SYNOPSIS
     smog [global flags] [command]

DESCRIPTION
     smog acts as a local SMTP server that accepts authenticated email
     submissions and relays them through the Google Gmail API using OAuth2.
     It is designed for legacy systems that can only send email via SMTP
     but need to integrate with a modern Gmail account.

     Before first use, the administrator must run `smog auth login` to
     authorize the application with Google.

GLOBAL FLAGS
     -c, --config <path>
           Specify a custom path to the smog.conf configuration file.

     -v, --verbose
           Enable verbose logging output to the console.

     -s, --silent
           Disable all console output except for fatal errors.

COMMANDS
     serve
           Starts the SMTP server and begins listening for connections.
           This is the primary operational mode and the default command.

     auth
           Manages Google API authorization.
           login     Initiate the interactive OAuth2 flow to authorize
                     smog to send emails on your behalf.
           revoke    Delete the stored API token. Re-authorization will
                     be required on the next run.

     config
           Manages the configuration file.
           create    Creates a new, default smog.conf file in the
                     platform-appropriate default location.
           show      Displays the currently loaded configuration.

     version
           Prints the version of smog.

     help
           Displays this help message.

FILES
     /etc/smog.conf
           Default configuration file path on Linux.

     ~/smog.conf
           Default configuration file path on macOS.

     <executable_dir>\smog.conf
           Default configuration file path on Windows.

EXAMPLES
     Run the SMTP server with the default configuration:
           $ smog serve

     Authorize smog with your Google account:
           $ smog auth login

     Run the server using a custom configuration file and verbose output:
           $ smog -v -c /etc/custom/smog.conf serve

AUTHOR
     Developed by Gemini for Google.
```

-----

### 4\. Authorization Flow (`smog auth login`)

This process is critical and is designed to work on both local machines with a GUI and remote/headless servers, inspired by `rclone`.

1.  **Prerequisites:** The user must first create a Google Cloud Project, enable the Gmail API, and download a `credentials.json` file for an "Desktop Application" OAuth 2.0 Client ID. The path to this file must be set in `smog.conf`.

2.  **Initiation:** The user runs `smog auth login`.

3.  **Startup Check:** `smog` checks for the `credentials.json` file at the path specified in the config. If not found, it exits with an error.

4.  **Flow Type Detection (Auto-detection):**

      * `smog` will attempt to start a tiny local web server on a high-numbered port (e.g., `53682`) and check if it can be accessed. If successful, it proceeds with the "Local Machine Flow." If not (e.g., in a minimal Docker container or remote SSH session without port forwarding), it falls back to the "Remote/Headless Flow."

5.  **Local Machine Flow (with browser access):**
    a.  `smog` starts a local web server listening on `http://127.0.0.1:53682`. This URL is automatically configured as the redirect URI.
    b.  It generates the Google authorization URL with the necessary scopes (`https://www.googleapis.com/auth/gmail.send`).
    c.  It prints to the console:
    \`\`\`
    Your browser has been opened to visit:
    https://www.google.com/url?sa=E\&source=gmail\&q=https://accounts.google.com/o/oauth2/v2/auth?....

    ````
    If your browser is on a different machine, please open the URL above.
    Waiting for authorization...
    ```
    ````

    d.  It attempts to open the URL in the system's default browser.
    e.  The user authenticates with Google and grants permission.
    f.  Google redirects the browser back to `http://127.0.0.1:53682/?code=...`.
    g.  The local web server inside `smog` captures the authorization code from the request, sends a success page to the browser, and then shuts down.
    h.  `smog` exchanges the code for an access token and a refresh token.
    i.  The tokens are saved to the file specified by `GoogleTokenPath` in the config. A success message is printed.

6.  **Remote/Headless Flow (no browser access):**
    a.  `smog` generates the Google authorization URL.
    b.  It prints to the console:
    \`\`\`
    You are running on a remote machine. Please follow these steps:

    ````
    1. Open this URL on any machine with a web browser:
       https://accounts.google.com/o/oauth2/v2/auth?....

    2. Authenticate with Google and grant permissions.

    3. Google will show you an authorization code. Copy it.

    Enter the authorization code here:
    ```
    ````

    c.  `smog` waits for the user to paste the code and press Enter.
    d.  `smog` exchanges the pasted code for an access token and a refresh token.
    e.  The tokens are saved to the file specified by `GoogleTokenPath`. A success message is printed.

-----

### 5\. Standard Use Flow (`smog serve`)

1.  **Startup:** User executes `smog` or `smog serve`.
2.  **Config Load:** `smog` searches for `smog.conf` in default locations or uses the path from the `-c` flag.
3.  **Validation Checks:**
      * **Authorization:** Checks if the token file specified by `GoogleTokenPath` exists and is valid. If not, it prints an error message instructing the user to run `smog auth login` and exits.
      * **Default Password:** Checks if `SMTPPassword` is still set to the default `"smoggmos"`. If so, it prints a security warning and refuses to start.
4.  **Logging Setup:** Initializes logging based on `LogLevel` and `LogPath` from the configuration.
5.  **SMTP Server Start:**
      * Starts a TCP listener on the `SMTPPort`.
      * Logs a message: `smog SMTP relay listening on port <SMTPPort>`.
6.  **Connection Handling (per connection):**
    a.  Accepts an incoming client connection.
    b.  Logs the new connection from the client's IP address.
    c.  **IP Filtering:** If `AllowedSubnets` is configured, it checks if the client IP is within a permitted range. If not, the connection is immediately dropped.
    d.  **SMTP Handshake:** The SMTP server library handles the `HELO/EHLO`, `AUTH PLAIN/LOGIN` commands.
    e.  **Authentication:** It validates the provided `SMTPUser` and `SMTPPassword` against the config. On failure, it returns an SMTP error and closes the connection.
    f.  **Message Reception:** It accepts `MAIL FROM`, `RCPT TO`, and `DATA` commands. It respects `MessageSizeLimitMB` and will reject the message with an appropriate SMTP error code if it's too large.
    g.  **Message Parsing:** Once the `DATA` section is complete, the entire raw email message (including all headers, body, and attachments) is stored in a buffer as a single RFC 2822 formatted string/byte slice.
7.  **Gmail API Forwarding:**
    a.  The application uses the stored OAuth2 token to create an authenticated Gmail service client. If the access token is expired, the underlying Google library will automatically use the refresh token to get a new one.
    b.  The raw email message from the buffer is **base64url encoded**. This is critical for transparency, as it ensures the message is sent byte-for-byte.
    c.  The encoded message is placed into a `gmail.Message` object, specifically in the `Raw` field.
    d.  The message is sent via the `users.messages.send` API endpoint, addressed to the original recipient from the `RCPT TO` command.
8.  **Logging:**
      * A log entry is created detailing the transaction: `timestamp`, `client_ip`, `from_address`, `to_address`, `message_id` (from Gmail API response), and `status` (e.g., "SUCCESS" or "FAILED: \<error\>"). Minimal logging would just show this one line per message. Verbose logging would include the entire SMTP conversation.
9.  **Connection Close:** The SMTP connection is closed gracefully.

-----

### 6\. Code Structure (Go Project Layout)

```
/smog
├── cmd/
│   └── smog/
│       └── main.go         // Entry point, CLI command definitions (Cobra)
├── internal/
│   ├── app/
│   │   └── server.go       // Main application logic, orchestrates all components
│   ├── auth/
│   │   └── google.go       // Handles the entire OAuth2 flow, token storage
│   ├── config/
│   │   └── config.go       // Structs and functions for loading/validating config (Viper)
│   ├── gmail/
│   │   └── client.go       // Wrapper for Gmail API calls (sending messages)
│   ├── log/
│   │   └── logger.go       // Logging setup and helpers
│   └── smtp/
│       └── relay.go        // The SMTP server implementation and its backend logic
├── scripts/
│   ├── install-service.sh  // Installation script for Linux (systemd)
│   ├── install-service.ps1 // Installation script for Windows (as a Service)
│   └── build.sh            // Cross-platform build script
├── go.mod
├── go.sum
└── README.md
```

**Key Go Dependencies:**

  * **CLI Framework:** `github.com/spf13/cobra`
  * **Configuration:** `github.com/spf13/viper`
  * **SMTP Server:** `github.com/emersion/go-smtp` (A robust, well-maintained library)
  * **Google API:** `google.golang.org/api/gmail/v1`
  * **Google Auth:** `golang.org/x/oauth2/google`

-----

### 7. Configuration File Options (`smog.conf`)

The configuration file will use the TOML format for readability. A default file will be generated by `smog config create`.

```toml
# smog - SMTP to Gmail Relay Configuration File

# --- Logging Settings ---
# LogLevel: Set the detail level for logs. Options: "Disabled", "Minimal", "Verbose".
# Minimal logs one line per email transaction. Verbose logs the full SMTP conversation.
LogLevel = "Minimal"

# LogPath: Path to the log file. Platform-specific defaults are used if empty.
# Linux: /var/log/smog.log
# macOS: ~/Library/Logs/smog.log
# Windows: C:\ProgramData\smog\smog.log
LogPath = ""


# --- Google API Settings ---
# GoogleCredentialsPath: Absolute path to the credentials.json file downloaded from Google Cloud.
# This is required for the initial authorization.
GoogleCredentialsPath = "/etc/smog/credentials.json"

# GoogleTokenPath: Path to store the generated OAuth2 token.
# Smog needs read/write access to this file.
GoogleTokenPath = "/etc/smog/token.json"


# --- SMTP Server Settings ---
# SMTPUser: The username that SMTP clients must use to authenticate.
SMTPUser = "smog"

# SMTPPassword: The password that SMTP clients must use.
# IMPORTANT: Change this from the default value before running!
SMTPPassword = "smoggmos"

# SMTPPort: The TCP port for the SMTP server to listen on.
# Port 25 may require root privileges on Linux/macOS. Use a higher port like 587 or 2525.
SMTPPort = 2525

# MessageSizeLimitMB: The maximum email size (in Megabytes) to accept.
# Emails larger than this will be rejected.
MessageSizeLimitMB = 25


# --- Security Settings ---
# AllowedSubnets: A list of allowed client IP addresses or CIDR subnets.
# If this list is not empty, only clients from these IPs can connect.
# Example: ["192.168.1.50", "10.0.0.0/8"]
AllowedSubnets = []
```

