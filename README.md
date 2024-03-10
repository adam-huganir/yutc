# `yutc` ~~yet-unnamed-template-cli~~

`/jʌːtsi/`

It's as good a name as any i guess.

# yutc Command Line Application

`yutc` is a command line application developed in Go. It is designed to parse
and merge data files, and apply them to templates. The application supports
reading data from both local files and URLs, and can output the results to a
file or stdout.

## Installation

Download
the [latest build](https://github.com/adam-huganir/yutc/releases/latest) or any
other version from
the [releases page](https://github.com/adam-huganir/yutc/releases).
Put it somewhere on your computer. Add it to your path. If you are using this
you probably
already know how to do that.

## Usage

You can use `yutc` by passing it a list of templates along with various options:

```bash
yutc [flags] <template ...>
```

Here are the available options:

- `--data`: Specifies a data file to parse and merge. This can be a file or a
  URL. This option can be specified multiple times, and the inputs will be
  merged. If the `--stdin` option is set, the data will be read from stdin as
  well. Any `yaml` compatible file (including `json`) is allowed as input.

- `--output`: Specifies the output file or directory. If not provided, the
  output will be written to stdout. This is required if there are more than
  one template file.

- `--overwrite`: If set, existing files will be overwritten.

- `--shared`: Specifies templates to be shared across all templates in the
  template list. This can be a file or a URL. This option can be specified
  multiple times. This is useful for sharing common `define` blocks across
  multiple templates, they can go here.

- `--stdin`: If set, data will be read from stdin. If `--data` is used as well
  the input will be merged after the data files are read.

- `--stdin-first`: If set, the data files (if provided) will be loaded and
  merged first, and then the data from stdin will be merged.

- `--version`: If set, the version of the application will be printed and the
  application will exit.

For example, to parse and merge data from a file and apply it to a template, you
can use the following command:

## TODO: Examples


## Why?

I had very specific requirements
that [gomplate](https://github.com/hairyhenderson/gomplate) and others weren't
quite able to meet.
So really i just made this for myself at my day-job, but if anyone else
finds it useful, here it is.
Enjoy the weird niche features!

Others will likely be more actively maintained, and are rad so check them out!
