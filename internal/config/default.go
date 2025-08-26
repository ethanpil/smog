package config

import "fmt"

var defaultConfig = fmt.Sprintf(`
# smog configuration file
# For more information, see: https://github.com/bodgit/smog

# LogLevel: Set the detail level for logs. Options: "Disabled", "Minimal", "Verbose".
LogLevel = "Minimal"

# LogPath: Path to the log file. Platform-specific defaults are used if empty.
# Examples:
#   Windows: C:\ProgramData\smog\smog.log
#   Linux: /var/log/smog/smog.log
#   macOS: /Library/Logs/smog/smog.log
LogPath = ""

# GoogleCredentialsPath: Absolute path to the credentials.json file downloaded from Google Cloud.
GoogleCredentialsPath = ""

# GoogleTokenPath: Path to store the generated OAuth2 token.
# If empty, a platform-specific default is used, e.g.,
#   Windows: %%APPDATA%%\smog\token.json
#   Linux: $HOME/.config/smog/token.json
#   macOS: $HOME/Library/Application Support/smog/token.json
GoogleTokenPath = ""

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
`, DefaultSMTPPassword)
