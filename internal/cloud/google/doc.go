// Package google provides a Google Compute Engine IMDS client and the typed
// instance identity claims.
//
//	client, _ := google.NewClient(google.Config{
//		HTTP: httpclient.Config{BaseURL: constants.Address(constants.GoogleIMDSAddress)},
//	})
//	defer client.Close()
//	identity, err := client.Collect(ctx)
//	instanceID := identity.Claims.ComputeEngine.InstanceID
//
// A successful call returns the raw identity JWT together with its decoded
// claims.
package google
