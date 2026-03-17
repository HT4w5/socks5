package payload

import (
	"bufio"
)

type Payload interface {
	Read(br *bufio.Reader) error
	Write(bw *bufio.Writer) error
}
