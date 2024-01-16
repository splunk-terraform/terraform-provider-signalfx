# Splunk Observability Cloud Terraform Provider (`signalfx`)

Splunk Observability Cloud is a Software as a Service (SaaS) solution for infrastructure monitoring (Splunk IM), application performance monitoring (Splunk APM), real user monitoring (Splunk RUM), and synthetic monitoring (Splunk Synthetic Monitoring). For more information, see [the official documentation](https://docs.splunk.com/observability/en/).

Use this Terraform provider to automate the configuration of Splunk Observability Cloud.

- Documentation: https://registry.terraform.io/providers/splunk-terraform/signalfx/latest/docs
- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) 0.12.x or higher
-	[Go](https://golang.org/doc/install) 1.19 or higher to build the provider plugin

## Build the provider

To build the provider, follow these steps:

1. Clone the repository to: `$GOPATH/src/github.com/splunk-terraform/terraform-provider-signalfx`:

   ```sh
   $ git clone git@github.com:splunk-terraform/terraform-provider-signalfx.git $GOPATH/src/github.com/splunk-terraform/terraform-provider-signalfx
   ```

1. Enter the provider directory and build the provider:

   ```sh
   $ cd $GOPATH/src/github.com/splunk-terraform/terraform-provider-signalfx
   $ make build
   ```

## Use the provider

If you're building the provider, follow the instructions to [install it as a plugin.](https://www.terraform.io/docs/plugins/basics.html#installing-a-plugin). After placing it into your plugins directory,  run `terraform init` to initialize it.

Further [usage documentation](https://www.terraform.io/docs/providers/signalfx/index.html) is available on the Terraform website.

## Develop the provider

If you wish to work on the provider, you need the following:

- [Go](http://www.golang.org) version 1.11 or higher
- Configured [GOPATH](http://golang.org/doc/code.html#GOPATH)
- `$GOPATH/bin` added to your `$PATH`

To compile the provider, run `make build`. This builds the provider and put its binary inside the `$GOPATH/bin` directory:

   ```sh
   $ make build
   ...
   $ $GOPATH/bin/terraform-provider-signalfx
   ...
   ```

To test the provider, run the following command:

   ```sh
   $ make test
   ```

To run the full suite of acceptance tests, run `make testacc`.

```sh
$ export SFX_API_URL=https://api.signalfx.com # or https://api.eu0.signalfx.com
$ export SFX_AUTH_TOKEN=XXXXXX
$ make testacc
```

> [!IMPORTANT]
> Acceptance tests create real resources, and often cost money to run.

### Run AWS integration tests

To run the AWS integration tests for CloudWatch Metric Streams and AWS logs synchronization, create an AWS IAM user with an access key and secret that Splunk Observability Cloud can use to manage AWS resources, and define the `SFX_TEST_AWS_ACCESS_KEY_ID` and `SFX_TEST_AWS_SECRET_ACCESS_KEY` environment variables. For example:

```sh
export SFX_TEST_AWS_ACCESS_KEY_ID=AKIAXXXXXX
export SFX_TEST_AWS_SECRET_ACCESS_KEY=XXXXXX
```

Grant the following permissions. Additional permissions may be required to capture logs from specific AWS services.

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

> [!NOTE]
> Use an IAM user instead of an IAM role, as the latter requires an External ID that is only known at AWS integration creation time.

## Release the provider

To release the provider, install https://goreleaser.com/install/ if you don't already have it, then follow these steps:

1. Update the changelog and create a release in GH (vx.y.z format) in pre-release state

1. `git pull` (Locally)

1. `export GPG_TTY=$(tty)` (avoid gpg terminal issues if using iTerm2)

1. `GITHUB_TOKEN=xxx GPG_FINGERPRINT=xxx goreleaser --rm-dist` (github token must have `repo` scope)

1. Go back to release in github and mark as released/published
