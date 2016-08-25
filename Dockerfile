from golang:1.6

RUN apt-get update
RUN apt-get install fping

ADD ./bin/infping /go/bin/infping

CMD infping

