Terraform `signalfx` Provider
=========================

- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 0.10.x
-	[Go](https://golang.org/doc/install) 1.11 (to build the provider plugin)

Building The Provider
---------------------

Clone repository to: `$GOPATH/src/github.com/splunk-terraform/terraform-provider-signalfx`

```sh
$ git clone git@github.com:splunk-terraform/terraform-provider-signalfx.git $GOPATH/src/github.com/splunk-terraform/terraform-provider-signalfx
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/splunk-terraform/terraform-provider-signalfx
$ make build
```

Using the provider
----------------------

If you want to test a local build of  the provider, follow the instructions to [Development Overrides for Provider Developers
](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers).

Example dev.tfrc
```terraform
provider_installation {
  dev_overrides {
    "splunk-terraform/signalfx" = "/Users/user/.go/bin/"
  }

  direct {}
}
```

Replace `/Users/user/.go/bin/` with a path pointing to the go bin directory (place
where `make build` stores compiled binaries) - `echo $GOPATH/bin` 

Now you can set `TF_CLI_CONFIG_FILE` variable to enable a development override only for shell sessions. For example:
```shell
export TF_CLI_CONFIG_FILE=/User/user/tmp/dev.tfrc
```

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
$ export SFX_API_URL=https://api.signalfx.com # or https://api.eu0.signalfx.com
$ export SFX_AUTH_TOKEN=XXXXXX
$ make testacc
```

To also run the AWS integration tests for CloudWatch Metric Streams and AWS logs synchronization, you must create an actual AWS IAM user with an access key and secret that SignalFx can use to manage AWS resources, and define the `SFX_TEST_AWS_ACCESS_KEY_ID` and `SFX_TEST_AWS_SECRET_ACCESS_KEY` environment variables:

```sh
export SFX_TEST_AWS_ACCESS_KEY_ID=AKIAXXXXXX
export SFX_TEST_AWS_SECRET_ACCESS_KEY=XXXXXX
```

The following permissions must be granted. Additional permissions may be required to capture logs from specific AWS services.

```
"cloudwatch:DeleteMetricStream",
"cloudwatch:GetMetricStream",
"cloudwatch:ListMetricStreams",
"cloudwatch:PutMetricStream",
"cloudwatch:StartMetricStreams",
"cloudwatch:StopMetricStreams",
"iam:PassRole",

"logs:DeleteSubscriptionFilter",
"logs:DescribeLogGroups",
"logs:DescribeSubscriptionFilters",
"logs:PutSubscriptionFilter",
"s3:GetBucketLogging",
"s3:GetBucketNotification",
"s3:PutBucketNotification"
```

See [Connect to AWS using the guided setup in Splunk Observability Cloud](https://docs.splunk.com/Observability/gdi/get-data-in/connect/aws/aws-wizardconfig.html) and [Enable CloudWatch Metric Streams](https://docs.splunk.com/Observability/gdi/get-data-in/connect/aws/aws-apiconfig.html#enable-cloudwatch-metric-streams) in Splunk documentation for more details about creating that IAM policy.

Note that we use an IAM user instead of an IAM role as the latter requires an External ID that is only known at AWS integration creation time.

Releasing the Provider
----------------------

Install https://goreleaser.com/install/ if you don't already have it.

 - Update changelog and create release in GH (vx.y.z format) in pre-release state
 - `git pull` (Locally)
 - `export GPG_TTY=$(tty)` (avoid gpg terminal issues if using iTerm2)
 - `GITHUB_TOKEN=xxx GPG_FINGERPRINT=xxx goreleaser --rm-dist` (github token must have `repo` scope)
 - Go back to release in github and mark as released/published
