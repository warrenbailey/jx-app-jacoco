FROM scratch
EXPOSE 8080
ENTRYPOINT ["/jenkins-x-spotbugs-reporter"]
COPY ./bin/ /