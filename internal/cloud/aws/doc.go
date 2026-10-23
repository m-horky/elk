// Package aws provides an AWS IMDSv2 client and the typed EC2 instance
// identity document.
//
//	client, _ := aws.NewClient(aws.Config{
//		HTTP: httpclient.Config{BaseURL: constants.Address(constants.AWSIMDSAddress)},
//	})
//	defer client.Close()
//	identity, err := client.InstanceIdentity(ctx)
//	region := identity.Document.Region
//
// A successful call returns the parsed document together with its verbatim
// base64 rsa2048 signature.
package aws
