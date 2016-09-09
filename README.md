# DIM

DIM is a Docker Image Management utility. It's the perfect companion for your self-hosted private registry and provides useful commands to manage your docker images :
- [Advanced search](#advanced-searches) 
- Image labels add / removal
- Image deletion (both locally and on your private registry)
- Show image details

DIM works in two ways that are complementary :
- server mode :  provides search feature for both the `docker` command line and `dim` client mode
- client mode : to manage images locally and on your private registry and to run advanced searches (only with dim server enabled)


# Installation

DIM is written in go so it is easy to install. Moreover a docker image is available to easily setup the server mode.

## Client installation

To run dim in client mode, simply download the binary and give it execution permission.
```bash
curl -L https://github.com/nhurel/dim/releases/download/latest/dim-linux-x64 -o dim
chmod a+x dim
./dim help
```

## Server installation

The easiest way to run dim server side is to run the private registry and dim with a docker-compose file like this :

```yml
# docker-compose.yml
version: '2'
services:
 docker-registry:
  container_name: registry
  restart: always
  image: registry:2.4.1
  ports:
    - 5000:5000
    - 5001:5001
  volumes:
    - registry.yml:/etc/docker/registry/config.yml
  networks:
     - registry
 dim:
  container_name: dim
  restart: always
  image: nhurel/dim
  ports:
    - 6000:6000
  networks:
    - registry
networks:
  registry:
    driver: bridge

```

This will start both the registry and dim in server mode. These 2 containers will be in the same network so they can talk to each other with hostnames `docker-registry` and `dim`.

By default, dim docker image is configured to index the registry available at http://docker-registry:5000 so it should work automatically. Otherwise, use the `REGISTRY_URL` environment variable to set the right registry url.

Configure the yml file that will be mounted as the registry's config.yml file to have it send events to dim so that dim can maintain its index up-to-date with :
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

Finally run an httpd server in front of these 2 services and make sure to redirect `/v1` and `/dim` URLs to the dim service and `/v2` URLs to the docker registry. You can also use this httpd instance to manage auhtorizations as described in [this docker recipe](https://docs.docker.com/registry/recipes/apache/)

Example apache httpd config for dim + docker regitry :
```bash
ProxyPass        /v1 http://dim:6000/v1
ProxyPassReverse /v1 http://dim:6000/v1

ProxyPass        /dim http://dim:6000/dim
ProxyPassReverse /dim http://dim:6000/dim

ProxyPass        /v2 http://docker-registry:5000/v2
ProxyPassReverse /v2 http://docker-registry:5000/v2

```

# Configuration

All dim commands may need to access a private registry, so the global command line flags are available :
- `--registry-url` : hostname or full URL to the docker registry
- `--registry-user` : username to authenticate on the registry
- `--registry-password` : password to authenticate on the registry

As it may be cumbersome to always provide these flags, those values can be given through the following environment variables :
- REGISTRY_URL
- REGISTRY_USER
- REGISTRY_PASSWORD

Finally, dim will also search for those settings in the `dim.yml` config file that can be located :
- in current directory
- in `$HOME/.dim/` directory  

# Managing images using dim

## Adding / Editing labels on an image :

Dim lets you add or edit your image labels with the `dim label` command.

### Update image labels locally
Using `-o` flag, dim will override the current image with the labeled one :
```bash
dim label add -o ubuntu:vivid my_label=value my_other_label=another.value
```

To make sure the image you're editing is up to date, you can use the `-p` flag to force dim to download the image before applying labels :

```bash
dim label add -o -p ubuntu:vivid my_label=value my_other_label=another.value
```

### Save the labeled image under a different tag
To leave the original image untouched and apply the label in a dedicated image, use the `-t` flag. For example this will create a new image `ubuntu:vivid_with_label` leaving `ubuntu:vivid` untouched :

```bash
dim label add ubuntu:vivid my_label=value my_other_label=another.value -t ubuntu:vivid_with_label
```

### Label an image and push it to your registry
Finally, dim can automatically push the labeled image to a registry with the `-r` flag :

```bash
# Add a label on an officiel docker hub image and save the result on your registry
dim label add ubuntu:vivid my_label=value my_other_label=another.value -t private-registry/ubuntu:vivid_with_label -r

# Add a label on your custom image and push the result in your registry
dim label add private-registry/my_image:latest my_label=value -p -o -r
```


## Removing labels on an image
Label removal is done with the same `dim label` command using the `-d` flag. Keep in mind that removing a label from on image is not possible with docker, so dim will simply put the label value to an empty value. Nevertheless, when `dim server` will index the image, it won't index empty labels so the image won't match query based on this label.

```bash
dim label -d private-registry/my_image:latest my_label_to_delete -p -o -r
```

# Searching an image

Whether you want to search images with `docker` command or `dim` command, you will need to deploy dim in server mode first.

## Running simple searches with docker command

Once you have your private registry and dim service up and running, you can search your private registry with :
```bash
docker search private-registry/my_query
```

This will search in all your images names and tags

## Running simple and advanced queries with dim

## Simple searches
Assuming you've configured the registry info for your dim client, `dim search` command let you search your images by name or tag with the simple syntax :

```bash
dim search image_name
```

Like all other dim command, you can specify only the `registry-url` setting and dim will ask you your username and password interactively :
```bash
dim search --registry-url=private-registry image_name
```

## Advanced searches

Using the `-a` flag, you can run advanced searches against your private registry.

### Search all images with a given label

You can find all images having a specific label key with the `Labels:` prefix :

```bash
dim search -a Labels:label_key
```

This supports also fuzzy searches, using the following syntax :

```bash
dim search -a Labels:label_*
dim search -a Labels:/.*bel_.*/
```

Finally, you can find  all images with a specific label value using the `Label.` prefix :

```bash
dim search -a Label.label_key:value
```

This also support fuzzy searches on values :

```bash
dim search -a Label.label_key:/val.*/
```


### Search all images with given environment variable

You can find all images having a specific environment variable with the `Envs:` prefix :

```bash
dim search -a Envs:JAVA_VERSION
```

Like labels searches, you can run fuzzy searches on envrionments variable names.

Also, you can search on a specific environment variable value with the `Env.` prefix :

```bash
dim search -a Env.JAVA_VERSION:/1.8.*/
```

### Search images by tag or by name
Use the `Tag:` or `Name:` prefix to search for images by tag or name

```bash
dim search -a Tag:vivid
dim search -a Name:ubuntu
```

### Combining search criteria
You can run more advanced queries by using `+` and `-` operators like :

```bash
# Find all images with java 1.8 except ones with the label REJECTED=true
dim search -a "+Env.JAVA_VERSION:/1.8.*/ -Label.REJECTED:true"
```
