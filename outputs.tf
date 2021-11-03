output "bucket_id" {
  value = aws_s3_bucket.test_bucket.id
}

output "ec2_instance_id" {
  value = aws_instance.python_server.id
}

output "ec2_instance_public_ip" {
  value = aws_instance.python_server.public_ip
}
