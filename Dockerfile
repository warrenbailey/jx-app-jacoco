FROM scratch
EXPOSE 8080
ENTRYPOINT ["/jx-app-jacoco"]
COPY ./tmp/ca-certificates.crt /etc/ssl/certs/
COPY ./bin/ /
