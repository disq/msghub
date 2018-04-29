![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)
![Tag](https://img.shields.io/github/tag/disq/msghub.svg)
[![godoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/disq/msghub)
[![Go Report](https://goreportcard.com/badge/github.com/disq/msghub)](https://goreportcard.com/report/github.com/disq/msghub)

# PoC Chat Server

This is a PoC chat server that will broadcast messages to other connected clients.

## Build

    go build ./cmd/msghub

## Running

Just do:

    ./msghub

## Options

    Usage of ./msghub:
      -addr string
            Listen on address (default ":49152")

## Usage

Connect with telnet. The welcome message on connection explains commands:

    w  ask for clients
    s  broadcast message (example: s 1,2 message)
    a  ask for id
    d  disconnect
    ?  this message

## Caveats

Newline and space characters are used as delimiters. To send arbitrary byte-arrays, base-64 encoding is recommended.
