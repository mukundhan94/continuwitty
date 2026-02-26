package mcptokens

import "errors"

var (
	errTokenPrefixMismatch           = errors.New("token prefix mismatch")
	errTokenPayloadFormatMismatch    = errors.New("token payload format mismatch")
	errTokenIdentifierLengthMismatch = errors.New("token identifier length mismatch")
	errTokenIdentifierInvalid        = errors.New("token identifier invalid")
	errTokenSecretMissing            = errors.New("token secret missing")
)
