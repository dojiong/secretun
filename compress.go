package secretun

import (
	"bytes"
	zlib "compress/zlib"
	"fmt"
	"io"
)

type ZlibEncoder struct {
	level int
}

func (z *ZlibEncoder) Init(cfg map[string]interface{}) error {
	if ilevel, ok := cfg["level"]; !ok {
		z.level = 6
	} else if z.level, ok = ilevel.(int); !ok {
		return fmt.Errorf("zlib.level invalid type (int desired)")
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
