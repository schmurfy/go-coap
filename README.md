[![Build Status](https://travis-ci.com/go-ocf/go-coap.svg?branch=master)](https://travis-ci.com/go-ocf/go-coap)
[![codecov](https://codecov.io/gh/go-ocf/go-coap/branch/master/graph/badge.svg)](https://codecov.io/gh/go-ocf/go-coap)
[![Go Report](https://goreportcard.com/badge/github.com/go-ocf/go-coap)](https://goreportcard.com/report/github.com/go-ocf/go-coap)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fgo-ocf%2Fgo-coap.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fgo-ocf%2Fgo-coap?ref=badge_shield)
[![backer](https://opencollective.com/go-coap/backers/badge.svg)](https://opencollective.com/go-coap#backer)
[![sponsors](https://opencollective.com/go-coap/sponsors/badge.svg)](https://opencollective.com/go-coap#sponsors)
[![contributors](https://img.shields.io/github/contributors/go-ocf/go-coap)](https://github.com/go-ocf/go-coap/graphs/contributors)
[![GitHub stars](https://img.shields.io/github/stars/go-ocf/go-coap)](https://github.com/go-ocf/go-coap/stargazers)
[![GitHub license](https://img.shields.io/github/license/go-ocf/go-coap)](https://github.com/go-ocf/go-coap/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/go-ocf/go-coap?status.svg)](https://godoc.org/github.com/go-ocf/go-coap)
[![Sourcegraph](https://sourcegraph.com/github.com/go-ocf/go-coap/-/badge.svg)](https://sourcegraph.com/github.com/go-ocf/go-coap?badge)

# Go-CoAP

The Constrained Application Protocol (CoAP) is a specialized web transfer protocol for use with constrained nodes and constrained networks in the Internet of Things.
The protocol is designed for machine-to-machine (M2M) applications such as smart energy and building automation.

The go-coap provides servers and clients for DTLS, TCP-TLS, UDP, TCP in golang language.

## Features
* CoAP over UDP [RFC 7252][coap].
* CoAP over TCP/TLS [RFC 8232][coap-tcp]
* Observe resources in CoAP [RFC 7641][coap-observe]
* Block-wise transfers in CoAP [RFC 7959][coap-block-wise-transfers]
* request multiplexer
* multicast
* CoAP NoResponse option in CoAP [RFC 7967][coap-noresponse]
* CoAP over DTLS [pion/dtls][pion-dtls]

[coap]: http://tools.ietf.org/html/rfc7252
[coap-tcp]: https://tools.ietf.org/html/rfc8323
[coap-block-wise-transfers]: https://tools.ietf.org/html/rfc7959
[coap-observe]: https://tools.ietf.org/html/rfc7641
[coap-noresponse]: https://tools.ietf.org/html/rfc7967
[pion-dtls]: https://github.com/pion/dtls

## Samples

### Simple

#### Server UDP/TCP
```go
	// Server
	
	// Middleware function, which will be called for each request.
	func loggingMiddleware(next mux.Handler) mux.Handler {
		return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
			log.Printf("ClientAddress %v, %v\n", w.Client().RemoteAddr(), r.String())
			next.ServeCOAP(w, r)
		})
	}
	
	// See /examples/simple/server/main.go
	func handleA(w mux.ResponseWriter, req *message.Message) {
		err := w.SetResponse(codes.GET, message.TextPlain, bytes.NewReader([]byte("hello world")))
		if err != nil {
			log.Printf("cannot set response: %v", err)
		}
	}

	func main() {
		r := mux.NewRouter()
		r.Use(loggingMiddleware)
		r.Handle("/a", mux.HandlerFunc(handleA))
		r.Handle("/b", mux.HandlerFunc(handleB))

		log.Fatal(coap.ListenAndServe("udp", ":5688", r))

		
		// for tcp
		// log.Fatal(coap.ListenAndServe("tcp", ":5688",  r))

		// for tcp-tls
		// log.Fatal(coap.ListenAndServeTLS("tcp", ":5688", &tls.Config{...}, r))

		// for udp-dtls
		// log.Fatal(coap.ListenAndServeDTLS("udp", ":5688", &dtls.Config{...}, r))
	}
```
#### Client
```go
	// Client
	// See /examples/simpler/client/main.go
	func main() {
		co, err := udp.Dial("localhost:5688")
		
		// for tcp
		// co, err := tcp.Dial("localhost:5688")
		
		// for tcp-tls
		// co, err := tcp.Dial("localhost:5688", tcp.WithTLS(&tls.Config{...}))

		// for dtls
		// co, err := dtls.Dial("localhost:5688", &dtls.Config{...}))

		if err != nil {
			log.Fatalf("Error dialing: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		resp, err := co.Get(ctx, "/a")
		if err != nil {
			log.Fatalf("Cannot get response: %v", err)
			return
		}
		log.Printf("Response: %+v", resp)
	}
```

### Observe / Notify

[Server](examples/observe/server/main.go) example.

[Client](examples/observe/client/main.go) example.

### Multicast

[Server](examples/mcast/server/main.go) example.

[Client](examples/mcast/client/main.go) example.

## Contributing

In order to run the tests that the CI will run locally, the following two commands can be used to build the Docker image and run the tests. When making changes, these are the tests that the CI will run, so please make sure that the tests work locally before committing.

```shell
$ docker build . --network=host -t go-coap:build --target build
$ docker run --mount type=bind,source="$(pwd)",target=/shared,readonly --network=host go-coap:build go test './...'
```

## License
Apache 2.0

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fgo-ocf%2Fgo-coap.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fgo-ocf%2Fgo-coap?ref=badge_large)

<h2 align="center">Sponsors</h2>

[Become a sponsor](https://opencollective.com/go-coap#sponsor) and get your logo on our README on Github with a link to your site.

<div align="center">

<a href="https://opencollective.com/go-coap/sponsor/0/website?requireActive=false" target="_blank"><img src="https://opencollective.com/go-coap/sponsor/0/avatar.svg?requireActive=false"></a>

</div>

<h2 align="center">Backers</h2>

[Become a backer](https://opencollective.com/go-coap#backer) and get your image on our README on Github with a link to your site.

<a href="https://opencollective.com/go-coap/backer/0/website?requireActive=false" target="_blank"><img src="https://opencollective.com/go-coap/backer/0/avatar.svg?requireActive=false"></a>
