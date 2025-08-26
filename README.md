# smog
A cross-platform SMTP to Gmail API relay tool.

## DESCRIPTION
smog acts as a local SMTP server that accepts authenticated email submissions and relays them through the Google Gmail API using OAuth2. It is designed for legacy systems that can only send email via SMTP  ut need to integrate with a modern Gmail account.

Before first use, the administrator must run `smog auth login` to authorize the application with Google. The `GoogleCredentialsPath` configuration option must be set to the path of the `credentials.json` file downloaded from the Google Cloud Console.

## USAGE
     smog [global flags] [command]

## GLOBAL FLAGS
     -c, --config <path> - Specify a custom path to the smog.conf configuration file.

     -v, --verbose
           Enable verbose logging output to the console.

     -s, --silent
           Disable all console output except for fatal errors.

## COMMANDS
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

## FILES
     smog.conf is the default configuration file. It is searched for in the
     current directory, and in the following platform-specific locations:

     Linux
           /etc/smog/smog.conf
           /var/lib/smog/smog.conf

     macOS
           /Library/Application Support/smog/smog.conf

     Windows
           C:\ProgramData\smog\smog.conf

## EXAMPLES
     Run the SMTP server with the default configuration:
           $ smog serve

     Authorize smog with your Google account:
           $ smog auth login

     Run the server using a custom configuration file and verbose output:
           $ smog -v -c /etc/custom/smog.conf serve

## LICENSE
    Copyright (C) 2025 Ethan Piliavin

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.

## AUTHOR / CREDITS
    Ethan Piliavin - Architect and Vibe Coder :)
    Google Gemini - Planning and Documentation
    Google Jules - Engineering
    
