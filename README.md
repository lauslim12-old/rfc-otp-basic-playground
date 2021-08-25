# Fullstack OTP

Nicholas Dwiarto's Bachelor Thesis.

Documentation will be written after the application is finished.

## Requirements

- [Docker and Docker Compose](https://www.docker.com/)
- [Go 1.16+](https://golang.org/)
- [Postman Agent](https://www.postman.com/downloads/)
- Shell that supports `make`, `curl`, and `sh`. WSL / Ubuntu / OS X should be able to do this just fine without any configuration.

## Installation

- Provision infrastructures.

```bash
docker-compose up -d
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
