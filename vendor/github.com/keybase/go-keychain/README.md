# Go Keychain

[![Build Status](https://github.com/keybase/go-keychain/actions/workflows/ci.yml/badge.svg)](https://github.com/keybase/go-keychain/actions)

A library for accessing the Keychain for macOS, iOS, and Linux in Go (golang).

Requires macOS 10.9 or greater and iOS 8 or greater. On Linux, communicates to
a provider of the DBUS SecretService spec like gnome-keyring or ksecretservice.

```go
import "github.com/keybase/go-keychain"
```

## Mac/iOS Usage

The API is meant to mirror the macOS/iOS Keychain API and is not necessarily idiomatic go.

#### Add Item

```go
item := keychain.NewItem()
item.SetSecClass(keychain.SecClassGenericPassword)
item.SetService("MyService")
item.SetAccount("gabriel")
item.SetLabel("A label")
item.SetAccessGroup("A123456789.group.com.mycorp")
item.SetData([]byte("toomanysecrets"))
item.SetSynchronizable(keychain.SynchronizableNo)
item.SetAccessible(keychain.AccessibleWhenUnlocked)
err := keychain.AddItem(item)

if err == keychain.ErrorDuplicateItem {
  // Duplicate
}
```

#### Query Item

Query for multiple results, returning attributes:

```go
query := keychain.NewItem()
query.SetSecClass(keychain.SecClassGenericPassword)
query.SetService(service)
query.SetAccount(account)
query.SetAccessGroup(accessGroup)
query.SetMatchLimit(keychain.MatchLimitAll)
query.SetReturnAttributes(true)
results, err := keychain.QueryItem(query)
if err != nil {
  // Error
} else {
  for _, r := range results {
    fmt.Printf("%#v\n", r)
  }
}
```

Query for a single result, returning data:

```go
query := keychain.NewItem()
query.SetSecClass(keychain.SecClassGenericPassword)
query.SetService(service)
query.SetAccount(account)
query.SetAccessGroup(accessGroup)
query.SetMatchLimit(keychain.MatchLimitOne)
query.SetReturnData(true)
results, err := keychain.QueryItem(query)
if err != nil {
  // Error
} else if len(results) != 1 {
  // Not found
} else {
  password := string(results[0].Data)
}
```

#### Delete Item

Delete a generic password item with service and account:

```go
item := keychain.NewItem()
item.SetSecClass(keychain.SecClassGenericPassword)
item.SetService(service)
item.SetAccount(account)
err := keychain.DeleteItem(item)
```

### Other

There are some convenience methods for generic password:

```go
// Create generic password item with service, account, label, password, access group
item := keychain.NewGenericPassword("MyService", "gabriel", "A label", []byte("toomanysecrets"), "A123456789.group.com.mycorp")
item.SetSynchronizable(keychain.SynchronizableNo)
item.SetAccessible(keychain.AccessibleWhenUnlocked)
err := keychain.AddItem(item)
if err == keychain.ErrorDuplicateItem {
  // Duplicate
}

password, err := keychain.GetGenericPassword("MyService", "gabriel", "A label", "A123456789.group.com.mycorp")

accounts, err := keychain.GetGenericPasswordAccounts("MyService")
// Should have 1 account == "gabriel"

err := keychain.DeleteGenericPasswordItem("MyService", "gabriel")
if err == keychain.ErrorItemNotFound {
  // Not found
}
```

## iOS

Bindable package in `bind`. iOS project in `ios`. Run that project to test iOS.

To re-generate framework:

```
(cd bind && gomobile bind -target=ios -tags=ios -o ../ios/bind.framework)
```

Post issues to: https://github.com/keybase/keybase-issues
