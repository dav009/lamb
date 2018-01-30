![](https://github.com/dav009/lamb/raw/master/sheep.png)

# Lamb

Lamb is a support tool when developing AWS Lambda Functions.


if you already have many lambda functions or you are working in a project using 
a few of those, most likely you already have lots of annoying cloudwatch tabs on your browser.

Lamb aggregates Cloudwatch logs and a list of lambda functions, so that you can get quick cloudwatch updates on your terminal.
This is useful when you are developing + testing your lambda function integrations.

## Demo

[![asciicast](https://asciinema.org/a/fP11ggkj7bbkYXjLmh9AsJwvL.png)](https://asciinema.org/a/fP11ggkj7bbkYXjLmh9AsJwvL)

[![asciicast](https://asciinema.org/a/hB1TVgZSwlKkiWiFkSDdmD0wT.png)](https://asciinema.org/a/hB1TVgZSwlKkiWiFkSDdmD0wT)



## Installation

`go get github.com/aws/aws-sdk-go`
`go get github.com/jroimartin/gocui`
`go get github.com/dav009/lamb`
`go install github.com/dav009/lamb`

## Usage


1. make sure you have loaded your aws credentials
2. do `lamb`
3. select your lambda and press enter to load/re-load a lambda function logs
4. press `tab` to switch to the log buffer to scroll up and down
5. press `tab` again to go back to the lambda-list buffer
6. press `enter` at anytime to refresh logs

if you have an overwhelming number of lambda functions you can do:
`lamb projectname` to only list lambdas containing `projectname` in their name
