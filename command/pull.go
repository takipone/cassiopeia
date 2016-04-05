package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

type PullCommand struct {
	Meta
}

func (c *PullCommand) Run(args []string) int {
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

	type LogstashActionIndex struct {
		Index string `json:"_index"`
		Type  string `json:"_type"`
	}
	type LogstashAction struct {
		Index LogstashActionIndex `json:"index"`
	}

	requestBody := ""
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
			bytes, err := json.Marshal(LogstashAction{LogstashActionIndex{
				Index: "logstash-" + timestamp.Format("06-01-02"),
				Type:  "data",
			}})
			if err != nil {
				c.Ui.Error(err.Error())
				return 1
			}
			// Output Logstash Action and Meta Record
			requestBody += string(bytes) + "\n"
			// Check whether data is JSON
			if strings.Index(data, "{") == 0 {
				trimedData := strings.Trim(data, "{}")
				trimedData += ", \"@timestamp\": \"" + timestamp.Format(time.RFC3339) + "\""
				data = "{" + trimedData + "}"
			}
			requestBody += data + "\n"
		}
		if *recs.MillisBehindLatest == 0 {
			break
		}
		si = recs.NextShardIterator
	}

	fmt.Println(requestBody)
	req, _ := http.NewRequest("POST", os.Getenv("CASSIOPEIA_ANALYZER_ENTRY"), bytes.NewBuffer([]byte(requestBody)))
	req.Header.Set("Content-Type", "application/json")
	hclient := &http.Client{}
	resp, err := hclient.Do(req)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))

	return 0
}

func (c *PullCommand) Synopsis() string {
	return "Fetch records from cloud transit and post to local analyzer"
}

func (c *PullCommand) Help() string {
	helpText := `

`
	return strings.TrimSpace(helpText)
}
