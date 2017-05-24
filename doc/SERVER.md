# Advanced server configuration

## Using HTTPS
By adding these 2 properties in your yml config, dim will listen on https protocol :
```yml
ssl-key-file: /path/to/keyfile.pem
ssl-cert-file: /path/to/certfile.pem
```


## Hooks

In server mode, dim lets you create advanced hooks when an image is pushed or deleted from your registry. Hooks are defined in the yaml configuration under the `index.hooks` key.
For example, the following configuration will send messages to a slack channel on push events :
```yml
index:
  hooks:
    - Event: push
      Action: |
        {{ if eq .Name "dim" }}
        {{ with $payload := printf `{"text": "A new dim image has been pushed : %s"}`  .FullName | withPayload }}
          {{ with $method := withMethod "POST" }}
            {{  sendRequest "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX" $method $payload }}
          {{ end }}
        {{ end }}
        {{else}}
        {{ with $payload := printf `{"text": "A random image has been pushed : %s"}`  .FullName | withPayload }}
          {{ with $method := withMethod "POST" }}
             {{  sendRequest "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX" $method $payload }}
          {{ end }}
        {{ end }}
        {{ end }}
```

As you can see, hooks are defined as Go temaplates. Image information is accessible from the template allowing you to write advanced rules to trigger whatever you may need

### Available hook actions :

The functions
- `info`
- `warn`
- `error`

will log with the corresponding level the text passed as an argument.

By default, the function `sendRequest` will do a GET to the given URL. Nevertheless, you can build options to tweak the http call :
- `withPayload` allows you to send any text to the target URL
- `withMethod` lets you change the HTTP method (GET, POST, PUT, HEAD...)
- `withHeader` lets you add headers to your request

You can pass as many options as you need to `sendRequest` method

### Available images fields

To write your conditions or to customize the hooks you have access to the following images information :
 - `.ID`
 - `.Name`
 - `.FullName` fullname of the image, composed by its repository name and its tag
 - `.Tag`
 - `.Comment`
 - `.Created` is the created time of the image
 - `.Author`
 - `.Label` is the map of all labels and their values
 - `.Labels` is the array of all label keys
 - `.Volumes`
 - `.ExposedPorts`
 - `.Env` is the map of all environment variable keys and their values
 - `.Envs` is the array of all environment variable keys
 - `.Size`

### Testing your hooks

To help you writing your hooks, you can use dim `hooktest` command. This command take an image name as argument and will dry-run every hook you have in your config file against that image.
```bash
$ dim hooktest dim:latest
Hook #0 would produce :
Would have sent payload {"text" : "A new dim image has been pushed : dim:latest} with method POST to https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX with headers map[]
```

## Authorizations
As Dim server is implemented as a reverse proxy between your dim client or docker client and the docker registry, it's the perfect place to add some access controls.

Currently, access controls are really basic to define. In your `dim.yml`, you can declare as many users as you need, with their username and encrypted password (see below on how to encrypt passwords).
Then, you can declare restrictions under the `server.security` key :
```yml
user: &alice
 Username: alice
 Password: 6a934b45144e3758911efa29ed68fb2d420fa7bd568739cdcda9251fa9609b1e
user: &bob
 Username: bob
 Password: 9b5665f9978886cbea4c163f650f57447f41b93a3a90ecd75ccf97cace6f79fc
user: &registry
 Username: registry
 Password 872491a30d60d598962de6e7b834ab76b2aa65fbab102c6ebaaae6acdc238822

server:
 security:
  # Grant access to search feature to user Alice and Bob
  - Path: /v1/search
    Users: [*alice,*bob]
  # Only registry should be able to call /dim/notify
  - Path: /dim/notify
    Users: [*registry]
  # Version is accessible to anyone
  - Path: /dim/version
    Users:
  # Only Bob can  interact with the registry
  - Path: /
    Users: [*bob]

```

Rules can be defined more precisely by also specifying an HTTP method. So, looking at the [docker registry API](https://docs.docker.com/registry/spec/api/) it becomes really easy to grant read access to a group of users  and push permission to another one :
```yml
server:
 security:
  - Path: /v2/.*/manifests/.*
    Method: HEAD
    Users: [*alice, *bob, ...]
  - Path: /v2/.*/blobs/.*
    Method: GET
    Users: [*alice, *bob, ...]
  - Path: /v2/.*/blobs/.*
    Method: POST
    Users: [*alice]
```

Note: As the second-level of the path is the repository name, you can easily tune permissions depending on repository names



### Rules processing

When dim grants access to a user, it simply reads the rules in the order they are declared and compares the given "Basic Auth" authentication with the allowed users for that rule.
So **you should always declare the most specific rules first, and the rules with the shortest path last**

### Generating encrypted password
Server Authentication is done using HTTP Basic Auth. Nevertheless, to avoid printing base64 encoded credentials in the server config file, the password are encrypted in sha256.

Dim provides a handy command to encrypt your password so you can use it in your server config file : `dim genpasswd`.
This command can be called with the clear password as a parameter :
```bash
$ dim genpasswd my-secret-password
a9c90c47c231afb31950169ccb89951337eb0689d31660e32c34835bb7018c0c
```

Or you can just call the command and it will prompt you the password you want to encrypt :
```bash
$ dim genpasswd
Password:
a9c90c47c231afb31950169ccb89951337eb0689d31660e32c34835bb7018c0c
```

