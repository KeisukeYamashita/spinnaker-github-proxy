# Spinnaker GitHub Proxy

![Test](https://github.com/KeisukeYamashita/spinnaker-github-proxy/workflows/Test/badge.svg?branch=master)

`spinnaker-github-proxy` proxies user info request from Spinnaker. By default, Spinnaker's GitHub Organization authentication only reads public members. 
This proxy will allow to fetch private members and return the judge if the user can access or not.

## How to run

### Run locally

Configure environment variables.

```console
$ cp .envrc.sample .envrc
$ vi .envrc
```

And the run.

```console
$ make run
```

### Run by Docker image

```console
$ docker run docker.pkg.github.com/keisukeyamashita/spinnaker-github-proxy/spinnaker-github-proxy
```

## Maintainer

* [KeisukeYamashita](https://github.com/KeisukeYamashita)

## (For maintainer) How to release

```console
$ git tag <version>
$ git push origin <version>
```
