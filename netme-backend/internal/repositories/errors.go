package repositories

import "errors"

// ErrEmailTakenByOtherProvider is returned when a Google sign-in email matches
// an existing account that uses a different auth provider.
var ErrEmailTakenByOtherProvider = errors.New("email is already registered with a different login method")
