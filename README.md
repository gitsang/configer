# Configer

Configer is an automatic configurator based on Viper and Cobra, supporting automatic or manual binding of environment variables and command line arguments. **Configure once, effective everywhere**

## Priority

Command-line arguments > Environment variables > Configuration files > Default values

## Feature

### Command-line arguments

| Feature                 | Supported |
| ----------------------- | --------- |
| Auto binding            | √         |
| Binding by tag          | √         |
| Configurable prefix     | √         |
| Configurable tokenizers | √         |
| Usage document          | √         |

### Environment variables

| Feature             | Supported |
| ------------------- | --------- |
| Auto binding        | √         |
| Binding by tag      | √         |
| Configurable prefix | √         |
| Configurable delim  | √         |

### Files

| Feature             | Supported |
| ------------------- | --------- |
| Auto Binding        | √         |
| Binding by tag      | -         |
| Configurable prefix | ×         |
| Configurable delim  | ×         |
| Multiple Files      | √         |
| YAML                | √         |
| JSON                | √         |

### Default

| Feature               | Supported |
| --------------------- | --------- |
| Load default from tag | √         |

## Example

See [./example/main.go](./example/main.go)

### Load from config file

```sh
go run . -c config.yaml
```

### See command-line arguments help

```sh
go run . --help
```

### Load from command-line arguments

```sh
go run . --example.log.level debug
```

### Load from environment variables

```sh
EXAMPLE_LOG_LEVEL=trace go run .
```
