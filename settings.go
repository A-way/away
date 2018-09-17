package main

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Settings struct {
	Remote  string
	Passkey string
	Port    string
}

func ExistSetting(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return false
}

func ReadSettings(filename string) (rs *Settings, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var obj = new(Settings)
	dc := gob.NewDecoder(f)
	if err := dc.Decode(obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func WriteSettings(rs *Settings, filename string) (err error) {
	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return err
	}

	tmp := filename + "." + strconv.Itoa(time.Now().Nanosecond())
	tmpf, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer tmpf.Close()

	ec := gob.NewEncoder(tmpf)
	if err := ec.Encode(rs); err != nil {
		return err
	}

	if err = tmpf.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, filename)
}
