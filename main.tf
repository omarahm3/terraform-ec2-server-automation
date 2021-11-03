terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.27"
    }
  }

  required_version = ">= 0.14.9"
}

# Rest of AWS credentials (key, secret) will be passed as ENV variables
provider "aws" {
  # Get these default credentials from environment
  region = var.aws.region
}

locals {
  common_tags = {
    Name  = "${var.name_tag_value}"
    Owner = "${var.owner_tag_value}"
  }
}

data "template_file" "ec2_init" {
  template = file("${path.module}/scripts/ec2_server_startup.sh")

  vars = {
    access_key = "${var.aws_access_key}"
    secret_key = "${var.aws_secret_key}"
  }
}


// Add security group to allow accessing the webserver on http, we don't care about https for now, as this will require nginx with proxy pass
// REF: https://learn.hashicorp.com/tutorials/terraform/resource?in=terraform/configuration-language#associate-security-group-with-instance
resource "aws_security_group" "web-sg" {
  name = "web-sg"

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "python_server" {
  # Using "Ubuntu Server 20.04 LTS (HVM), SSD Volume Type 64bit" AMI
  ami                    = "ami-05f7491af5eef733a"
  instance_type          = "t2.micro"
  tags                   = local.common_tags
  user_data              = data.template_file.ec2_init.rendered
  vpc_security_group_ids = [aws_security_group.web-sg.id]
}

resource "aws_s3_bucket" "test_bucket" {
  bucket = var.s3_bucket_name
  tags   = local.common_tags
}
