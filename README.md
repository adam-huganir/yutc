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
Usage of yutc:
  -c, --common-templates string        Templates to be shared across all arguments in template list. Can be a file or a URL. Can be specified multiple times.
  -d, --data string                    Data file to parse and merge. Can be a file or a URL. Can be specified multiple times and the inputs will be merged.
  -o, --output string                  Output file/directory, defaults to stdout (default "-")
  -w, --overwrite                      Overwrite existing files
      --version                        Print the version and exit
```

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
     -d talosPatches/local-storage.yaml \\
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
