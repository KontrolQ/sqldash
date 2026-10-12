package members

import "errors"

var Refused = errors.New("You cannot manage that user, role or database access.")
var Invalid = errors.New("Check the submitted details and try again.")
var LinkInvalid = errors.New("This link is invalid, expired, cancelled or already used.")
var PasswordWrong = errors.New("That is not your current password.")
var DependenciesInvalid = errors.New("SQL execution and token management require write permission.")
var Duplicate = errors.New("That username or email address is already in use.")

const InvitationSubject = "Your sqldash invitation"
const RecoverySubject = "Your sqldash recovery link"
const OwnershipSubject = "sqldash ownership transfer request"
