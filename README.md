# fb is a tool to harvest content from public facebook accounts

## Setup

In order to run, you need to set the appID, appSecret and redirectURI variables with details from your facebook developer account. The first step is to get these details by setting up a facebook developer account. 

Once you have them, you can add them in three ways:

  - by editing the fb.go script, 
  - by defining "FB_APP_ID", "FB_APP_SECRET", "FB_REDIRECT" environment variables with those values
  - or by including a config.go file with contents like this:

```golang
package main

func init() {
    appID = "XXXXXXXXXXX"
    appSecret = "XXXXXXXXXXX"
    redirectURI = "https://www.example.com/"
}
```

Once done, use `go install` to build.

## Usage

You can harvest a feed with just `fb PUBLICACCOUNT`. 

Use the `-l` and `-c` flags to control whether like and comment data (user names etc.) is included with the feed harvest.

Use the `-v` and `-p` flags to get lists of video and photo content (for downloading separately using tools like youtubedl and wget). 



