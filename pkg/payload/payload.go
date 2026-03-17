package payload

import "io"

type Payload interface {
	Read(r io.Reader) error
	Write(w io.Writer) error
}
