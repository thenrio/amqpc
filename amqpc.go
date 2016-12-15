package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/template"
	"time"
)

// types
type headers []string

func (h *headers) String() string {
	return fmt.Sprintf("%s", []string(*h))
}

func (h *headers) Set(s string) error {
	*h = append(*h, s)
	return nil
}

const version = "0.1.0"

var options struct {
	concurrency int
	contentType string
	count       int
	exchange    string
	headers     headers
	interval    int
	period      int
	silent      bool
	uri         string
	version     bool
}

func parsecli() []string {
	flag.BoolVar(&options.silent, "silent", false,
		"silent (mute)")
	flag.BoolVar(&options.version, "version", false,
		"print version and exit")
	flag.IntVar(&options.concurrency, "concurrency", 1,
		"number of client (connections)")
	flag.IntVar(&options.count, "count", 1,
		"count of messages to send, use 0 for infinite loop")
	flag.IntVar(&options.interval, "interval", 0,
		"interval at which send messages (in ms)")
	flag.IntVar(&options.period, "period", 0,
		"concurrency period in ms - Interval at which spawn new Producer when concurrency is set")
	flag.StringVar(&options.contentType, "content-type",
		"application/octet-stream", "content-type is not in headers amqp protocol...")
	flag.StringVar(&options.exchange, "exchange", "",
		"exchange on which to pub")
	flag.StringVar(&options.uri, "uri", "amqp://guest:guest@localhost:5672",
		"server uri")
	flag.Var(&options.headers, "header",
		"header, value is k:v, that set header[k]=v (see usage for details)")

	flag.Usage = func() {
		s := `
amqpc [options] routingkey < file
version: %s

file is processed using text.template with one argument, the index of message.
index starts at 1.

header option
=============
header option MAY be used many times.
option value MUST match k:v pattern, where k is the key of header, and v its value.
this is forwarded as header of message in amqp protocol.

datatype of value
-----------------
value is sent as a string unless it matches ^\[(.*)\]$ regular expression.
then what is sent is is comma separated list of string of first group.

this is MAY be useful to look like an http header struct.
(and it has the downcase of NOT being able to send [a] as a header value)

examples
========
pub 1 message to somewhere with content-type:application/vnd.me.awesome.1+json

	echo 'message nº{{ . }}' | amqpc --content-type=application/vnd.me.awesome.1+json somewhere

pub 1 message to somewhere with header include, as a list of values a, b, c

	echo 'message nº{{ . }}' | amqpc --header=include:[a,b,c] somewhere

pub 1 message to somewhere with header include being a,b,c

see
* http://golang.org/pkg/text/template/
* https://golang.org/pkg/fmt/
`
		fmt.Printf(s, version)
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()
	if options.version {
		fmt.Println(version)
		os.Exit(0)
	}
	if options.silent {
		log.SetOutput(ioutil.Discard)
	}
	return flag.Args()
}

func main() {
	done := make(chan error)

	args := parsecli()

	routingKey := args[0]
	bytes, _ := ioutil.ReadAll(os.Stdin)
	body := string(bytes[:])
	for i := 0; i < options.concurrency; i++ {
		if options.period > 0 {
			time.Sleep(time.Duration(options.period) * time.Millisecond)
		}
		go startProducer(done, options.uri, options.exchange, routingKey, options.count, options.interval, options.contentType, []string(options.headers), body)
	}

	err := <-done
	if err != nil {
		log.Fatalf("Error : %s", err)
	}

	log.Printf("Exiting...")
}

func startProducer(done chan error, uri string, exchange string, routingKey string, messageCount, interval int, contentType string, headers []string, body string) {
	var (
		p   *Producer = nil
		err error     = nil
	)

	if err != nil {
		log.Fatalf("Error while starting producer : %s", err)
	}

	for {
		p, err = NewProducer(uri, exchange, routingKey, contentType, headers)
		if err != nil {
			log.Printf("Error while starting producer : %s", err)
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	i := 1
	duration := time.Duration(interval) * time.Millisecond
	template := template.Must(template.New("body").Parse(body))

	for {
		p.Publish(_body(template, i))

		i++
		if messageCount != 0 && i > messageCount {
			break
		}

		time.Sleep(duration)
	}

	done <- nil
}

func _body(template *template.Template, i int) string {
	var buffer bytes.Buffer
	if err := template.Execute(&buffer, i); err != nil {
		panic(err)
	}
	return buffer.String()
}
