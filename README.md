# Terraform EC2 & S3 python web server infrastructure
This a simple Terraform code to spin up an EC2 instance & S3 Bucket with the following tags:

- Name: `TF_VAR_name_tag_value`
- Owner: `TF_VAR_owner_tag_value`

with a simple Flask web server deployed to the EC2 instance using a bash script that run as a template file to setup python3 and git, then it will clone [the server](https://github.com/omarking05/flask-ec2-web-server) and then spin it up. Making it accessible on EC2 instance public IP on port 80.

## Setup
Basically you'll need the following to be able to run everything smoothly, versions are the one i used when developing this task, i didn't actually try the compatibility on any other version:

- Terraform: 1.0.7 (Required)
- GoLang: 1.16.7 (Required)
- Python: 3.8.10 (Required)
- Pip: 20.0.2 (Required)

## Run

Terraform script is generic and it accepts input via environment variables, and since it is interacting with AWS resources, you're required to pass the authentication credentials as an ENV variables as this is the approach i chose here.

*Please note that there are couple of other approaches, you can check [here](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#authentication)*

So you're expected to have these variables exported or passed as a prefix to your terraform command:

- `AWS_ACCESS_KEY_ID=<YOUR_AWS_ACCESS_KEY>` (Required)
- `AWS_SECRET_ACCESS_KEY=<YOUR_AWS_SECRET_KEY>` (Required)
- `TF_VAR_aws_access_key=$AWS_ACCESS_KEY_ID` (Required) (Default=EMPTY)
- `TF_VAR_aws_secret_key=$AWS_SECRET_ACCESS_KEY` (Required) (Default=EMPTY)
- `TF_VAR_name_tag_value="My Name Tag"` (Optional) (Default=MrGeek)
- `TF_VAR_owner_tag_value="My Group Tag"` (Optional) (Default=MRG)
- `TF_VAR_s3_bucket_name="My Awesome Bucket"` (Optional) (Default=mrg-bucket)

By default this script will spin everything under AWS region `eu-central-1` 

### Terraform Run

If you have ENV variables exported in your shell, you can simply just run it by
```
terraform init
terraform plan
terraform apply --auto-approve
terraform destroy --auto-approve
```

Or if you don't, you can just prefix the command with the ENV variables needed
```
TF_VAR_name_tag_value="John" TF_VAR_owner_tag_value="SuperCo" TF_VAR_aws_access_key=$AWS_ACCESS_KEY_ID TF_VAR_aws_secret_key=$AWS_SECRET_ACCESS_KEY terraform apply --auto-approve
```

### Tests Run

You can run the tests by:
```
cd test
go test -timeout 30m
```

Or you can always prefix the command with no exported ENV variables by
```
TF_VAR_name_tag_value="John" TF_VAR_owner_tag_value="SuperCo" TF_VAR_aws_access_key=$AWS_ACCESS_KEY_ID TF_VAR_aws_secret_key=$AWS_SECRET_ACCESS_KEY go test -timeout 30m
```

## Technical Description
Terraform script will do the following:

- Create a EC2 template file `ec2_init` that will run [ec2_server_startup.sh](./scripts/ec2_server_startup.sh) passing AWS access and secret keys
- Create a security group `web-sg` to allow traffic on port 80 [Ref](https://learn.hashicorp.com/tutorials/terraform/resource?in=terraform/configuration-language#associate-security-group-with-instance)
- Create a `t2.micro` EC2 instance that is using `ami-05f7491af5eef733a` and spin it up by on the default VPC and associating it with the created security group `web-sg` and template file `ec2_init`, and it will also assign the tags (`TF_VAR_name_tag_value`, `TF_VAR_owner_tag_value`) to the instance
- Create S3 bucket `TF_VAR_s3_bucket_name` beside assigning the tags (`TF_VAR_name_tag_value`, `TF_VAR_owner_tag_value`) to it
- It will output the following:
	- `bucket_id`: The bucket name that was actually created on AWS
	- `ec2_instance_id`: The instance ID that was created
	- `ec2_instance_public_ip`: The instance public IP, in which you can check by going to port 80 on this IP and check if the web server is up & running, usually this takes ~2min to have the server setup and started

Test are written using Terratest, in which we have a single function that will start testing stuff that Terraform just created, it will first test the creation of the EC2 instance and check for this scenarios:

- AWS has actually the EC2 instance
- Instance has `Name` tag that is equal to `TF_VAR_name_tag_value`
- Instance has `Owner` tag that is equal to `TF_VAR_owner_tag_value`

Then it will test the created S3 bucket, by checking these scenarios:

- Check that the bucket actually exists
- Check that the bucket has the following tags `Name`, and `Owner` set to these values respectively `TF_VAR_name_tag_value`, and `TF_VAR_owner_tag_value`

Bucket name in test is created using the default bucket name `mrg-bucket` appended with some random lowercase unique ID, to ensure that when creating and destroying this bucket we don't have any errors regarding a conflict on names

After that test will sleep for 1min to give a chance for [ec2_server_startup.sh](./scripts/ec2_server_startup.sh) to run and to install all needed dependencies and to start the Flask web server.
Once it is completed it will then begin to test the actual web server, say for example that our created EC2 instance has this public IP `1.2.3.4`

- Send a request to `http://1.2.3.4` and check the response that is containing the word `up & running` to make sure that it is responding
- Send a request to `http://1.2.3.4/tags` and check that the response contains these:
	- `name` and `TF_VAR_name_tag_value`
	- `owner` and `TF_VAR_owner_tag_value`
- Send a request to `http://1.2.3.4/shutdown` and make sure that the request is returning `200` and the response body contains the instance ID and the word `shutdown` in it
- Sleep the test execution for 30seconds
- Send a request again to the server home URL but this time append a random query string `http://1.2.3.4/?q=diu23224` to avoid server-side caching and check if the site is still reachable

## Github Workflow

The way that everything is setup on this repository is that an Github action is running on a push to `master` branch, and it will run the Go test on each commit, ENV variables are setup as secrets as you can see on [test-infra.yml](./.github/workflows/test-infra.yml)

```
    env:
      AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
      AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
      TF_VAR_name_tag_value: ${{ secrets.NAME_TAG_VALUE }}
      TF_VAR_owner_tag_value: ${{ secrets.OWNER_TAG_VALUE }}
      TF_VAR_aws_access_key: ${{ secrets.AWS_ACCESS_KEY_ID }}
      TF_VAR_aws_secret_key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
```

It installs the required Go modules, and then run the tests.

*Please note that only `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` variables are the ones needed to be secrets, but other variables were intentionally hidden since this is a public and i would like to keep this repository generic for future uses*
