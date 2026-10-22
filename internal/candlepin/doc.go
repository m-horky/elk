// Package candlepin contains the Candlepin client. The API surface is
// intentionally small while the Candlepin wire contracts are being confirmed.
//
// Construct a client from HTTP settings and credentials, then call its methods:
//
//	client, err := candlepin.NewBasicClient(cfg, candlepin.BasicCredentials{
//		Username: "user",
//		Password: "secret",
//	})
//	if err != nil {
//		return err
//	}
//	defer client.Close()
//
//	owners, err := client.ListUserOwners(ctx, "user")
package candlepin
