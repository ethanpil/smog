//go:build darwin
// +build darwin

package config

import "fmt"

var defaultConfig = fmt.Sprintf(`
# smog configuration file
# For more information, see: https://github.com/bodgit/smog

# LogLevel: Set the detail level for logs. Options: "Disabled", "Minimal", "Verbose".
LogLevel = "Minimal"

# LogPath: Path to the log file. Platform-specific defaults are used if empty.
# Example: /Library/Logs/smog/smog.log
LogPath = ""

# GoogleCredentialsPath: Absolute path to the credentials.json file downloaded from Google Cloud.
GoogleCredentialsPath = "/Library/Application Support/smog/credentials.json"

# GoogleTokenPath: Path to store the generated OAuth2 token.
GoogleTokenPath = "/Library/Application Support/smog/token.json"

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

# ReadTimeout: The maximum duration in seconds for reading the entire request.
ReadTimeout = 10

# WriteTimeout: The maximum duration in seconds for writing the response.
WriteTimeout = 10

# MaxRecipients: The maximum number of recipients for a single email.
MaxRecipients = 50

# AllowInsecureAuth: Allow insecure authentication methods.
AllowInsecureAuth = true
`, DefaultSMTPPassword)
