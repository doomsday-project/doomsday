package vaultkv_test

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/cloudfoundry-community/vaultkv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sys", func() {

	//Uses SealStatus to get seal state
	var AssertStatusSealed = func(expected bool) func() {
		return func() {
			state, err := vault.SealStatus()
			Expect(err).NotTo(HaveOccurred())
			Expect(state).ToNot(BeNil())
			Expect(state.Sealed).To(Equal(expected))
		}
	}

	Describe("SealStatus", func() {
		JustBeforeEach(func() {
			_, err = vault.SealStatus()
		})

		When("Vault is uninitialized", func() {
			It("should return an ErrUninitialized", AssertErrorOfType(&vaultkv.ErrUninitialized{}))
		})
	})

	Describe("Initialization", func() {
		var output *vaultkv.InitVaultOutput
		var input vaultkv.InitConfig
		JustBeforeEach(func() {
			output, err = vault.InitVault(input)
		})

		var AssertHasRootToken = func() func() {
			return func() {
				Expect(output).ToNot(BeNil())
				Expect(output.RootToken).ToNot(BeEmpty())
			}
		}

		var AssertHasUnsealKeys = func(numKeys int) func() {
			return func() {
				Expect(output).ToNot(BeNil())
				Expect(output.Keys).To(HaveLen(numKeys))
				Expect(output.KeysBase64).To(HaveLen(numKeys))
				for i, key := range output.Keys {
					//Decode the base64
					buf := strings.NewReader(output.KeysBase64[i])
					b64decoder := base64.NewDecoder(base64.StdEncoding, buf)
					b64decoded, err := ioutil.ReadAll(b64decoder)
					Expect(err).NotTo(HaveOccurred(), "should not have erred on decoding base64")
					//Encode into hex
					hexEncoded := hex.EncodeToString(b64decoded)
					Expect(string(hexEncoded)).To(Equal(key),
						fmt.Sprintf("base64 string `%s' does not decode to the same string as the hex string `%s' decodes", output.KeysBase64[i], key))
				}
			}
		}

		var AssertInitializationStatus = func(expected bool) func() {
			return func() {
				actual, err := vault.IsInitialized()
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(Equal(expected))
			}
		}

		When("the Vault is not initialized", func() {
			When("there's only one secret share", func() {
				BeforeEach(func() {
					input = vaultkv.InitConfig{
						Shares:    1,
						Threshold: 1,
					}
				})

				It("should not err", AssertNoError())
				It("should return a root token", AssertHasRootToken())
				It("should have one unseal key", AssertHasUnsealKeys(1))
				It("should be initialized", AssertInitializationStatus(true))

				Describe("Unseal with an InitVaultOutput", func() {
					JustBeforeEach(func() {
						err = output.Unseal()
					})

					It("should not return an error", AssertNoError())
					Specify("SealStatus should return unsealed", func() {
						sealState, err := vault.SealStatus()
						Expect(err).NotTo(HaveOccurred())
						Expect(sealState).NotTo(BeNil())
						Expect(sealState.Sealed).To(BeFalse())
					})
				})
			})

			When("there are multiple secret shares", func() {
				BeforeEach(func() {
					input = vaultkv.InitConfig{
						Shares:    3,
						Threshold: 2,
					}
				})

				It("should not err", AssertNoError())
				It("should return a root token", AssertHasRootToken())
				It("should have three unseal keys", AssertHasUnsealKeys(3))
				It("should be initialized", AssertInitializationStatus(true))
			})

			When("0 secret shares are requested", func() {
				BeforeEach(func() {
					input = vaultkv.InitConfig{
						Shares:    0,
						Threshold: 0,
					}

					It("should return an ErrBadRequest", AssertErrorOfType(&vaultkv.ErrBadRequest{}))
					It("should be initialized", AssertInitializationStatus(false))
				})
			})

			When("the threshold is larger than the number of shares", func() {
				BeforeEach(func() {
					input = vaultkv.InitConfig{
						Shares:    3,
						Threshold: 4,
					}

					It("should return an ErrBadRequest", AssertErrorOfType(&vaultkv.ErrBadRequest{}))
					It("should be initialized", AssertInitializationStatus(false))
				})
			})
		})

		When("the Vault has already been initialized", func() {
			BeforeEach(func() {
				input = vaultkv.InitConfig{
					Shares:    1,
					Threshold: 1,
				}
				_, err = vault.InitVault(input)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an ErrBadRequest", AssertErrorOfType(&vaultkv.ErrBadRequest{}))
			It("should be initialized", AssertInitializationStatus(true))
		})
	})

	Describe("Unseal", func() {
		var output *vaultkv.SealState
		var unsealKey string

		BeforeEach(func() {
			unsealKey = "pLacEhoLdeR="
		})
		JustBeforeEach(func() {
			output, err = vault.Unseal(unsealKey)
		})

		var AssertSealed = func(expected bool) func() {
			return func() {
				Expect(output).ToNot(BeNil())
				Expect(output.Sealed).To(Equal(expected))
			}
		}

		var AssertProgressIs = func(expected int) func() {
			return func() {
				state, err := vault.SealStatus()
				Expect(err).NotTo(HaveOccurred())
				Expect(state.Progress).To(Equal(expected))
			}
		}

		When("the vault is uninitialized", func() {
			It("should return an ErrUninitialized", AssertErrorOfType(&vaultkv.ErrUninitialized{}))
		})

		When("the vault is initialized", func() {
			var initOut *vaultkv.InitVaultOutput
			Context("with one share", func() {
				BeforeEach(func() {
					initOut, err = vault.InitVault(vaultkv.InitConfig{
						Shares:    1,
						Threshold: 1,
					})
				})

				When("unseal key is correct", func() {
					BeforeEach(func() {
						unsealKey = initOut.Keys[0]
					})

					It("should not return an error", AssertNoError())
					Specify("Unseal should return that Vault is unsealed", AssertSealed(false))
					Specify("SealStatus should return that the Vault is unsealed", AssertStatusSealed(false))

					When("the auth token is missing", func() {
						JustBeforeEach(func() {
							vault.AuthToken = ""
						})

						Specify("SealStatus should return that the vault is unsealed", AssertStatusSealed(false))
					})

					When("an unseal is attempted after the vault is unsealed", func() {
						BeforeEach(func() {
							_, err = vault.Unseal(unsealKey)
							Expect(err).NotTo(HaveOccurred())
						})

						It("should not return an error", AssertNoError())
						Specify("Unseal should return that Vault is unsealed", AssertSealed(false))
						Specify("SealStatus should return that the Vault is unsealed", AssertStatusSealed(false))
					})

					When("an unseal reset is requested after the vault is unsealed", func() {
						It("should not return an error", AssertNoError())
						Specify("Unseal should return that Vault is unsealed", AssertSealed(false))
						Specify("SealStatus should return that the Vault is unsealed", AssertStatusSealed(false))
					})
				})

				When("the unseal key is wrong", func() {
					BeforeEach(func() {
						unsealKey = initOut.Keys[0]
						replacementChar := "a"
						if unsealKey[0] == 'a' {
							replacementChar = "b"
						}

						unsealKey = fmt.Sprintf("%s%s", replacementChar, unsealKey[1:])
					})

					It("should return an ErrBadRequest", AssertErrorOfType(&vaultkv.ErrBadRequest{}))
					Specify("SealStatus should return that the Vault is still sealed", AssertStatusSealed(true))
				})

				When("the unseal key is improperly formatted", func() {
					It("should return an ErrBadRequest", AssertErrorOfType(&vaultkv.ErrBadRequest{}))
					Specify("SealStatus should return that the Vault is still sealed", AssertStatusSealed(true))
				})
			})

			Context("with a threshold greater than one", func() {
				BeforeEach(func() {
					initOut, err = vault.InitVault(vaultkv.InitConfig{
						Shares:    3,
						Threshold: 3,
					})
				})

				When("the unseal key is improperly formatted", func() {
					It("should return an ErrBadRequest", AssertErrorOfType(&vaultkv.ErrBadRequest{}))
					It("should not have increased the progress count", AssertProgressIs(0))
					Specify("SealStatus should return that the Vault is still sealed", AssertStatusSealed(true))
				})

				When("the unseal key is correct", func() {
					BeforeEach(func() {
						unsealKey = initOut.Keys[0]
					})

					It("should not return an error", AssertNoError())
					It("should increase the progress count", AssertProgressIs(1))
					Specify("Unseal should return that the vault is still sealed", AssertSealed(true))
					Specify("SealStatus should return that the Vault is still sealed", AssertStatusSealed(true))

					Context("and then another key is given", func() {
						JustBeforeEach(func() {
							output, err = vault.Unseal(initOut.Keys[1])
						})

						It("should not return an error", AssertNoError())
						It("should increase the progress count", AssertProgressIs(2))
						Specify("Unseal should return that the vault is still sealed", AssertSealed(true))
						Specify("SealStatus should return that the Vault is still sealed", AssertStatusSealed(true))
					})

					Context("and then the unseal attempt is reset", func() {
						JustBeforeEach(func() {
							err = vault.ResetUnseal()
						})

						It("should not return an error", AssertNoError())
						It("should reset the progress count", AssertProgressIs(0))
					})
				})
			})
		})
	})

	Describe("Seal", func() {
		JustBeforeEach(func() {
			err = vault.Seal()
		})

		When("the vault is not initialized", func() {
			It("should not return an error", AssertNoError())
		})

		When("the vault is initialized", func() {
			var initOut *vaultkv.InitVaultOutput
			BeforeEach(func() {
				initOut, err = vault.InitVault(vaultkv.InitConfig{
					Shares:    1,
					Threshold: 1,
				})
			})
			When("the vault is already sealed", func() {
				It("should not return an error", AssertNoError())
				Specify("The vault should be sealed", AssertStatusSealed(true))
			})

			When("the vault is unsealed", func() {
				BeforeEach(func() {
					sealState, err := vault.Unseal(initOut.Keys[0])
					Expect(err).NotTo(HaveOccurred())
					Expect(sealState).NotTo(BeNil())
					Expect(sealState.Sealed).To(BeFalse())
				})
				It("should not return an error", AssertNoError())
				Specify("The vault should be sealed", AssertStatusSealed(true))

				Context("but the user is not authenticated", func() {
					BeforeEach(func() {
						vault.AuthToken = ""
					})

					It("should return ErrForbidden", AssertErrorOfType(&vaultkv.ErrForbidden{}))
					Specify("The vault should remain unsealed", AssertStatusSealed(false))
				})
			})

		})
	})

	Describe("Health", func() {
		JustBeforeEach(func() {
			err = vault.Health(true)
		})

		When("the vault is not initialized", func() {
			It("should return ErrUninitialized", AssertErrorOfType(&vaultkv.ErrUninitialized{}))
		})

		When("the vault is initialized", func() {
			var initOut *vaultkv.InitVaultOutput
			BeforeEach(func() {
				initOut, err = vault.InitVault(vaultkv.InitConfig{
					Shares:    1,
					Threshold: 1,
				})
			})

			When("the vault is sealed", func() {
				It("should return ErrSealed", AssertErrorOfType(&vaultkv.ErrSealed{}))
			})

			When("the vault is unsealed", func() {
				BeforeEach(func() {
					sealState, err := vault.Unseal(initOut.Keys[0])
					Expect(err).NotTo(HaveOccurred())
					Expect(sealState).NotTo(BeNil())
					Expect(sealState.Sealed).To(BeFalse())
				})

				It("should not return an error", AssertNoError())

				When("the auth token is wrong", func() {
					BeforeEach(func() {
						vault.AuthToken = ""
					})

					It("should not return an error", AssertNoError())
				})
			})
		})
	})
})
