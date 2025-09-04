FROM golang:alpine as build

WORKDIR /root
COPY . /root
RUN go build .

FROM alpine:latest

WORKDIR /app
COPY --from=build /root/orakel-webpage /app

CMD /app/orakel-webpage
