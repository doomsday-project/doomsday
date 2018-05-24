package vaultkv

import (
	"encoding/json"
	"strings"
)

//Rekey represents a rekey operation currently in progress in the Vault. This
// wraps an otherwise cumbersome rekey API. Remaining() can be called to see
// how many keys are still required by the rekey, and then those many keys
// can be sent through one or more calls to Submit(). This should be created
// through a call to NewRekey or CurrentRekey. Using an uninitialized Rekey
// struct will lead to undefined behavior.
type Rekey struct {
	client *Client
	state  RekeyState
	keys   []string
}

//RekeyConfig is given to NewRekey to configure the parameters of the rekey
//operation to be started.
type RekeyConfig struct {
	Shares    int      `json:"secret_shares"`
	Threshold int      `json:"secret_threshold"`
	PGPKeys   []string `json:"pgp_keys,omitempty"`
	Backup    bool     `json:"backup,omitempty"`
}

//RekeyState gives the state of the rekey operation as of the last call to
//Submit, NewRekey, or CurrentRekey.
type RekeyState struct {
	Started          bool   `json:"started"`
	Nonce            string `json:"nonce"`
	PendingThreshold int    `json:"t"`
	PendingShares    int    `json:"n"`
	//The number of keys given so far in this rekey operation
	Progress int `json:"progress"`
	//The total number of keys needed for this rekey operation
	Required        int      `json:"required"`
	PGPFingerprints []string `json:"pgp_fingerprints"`
	Backup          bool     `json:"backup"`
}

//NewRekey will cancel the current rekey if one is in progress. If starting a
// rekey is successful, a *Rekey is returned containing the necessary state
// for submitting keys for this rekey operation.
func (v *Client) NewRekey(conf RekeyConfig) (*Rekey, error) {
	err := v.rekeyCancel()
	if err != nil {
		return nil, err
	}

	err = v.rekeyStart(conf)
	if err != nil {
		return nil, err
	}

	return v.CurrentRekey()
}

//CurrentRekey returns a *Rekey with the state necessary to continue a rekey
// operation if one is in progress. If no rekey is in progress, *ErrNotFound
// is returned and no *Rekey is returned.
func (v *Client) CurrentRekey() (*Rekey, error) {
	var state RekeyState
	err := v.doSysRequest("GET", "/sys/rekey/init", nil, &state)
	if err != nil {
		return nil, err
	}

	if !state.Started {
		return nil, &ErrNotFound{message: "No rekey in progress"}
	}

	return &Rekey{
		client: v,
		state:  state,
	}, nil
}

func (v *Client) rekeyStart(conf RekeyConfig) error {
	return v.doSysRequest("PUT", "/sys/rekey/init", &conf, nil)
}

//Cancel tells Vault to forget about the current rekey operation
func (r *Rekey) Cancel() error {
	return r.client.rekeyCancel()
}

func (v *Client) rekeyCancel() error {
	return v.doSysRequest("DELETE", "/sys/rekey/init", nil, nil)
}

//Submit gives keys to the rekey operation specified by this *Rekey object. Any
//keys beyond the current required amount are ignored. If the Rekey is
//successful after all keys have been sent, then done will be returned as true.
//If the threshold is reached and any of the keys were incorrect, an
//*ErrBadRequest is returned and done is false. In this case, the rekey is not
//cancelled, but is instead reset. No error is given for an incorrect key
//before the threshold is reached. An *ErrBadRequest may also be returned if
//there is no longer any rekey in progress, but in this case, done will be
//returned as true.
func (r *Rekey) Submit(keys ...string) (done bool, err error) {
	for _, key := range keys {
		var result interface{}
		result, err = r.client.rekeySubmit(key, r.state.Nonce)
		if err != nil {
			if ebr, is400 := err.(*ErrBadRequest); is400 {
				r.state.Progress = 0
				//I really hate error string checking, but there's no good way that doesn't
				//require another API call (which could, in turn, err, and leave us in a
				//wrong state)
				if strings.Contains(ebr.message, "no rekey in progress") {
					done = true
				}
			}

			return
		}

		switch v := result.(type) {
		case *RekeyState:
			r.state = *v
		case *rekeyKeys:
			r.keys = v.Keys
			r.state = RekeyState{}
			return true, nil

		default:
			panic("rekeySubmit gave an unknown type")
		}
	}

	return false, nil
}

type rekeyKeys struct {
	Keys       []string `json:"keys"`
	KeysBase64 []string `json:"keys_base64"`
}

func (v *Client) rekeySubmit(key string, nonce string) (ret interface{}, err error) {
	tempMap := make(map[string]interface{})
	err = v.doSysRequest(
		"PUT",
		"/sys/rekey/update",
		&struct {
			Key   string `json:"key"`
			Nonce string `json:"nonce"`
		}{
			Key:   key,
			Nonce: nonce,
		},
		&tempMap,
	)
	if err != nil {
		return
	}

	jBytes, err := json.Marshal(&tempMap)
	if err != nil {
		return
	}

	var unmarshalTarget interface{} = &RekeyState{}
	if _, isComplete := tempMap["complete"]; isComplete {
		unmarshalTarget = &rekeyKeys{}
	}

	err = json.Unmarshal(jBytes, &unmarshalTarget)
	if err != nil {
		return
	}

	return unmarshalTarget, err
}

//Remaining returns the number of keys yet required by this rekey operation.
//This does not refresh state. If you believe that an external agent may have
//changed the state of the rekey, get a new rekey object with CurrentRekey, or
//Submit another key.
func (r *Rekey) Remaining() int {
	return r.state.Required - r.state.Progress
}

//State returns the current state of the rekey operation. This does not refresh
// state. If you believe that an external agent may have changed the state of
// the rekey, get a new rekey object with CurrentRekey, or Submit another key.
func (r *Rekey) State() RekeyState {
	return r.state
}

//Keys returns the new keys from this rekey operation if the operation has been
//successful. The return value is undefined if the rekey operation is not yet
//successful.
func (r *Rekey) Keys() []string {
	return r.keys
}
