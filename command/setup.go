package command

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

type SetupCommand struct {
	Meta
}

func (c *SetupCommand) Synopsis() string {
	return "Initialize(Create) components"
}
func (c *SetupCommand) Help() string {
	helpText := `

`
	return strings.TrimSpace(helpText)
}

func (c *SetupCommand) Run(args []string) int {
	const cfnStackName string = "Cassiopeia"
	const cfnTemplateURL string = "https://raw.githubusercontent.com/takipone/cassiopeia/master/resources/transit.template"

	// Create Transit(Kinesis Stream and IAM Role by CloudFormation)
	svc := cloudformation.New(session.New(), &aws.Config{Region: aws.String("ap-northeast-1")})

	params := &cloudformation.CreateStackInput{
		StackName:    aws.String(cfnStackName),
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		Parameters: []*cloudformation.Parameter{{
			ParameterKey:     aws.String("ShardCount"),
			ParameterValue:   aws.String("1"),
			UsePreviousValue: aws.Bool(true),
		}},
		TemplateBody: aws.String(`
{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Cassiopeia datastore and credentials stack",
  "Parameters": {
    "ShardCount": {
      "Description": "Number of Shards.",
      "Type": "Number",
      "Default": "1"
    }
  },
  "Resources": {
    "CloudTransit": {
      "Type": "AWS::Kinesis::Stream",
      "Properties": {
        "ShardCount": {
          "Ref": "ShardCount"
        }
      }
    },
    "EdgeTransitUser": {
      "Type": "AWS::IAM::User",
      "Properties": {
        "Policies" : [ {
           "PolicyName" : "AllowKinesisPutRecord",
           "PolicyDocument" : {
              "Version": "2012-10-17",
              "Statement" : [ {
                 "Effect" : "Allow",
                 "Action" : "kinesis:Put*",
                 "Resource" : [ {
                    "Fn::GetAtt" : [ "CloudTransit", "Arn" ]
                 } ]
              } ]
           }
        } ]
      }
    },
    "ProcessorUser": {
      "Type": "AWS::IAM::User",
      "Properties": {
        "Policies" : [ {
          "PolicyName" : "AllowKinesisGetRecord",
          "PolicyDocument" : {
            "Version": "2012-10-17",
            "Statement" : [
              {
                "Effect" : "Allow",
                "Action" : "kinesis:Get*",
                "Resource" : { "Fn::GetAtt" : [ "CloudTransit", "Arn" ] } 
              },
              {
                "Effect": "Allow",
                "Action": [
                  "logs:CreateLogGroup",
                  "logs:CreateLogStream",
                  "logs:PutLogEvents"
                ],
                "Resource": "*"
              }
            ]
          }
        } ]
      }
    },
    "EdgeTransitAccessKey": {
       "Type" : "AWS::IAM::AccessKey",
       "Properties" : {
          "UserName" : { "Ref" : "EdgeTransitUser" }
       }
    },
    "ProcessorAccessKey": {
       "Type" : "AWS::IAM::AccessKey",
       "Properties" : {
          "UserName" : { "Ref" : "ProcessorUser" }
       }
    }
  },
  "Outputs": {
    "CloudTransit": { "Value": { "Ref": "CloudTransit" } },
    "EdgeTransitAccessKey": { "Value": { "Ref": "EdgeTransitAccessKey" } },
    "EdgeTransitSecretKey": { "Value": { "Fn::GetAtt" : [ "EdgeTransitAccessKey", "SecretAccessKey" ] } },
    "ProcessorAccessKey": { "Value": { "Ref": "ProcessorAccessKey" } },
    "ProcessorSecretKey": { "Value": { "Fn::GetAtt" : [ "ProcessorAccessKey", "SecretAccessKey" ] } }
  }
}
`),
	}
	resp, err := svc.CreateStack(params)

	if err != nil {
		fmt.Println(err.Error())
		return 1
	}
	fmt.Println(resp)

	return 0
}
