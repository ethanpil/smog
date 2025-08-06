smog is a cross platform tool written in Go which is an SMTP relay. The tool acts like an SMTP server, accepting incoming SMTP connections on a specified port with configured username and password authentication. When an email is received via SMTP it will then send it to the original destination address via Google's GMAIL API via the Oauth2 requirements.

smog is run on the command-line, executed with the command `smog` as well as the additional parameters needed to perform the functions requested. Once properly configured, it can also be setup as a service.

Before using SMOG, the administrator needs to setup SMOG by providing the configuration settings. Additionally, the user needs to go through the authorization flow to authorize the application for use by Google and to store the user's access and refresh tokens. Once this has been accomplished then standard operation can commence. The authorization flow should be done completely VIA CLI. Use the rclone authorization options and process as a guide. If a URL is needed then we need to be able to authenticate the URL locally or remotely like rclone remote authorization.

When the application starts - check if the authorization flow has been completed. If not then we need to work through the authorization. Provide the callback URL and instructions via the terminal to complete this interactively. There should be an option to delete the authorization. Then we can reauthorize.

If the configuration still contains the default SMTP password, refuse to continue with an error message until a new SMTP password is set. Configuration should allow SMTP login restriction by IP Address or subnet.

Now, listen on SMTPPort for incoming SMTP connections. With a valid SMTPUser and SMTPPassword, accept a message. Forward this message via the Gmail API to the original recipient address. Include all message content from, to and subject plus attachments as received by the SMTP server. The email data formatted for the API and sent to Gmail should be completely transparent with no trace of information modification or evidence from the intermediary SMTP server. Log the entire interaction per the log specifications.

Default configuration settings will be searched for by smog upon run. On linux system the configuration will be in /etc/smog.conf on windows in ~/smog.conf on MacOS and in the application's executable folder on Windows. The config file path can be specified with -c for example smog -c /etc/smog/smog.conf

There should be options for verbose and silent operation, as well as verbose and minimal logging with smart default locations for logfiles. 

Configuration File should have the following settings at minimum: (Add any that may be necessary)
 - LogLevel ;Disabled, Minimal or Verbose
 - LogPath ;Path to log file with sensible defaults per platform
 - GoogleCredientialsPath ;Path to credentials.json for API authorization
 - GoogleTokenPath ;Path to search for and store OAuth token for ongoing API usage 
 - SMTPUser  ;The username which SMTP clients should connect with (Default: smog)
 - SMTPPassword ;The password which SMTP clients should connect with (Default: smoggmos) 
 - SMTPPort ;The port which SMTP clients should connect with (Default: 25)
 - MessageSizeLimitMB ; Reject all incoming messages larger than this value in Megabytes (Default 10)

Create a development plan for this application, including the command line options, man page, authorization flow, use flow, code structure and configuration file options to make the application robust whole flexible and configurable.
