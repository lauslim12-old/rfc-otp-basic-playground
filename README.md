# Fullstack OTP

Nicholas Dwiarto's Bachelor Thesis.

Documentation will be written after the application is finished.

## Requirements

- [Docker and Docker Compose](https://www.docker.com/)
- [Go 1.16+](https://golang.org/)
- [Postman Agent](https://www.postman.com/downloads/)
- [Direnv](https://direnv.net/)
- Shell that supports `make`, `curl`, and `sh`. WSL / Ubuntu / OS X should be able to do this just fine without any configuration.

## Installation

- Go to the project directory.

```bash
cd fullstack-otp
```

- Initialize environment variables. It is recommended that you use `direnv` in order to set all of them. The `.envrc` provided here should help you get started right away. Remember the rename `.envrc.example` to `.envrc` so `direnv` can read it properly.

```bash
mv .envrc.example .envrc
nano .envrc
```

- Allow `direnv` in your project. You are going to need these environment variables, so it is the best not to forget about it.

```bash
direnv allow .
eval "$(direnv hook zsh)" # or your favorite shell
```

- Provision infrastructures.

```bash
make start-infrastructure
```

- Run Go application.

```bash
make start
```

- Run unit tests.

```bash
make test
```

- Run integration tests, either with `make e2e` or Postman.

- Stop infrastructures.

```bash
make stop-infrastructure
```
