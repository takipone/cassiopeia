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
const analyzerDockerImage string = "sebp/elk"
const analyzerDockerImageTag string = "latest"
const analyzerContainerName string = "cassiopeia_analyzer"

type SetupCommand struct {
	Meta
}

func (c *SetupCommand) Run(args []string) int {
	// Create CloudTransit(Kinesis Stream and IAM Role by CloudFormation)
	c.Ui.Output("Check or create cassiopeia components")

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
		c.Ui.Info("-> CloudTransit creating.")
	} else {
		c.Ui.Output("-> CloudTransit already exists.")
	}

	// Wait for the cloud transit has created.
	ct := map[string]string{}
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
		c.Ui.Info("-> .")
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

	// Create SORACOM Credential if not exists.
	creds, _, err := ac.ListCredentials()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	flag = false
	var cred soracom.Credential
	for _, value := range creds {
		if value.CredentialId == soracomName {
			cred = value
			c.Ui.Output("-> EdgeTransit(SORACOM) Credential: already exists.")
			flag = true
			break
		}
	}
	if !flag {
		co := &soracom.CredentialOptions{
			Type:        "aws-credentials",
			Description: "Cassiopeia credential",
			Credentials: soracom.Credentials{
				AccessKeyId:     ct["EdgeTransitAccessKey"],
				SecretAccessKey: ct["EdgeTransitSecretKey"],
			},
		}
		cred, err := ac.CreateCredentialWithName(soracomName, co)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
		c.Ui.Info("-> EdgeTransit(SORACOM) Credential: " + cred.CredentialId + " created.")
	}

	// Create SORACOM Group if not exists.
	lgo := &soracom.ListGroupsOptions{
		TagName:  "name",
		TagValue: soracomName,
	}
	groups, _, err := ac.ListGroups(lgo)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	var g *soracom.Group
	if len(groups) == 0 {
		g, err = ac.CreateGroupWithName(soracomName)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
		c.Ui.Info("-> EdgeTransit(SORACOM) Configuration: " + soracomName + " created.")
	} else {
		g = &groups[0]
		c.Ui.Output("-> EdgeTransit(SORACOM) Configuration: already exists.")
	}

	gc := []soracom.GroupConfig{
		{
			Key:   "enabled",
			Value: "true",
		}, {
			Key:   "credentialsId",
			Value: cred.CredentialId,
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
	c.Ui.Info("-> EdgeTransit Configuration updated.")

	// Setup Analyzer(Local/Docker)
	client, err := docker.NewClientFromEnv()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Pull Docker image if not exists.
	lio := docker.ListImagesOptions{
		Filter: analyzerDockerImage,
	}
	images, err := client.ListImages(lio)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if len(images) == 0 {
		var buf bytes.Buffer
		opts := docker.PullImageOptions{
			Repository:   analyzerDockerImage,
			Tag:          analyzerDockerImageTag,
			OutputStream: &buf,
		}
		err = client.PullImage(opts, docker.AuthConfiguration{})
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
	} else {
		c.Ui.Output("-> Analyzer docker image found.")
	}
	// Create and start container if not exists.
	containers, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	flag = false
	for _, v := range containers {
		if v.Names[0] == "/"+analyzerContainerName {
			flag = true
			break
		}
	}
	if !flag {
		cco := docker.CreateContainerOptions{
			Name:   analyzerContainerName,
			Config: &docker.Config{Image: analyzerDockerImage},
		}
		container, err := client.CreateContainer(cco)
		portBindings := map[docker.Port][]docker.PortBinding{
			"5000/tcp": {docker.PortBinding{HostIP: "0.0.0.0", HostPort: "5000"}},
			"5044/tcp": {docker.PortBinding{HostIP: "0.0.0.0", HostPort: "5044"}},
			"5601/tcp": {docker.PortBinding{HostIP: "0.0.0.0", HostPort: "5601"}},
			"9200/tcp": {docker.PortBinding{HostIP: "0.0.0.0", HostPort: "9200"}},
		}
		hostConfig := docker.HostConfig{
			PortBindings: portBindings,
		}
		err = client.StartContainer(container.ID, &hostConfig)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
	} else {
		c.Ui.Output("-> Analyzer docker container found.")
	}

	c.Ui.Output("Cassiopeia setup has completed.")
	c.Ui.Output("")

	c.Ui.Output("Next steps below")
	// Output environment values
	dockerUrlArr := strings.Split(os.Getenv("DOCKER_HOST"), ":")
	dockerIp := strings.TrimLeft(dockerUrlArr[1], "/")
	c.Ui.Output("- Set values into environment values")
	c.Ui.Info("  export CASSIOPEIA_TRANSIT=" + ct["CloudTransit"])
	c.Ui.Info("  export CASSIOPEIA_ANALYZER_ENTRY=http://" + dockerIp + ":9200/_bulk")
	c.Ui.Info("  export CASSIOPEIA_ANALYZER_URL=http://" + dockerIp + ":5601/")
	c.Ui.Output("- Send data to edge transit.")
	c.Ui.Output("    POST `http://funnel.soracom.io/`")
	c.Ui.Output("    or `tcp://funnel.soracom.io:23080/`")
	c.Ui.Output("    or `udp://funnel.soracom.io:23080/`")
	c.Ui.Output("- Fetch from transit to analyzer.")
	c.Ui.Output("    `cas pull`")
	c.Ui.Output("    `cas fetch | some commands`")
	c.Ui.Output("- Open analyzer.")
	c.Ui.Output("    `cas open`")

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
