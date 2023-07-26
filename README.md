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

```sh
brew install python3
```

## Running Tests

The environment variables required to run the tests are listed in `.envrc.template`.

### GCP
The GCP integration tests require access to a GCP service account file.
It should look like this:

```json
{
  "type": "service_account",
  "project_id": " <project-id>",
  "private_key_id": "<key-id>",
  "private_key": " <private-key> ",
  "client_email": " <email> ",
  "client_id": " <id>",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://accounts.google.com/o/oauth2/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": " <cert-url>"
}
```

### AWS S3 

The S3 integration tests require access to a bucket called `service-backup-integration-test` with all permissions. 
Example policy is as follows:

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

