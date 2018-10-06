FROM scratch
EXPOSE 8080
ENTRYPOINT ["/ext-jacoco"]
COPY ./bin/ /