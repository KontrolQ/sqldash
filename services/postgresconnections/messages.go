package postgresconnections

const (
	Invalid                = "Choose a label, a valid connection profile and a future expiry, or leave the expiry empty."
	Missing                = "No such connection."
	Refused                = "The connection could not be updated. It remains unavailable until recovery or revocation completes."
	AuthorityRefused       = "You no longer have permission to manage this connection profile."
	Created                = "The connection was created."
	Rotated                = "The password was rotated."
	Revoked                = "The connection was revoked."
	LostPassword           = "The password could not be displayed. Rotate it to receive a new password."
	SingleConnection       = "Each PostgreSQL database has one connection. Rotate its password instead of creating another user."
	PasswordUnavailable    = "The managed password is unavailable or has changed outside the dashboard. Rotate it to create a new password."
	PrimaryCannotBeRevoked = "Delete the database to remove its connection, or rotate the password to disconnect existing clients."
)
