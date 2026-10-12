package access

import "errors"

var Unavailable = errors.New("Your access could not be checked. Please try again.")
var Refused = errors.New("You do not have permission to perform that action.")
