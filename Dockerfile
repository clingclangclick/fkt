FROM golang:1.21-alpine3.18

WORKDIR /src

COPY . . 
RUN go build -o /bin/fkt .

ENTRYPOINT ["/bin/fkt"]