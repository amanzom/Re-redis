Re-redis
===

Re-redis, as its name suggests is an in-memory database inspired by redis. It implements some of the redis's core features in golang.

## Key features of Re-redis
1. Re-redis speaks redis dialect as it implements [RESP](https://redis.io/docs/latest/develop/reference/protocol-spec/), you can connect to it with any Redis Client and the simplest way it to use a [Redis CLI](https://redis.io/docs/manual/cli/). Programmatically, depending on the language you prefer, you can use your favourite Redis library to connect.
2. Single-threaded - uses [IO multiplexing](https://wiki.c2.com/?IoMultiplexing) and [Event Loop](https://en.wikipedia.org/wiki/Event_loop) to support concurrent clients, using [KQUEUE](https://man.freebsd.org/cgi/man.cgi?kqueue) for [OSX (Darwin) based environment](https://en.wikipedia.org/wiki/MacOS) and [Epool](https://en.wikipedia.org/wiki/Epoll#:~:text=epoll%20is%20a%20Linux%20kernel,45%20of%20the%20Linux%20kernel.) for  [Linux based environment](https://en.wikipedia.org/wiki/Comparison_of_Linux_distributions).
3. Key commands supported - PING, SET, GET, TTL, EXPIRE, DEL, BGWRITEAOF, INCR, INFO, MULTI, EXEC, DISCARD.
4. [Active and passive](https://redis.io/docs/latest/commands/expire/#:~:text=How%20Redis%20expires%20keys,will%20never%20be%20accessed%20again.) deletion of expired keys.
5. [Pipeling](https://redis.io/docs/latest/develop/use/pipelining/) support where we can issue multiple commands at once without waiting for the response to each individual command.
6. [Persistance](https://redis.io/docs/latest/operate/oss_and_stack/management/persistence/) support via AOF, to support reconstruction of key-value store due to unexpected downtime.
7. [Object encoding](https://redis.io/docs/latest/commands/object-encoding/), currently supports only string object with its corresponding encodings- raw, int and embedded string.
8. [Keys eviction](https://redis.io/docs/latest/develop/reference/eviction/) using Approximated LRU and all keys random eviction algorithms
9. [Transactions](https://redis.io/docs/latest/develop/interact/transactions/) support using MULTI, EXEC and DISCARD commands
10. Background rewrite of AOF using [BGWRITEAOF](https://redis.io/docs/latest/commands/bgrewriteaof/) command.

## Get started

### Using Docker

The easiest way to get started with Re-redis is using [Docker](https://www.docker.com/) by running the following command.

```
$ make dev
```

### Setting up

To run Re-redis for local development or running from source, you will need

1. [Golang](https://go.dev/)
2. Any of the below supported platform environment:
    1. [Linux based environment](https://en.wikipedia.org/wiki/Comparison_of_Linux_distributions)
    2. [OSX (Darwin) based environment](https://en.wikipedia.org/wiki/MacOS)

```
$ git clone https://github.com/amanzom/Re-redis
$ cd Re-redis
$ go run main.go
```

## Re-redis Playground:
[Re-Redis Playground](http://15.207.107.93:8083/) is an interactive site that allows users to experiment with Redis features in real-time on a hosted Re-Redis server.
Give it a try [here](http://15.207.107.93:8083/).
