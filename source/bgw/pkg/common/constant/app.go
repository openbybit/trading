package constant

const (
	IdxAppTypeUNSPECIFIED = 0
	IdxAppTypeFUTURES     = 1
	IdxAppTypeSPOT        = 2
	IdxAppTypeINVESTMENT  = 3
	IdxAppTypeOPTION      = 4
	IdxAppTypeUNIFIED     = 5
)

const (
	AppTypeFUTURES    = "futures"
	AppTypeSPOT       = "spot"
	AppTypeINVESTMENT = "investment"
	AppTypeOPTION     = "option"
)

var AppType = map[string]int32{
	AppTypeFUTURES:    IdxAppTypeFUTURES,
	AppTypeSPOT:       IdxAppTypeSPOT,
	AppTypeINVESTMENT: IdxAppTypeINVESTMENT,
	AppTypeOPTION:     IdxAppTypeOPTION,
}
