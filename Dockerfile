FROM golang:1.17.0-alpine3.14

COPY build/collector /bin/

CMD ["/bin/collector"]
