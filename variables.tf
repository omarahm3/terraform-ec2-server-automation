variable "aws" {
  type = map(any)
  default = {
    region = "eu-central-1"
  }
}

variable "name_tag_value" {
  description = "Name tag value"
  default     = "MrGeek"
}

variable "s3_bucket_name" {
  description = "S3 Bucket Name"
  default     = "mrg-bucket"
}

variable "owner_tag_value" {
  description = "Owner tag value"
  default     = "MRG"
}

variable "aws_access_key" {
  description = "AWS access key"
  default     = ""
}

variable "aws_secret_key" {
  description = "AWS secret key"
  default     = ""
}
