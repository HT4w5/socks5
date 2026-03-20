package payload

import "errors"

var (
	ErrTooShort               = errors.New("datagram too short")
	ErrUnsupportedAddressType = errors.New("address type not supported")
)
