ARG GOLANG_BUILD_IMAGE_TAG=1.21-alpine

FROM golang:${GOLANG_BUILD_IMAGE_TAG} as sops

RUN --mount=type=cache,target=/root/.cache/go-build,target=/go/mod/pkg go install github.com/getsops/sops/v3/cmd/sops@v3.8.1


FROM golang:${GOLANG_BUILD_IMAGE_TAG} as fkt

WORKDIR /go/fkt
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build,target=/go/mod/pkg go build -mod vendor -o $GOPATH/bin/fkt .


FROM gcr.io/distroless/static

COPY --from=sops /go/bin/sops /bin/sops
COPY --from=fkt /go/bin/fkt /bin/fkt
WORKDIR /src

ENTRYPOINT ["/bin/fkt"]
