package command

import (
	"bytes"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/fsouza/go-dockerclient"
	"github.com/takipone/soracom-sdk-go"
)

const cfnStackName string = "Cassiopeia"
const soracomName string = "Cassiopeia"

type SetupCommand struct {
	Meta
}

func (c *SetupCommand) Run(args []string) int {
	// Create CloudTransit(Kinesis Stream and IAM Role by CloudFormation)
	svc := cloudformation.New(session.New(), &aws.Config{Region: aws.String("ap-northeast-1")})
	params := &cloudformation.ListStacksInput{
		StackStatusFilter: []*string{
			aws.String("CREATE_COMPLETE"),
		},
	}
	lso, err := svc.ListStacks(params)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	var flag bool = false
	for _, v := range lso.StackSummaries {
		if *v.StackName == cfnStackName {
			flag = true
			break
		}
	}
	if !flag {
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
		_, err = svc.CreateStack(params)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
		c.Ui.Info("CloudTransit creating.")
	} else {
		c.Ui.Info("CloudTransit already exists.")
	}

	ct := map[string]string{}
	// Wait for the cloud transit has created.
	for {
		params := &cloudformation.DescribeStacksInput{
			StackName: aws.String(cfnStackName),
		}
		resp, err := svc.DescribeStacks(params)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
		if *resp.Stacks[0].StackStatus == "CREATE_COMPLETE" {
			for _, v := range resp.Stacks[0].Outputs {
				ct[*v.OutputKey] = *v.OutputValue
			}
			break
		}
		c.Ui.Info(".")
		// fmt.Print(".")
		time.Sleep(30 * time.Second)
	}

	// Setup EdgeTransit
	ac := soracom.NewAPIClient(nil)
	email := os.Getenv("SORACOM_EMAIL")
	password := os.Getenv("SORACOM_PASSWORD")
	if email == "" {
		c.Ui.Error("SORACOM_EMAIL env var is required")
		return 1
	}
	if password == "" {
		c.Ui.Error("SORACOM_PASSWORD env var is required")
		return 1
	}

	err = ac.Auth(email, password)
	if err != nil {
		c.Ui.Error("SORACOM Auth err: %v\n" + err.Error())
		return 1
	}

	// Set AWS Credential into SORACOM Credential.
	o := &soracom.CredentialOptions{
		Type:        "aws-credentials",
		Description: "Cassiopeia credential",
		Credentials: soracom.Credentials{
			AccessKeyId:     ct["EdgeTransitAccessKey"],
			SecretAccessKey: ct["EdgeTransitSecretKey"],
		},
	}

	cr, err := ac.CreateCredentialWithName(soracomName, o)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	c.Ui.Info("EdgeTransit(SORACOM) Credential: " + cr.CredentialId + " created.")

	g, err := ac.CreateGroupWithName(soracomName)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	c.Ui.Info("EdgeTransit(SORACOM) Configuration: " + soracomName + " created.")
	gc := []soracom.GroupConfig{
		{
			Key:   "enabled",
			Value: "true",
		}, {
			Key:   "credentialsId",
			Value: cr.CredentialId,
		}, {
			Key: "destination",
			Value: soracom.FunnelDestinationConfig{
				Provider:    "aws",
				Service:     "kinesis",
				ResourceUrl: "https://kinesis.ap-northeast-1.amazonaws.com/" + ct["CloudTransit"],
			},
		}, {
			Key:   "contentType",
			Value: "",
		},
	}
	_, err = ac.UpdateGroupConfigurations(g.GroupID, "SoracomFunnel", gc)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	c.Ui.Info("EdgeTransit Configuration updated.")

	// Setup Analyzer(Local/Docker)
	client, err := docker.NewClientFromEnv()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	var buf bytes.Buffer
	opts := docker.PullImageOptions{
		Repository:   "sebp/elk",
		Tag:          "latest",
		OutputStream: &buf,
	}
	err = client.PullImage(opts, docker.AuthConfiguration{})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	return 0
}

func (c *SetupCommand) Synopsis() string {
	return "Initialize(Create) components"
}
func (c *SetupCommand) Help() string {
	helpText := `
Usage: cas setup [options]

	Initialize(Create) components.

Options:

`
	return strings.TrimSpace(helpText)
}
