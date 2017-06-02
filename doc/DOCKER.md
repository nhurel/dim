# Docker Image Manager

DIM is a Docker Image Management utility. It's the perfect companion for your private docker registry as it brings :
- [Search](#searching-an-image) and [advanced search](#advanced-searches)
- Authentication and access controls
- Hooks on image push / deletion

This image is meant to ease the deployment of a docker registry behind a dim server.
Full documentation about dim itself and and the client command line is available on the [github project](https://github.com/nhurel/dim)

## Deploying private registry with dim

The easiest way to run dim server side is to run the private registry and dim with a docker-compose file like this :

```yml
# docker-compose.yml
version: '2'
services:
 docker-registry:
  container_name: registry
  restart: always
  image: registry:2.5.1
  volumes:
    - registry.yml:/etc/docker/registry/config.yml
  networks:
     - registry
 dim:
  container_name: dim
  restart: always
  image: nhurel/dim
  volumes:
    - dim.yml:/dim.yml
  ports:
    - 80:6000
  networks:
    - registry
networks:
  registry:
    driver: bridge
```

This will start both the registry and dim in server mode. These 2 containers will be in the same network so they can talk to each other with hostnames `docker-registry` and `dim`.

By default, dim docker image is configured to index the registry available at http://docker-registry:5000 so it should work automatically. Otherwise, use the `REGISTRY_URL` environment variable to set the right registry url.

Also, this configuration let you edit configuration of the private registry in a `registry.yml` file and the configuration of dim with a `dim.yml` file

Configure the `registry.yml` file that will be mounted as the registry's config.yml file to have it send events to dim so that dim can maintain its index up-to-date with :
```yml
notifications:
  endpoints:
    - name: dim-listener
      disabled: false
      url: http://dim:6000/dim/notify
      timeout: 1s
      threshold: 5
      backoff: 5000
```

**Congratulations : You now have a docker registry accessible on port 80 that provides a search endpoint !**

For more info about dim server configuration see [SERVER.md](doc/SERVER.md) in `doc` directory of the [github project](https://github.com/nhurel/dim).
It will give you instructions about configuring hooks and how to manage authorizations


## HTTPS support
Dim server can use HTTPS. To do so :
- Add your cert file and key file in the docker container using `volumes`
- Edit your `dim.yml` to set the `ssl-cert-file` and `ssl-key-file` properties
- Update private registry configuration to push notifications over HTTPS
- Map port 6000 of dim container on port 443
