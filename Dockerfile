FROM golang:1.11-alpine3.8 AS build-env
ADD . /src
RUN apk add build-base git \
  && cd /src \
  && go build -o external-ambassador

FROM alpine:3.8
WORKDIR /app
RUN apk add ca-certificates
COPY --from=build-env /src/external-ambassador /app/
EXPOSE 8080
ENTRYPOINT "external-ambassador"
