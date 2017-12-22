# service-backup
Utility to provide mechanism for backing up services

## Usage

This is intended to be used with the
[service-backup-release](https://github.com/pivotal-cf/service-backup-release). Further instructions can be found in that repository.

## Development

S3 requires the AWS CLI:

```sh
brew install awscli
```

Azure requires the [`blobxfer`](https://github.com/Azure/blobxfer) CLI for batch uploads:

```sh
brew install python
pip install blobxfer
```

> NB: blobxfer version 1.0.0 or above is required

The S3 integration tests require access to a bucket called `service-backup-integration-test` with all permissions. Example policy is as follows:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "servicebackupintegrationtest",
      "Effect": "Allow",
      "Action": [
          "s3:*"
      ],
      "Resource": [
          "arn:aws:s3:::service-backup-*/*",
          "arn:aws:s3:::service-backup-*"
      ]
    }
  ]
}
```

The environment variables required to run the tests are listed in `.envrc.template`.
