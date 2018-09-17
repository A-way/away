package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	ModeRule rune = 42
	ModeAway rune = 126
	ModePass rune = 64
	ModeDrop rune = 33
)

type Rule struct {
	Rule string
	Mode rune
}

func NewRule(rule string) (*Rule, error) {
	m := []rune(rule[0:1])[0]
	if m == ModeRule {
		return nil, fmt.Errorf("Rule mode must be in [~, !, @], %s", rule)
	}
	r := &Rule{
		Rule: rule[1:],
		Mode: m,
	}
	return r, nil
}

func (r *Rule) String() string {
	return string(r.Mode) + r.Rule
}

type Away struct {
	rules    sync.Map
	mode     rune
	filename string
}

func NewAway(mode rune, filename string) *Away {
	a := &Away{
		mode:     mode,
		filename: filename,
	}
	return a
}

func (a *Away) Mode() rune {
	return a.mode
}

func (a *Away) ChangeMode(m rune) {
	a.mode = m
}

func (a *Away) ResloveMode(addr *Addr) rune {
	if a.mode == ModeRule {
		s := addr.Host()
		for {
			if r, ok := a.rules.Load(s); ok {
				return r.(*Rule).Mode
			}
			i := strings.IndexRune(s, '.')
			if i >= 0 {
				s = s[i+1:]
			} else {
				return ModeRule
			}
		}
	} else {
		return a.mode
	}
}

func (a *Away) AddRule(r string) error {
	nr, err := NewRule(r)
	if err != nil {
		return err
	}
	a.rules.Store(nr.Rule, nr)
	return nil
}

func (a *Away) DeleteRule(r string) {
	a.rules.Delete(r[1:])
}

func (a *Away) SortRules() []string {
	rs := make([]string, 0, 30)
	a.rules.Range(func(_, r interface{}) bool {
		rs = append(rs, r.(*Rule).String())
		return true
	})
	sort.Slice(rs, func(i, j int) bool {
		return rs[i][1:] < rs[j][1:]
	})
	return rs
}

func (a *Away) LoadRules() (int, error) {
	file, err := os.Open(a.filename)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	i := 0
	s := bufio.NewScanner(file)
	for s.Scan() {
		rule := s.Text()
		nr, err := NewRule(rule)
		if err != nil {
			log.Warn(err)
		}
		a.rules.Store(nr.Rule, nr)
		i++
	}
	return i, nil
}

func (a *Away) WriteRules() error {
	tmp := a.filename + "." + strconv.Itoa(time.Now().Nanosecond())
	tmpf, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer tmpf.Close()

	rs := a.SortRules()
	for _, r := range rs {
		if _, e := fmt.Fprintln(tmpf, r); e != nil {
			return e
		}
	}

	return os.Rename(tmp, a.filename)
}
