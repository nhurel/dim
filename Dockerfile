FROM scratch
COPY ca-certificates.crt /etc/ssl/certs/
EXPOSE 6000

ENV REGISTRY_URL=http://docker-registry:5000

ENTRYPOINT ["/dim"]
CMD ["server"]

COPY dim /dim
