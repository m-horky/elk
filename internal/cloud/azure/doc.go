// Package azure provides a Microsoft Azure IMDS client and the typed instance
// and attested documents.
//
//	client, _ := azure.NewClient(azure.Config{
//		HTTP: httpclient.Config{BaseURL: constants.Address(constants.AzureIMDSAddress)},
//	})
//	defer client.Close()
//	metadata, err := client.Collect(ctx)
//	vmid := metadata.Instance.Compute.VMID
//
// A successful call returns the parsed instance document together with the
// attested document.
package azure
