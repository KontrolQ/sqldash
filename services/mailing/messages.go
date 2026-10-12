package mailing

import "errors"

var SettingsInvalid = errors.New("Provide a host, valid port, TLS security mode and sender email.")
var DeliveryFailed = errors.New("Email could not be delivered. The link remains available to copy.")
var SecretUnavailable = errors.New("The saved SMTP credential could not be read.")
