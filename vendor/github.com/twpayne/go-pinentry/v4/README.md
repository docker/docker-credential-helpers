# go-pinentry

[![PkgGoDev](https://pkg.go.dev/badge/github.com/twpayne/go-pinentry/v4)](https://pkg.go.dev/github.com/twpayne/go-pinentry/v4)

Package `pinentry` provides a client to [GnuPG's
pinentry](https://www.gnupg.org/related_software/pinentry/index.html).

## Key Features

* Support for all `pinentry` features.
* Idiomatic Go API.
* Well tested.

## Example

```go
	client, err := pinentry.NewClient(
		pinentry.WithBinaryNameFromGnuPGAgentConf(),
		pinentry.WithDesc("My description"),
		pinentry.WithGPGTTY(),
		pinentry.WithPrompt("My prompt:"),
		pinentry.WithTitle("My title"),
	)
	if err != nil {
		return err
	}
	defer client.Close()

	switch result, err := client.GetPIN(); {
	case pinentry.IsCancelled(err):
		fmt.Println("Cancelled")
	case err != nil:
		return err
	case result.PasswordFromCache:
		fmt.Printf("PIN: %s (from cache)\n", result.PIN)
	default:
		fmt.Printf("PIN: %s\n", result.PIN)
	}
```

## License

MIT
