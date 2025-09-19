# aws-go-tools

This document assumes you are on the Mac OS.  You may need to adapt it for Linux and/or Windows


This doc is a placeholder / WIP

TODO:


Install Homebrew

MacOS
```
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

NOTE: If Homebrew is already installed, it would be a good idea to update all of you packages before proceeding.

```
brew update && brew upgrade
```


Install needed apps

```
brew install aws-iam-authenticator awscli git go golangci-lint gitleaks
```

```
pyenv python@3.13
```

Install aws-okta-processor

You may need pyenv, python@3.13, & pip

See for instructions: https://github.com/godaddy/aws-okta-processor



Install Go modules

```
go install github.com/jandelgado/gcov2lcov@latest
```


Building

```
make release
````



Troubleshooting:

Q: I get the following error.
2025/09/17 17:42:03 Unable to get caller identity: operation error STS: GetCallerIdentity, exceeded maximum number of attempts, 3, get identity: get credentials: failed to refresh cached credentials, no EC2 IMDS role found, operation error ec2imds: GetMetadata, exceeded maximum number of attempts, 3, request send failed, Get "http://169.254.169.254/latest/meta-data/iam/security-credentials/": dial tcp 169.254.169.254:80: connect: host is down

A: Make sure you setup ~/.aws/credentials with the following
[default]
credential_process = aws-okta-processor authenticate -u [USERNAME] -o godaddy.okta.com -k default -d 7200
