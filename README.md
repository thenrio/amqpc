amqpc
=====
fork of [gocardless/amqpc](https://github.com/gocardless/amqpc).
this is mostly a cli built from [streadway/amqp](https://github.com/streadway/amqp).

has nuke change from original : it is _only_ able to produce _maybe unique_ messages from stdin.

    amqpc [options] routingkey < file

for details see `amqpc --help` or source code.
