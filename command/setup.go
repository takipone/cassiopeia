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

  // Create Transit(Kinesis Stream and IAM Role by CloudFormation)
  svc := cloudformation.New(session.New())

  params := &cloudformation.CreateStackInput{
    StackName: aws.String("StackName"),
    Capabilities: []*string{
      aws.String("Capability"), // Required
      // More values...
    },
    Parameters: []*cloudformation.Parameter{
      { // Required
        ParameterKey:     aws.String("ParameterKey"),
        ParameterValue:   aws.String("ParameterValue"),
        UsePreviousValue: aws.Bool(true),
      },
      // More values...
    },
    TemplateBody:     aws.String("TemplateBody"),
    TemplateURL:      aws.String("TemplateURL"),
  }
  resp, err := svc.CreateStack(params)

  if err != nil {
    fmt.Println(err.Error())
    return
  }
  fmt.Println(resp)

	return 0
}

