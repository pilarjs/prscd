# 🌎 Prscd

The open source backend for Pilar.js

## 🎯 Roadmap

- [x] Websocket arraybuffer support
- [x] Zero-copy upgrade to WebSocket
- [x] SO_REUSEPORT on Darwin and Linux
- [x] Implement WebSocket native Ping/Pong frame to keep alive
- [x] WebTransport Datagram support, unreliable but fast communication
- [x] Geo-distributed System by YoMo
- [ ] WebTransport Stream support, reliable
- [ ] reuse goroutine
- [ ] pprof support

## 📚 Usage

Step 1: Download `prscd` for your OS and Arch, currenty support `macOS`, `Linux` and `Windows` with amd64 or aarch64.

```bash
curl -fsSL https://bina.egoist.dev/pilarjs/prscd | sh                                                      

  ==> Using auto generated config because pilarjs/prscd doesn't have a bina.json file in its release
  ==> This might not work for some projects
  ==> Resolved version latest to v0.1
  ==> Downloading asset for darwin arm64
  ==> Permissions required for installation to /usr/local/bin
Password:
  ==> Installation complete
```

Step 2: Downlaod prepared SSL certicates of domain `lo.yomo.dev`

Pilarjs provides a domain `lo.yomo.dev` for local development, this domain is always resolved to `127.0.0.1`, so
frontend developers can crafting realtime web applications in https enviroment.

There are two files needed:

- [lo.yomo.dev.cert](https://raw.githubusercontent.com/pilarjs/prscd/main/lo.yomo.dev.cert)
- [lo.yomo.dev.key](https://raw.githubusercontent.com/pilarjs/prscd/main/lo.yomo.dev.key)

Step 3: Create `.env` file:

```sh
# debug mode
DEBUG=true

# WebTransport and WebSocket will be served over 8443 port
DOMAIN=lo.yomo.dev
PORT=8443

# Mesh node identifier
MESH_ID=dev

# Integrate the yomo zipper
WITH_YOMO_ZIPPER=true

# YoMo settings, see https://yomo.run for details
YOMO_ZIPPER=127.0.0.1:9000
YOMO_SNDR_NAME=prscd-sender
YOMO_RCVR_NAME=prscd-receiver

# Observerbility
#OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318

# SSl certs
CERT_FILE=./lo.yomo.dev.cert
KEY_FILE=./lo.yomo.dev.key
```

## 🥷🏻 Development

1. Start prscd service in terminal-2：`make dev`
1. Open `webtransport.html` by Chrome with Dev Tools
1. Open `websocket.html` by Chrome with Dev Tools

![](https://github.com/fanweixiao/gifs-repo/blob/main/prscd-readme.gif)

## 🦸🏻 Self-hosting

Compile:

```bash
make dist
```

## ☕️ FAQ

### how to generate SSL for your own domain

1. `brew install certbot`
2. `sudo certbot certonly --manual --preferred-challenges dns -d prscd.example.com`
3. create a TXT record followed the instruction by certbot
4. `nslookup -type=TXT _acme-challenge.prscd.example.com` to verify the process
5. `sudo chown -Rv "$(whoami)":staff /etc/letsencrypt/` set permission
6. cert and key: `/etc/letsencrypt/live/prscd.example.com/{fullchain, privkey}.pem`
7. verify the expiratioin time: `openssl x509 -enddate -noout -in prscd.example.com.cert.pem`

### if you are behind a proxy on Mac

Most of proxy applications drop UDP packets, which means developers can not route WebTransport or HTTP/3 requests, 
so if you are a macOS user, this bash script can helped bypass `lo.yomo.dev` domain to proxy.

```bash
networksetup -setproxybypassdomains "Wi-Fi" $(networksetup -getproxybypassdomains "Wi-Fi" | awk '{ printf "\"%s\" ", $0 }') "lo.yomo.dev"
```

### Integrate to your own Auth system

Currently, provide `public_key` for authentication, the endpoint looks like: `/v1?app_id=<USER_CLIENT_ID>&public_key=<PUBLIC_KEY>`

### Live inspection

Execute `make dev` in terminal-1:

```bash
$ make dev
go run -race main.go
pid: 20079
Listening SIGUSR1, SIGUSR2, SIGTERM/SIGINT...
```

Open terminal-2, execute:

```bash
$ kill -SIGUSR1 20079
$ kill -SIGUSR2 20079
```

The output of terminal-1 will looks like:

```bash
$ make dev
go run -race main.go
pid: 20079
Listening SIGUSR1, SIGUSR2, SIGTERM/SIGINT...
Received signal: user defined signal 1
SIGUSR1
Dump start --------
Peers: 1
Channel:room-1
	Peer:127.0.0.1:62577
Dump doen --------
Received signal: user defined signal 2
	NumGC = 0
```

### Configure Firewall of Cloud Provider

TCP and UDP on the `PORT` shall has to be allowed in security rules.
