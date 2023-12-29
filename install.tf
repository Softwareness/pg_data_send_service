resource "aws_s3_bucket" "data_bucket" {
  bucket = "data-service-import-storage"

  tags = {
    Name        = "MijnTerraformBucket"
    Environment = "Dev"
  }
}


#####################################################################
# Service that gets triggerd by RDS when new order is added
#####################################################################

resource "aws_iam_policy" "lambda_s3_access" {
  name        = "lambda_s3_access_policy"
  description = "Beleid dat Lambda-functie toegang geeft tot S3-bucket"

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Action = [
          "s3:GetObject",
          "s3:ListBucket"
        ],
        Resource = [
          "arn:aws:s3:::data-service-import-storage",
          "arn:aws:s3:::data-service-import-storage/*"
        ]
      }
    ]
  })
}


resource "aws_iam_role" "lambda_role" {
  name = "lambda_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Action = "sts:AssumeRole",
        Effect = "Allow",
        Principal = {
          Service = "lambda.amazonaws.com"
        },
      },
    ],
  })
}

resource "aws_iam_role_policy_attachment" "lambda_s3_access_attach" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = aws_iam_policy.lambda_s3_access.arn
}


resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_lambda_permission" "allow_bucket_payload" {
  statement_id  = "AllowExecutionFromS3Bucket"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.payload.function_name
  principal     = "s3.amazonaws.com"
  source_arn    = aws_s3_bucket.data_bucket.arn
}

resource "aws_lambda_function" "payload" {
  function_name = "pg-data-payload-service"
  role          = aws_iam_role.lambda_role.arn

  handler = "main" # update dit afhankelijk van je runtime en handler
  runtime = "go1.x"     # update dit afhankelijk van je runtime

  filename         = "main.zip"
  source_code_hash = filebase64sha256("main.zip")
  environment {
    variables = {
      GITHUB_TOKEN = "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"
      REPO_OWNER   = "Softwareness"
      REPO_NAME    = "bmwpoc"
    }
  }
}

resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = aws_s3_bucket.data_bucket.id

  lambda_function {
    lambda_function_arn = aws_lambda_function.payload.arn
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "process/"
  }
  depends_on = [
      aws_lambda_permission.allow_bucket_payload,
      aws_lambda_function.payload
    ]
}
