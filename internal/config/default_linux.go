//go:build linux
// +build linux

package config

import "fmt"

var defaultConfig = fmt.Sprintf(`
# smog - SMTP to Gmail Relay Configuration File

# --- Logging Settings ---
# LogLevel: Set the detail level for logs. Options: "Disabled", "Minimal", "Verbose".
LogLevel = "Minimal"

# LogPath: Path to the log file. If empty, logs are written to platform-specific locations.
# Linux: /var/log/smog/smog.log
LogPath = ""


# --- Google API Settings ---
# GoogleCredentialsPath: Absolute path to the credentials.json file downloaded from Google Cloud.
# This is required for the initial authorization.
GoogleCredentialsPath = "/etc/smog/credentials.json"

# GoogleTokenPath: Path to store the generated OAuth2 token.
# If left empty, it defaults to a path in the user's config directory (e.g., ~/.config/smog/token.json).
GoogleTokenPath = ""


# --- SMTP Server Settings ---
# SMTPUser: The username that SMTP clients must use to authenticate.
SMTPUser = "smog"

# SMTPPassword: The password that SMTP clients must use.
SMTPPassword = "%s"

# SMTPPort: The TCP port for the SMTP server to listen on.
SMTPPort = 2525

# MessageSizeLimitMB: The maximum email size (in Megabytes) to accept.
MessageSizeLimitMB = 10

# AllowedSubnets: A list of allowed client IP addresses or CIDR subnets.
# Example: AllowedSubnets = ["192.168.1.0/24", "127.0.0.1"]
AllowedSubnets = []


# --- Advanced SMTP Settings ---
# ReadTimeout: The maximum duration in seconds for reading an entire SMTP request.
# Helps prevent slow client attacks (e.g., Slowloris).
ReadTimeout = 10

# WriteTimeout: The maximum duration in seconds for writing an entire SMTP response.
WriteTimeout = 10

# MaxRecipients: The maximum number of recipients allowed for a single email.
# The server will reject messages with more recipients than this value.
MaxRecipients = 50

# AllowInsecureAuth: Allow insecure authentication methods like AUTH PLAIN over
# non-TLS connections. This is not recommended and should only be enabled for
# legacy clients that do not support STARTTLS.
AllowInsecureAuth = true
`, DefaultSMTPPassword)
