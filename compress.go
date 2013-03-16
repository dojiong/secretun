package secretun

import (
	"bytes"
	zlib "compress/zlib"
	"io"
)

type ZlibEncoder struct {
	level int
}

func (z *ZlibEncoder) Init(cfg Config) error {
	if err := cfg.Get("level", &z.level); err != nil {
		if err.(*ConfigError).Errno == ErrMissing {
			z.level = 6
		} else {
			return err
		}
	}

	return nil
}

func (z *ZlibEncoder) Encode(data []byte) ([]byte, error) {
	w := new(bytes.Buffer)
	dec, err := zlib.NewWriterLevel(w, z.level)
	if err != nil {
		return nil, err
	}
	defer dec.Close()

	if _, err = dec.Write(data); err != nil {
		return nil, err
	}
	if err = dec.Flush(); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (z *ZlibEncoder) Decode(data []byte) ([]byte, error) {
	w := new(bytes.Buffer)
	if r, err := zlib.NewReader(bytes.NewBuffer(data)); err != nil {
		return nil, err
	} else {
		if _, err = io.Copy(w, r); err != nil {
			if err != io.ErrUnexpectedEOF && err != io.EOF {
				return nil, err
			}
		}
	}
	return w.Bytes(), nil
}

func init() {
	RegisterEncoder("zlib", ZlibEncoder{})
}
