FROM scratch
EXPOSE 8080
ENTRYPOINT ["/ext-jacoco"]
COPY ./tmp/ca-certificates.crt /etc/ssl/certs/
COPY ./bin/ /
