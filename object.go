package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/codegangsta/cli"
	"github.com/mitchellh/colorstring"
)

func createObject(bucketname string, r io.Reader) map[string]interface{} {
	path := fmt.Sprintf("/apps/%s/buckets/%s/objects", globalConfig.AppId, bucketname)
	headers := globalConfig.HttpHeadersWithAuthorization("application/json")
	body := HttpPost(path, headers, r).Bytes()
	var j map[string]interface{}
	json.Unmarshal(body, &j)
	return j
}

func CreateObject(bucketname string) {
	r := OptionalReader(func() io.Reader { return strings.NewReader("{}") })
	j := createObject(bucketname, r)
	fmt.Printf("%s\n", j["objectID"])
}

func ReadObject(bucketname, objectId, templstr string) {
	var templ *template.Template
	if templstr != "" {
		t, err := template.New("").Parse(templstr)
		if err != nil {
			log.Fatalf("%v", err)
		}
		templ = t
	}

	path := fmt.Sprintf("/apps/%s/buckets/%s/objects/%s", globalConfig.AppId, bucketname, objectId)
	headers := globalConfig.HttpHeadersWithAuthorization("")
	body := HttpGet(path, headers).Bytes()

	if templ == nil {
		fmt.Printf("%s\n", string(body))
	} else {
		var j map[string]interface{}
		json.Unmarshal(body, &j)
		templ.Execute(os.Stdout, j)
	}
}

func QueryObject(bucketname string) {
	path := fmt.Sprintf("/apps/%s/buckets/%s/query", globalConfig.AppId, bucketname)
	headers := globalConfig.HttpHeadersWithAuthorization("application/vnd.kii.QueryRequest+json")
	r := OptionalReader(func() io.Reader { return strings.NewReader(`{"bucketQuery":{"clause":{"type":"all"}}}`) })
	body := HttpPost(path, headers, r).Bytes()

	fmt.Printf("%s\n", string(body))
}

func ReplaceObject(bucketname string) {
	path := fmt.Sprintf("/apps/%s/buckets/%s/objects", globalConfig.AppId, bucketname)
	headers := globalConfig.HttpHeadersWithAuthorization("application/json")
	r := OptionalReader(func() io.Reader { return strings.NewReader("{}") })
	body := HttpPut(path, headers, r).Bytes()

	var j map[string]interface{}
	json.Unmarshal(body, &j)
	fmt.Printf("%s\n", j["objectID"])
}

func DeleteObject(bucketname, objectId string) {
	path := fmt.Sprintf("/apps/%s/buckets/%s/objects/%s", globalConfig.AppId, bucketname, objectId)
	headers := globalConfig.HttpHeadersWithAuthorization("")
	body := HttpDelete(path, headers).Bytes()
	fmt.Printf("%s\n", string(body))
}

func attachObjectBody(bucketname, objectId, conttype string) []byte {
	path := fmt.Sprintf("/apps/%s/buckets/%s/objects/%v/body", globalConfig.AppId, bucketname, objectId)
	headers := globalConfig.HttpHeadersWithAuthorization(conttype)
	r := OptionalReader(func() io.Reader {
		log.Fatalf(colorstring.Color("[red]object body must be given thru stdin"))
		return nil
	})
	return HttpPut(path, headers, r).Bytes()
}

func AttachObjectBody(bucketname, objectId, conttype string) {
	body := attachObjectBody(bucketname, objectId, conttype)
	fmt.Printf("%v", string(body))
}

func publishObjectBody(bucketname, objectId string) []byte {
	path := fmt.Sprintf("/apps/%s/buckets/%s/objects/%v/body/publish", globalConfig.AppId, bucketname, objectId)
	headers := globalConfig.HttpHeadersWithAuthorization("application/vnd.kii.ObjectBodyPublicationRequest+json")
	req := map[string]int64{"expiresIn": 60 * 3 /*sec*/}
	j, _ := json.Marshal(req)
	return HttpPost(path, headers, bytes.NewReader(j)).Bytes()
}

func PublishObjectBody(bucketname, objectId string) {
	body := publishObjectBody(bucketname, objectId)
	fmt.Printf("%v", string(body))
}

func CreateObjectAndPublishBody(bucketname, conttype string) {
	r := strings.NewReader("{}")
	j := createObject(bucketname, r)
	objId := j["objectID"].(string)

	r0 := attachObjectBody(bucketname, objId, conttype)
	var a map[string]int64
	json.Unmarshal(r0, &a)
	logger.Printf("modifiedAt: %v", a["modifiedAt"])

	r1 := publishObjectBody(bucketname, objId)
	var res struct {
		PublicationID string `json:"publicationID"`
		URL           string `json:"url"`
	}
	json.Unmarshal(r1, &res)
	logger.Printf("publicationID: %v", res.PublicationID)
	logger.Printf("url: %v", res.URL)

	fmt.Printf("%v\n", res.URL)
}

var ObjectCommands = []cli.Command{
	{
		Name:  "object:create",
		Usage: "Create an object in application scope",
		Action: func(c *cli.Context) {
			ShowCommandHelp(1, c)
			CreateObject(c.Args()[0])
		},
	},
	{
		Name:        "object:read",
		Usage:       "Read the object in application scope",
		Description: "args: <bucket> <object-id>",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "template", Value: "", Usage: "Template for output"},
		},
		Action: func(c *cli.Context) {
			ShowCommandHelp(2, c)
			ReadObject(c.Args()[0], c.Args()[1], c.String("template"))
		},
	},
	{
		Name:        "object:query",
		Usage:       "Query objects in a bucket of application scope",
		Description: "args: <bucket>",
		Action: func(c *cli.Context) {
			ShowCommandHelp(1, c)
			QueryObject(c.Args()[0])
		},
	},
	//{
	//	Name:        "object:replace",
	//	Usage:       "Replate the object in application scope with a new one",
	//	Description: "args: <bucket> <object-id>",
	//	Action: func(c *cli.Context) {
	//		ShowCommandHelp(2, c)
	//		ReplaceObject(c.Args()[0], c.Args()[1])
	//	},
	//},
	{
		Name:        "object:delete",
		Usage:       "Delete the object in application scope",
		Description: "args: <bucket> <object-id>",
		Action: func(c *cli.Context) {
			ShowCommandHelp(2, c)
			DeleteObject(c.Args()[0], c.Args()[1])
		},
	},
	{
		Name:  "object:body-attach",
		Usage: "Attach body to an object in application scope",
		Description: `args: <bucket> <object-id> <content-type>

   ex)
     dogs 4c8aaf60-3166-11e4-a448-12315004cc43 image/png < mydog.png
`,
		Action: func(c *cli.Context) {
			ShowCommandHelp(3, c)
			AttachObjectBody(c.Args()[0], c.Args()[1], c.Args()[2])
		},
	},
	{
		Name:        "object:body-publish",
		Usage:       "Publish a body of object in application scope",
		Description: `args: <bucket> <object-id>`,
		Action: func(c *cli.Context) {
			ShowCommandHelp(2, c)
			PublishObjectBody(c.Args()[0], c.Args()[1])
		},
	},
	{
		Name:  "object:publish",
		Usage: "Publish a body creating a new object into the bucket in application scope",
		Description: `args: <bucket> <content-type>

   Runs object:create, object-body-attach and object:body-publish in order.
   It's expected body is given thru stdin.

   ex)
     dogs image/png < mydog.png`,
		Action: func(c *cli.Context) {
			ShowCommandHelp(2, c)
			CreateObjectAndPublishBody(c.Args()[0], c.Args()[1])
		},
	},
}
