package internal

import (
	yutc "github.com/adam-huganir/yutc/pkg"
	"text/template"
)

var FuncMap = template.FuncMap{
	"toYaml":       yutc.ToYaml,
	"fromYaml":     yutc.FromYaml,
	"mustToYaml":   yutc.MustToYaml,
	"mustFromYaml": yutc.MustFromYaml,
	"toToml":       yutc.ToToml,
	"fromToml":     yutc.FromToml,
	"mustToToml":   yutc.MustToToml,
	"mustFromToml": yutc.MustFromToml,
	// "stringMap":    yutc.stringMap, // not imp
	"wrapText":     yutc.WrapText,
	"wrapComment":  yutc.WrapComment,
	"fileGlob":     yutc.PathGlob,
	"fileStat":     yutc.PathStat,
	"fileRead":     yutc.FileRead,
	"fileReadN":    yutc.FileReadN,
	"type":         yutc.TypeOf,
	"pathAbsolute": yutc.PathAbsolute,
	"pathIsDir":    yutc.PathIsDir,
	"pathIsFile":   yutc.PathIsFile,
	"pathExists":   yutc.PathExists,
}
