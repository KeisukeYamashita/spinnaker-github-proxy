# Spinnaker GitHub Proxy

`spinnaker-github-proxy` proxies user info request from Spinnaker. By default, Spinnaker's GitHub Organization authentication only reads public members. 
This proxy will allow to fetch private members and return the judge if the user can access or not.

## How to run

### Run locally

Configure environment variables.

```console
cp .envrc.sample .envrc
vi .envrc
```

And the run.

```console
$ make run
```

## Maintainer

* [KeisukeYamashita](https://github.com/KeisukeYamashita)

## (For maintainer) How to release

```console
$ git tag <version>
$ git push origin <version>
```
