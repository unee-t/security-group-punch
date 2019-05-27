package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {

	var profile string
	var securityGroup string
	var region string

	flag.StringVar(&profile, "profile", os.Getenv("AWS_PROFILE"), "Set AWS_PROFILE")
	flag.StringVar(&securityGroup, "sg", "", "Set security group")
	flag.StringVar(&region, "region", "ap-southeast-1", "Set AWS_REGION")
	flag.Parse()

	if profile == "" {
		profile = "uneet-dev"
	}

	if securityGroup == "" {
		switch profile {
		case "uneet-dev":
			securityGroup = "sg-66390301"
		case "uneet-demo":
			securityGroup = "sg-6f66d316"
		case "uneet-prod":
			securityGroup = "sg-9f5b5ef8"
		default:
			log.Fatalf("Unknown profile: %s", profile)
		}
	}
	log.Println("Profile:", profile, "Region:", region, "Security group:", securityGroup)

	creds := credentials.NewChainCredentials(
		[]credentials.Provider{
			&credentials.EnvProvider{},
			&credentials.SharedCredentialsProvider{Filename: "", Profile: profile},
		})

	cfg := &aws.Config{
		Region:                        aws.String(region),
		Credentials:                   creds,
		CredentialsChainVerboseErrors: aws.Bool(true),
	}

	sess, err := session.NewSession(cfg)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Get("https://checkip.amazonaws.com")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	fromIP := strings.TrimSpace(string(body))
	addrs, _ := net.LookupAddr(fromIP)
	if err != nil {
		log.Fatal(err)
	}
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	name := u.Name
	if name == "" {
		name = u.Username
	}
	log.Println("Name", name, "from IP", fromIP, addrs)

	svc := ec2.New(sess)
	input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(securityGroup),
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(0),
				ToPort:     aws.Int64(65535),
				IpProtocol: aws.String("TCP"),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      aws.String(fmt.Sprintf("%s/32", fromIP)),
						Description: aws.String(fmt.Sprintf("%s on %s", name, addrs[0])),
					},
				},
			},
		},
	}

	_, err = svc.AuthorizeSecurityGroupIngress(input)
	if err != nil {
		log.Fatal(err)
	}

}
