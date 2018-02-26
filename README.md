# Mini Keystore

A toy implementation of an in memory keystore.

All commands are assessable over HTTP using standard HTTP verbs (`GET`, `PUT`, `DELETE`, `POST`). Keys in the keystore are described by the path - `/somekey` describes the key `somekey`. Keys must not contain slashes.

## Commands

### Set key

To set a key, HTTP PUT with the key in the path and the value in a JSON document. `value` can be a string, a list of strings, or a map of string keys -> string values.

```
PUT /stringkey HTTP/1.1
Content-Type: application/json; charset=utf-8
Content-Length: 26

{"value":"a string value"}

```

```
PUT /mapkey HTTP/1.1
Content-Type: application/json; charset=utf-8
Content-Length: 37

{"value":{"key1":"one","key2":"two"}}
```

```
PUT /listkey HTTP/1.1
Content-Type: application/json; charset=utf-8
Content-Length: 35

{"value":["thing one","Thing two"]}
```

### Get Key

To retrieve a value, use an HTTP GET with the key in the path

```
GET /listkey HTTP/1.1
Content-Type: application/json; charset=utf-8


HTTP/1.1 200 OK
Content-Type: application/json; charset=UTF-8
Content-Length: 35

{"value":["thing one","Thing two"]}
```

### Delete Key

```
DELETE /somekey HTTP/1.1
Content-Type: application/json; charset=utf-8
```

## List Type Commands

### Append a value to a list

This will push a value on to the end of a list. If they key does not exist, it will be created. If the key exists and is not a list type, an error will be returned.

```
POST /listkey HTTP/1.1
Content-Type: application/json; charset=utf-8
Content-Length: 37

{"value":"On The End","cmd":"append"}
```

### Pop a value off the end of a list

This will remove a value from the end of a list and return it. If the list is empty or the key is not a list type, an error will be returned

```
POST /listkey HTTP/1.1
Content-Type: application/json; charset=utf-8
Content-Length: 13

{"cmd":"pop"}
```

## Map Type Commands

### Set a map key

This will set a key value pair on a map type. If the key does not exist, it will be created. If the key is not a map type, an error will be returned

```
POST /mapkey HTTP/1.1
Content-Type: application/json; charset=utf-8
Content-Length: 42

{"key":"foo","cmd":"mapset","value":"bar"}
```

### Get a map key

This gets a map value by key.

```
POST /mapkey HTTP/1.1
Content-Type: application/json; charset=utf-8
Host: localhost:8787
Connection: close
User-Agent: Paw/3.1.5 (Macintosh; OS X/10.12.6) GCDHTTPRequest
Content-Length: 28

{"key":"foo","cmd":"mapget"}
```

### Delete a map key

This deletes a key from a map.

```
POST /mapkey HTTP/1.1
Content-Type: application/json; charset=utf-8
Content-Length: 31

{"key":"foo","cmd":"mapdelete"}
```

## Index Commands

### Get Index Keys

Index keys can be retrieved using a simple glob search. The `*` character is used as a universal wild card, and will match any character. Searching for `foo*` will match `foo`, `foobar`, and `foo:bar` but not `barfoo`. Searching for `*` along will return all keys. Searching is case sensitive.

```
POST / HTTP/1.1
Content-Type: application/json; charset=utf-8
Content-Length: 28

{"key":"foo*","cmd":"index"}
```

## TODO

- [ ] Add authentication
- [ ] Add file backed storage (currently, this is an in-memory only key store, and all values will be lost if the server is shut down)
- [ ] Better performance on index searches. Any wildcard search (except for `*` by itself) requires all index keys to be searched