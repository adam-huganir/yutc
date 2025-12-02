# UV Python Project Example

This example demonstrates how to use `yutc` to generate a Python project that uses `uv` for package management.

## Usage

To generate the project, run the following command from the root of the `yutc` repository:

```bash
yutc -t examples/uv-python-project/pyproject.toml -d examples/uv-python-project/data.yaml -o .
```

This will generate a `pyproject.toml` file in the `uv-python-project` directory.

To install dependencies and run the project, you can use `uv`:

```bash
cd examples/uv-python-project
uv pip install -e .
```
