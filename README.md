Terraform `signalfx` Provider
=========================

- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 0.10.x
-	[Go](https://golang.org/doc/install) 1.11 (to build the provider plugin)

Building The Provider
---------------------

Clone repository to: `$GOPATH/src/github.com/splunk-terraform/terraform-provider-signalfx`

```sh
$ git clone git@github.com:terraform-providers/terraform-provider-signalfx $GOPATH/src/github.com/splunk-terraform/terraform-provider-signalfx
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/splunk-terraform/terraform-provider-signalfx
$ make build
```

Using the provider
----------------------
If you're building the provider, follow the instructions to [install it as a plugin.](https://www.terraform.io/docs/plugins/basics.html#installing-a-plugin) After placing it into your plugins directory,  run `terraform init` to initialize it.

Further [usage documentation is available on the Terraform website](https://www.terraform.io/docs/providers/signalfx/index.html).

Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.11+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make bin
...
$ $GOPATH/bin/terraform-provider-signalfx
...
```

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```

Releasing the Provider
----------------------

Install https://goreleaser.com/install/ if you don't already have it.

 - Update changelog and create release in GH (vx.y.z format) in pre-release state
 - `git pull` (Locally)
 - `export GPG_TTY=$(tty)` (avoid gpg terminal issues if using iTerm2)
 - `GITHUB_TOKEN=xxx GPG_FINGERPRINT=xxx goreleaser --rm-dist` (github token must have `repo` scope)
 - Go back to release in github and mark as released/published
