# nsqauth

https://nsq.io/components/nsqd.html#auth

To configure nsqd to require authorization you need to specify the --auth-http-address=host:port flag with an Auth Server that conforms to the Auth HTTP protocol.


*NOTE:* It is expected when using authorization that only the nsqd TCP protocol is exposed to external clients, not the HTTP(S) endpoints. See the note below about exposing stats and lookup to clients with auth.


The Auth Server must accept an HTTP request on:

```
/auth?remote_ip=...&tls=...&secret=...
```

And return a response in the following format:

```
{
  "ttl": 3600,
  "identity": "username",
  "identity_url": "https://....",
  "authorizations": [
    {
      "permissions": [
        "subscribe",
        "publish"
      ],
      "topic": ".*",
      "channels": [
        ".*"
      ]
    }
  ]
}
```


# RUN

```
$ go run . --file=demoauth.csv --log=error
```