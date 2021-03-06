FROM golang:1.14 AS build

ARG COMMIT=""

WORKDIR /src
# enable modules caching in separate layer
COPY go.mod go.sum ./
RUN go mod download
COPY . ./

RUN make binary COMMIT=$COMMIT

FROM debian:10.2-slim

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y \
        ca-certificates; \
    apt-get clean; \
    rm -rf /var/lib/apt/lists/*; \
    groupadd -r bee --gid 999; \
    useradd -r -g bee --uid 999 --no-log-init -m bee;

COPY --from=build /src/dist/bee /usr/local/bin/bee

EXPOSE 6060 7070 8080
USER bee
WORKDIR /home/bee

ENTRYPOINT ["bee"]
