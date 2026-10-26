// Package imds provides the HTTP client construction shared by the provider
// IMDS clients under internal/cloud.
//
//	client, _ := imds.NewClient(httpclient.Config{
//		BaseURL: constants.Address(constants.AWSIMDSAddress),
//	})
//	defer client.Close()
//	response, _ := client.Do(ctx, httpclient.Request{
//		Path: "/latest/dynamic/instance-identity/document",
//	})
//
// NewClient applies common defaults to the configuration: connect and request timeouts,
// a response cap, and no proxy, so metadata traffic stays link-local.
package imds
