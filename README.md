# `yutc` ~~yet-unnamed-template-cli~~

[![GitHub version](https://badge.fury.io/gh/adam-huganir%2Fyutc.svg)](https://badge.fury.io/gh/adam-huganir%2Fyutc)

It's as good a name as any i guess. `/jʌːtsi/`

`yutc` is a templating command line interface written in (surprise, surprise) Go.
It is designed to parse and merge data files, and apply them to templates.
The application supports reading data from both local files and URLs,
and can output the results to a file or stdout.

## Installation

Download the [latest build](https://github.com/adam-huganir/yutc/releases/latest) or any
other version from the [releases page](https://github.com/adam-huganir/yutc/releases).
Put it somewhere on your computer. Add it to your path. If you are using this you probably
already know how to do that.

## Usage

You can use `yutc` by passing it a list of templates along with various options:

```bash
yutc [OPTIONS]... [ <templates> ... ]
```
```
Flags:
  -d, --data stringArray               Data file to parse and merge. Can be a file or a URL. Can be specified multiple times and the inputs will be merged.
  -c, --common-templates stringArray   Templates to be shared across all arguments in template list. Can be a file or a URL. Can be specified multiple times.
  -o, --output string                  Output file/directory, defaults to stdout (default "-")
      --include-filenames              Exec any filenames with go templates
  -w, --overwrite                      Overwrite existing files
      --bearer-auth string             Bearer token for any URL authentication
      --basic-auth string              Basic auth for any URL authentication
      --version                        Print the version and exit
  -v, --verbose                        Verbose output
  -h, --help                           help for yutc
```

## Custom Template Functions


### `toYaml` and `mustToYaml`

`toYaml` is a custom template function that converts the input to a yaml representation.
similar to the `toYaml` in `helm`.

`mustToYaml` is also available, which will panic if the input cannot be converted to yaml.

```gotemplate
{{ . | toYaml }}
```
### `fromYaml` and `mustFromYaml`

`fromYaml` is a custom template function that converts the input to a go object.
similar to the `fromYaml` in `helm`.

`mustFromYaml` is also available, which will panic if the input cannot be converted to a go object.

```gotemplate
{{ fromYaml . | .SomeField | toString }}
```
### `yamlOptions`

`yamlOptions` allows you to set options for the yaml encoder. These settings will be global across all calls
to `toYaml` and `mustToYaml`. See [the documentation for goccy/go-yaml](https://pkg.go.dev/github.com/goccy/go-yaml#EncodeOption) for details.
`DecodeOption` support will be added in the future.

```gotemplate
{{ yamlOptions (dict "indent" 2) }}
{{ toYaml $someData }}
```
```
### `wrapText` and `wrapComment`

`wrapText` uses [textwrap](https://github.com/isbm/textwrap) to wrap text to a specified width.

`wrapComment` is a wrapper around `wrapText` that adds a comment character to the beginning of each line.

```gotemplate
{{ wrapText 80 .SomeText }}

{{ wrapComment "#" 80 .SomeText }}
```
### `toToml` and `mustToToml`

`toToml` is a custom template function that converts the input to a TOML representation.
Similar to `toYaml` but for TOML format.

`mustToToml` is also available, which will panic if the input cannot be converted to TOML.

```gotemplate
{{ . | toToml }}
```
### `fromToml` and `mustFromToml`

`fromToml` is a custom template function that converts TOML input to a go object.
Similar to `fromYaml` but for TOML format.

`mustFromToml` is also available, which will panic if the input cannot be converted to a go object.

```gotemplate
{{ fromToml . | .SomeField | toString }}
```
### File Functions: `fileGlob`, `fileStat`, `fileRead`, `fileReadN`

**`fileGlob`** - Returns a list of files matching a glob pattern.

**`fileStat`** - Returns file statistics as a map with keys like Mode, Size, ModTime, etc.

**`fileRead`** - Reads the entire contents of a file as a string.

**`fileReadN`** - Reads the first N bytes of a file as a string.

```gotemplate
{{- range fileGlob "*.txt" }}
File: {{ . }}
{{- end }}

{{- $stat := fileStat "myfile.txt" }}
Size: {{ $stat.Size }} bytes

{{ fileRead "config.txt" }}

{{ fileReadN 100 "largefile.txt" }}
```
### Path Functions: `pathAbsolute`, `pathIsDir`, `pathIsFile`, `pathExists`

**`pathAbsolute`** - Returns the absolute path of the given path.

**`pathIsDir`** - Returns true if the path is a directory.

**`pathIsFile`** - Returns true if the path is a file.

**`pathExists`** - Returns true if the path exists.

```gotemplate
{{ pathAbsolute "./relative/path" }}

{{- if pathExists "myfile.txt" }}
{{- if pathIsFile "myfile.txt" }}
File exists: {{ pathAbsolute "myfile.txt" }}
{{- end }}
{{- end }}

{{- if pathIsDir "mydirectory" }}
Directory found!
{{- end }}
```
### `type`

`type` returns the Go type of the given value as a string.

```gotemplate
{{ type . }}
{{ type "hello" }}
{{ type 42 }}
```
### `include`

`include` allows you to include and render other templates as text within the current template.
This is the exact same as the `include` function in Helm.

```gotemplate
{{ include "shared-template" . }}
```
### `tpl`

`tpl`, also from Helm

```gotemplate
{{ tpl $my_template . }}

## Examples


### Merging many yaml/json files together and outputting them to
a file

```bash
yutc -o patch.yaml \
     -d ./talosPatches/controlplane-workloads.yaml \
     -d talosPatches/disable-cni.yaml \
     -d talosPatches/disable-discovery.yaml \
     -d talosPatches/install-disk.yaml \
     -d talosPatches/kubelet.yaml \
     -d talosPatches/local-storage.yaml \
     -d talosPatches/names.yaml \
      <(echo "{{ . | toYaml }}")
```

alternate form using matching

```bash
    yutc -o patch.yaml \
         --data ./talosPatches \
         --data-match './talosPatches/.*\.yaml' \
          <(echo "{{ . | toYaml }}")
```
### Listing files in a directory

For some reason you want to list the files in a directory and embed them in a file in a custom format:

```template
{{- $files := fileGlob "./*/*" -}}
{{- range $path := $files }}
{{- $stat := fileStat $path }}
{{- $username := (env "USERNAME" | default (env "USER") )}}
{{- $usernameFString := printf "%s%d%s  " "%-" (len $username) "s"}}
 {{ printf "%-12s" $stat.Mode }}{{ printf $usernameFString $username }}{{ pathAbsolute $path}}
{{- end }}
```
```
 -rw-rw-rw-  adam  C:\Users\adam\code\yet-unnamed-template-cli\.git\COMMIT_EDITMSG
 -rw-rw-rw-  adam  C:\Users\adam\code\yet-unnamed-template-cli\.git\FETCH_HEAD
 -rw-rw-rw-  adam  C:\Users\adam\code\yet-unnamed-template-cli\.git\HEAD
 -rw-rw-rw-  adam  C:\Users\adam\code\yet-unnamed-template-cli\.git\ORIG_HEAD
 -rw-rw-rw-  adam  C:\Users\adam\code\yet-unnamed-template-cli\.git\config
 -rw-rw-rw-  adam  C:\Users\adam\code\yet-unnamed-template-cli\.git\description
 drwxrwxrwx  adam  C:\Users\adam\code\yet-unnamed-template-cli\.git\hooks
 ........
 ```
### Merging 2 data files and applying them to a template

```pwsh
 yutc --data .\testFiles\data\data1.yaml --data .\testFiles\data\data2.yaml .\testFiles\templates\simpleTemplate.tmpl
```

```
JSON representation of the merged input:
---
{
  "ditto": [
    "woohooo",
    "yipeee"
  ],
  "dogs": [],
  "thisIsNew": 1000,
  "thisWillMerge": {
    "value23": 23,
    "value24": 24
  }
}
---
Yaml representation:
---
ditto:
    - woohooo
    - yipeee
dogs: []
thisIsNew: 1000
thisWillMerge:
    value23: 23
    value24: 24
---
```
### Using top-level keys to store data from files separately

```bash
 yutc --data "key=data1,src=./testFiles/data/data1.yaml" \
      --data "key=data2,src=./testFiles/data/data3.json" \
       ./testFiles/templates/simpleTemplate.tmpl
```
We see below that we are able to specify a key for each data file, and the data is not merged at the top level.
```
Unmerged data from data 1: {"dogs":[{"breed":"Labrador","name":"Fido","owner":{"name":"John Doe"},"vaccinations":["rabies"]}],"thisWillMerge":{"value23":"not 23","value24":24}}
Unmerged data from data 2: {"ditto":["woohooo","yipeee"],"dogs":[],"thisIsNew":1000,"thisWillMerge":{"value23":23}}
```
### Rendering this documentation

See README.data.yaml and README.md.tmpl for the source data and template

## Why?

I had very specific requirements
that [gomplate](https://github.com/hairyhenderson/gomplate), [gucci](https://github.com/noqcks/gucci), and
others weren't quite able to meet.
Both of those a great apps, and if you
So really i just made this for myself at my day-job, but if anyone else
finds it useful, here it is.
Enjoy the weird niche features!


Others will likely be more actively maintained, and are rad so check them out!
