package vaultkv_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-community/vaultkv"
)

var _ = Describe("Rekey", func() {
	When("the vault is not initialized", func() {
		Describe("Starting a new rekey operation", func() {
			JustBeforeEach(func() {
				_, err = vault.NewRekey(vaultkv.RekeyConfig{
					Shares:    1,
					Threshold: 1,
				})
			})

			It("should return ErrUninitialized", AssertErrorOfType(&vaultkv.ErrUninitialized{}))
		})

		Describe("Getting the current rekey operation", func() {
			JustBeforeEach(func() {
				_, err = vault.CurrentRekey()
			})

			It("should return ErrUninitialized", AssertErrorOfType(&vaultkv.ErrUninitialized{}))
		})
	})

	When("the vault is initialized", func() {
		var initShares, initThreshold int
		var initOutput *vaultkv.InitVaultOutput
		BeforeEach(func() {
			initShares = 1
			initThreshold = 1
		})

		JustBeforeEach(func() {
			initOutput, err = vault.InitVault(vaultkv.InitConfig{
				Shares:    initShares,
				Threshold: initThreshold,
			})
			AssertNoError()()
		})

		When("Vault is sealed", func() {
			Describe("Starting a new rekey operation", func() {
				JustBeforeEach(func() {
					_, err = vault.NewRekey(vaultkv.RekeyConfig{
						Shares:    1,
						Threshold: 1,
					})
				})

				It("should return ErrSealed", AssertErrorOfType(&vaultkv.ErrSealed{}))
			})

			Describe("Getting the current rekey operation", func() {
				JustBeforeEach(func() {
					_, err = vault.CurrentRekey()
				})

				It("should return ErrSealed", AssertErrorOfType(&vaultkv.ErrSealed{}))
			})
		})

		When("Vault is unsealed", func() {
			JustBeforeEach(func() {
				err = initOutput.Unseal()
				AssertNoError()()
			})

			Describe("CurrentRekey with no rekey in progress", func() {
				JustBeforeEach(func() {
					_, err = vault.CurrentRekey()
				})

				It("should return ErrNotFound", AssertErrorOfType(&vaultkv.ErrNotFound{}))
			})

			Describe("Starting a new rekey operation", func() {
				var rekeyConf vaultkv.RekeyConfig
				var rekey *vaultkv.Rekey

				var AssertRemaining = func(rem int) func() {
					return func() {
						Expect(rekey.Remaining()).To(Equal(rem))
					}
				}

				var AssertHasKeys = func(numKeys int) func() {
					return func() {
						Expect(rekey.Keys()).To(HaveLen(numKeys))
					}
				}

				JustBeforeEach(func() {
					rekey, err = vault.NewRekey(rekeyConf)
				})

				Context("With one key in the previous initialization", func() {
					Context("With one share and threshold of one requested", func() {
						BeforeEach(func() {
							rekeyConf.Shares = 1
							rekeyConf.Threshold = 1
						})

						It("should start a rekey with no error", AssertNoError())
						Specify("remaining should report one", AssertRemaining(1))

						Describe("State", func() {
							var state vaultkv.RekeyState
							JustBeforeEach(func() {
								state = rekey.State()
								Expect(state).NotTo(BeNil())
							})

							It("should have PendingShares as one", func() { Expect(state.PendingShares).To(Equal(1)) })
							It("should have PendingThreshold as one", func() { Expect(state.PendingThreshold).To(Equal(1)) })
							It("should have Required as one", func() { Expect(state.Required).To(Equal(1)) })
							It("should have Progress as zero", func() { Expect(state.Progress).To(Equal(0)) })
						})

						Describe("Submitting one key", func() {
							var rekeyDone bool

							JustBeforeEach(func() {
								rekeyDone, err = rekey.Submit(initOutput.Keys[0])
							})

							It("should not have returned an error", AssertNoError())
							It("should say that the rekey is done", func() { Expect(rekeyDone).To(BeTrue()) })

							Describe("Keys", func() {
								var rekeyKeys []string
								JustBeforeEach(func() {
									rekeyKeys = rekey.Keys()
								})

								It("should have one key", func() { Expect(rekeyKeys).To(HaveLen(1)) })
								Specify("Remaining should return zero", AssertRemaining(0))
							})
						})

						Describe("Submitting too many keys", func() {
							var rekeyDone bool
							JustBeforeEach(func() {
								rekeyDone, err = rekey.Submit(initOutput.Keys[0], "a", "b", "c")
							})

							It("should not have returned an error", AssertNoError())
							It("should say that the rekey is done", func() { Expect(rekeyDone).To(BeTrue()) })
						})

						Describe("Submitting an incorrect key", func() {
							var rekeyDone bool
							JustBeforeEach(func() {
								//If this is somehow your unseal key, then I'm sorry
								rekeyDone, err = rekey.Submit("k8vk0IdoDeNAJl5JDJ282eehqIbRLv5WWoBy6ppBK9c=")
							})

							It("should return ErrBadRequest", AssertErrorOfType(&vaultkv.ErrBadRequest{}))
							It("should not claim to be done", func() { Expect(rekeyDone).To(BeFalse()) })
						})
					})

					Context("with improper rekey parameters", func() {
						BeforeEach(func() {
							rekeyConf.Shares = 1
							rekeyConf.Threshold = 2
						})
						It("should return ErrBadRequest", AssertErrorOfType(&vaultkv.ErrBadRequest{}))
					})

				})

				Context("With multiple keys in the previous initialization", func() {
					BeforeEach(func() {
						initShares = 3
						initThreshold = 3
					})

					Context("With one share and threshold of one requested", func() {
						BeforeEach(func() {
							rekeyConf.Shares = 1
							rekeyConf.Threshold = 1
						})

						It("should not return an error", AssertNoError())
						Specify("Remaining should return three", AssertRemaining(3))

						Describe("Submitting one key", func() {
							var rekeyDone bool
							JustBeforeEach(func() {
								rekeyDone, err = rekey.Submit(initOutput.Keys[0])
							})

							It("should not return an error", AssertNoError())
							It("should not consider the rekey done", func() { Expect(rekeyDone).To(BeFalse()) })
							Specify("Remaining should say two", AssertRemaining(2))

							Describe("Getting the existing rekey operation with CurrentRekey", func() {
								JustBeforeEach(func() {
									rekey, err = vault.CurrentRekey()
								})

								It("should not return an error", AssertNoError())
								It("should return a non-nil rekey", func() { Expect(rekey).NotTo(BeNil()) })
								Specify("The rekey object should specify two keys remaining", AssertRemaining(2))
							})

							Describe("Cancelling the rekey", func() {
								JustBeforeEach(func() {
									err = rekey.Cancel()
								})

								It("should not return an error", AssertNoError())

								Describe("Submitting after the cancellation", func() {
									var rekeyDone bool
									JustBeforeEach(func() {
										rekeyDone, err = rekey.Submit(initOutput.Keys[0])
									})

									It("should report being done", func() { Expect(rekeyDone).To(BeTrue()) })
									It("should return an ErrBadRequest", AssertErrorOfType(&vaultkv.ErrBadRequest{}))
								})
							})
						})

						Describe("Submitting all necessary keys", func() {
							var rekeyDone bool
							Context("All at once", func() {
								JustBeforeEach(func() {
									rekeyDone, err = rekey.Submit(initOutput.Keys...)
								})

								It("should not return an error", AssertNoError())
								It("should consider the rekey done", func() { Expect(rekeyDone).To(BeTrue()) })
								Specify("remaining should return 0", AssertRemaining(0))
								Specify("Keys should return 1 key", AssertHasKeys(1))
							})

							Context("One Submit call at a time", func() {
								var rekeyDone bool
								JustBeforeEach(func() {
									for _, key := range initOutput.Keys {
										rekeyDone, err = rekey.Submit(key)
										AssertNoError()
									}
								})

								It("should consider the rekey done", func() { Expect(rekeyDone).To(BeTrue()) })
								Specify("remaining should return 0", AssertRemaining(0))
								Specify("Keys should return 1 key", AssertHasKeys(1))
							})
						})
					})
				})
			})
		})
	})
})
