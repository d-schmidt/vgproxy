# vgproxy
A [vpngate.net](http://www.vpngate.net/en/) mirror server, to host yourself, when the [csv](http://www.vpngate.net/api/iphone/) is not available.

Written in golang, this server caches the csv, removes comments to make it smaller (750kB to 120kB), supports https and gzip.

Start it with: `vgproxy -url http://www.vpngate.net/api/iphone/`

### Full options:
`vgproxy -url http://www.vpngate.net/api/iphone/ -addr 127.0.0.1 -port 8080  -cert cert.pem -key key.pem -sleep 300 -gzip`

```
  -addr string
        addr to bind HTTP and HTTPS to (default: none i.e. 0.0.0.0)
  -cert string
        cert file for HTTPS on port 443 (default: none)
  -gzip
        enable gzip support (default: disabled)
  -key string
        key file for HTTPS on port 443 (default: none)
  -port int
        port to bind HTTP to (default 80)
  -sleep int
        seconds between refresh of cached csv (default 600)
  -url string
        remote csv url to connect to (default: none)
```

### Compile
install go 1.8 and
```
go get github.com/natefinch/lumberjack
go get github.com/NYTimes/gziphandler
go build vgproxy.go
```

### License
[MIT](https://github.com/d-schmidt/vgproxy/blob/master/LICENSE)
