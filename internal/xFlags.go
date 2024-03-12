package internal

import (
	"flag"
	"strings"
)

type XFlag interface {
	String() string
	New()
	NewVar()
}

type StringFlag struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
	Default *string  `json:"default"`
	Help    string   `json:"help"`
}

func (sf *StringFlag) String() string {
	long := "--" + sf.Name
	flagArgs := []string{long}
	if len(sf.Aliases) > 0 {
		for _, alias := range sf.Aliases {
			flagArgs = append(flagArgs, "-"+alias)
		}
	}
	return long + " " + strings.Join(flagArgs, " ")
}

func (sf *StringFlag) New() {
	flag.String(sf.Name, *sf.Default, sf.Help)
	for _, alias := range sf.Aliases {
		flag.String(alias, *sf.Default, sf.Help)
	}
}

func (sf *StringFlag) NewVar(v *string) {
	flag.StringVar(v, sf.Name, *sf.Default, sf.Help)
	for _, alias := range sf.Aliases {
		flag.StringVar(v, alias, *sf.Default, sf.Help)
	}
}

type StringSliceFlag struct {
	Name    string             `json:"name"`
	Aliases []string           `json:"aliases"`
	Default RepeatedStringFlag `json:"default"`
	Help    string             `json:"help"`
}

func (ssf *StringSliceFlag) String() string {
	long := "--" + ssf.Name
	flagArgs := []string{long}
	if len(ssf.Aliases) > 0 {
		for _, alias := range ssf.Aliases {
			flagArgs = append(flagArgs, "-"+alias)
		}
	}
	return long + " " + strings.Join(flagArgs, " ")
}

func (ssf *StringSliceFlag) New() {
	flag.String(ssf.Name, strings.Join(ssf.Default, ","), ssf.Help)
	for _, alias := range ssf.Aliases {
		flag.String(alias, strings.Join(ssf.Default, ","), ssf.Help)
	}
}

func (ssf *StringSliceFlag) NewVar(v *RepeatedStringFlag) {
	flag.Var(v, ssf.Name, ssf.Help)
	for _, alias := range ssf.Aliases {
		flag.Var(v, alias, ssf.Help)
	}
}

type BoolFlag struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
	Default bool     `json:"default"`
	Help    string   `json:"help"`
}

func (bf *BoolFlag) String() string {
	long := "--" + bf.Name
	flagArgs := []string{long}
	if len(bf.Aliases) > 0 {
		for _, alias := range bf.Aliases {
			flagArgs = append(flagArgs, "-"+alias)
		}
	}
	return long + " " + strings.Join(flagArgs, " ")
}

func (bf *BoolFlag) New() {
	flag.Bool(bf.Name, bf.Default, bf.Help)
	for _, alias := range bf.Aliases {
		flag.Bool(alias, bf.Default, bf.Help)
	}
}

func (bf *BoolFlag) NewVar(v *bool) {
	flag.BoolVar(v, bf.Name, bf.Default, bf.Help)
	for _, alias := range bf.Aliases {
		flag.BoolVar(v, alias, bf.Default, bf.Help)
	}
}
