package command

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

type FetchCommand struct {
	Meta
}

func (c *FetchCommand) Run(args []string) int {
	svc := kinesis.New(session.New(), &aws.Config{Region: aws.String("ap-northeast-1")})

	siParams := &kinesis.GetShardIteratorInput{
		ShardId:           aws.String("shardId-000000000000"),
		ShardIteratorType: aws.String("TRIM_HORIZON"),
		StreamName:        aws.String(os.Getenv("CASSIOPEIA_TRANSIT")),
	}
	gsi, err := svc.GetShardIterator(siParams)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	si := gsi.ShardIterator

	type Output struct {
		Timestamp string `json:"timestamp"`
		Data      string `json:"data"`
	}

	for {
		rParams := &kinesis.GetRecordsInput{
			ShardIterator: aws.String(*si),
		}
		recs, err := svc.GetRecords(rParams)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
		for _, rec := range recs.Records {
			timestamp := rec.ApproximateArrivalTimestamp
			data := string(rec.Data)
			bytes, err := json.Marshal(Output{
				Timestamp: timestamp.Format(time.RFC3339),
				Data:      data,
			})
			if err != nil {
				c.Ui.Error(err.Error())
				return 1
			}
			fmt.Println(string(bytes))
		}
		if *recs.MillisBehindLatest == 0 {
			break
		}
		si = recs.NextShardIterator
	}

	return 0
}

func (c *FetchCommand) Synopsis() string {
	return "Fetch records from cloud transit to stdout."
}

func (c *FetchCommand) Help() string {
	helpText := `

`
	return strings.TrimSpace(helpText)
}
