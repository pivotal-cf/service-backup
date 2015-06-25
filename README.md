# service-backup
Utility to provide mechanism for backing up services

## Usage

This is intended to be used with the
[service-backup-release](https://github.com/pivotal-cf-experimental/service-backup-release). Further instructions can be found in that repository.

## Development

This tool shells out to the [aws cli](http://aws.amazon.com/documentation/cli/) which
requires python. Both of these things must be installed and on the path.

The integration tests require access to a bucket called `service-backup-integration-test` with all permissions. Example policy is as follows:

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

The integration tests also require environment variables as follows:

```sh
AWS_ACCESS_KEY_ID=my-access-key-id
AWS_SECRET_ACCESS_KEY=my-secret-access-key
```
