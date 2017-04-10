# fb is a tool to harvest content from public facebook accounts

## Setup

In order to run, you need to set the appID, appSecret and redirectURI variables with details from your facebook developer account. You can do this by editing the fb.go script or by including your own config.go file with contents like this:

package main

func init() {
    appID = "XXXXXXXXXXX"
    appSecret = "XXXXXXXXXXX"
    redirectURI = "https://www.example.com/"
}

Once done, use `go install` to build.

## Usage

You can harvest a feed with just `fb PUBLICACCOUNT`. 

Use the `-l` and `-c` flags to control whether like and comment data (user names etc.) is included with the feed harvest.

Use the `-v` and `-p` flags to get lists of video and photo content (for downloading separately using tools like youtubedl and wget). 



