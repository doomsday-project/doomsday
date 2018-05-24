package vaultkv

type GenRootToken struct {
	otp string
}

//Gen gives the state of the rekey operation as of the last call to
//Submit, NewRekey, or CurrentRekey.
type GenRootTokenState struct {
	Started bool   `json:"started"`
	Nonce   string `json:"nonce"`
	//The number of keys given so far in this rekey operation
	Progress int `json:"progress"`
	//The total number of keys needed for this rekey operation
	Required       int    `json:"required"`
	PGPFingerprint string `json:"pgp_fingerprint"`
	//EncodedToken is what EncodedRootToken is called in 0.9.0+
	EncodedToken     string `json:"encoded_token"`
	EncodedRootToken string `json:"encoded_root_token"`
	Complete         bool   `json:"complete"`
}

func NewGenRootToken() *GenRootToken {

}
