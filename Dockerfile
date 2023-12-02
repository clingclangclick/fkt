FROM golang:1.21-alpine3.18 as fkt

WORKDIR /src
COPY . . 
RUN go build -o /bin/fkt .

FROM golang:1.21-alpine3.18

COPY --from=fkt /bin/fkt /bin/fkt
RUN go install github.com/getsops/sops/v3/cmd/sops@v3.8.1 && \
    rm -rf $GOPATH/src

WORKDIR /src

ENTRYPOINT ["/bin/fkt"]