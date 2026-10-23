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
// NewClient applies the shared IMDS defaults to the configuration: 30 second
// connect and request timeouts, a 512 KiB response cap, and no proxy, so
// metadata traffic stays on the link-local path.
package imds
