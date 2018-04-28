# PoC Chat Server

This is a PoC chat server that will broadcast messages to other connected clients.

## Build

    go build ./cmd/server

## Running

Just do:

    ./server

## Options

    Usage of ./server:
      -addr string
            Listen on address (default ":49152")

## Usage

Connect with telnet. The welcome message on connection explains commands:

    w  ask for clients
    s  broadcast message (example: s 1,2 message)
    a  ask for id
    d  disconnect

## Caveats

Newline and space characters are used as delimiters. To send arbitrary byte-arrays, base-64 encoding is recommended.
