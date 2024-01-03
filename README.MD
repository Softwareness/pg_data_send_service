# Postgress Data Send Service

This service uses AWS Lambda and takes the JSON order file from AWS S3 which is created by the Postgress Data Import Service and sends the data as payload to Github.

## Installation

Service is deployed with Terraform

```bash
terraform apply
```
Uninstall

```bash
terraform destroy
```

## Usage

Compile the Go code and zip it so Terraform can deploy the AWS Lambda

```bash
./compile.sh