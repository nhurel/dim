FROM scratch
COPY dim /dim
COPY ca-certificates.crt /etc/ssl/certs/
EXPOSE 6000

ENV REGISTRY_URL=http://registry:5000

ENTRYPOINT ["/dim"]
CMD ["server"]