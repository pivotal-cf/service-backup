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

Azure requires the [`blobxfer`](https://github.com/Azure/azure-batch-samples/tree/master/Python/Storage) CLI for batch uploads:

```sh
brew install python
pip install blobxfer
```

> NB: blobxfer version 0.11.1 or above is required to set the storage account key using an environment variable

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
