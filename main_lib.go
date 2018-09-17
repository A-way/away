// +build lib

package main

/*
#include <stdio.h>
#include <stdlib.h>
#include "Away.h"

static inline char** away_rules_alloc(int count) {
	return malloc(count  * sizeof(char*));
}

static inline void away_rules_set(char **rs, int index, char *r) {
	rs[index] = r;
}
*/
import "C"

import (
	"log/syslog"
	"os"
	"os/signal"
	"path"
	"syscall"
	"unsafe"

	log "github.com/sirupsen/logrus"
	slog "github.com/sirupsen/logrus/hooks/syslog"
)

func main() {
}

const AppName = "Away"

var away *Away
var dataPath string

//export away_initialize
func away_initialize(dir *C.char) C.int {
	signal.Ignore(syscall.SIGPIPE) // https://github.com/golang/go/issues/17393

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, DisableColors: true, TimestampFormat: "2006/01/02 15:04:05.000"})
	if h, err := slog.NewSyslogHook("", "", syslog.LOG_INFO, AppName); err == nil {
		log.AddHook(h)
	}

	p := C.GoString(dir)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		log.Error(err)
		return -1
	}
	dataPath = p

	rulefile := path.Join(dataPath, "rules")
	away = NewAway(ModeRule, rulefile)
	n, err := away.LoadRules()
	if err != nil {
		log.Error(err)
		return -1
	} else {
		log.Infof("Initilize [%d] rules.", n)
	}

	return 0
}

var socksSrv *SocksSrv

func startSocksSrv(s *Settings, a *Away) error {
	if socksSrv != nil {
		socksSrv.Stop()
	}

	srv, err := NewSocksSrv(s, a)
	if err != nil {
		return err
	}
	socksSrv = srv
	go socksSrv.Start()
	return nil
}

//export away_start
func away_start() C.int {
	s, err := ReadSettings(settingsFilename())
	if err != nil {
		log.Error(err)
		return -1
	}

	if err := startSocksSrv(s, away); err != nil {
		log.Error(err)
		return -1
	}
	return 0
}

func settingsFilename() string {
	return path.Join(dataPath, "settings")
}

//export away_settings_exist
func away_settings_exist() C.int {
	if ExistSetting(settingsFilename()) {
		return 1
	}
	return 0
}

//export away_settings_get
func away_settings_get(s *C.struct_settings) {
	if rs, err := ReadSettings(settingsFilename()); err == nil {
		s.remote = C.CString(rs.Remote)
		s.passkey = C.CString(rs.Passkey)
		s.port = C.CString(rs.Port)
	}
}

//export away_settings_change
func away_settings_change(s C.struct_settings) C.int {
	st := &Settings{
		Remote:  C.GoString(s.remote),
		Passkey: C.GoString(s.passkey),
		Port:    C.GoString(s.port),
	}
	if err := WriteSettings(st, settingsFilename()); err != nil {
		log.Errorln(err)
		return -1
	}

	if err := startSocksSrv(st, away); err != nil {
		log.Error(err)
		return -1
	}
	return 0
}

//export away_settings_free
func away_settings_free(s *C.struct_settings) {
	C.free(unsafe.Pointer(s.remote))
	C.free(unsafe.Pointer(s.passkey))
	C.free(unsafe.Pointer(s.port))
}

//export away_rule_add
func away_rule_add(r *C.char) C.int {
	rule := C.GoString(r)
	if err := away.AddRule(rule); err != nil {
		log.Errorf("fail to add rule %s, %s", rule, err)
		return -1
	}
	if err := away.WriteRules(); err != nil {
		log.Errorf("fail to add rule %s, %s", rule, err)
		return -1
	}
	return 0
}

//export away_rule_del
func away_rule_del(r *C.char) C.int {
	rule := C.GoString(r)
	away.DeleteRule(rule)
	if err := away.WriteRules(); err != nil {
		log.Errorf("fail to delete rule %s, %s", rule, err)
		return -1
	}
	return 0
}

//export away_rules_get
func away_rules_get() **C.char {
	rules := away.SortRules()
	l := len(rules)
	if l == 0 {
		return nil
	}
	rs := C.away_rules_alloc(C.int(l + 1))
	for i, r := range rules {
		C.away_rules_set(rs, C.int(i), C.CString(r))
	}
	C.away_rules_set(rs, C.int(l), nil)
	return rs
}

//export away_mode_change
func away_mode_change(m C.enum_away_mode) C.int {
	away.ChangeMode(rune(m))
	return 0
}
