# grpcall
> functionality for protocol generic, http->grpc

## release note
- refactor response message, return proto raw message instead of encoded bytes.
- add ErrReqUnmarshalFailed error when marshal request to proto failed.
 