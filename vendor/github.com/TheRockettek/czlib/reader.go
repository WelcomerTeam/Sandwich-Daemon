// Pulled from https://github.com/youtube/vitess 229422035ca0c716ad0c1397ea1351fe62b0d35a
// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package czlib

import "io"

// err starts out as nil
// we will call inflateEnd when we set err to a value:
// - whatever error is returned by the underlying reader
// - io.EOF if Close was called
type reader struct {
	r      io.Reader
	in     []byte
	strm   *Zstream
	xstrm  bool
	err    error
	skipIn bool
	isEOF  bool
}

type Resetter interface {
	Reset(io.Reader)
}

// NewReader creates a new io.ReadCloser. Reads from the returned io.ReadCloser
//read and decompress data from r. The implementation buffers input and may read
// more data than necessary from r.
// It is the caller's responsibility to call Close on the ReadCloser when done.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	return NewReaderBuffer(r, DEFAULT_COMPRESSED_BUFFER_SIZE)
}

// NewReaderBuffer has the same behavior as NewReader but the user can provide
// a custom buffer size.
func NewReaderBuffer(r io.Reader, bufferSize int) (io.ReadCloser, error) {
	strm, err := NewZstream()
	if err != nil {
		return nil, err
	}
	return newReader(r, strm, true, bufferSize)
}

// NewStreamReader has the same behavior as NewReader but the user can provide
// a custom zstream context.
func NewStreamReader(r io.Reader, strm *Zstream) (io.ReadCloser, error) {
	return NewStreamReaderBuffer(r, strm, DEFAULT_COMPRESSED_BUFFER_SIZE)
}

// NewStreamReaderBuffer has the same behavior as NewStreamReader but the user
// can provide a custom zstream context and buffer size.
func NewStreamReaderBuffer(r io.Reader, strm *Zstream, bufferSize int) (io.ReadCloser, error) {
	return newReader(r, strm, false, bufferSize)
}

func newReader(r io.Reader, strm *Zstream, xstrm bool, bufferSize int) (io.ReadCloser, error) {
	z := &reader{in: make([]byte, bufferSize), strm: strm, xstrm: xstrm}
	z.Reset(r)
	return z, nil
}

func (z *reader) Reset(r io.Reader) {
	z.r = r
	z.err = nil
	z.isEOF = false
	z.strm.Reset()
}

func (z *reader) Read(p []byte) (int, error) {
	if z.err != nil {
		return 0, z.err
	}

	if len(p) == 0 {
		return 0, nil
	}

	// read and deflate until the output buffer is full
	z.strm.setOutBuf(p, len(p))

	for {
		// if we have no data to inflate, read more
		if !z.skipIn && z.strm.availIn() == 0 {
			var n int
			if z.isEOF {
				z.err = io.EOF
				return 0, z.err
			}
			n, z.err = z.r.Read(z.in)

			// If we got data and EOF, pretend we didn't get the
			// EOF.  That way we will return the right values
			// upstream.  Note this will trigger another read
			// later on, that should return (0, EOF).
			if z.err == io.EOF {
				z.isEOF = true

				z.err = nil
			}

			// FIXME(alainjobart) this code is not compliant with
			// the Reader interface. We should process all the
			// data we got from the reader, and then return the
			// error, whatever it is.
			if z.err != nil && z.err != io.EOF {
				if z.xstrm {
					z.strm.inflateEnd()
				}
				return 0, z.err
			}

			z.strm.setInBuf(z.in, n)
		} else {
			z.skipIn = false
		}

		// inflate some
		flush := zNoFlush
		if z.isEOF {
			flush = zSyncFlush
		}

		ret, err := z.strm.inflate(flush)
		if err != nil {
			if z.xstrm {
				z.strm.inflateEnd()
			}
			z.err = err
			return 0, z.err
		}

		// if we read something, we're good
		have := len(p) - z.strm.availOut()
		if have > 0 {
			z.skipIn = ret == Z_OK && z.strm.availOut() == 0
			return have, z.err
		}
	}
}

// Close closes the Reader. It does not close the underlying io.Reader.
func (z *reader) Close() error {
	if z.err != nil {
		if z.err != io.EOF {
			return z.err
		}
		return nil
	}
	if z.xstrm {
		z.strm.inflateEnd()
	}
	z.err = io.EOF
	return nil
}
