FROM alpine:3.9

RUN apk --update add ca-certificates

COPY ./build/jx-app-jacoco /jx-app-jacoco

EXPOSE 8080
ENTRYPOINT ["/jx-app-jacoco"]

