package inmemory

import (
	"bytes"
	"compress/lzw"
	"encoding/json"
	"fmt"
	"io"
)

type db[T any] struct {
	storage map[string][]byte
}

func NewDB[T any]() *db[T] {
	return &db[T]{
		storage: make(map[string][]byte),
	}
}

func (d *db[T]) Get(key string) (*T, error) {
	data, ok := d.storage[key]
	if !ok {
		return nil, fmt.Errorf("cannot find entry")
	}
	return decompressData[T](data)
}

func (d *db[T]) Delete(key string) {
	delete(d.storage, key)
}

func (d *db[T]) Keys() []string {
	keys := make([]string, 0, len(d.storage))
	for k := range d.storage {
		keys = append(keys, k)
	}
	return keys
}

func (d *db[T]) Store(key string, obj T) error {
	data, err := compressData[T](obj)
	if err != nil {
		return err
	}
	d.storage[key] = data
	return nil
}

func compressData[T any](obj T) ([]byte, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	com := lzw.NewWriter(&buf, lzw.LSB, 8)

	_, err = com.Write(data)
	if err != nil {
		return nil, err
	}

	err = com.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decompressData[T any](data []byte) (*T, error) {
	r := bytes.NewReader(data)
	decomp := lzw.NewReader(r, lzw.LSB, 8)

	data, err := io.ReadAll(decomp)
	if err != nil {
		return nil, err
	}

	err = decomp.Close()
	if err != nil {
		return nil, err
	}

	var obj T
	err = json.Unmarshal(data, &obj)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}
