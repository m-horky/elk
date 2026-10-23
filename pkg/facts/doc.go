// Package facts provides the public system facts model used by Elk.
//
// Facts are represented independently of any service-specific wire format.
// Applications collect the current facts with Collect and read the fields they
// need. Service clients are responsible for converting the model to their own
// request formats.
//
// Collection is best-effort: each source is queried independently and a
// failing source leaves its fields unset instead of aborting the operation.
//
//	collected := facts.Collect(context.Background())
//	if collected.UnameMachine != nil {
//		fmt.Println(*collected.UnameMachine)
//	}
package facts
