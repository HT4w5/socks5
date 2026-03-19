package server

import "io"

func copy(dst io.Writer, src io.Reader, errCh chan error) {
	_, err := io.Copy(dst, src)
	errCh <- err
}
