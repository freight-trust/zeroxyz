
package turbokeeperplugins

// EventOperation enumerates operation types on events
type EventOperation int

// SecurityModule is a code plug-point that can be implemented using a go plugin module.
//  Build your plugin with a "SecurityModule" export that implements this interface,
//  and configure the dynamic load path of your module in the configuration.
type SecurityModule interface {

	// VerifyToken - Authentication plugpoint. Verfies a token and returns a context object to store that will be returned to authorization points
	VerifyToken(string) (interface{}, error)

	// AuthRPC - Authorization plugpoint for a synchronous RPC call
	AuthRPC(authCtx interface{}, method string, args ...interface{}) error
	// AuthRPCSubscribe - Authorization plugpoint for subscribe RPC call
	AuthRPCSubscribe(authCtx interface{}, namespace string, channel interface{}, args ...interface{}) error
	// AuthEventStreams - Authorization plugpoint for event management system (single permission currently - evolution likely as requirements evolve)
	AuthEventStreams(authCtx interface{}) error
	// AuthListAsyncReplies - Authorization plugpoint for listing replies in the reply store (containing receipts and/or errors)
	AuthListAsyncReplies(authCtx interface{}) error
	// AuthReadAsyncReplyByUUID - Authorization plugpoint for getting an individual reply by UUID (containing an individual receipt/error)
	AuthReadAsyncReplyByUUID(authCtx interface{}) error
}
