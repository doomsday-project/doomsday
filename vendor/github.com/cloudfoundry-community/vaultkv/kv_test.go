package vaultkv_test

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cloudfoundry-community/vaultkv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kv", func() {
	//This is a hack because I don't want to refactor _everything_ in these tests
	var getAssertionPath string

	var AssertGetEquals = func(expected map[string]string) func() {
		return func() {
			output := make(map[string]string)
			err = vault.Get(getAssertionPath, &output)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal(expected))
		}
	}

	var AssertExists = func(exists bool) func() {
		var fn func()
		if exists {
			fn = func() {
				err = vault.Get(getAssertionPath, nil)
				AssertNoError()()
			}
		} else {
			fn = func() {
				err = vault.Get(getAssertionPath, nil)
				AssertErrorOfType(&vaultkv.ErrNotFound{})
			}
		}

		return fn
	}

	When("the vault is not initialized", func() {
		Describe("Get", func() {
			JustBeforeEach(func() {
				err = vault.Get("secret/sure/whatever", nil)
			})
			It("should return ErrUninitialized", AssertErrorOfType(&vaultkv.ErrUninitialized{}))
		})

		Describe("Set", func() {
			JustBeforeEach(func() {
				err = vault.Set("secret/sure/whatever", map[string]string{"foo": "bar"})
			})
			It("should return ErrUninitialized", AssertErrorOfType(&vaultkv.ErrUninitialized{}))
		})

		Describe("Delete", func() {
			JustBeforeEach(func() {
				err = vault.Delete("secret/sure/whatever")
			})
			It("should return ErrUninitialized", AssertErrorOfType(&vaultkv.ErrUninitialized{}))
		})

		Describe("List", func() {
			JustBeforeEach(func() {
				_, err = vault.List("secret/sure/whatever")
			})

			It("should return ErrUninitialized", AssertErrorOfType(&vaultkv.ErrUninitialized{}))
		})
	})

	When("the vault is initialized", func() {
		var initOut *vaultkv.InitVaultOutput
		BeforeEach(func() {
			initOut, err = vault.InitVault(vaultkv.InitConfig{
				Shares:    1,
				Threshold: 1,
			})
			AssertNoError()()
		})

		When("the vault is sealed", func() {
			Describe("Get", func() {
				JustBeforeEach(func() {
					err = vault.Get("secret/sure/whatever", nil)
				})
				It("should return ErrSealed", AssertErrorOfType(&vaultkv.ErrSealed{}))
			})

			Describe("Set", func() {
				JustBeforeEach(func() {
					err = vault.Set("secret/sure/whatever", map[string]string{"foo": "bar"})
				})
				It("should return ErrSealed", AssertErrorOfType(&vaultkv.ErrSealed{}))
			})

			Describe("Delete", func() {
				JustBeforeEach(func() {
					err = vault.Delete("secret/sure/whatever")
				})
				It("should return ErrSealed", AssertErrorOfType(&vaultkv.ErrSealed{}))
			})

			Describe("List", func() {
				JustBeforeEach(func() {
					_, err = vault.List("secret/sure/whatever")
				})

				It("should return ErrSealed", AssertErrorOfType(&vaultkv.ErrSealed{}))
			})
		})

		When("the vault is unsealed", func() {
			BeforeEach(func() {
				_, err = vault.Unseal(initOut.Keys[0])
				AssertNoError()
			})

			Describe("Setting something in the Vault", func() {
				var testPath string
				var testValue map[string]string
				BeforeEach(func() {
					testPath = "secret/foo"
					testValue = map[string]string{
						"foo":  "bar",
						"beep": "boop",
					}
				})

				JustBeforeEach(func() {
					err = vault.Set(testPath, testValue)
				})

				When("the value is nil", func() {
					BeforeEach(func() {
						testValue = nil
						getAssertionPath = testPath
					})

					It("should return ErrBadRequest", AssertErrorOfType(&vaultkv.ErrBadRequest{}))
					Specify("Get should find no key at this path", AssertExists(false))
				})

				When("the path is doesn't correspond to a mounted backend", func() {
					BeforeEach(func() {
						testPath = "notabackend/foo"
					})

					It("should return ErrNotFound", AssertErrorOfType(&vaultkv.ErrNotFound{}))
					Specify("Get should find no key at this path", AssertExists(false))
				})

				When("the path has a leading slash", func() {
					BeforeEach(func() {
						testPath = "/secret/foo"
						getAssertionPath = strings.TrimPrefix(testPath, "/")
					})

					It("should not return an error", AssertNoError())
					Specify("Get should find the key at the path without a slash",
						AssertExists(true))
					Specify("Get should find the inserted value at the path without a slash",
						AssertGetEquals(map[string]string{"foo": "bar", "beep": "boop"}))
				})

				When("the path has a trailing slash", func() {
					BeforeEach(func() {
						testPath = "secret/foo/"
						getAssertionPath = strings.TrimSuffix(testPath, "/")
					})

					It("should not return an error", AssertNoError())
					Specify("Get should find the key at the path without a slash",
						AssertExists(true))
					Specify("Get should find the inserted value at the path without a slash",
						AssertGetEquals(map[string]string{"foo": "bar", "beep": "boop"}))
				})

				When("setting an already set key", func() {
					var secondTestValue map[string]string
					BeforeEach(func() {
						secondTestValue = map[string]string{
							"thisisanotherkey": "thisisanothervalue",
						}
						getAssertionPath = testPath
					})

					JustBeforeEach(func() {
						err = vault.Set(testPath, secondTestValue)
					})

					It("should not return an error", AssertNoError())
					Specify("Get should find the value that was added second",
						AssertGetEquals(map[string]string{"thisisanotherkey": "thisisanothervalue"}))
				})

				Describe("Get", func() {
					var getTestPath string
					var getOutputValue map[string]string

					var AssertGetEqualsSet = func() func() {
						return func() {
							Expect(getOutputValue).To(Equal(testValue))
						}
					}
					BeforeEach(func() {
						getOutputValue = make(map[string]string)
					})

					JustBeforeEach(func() {
						err = vault.Get(getTestPath, &getOutputValue)
					})

					When("the key exists", func() {
						BeforeEach(func() {
							getTestPath = testPath
						})

						It("should not return an error", AssertNoError())
						It("should return the same value as what was inserted", AssertGetEqualsSet())
					})

					When("the key doesn't exist", func() {
						BeforeEach(func() {
							getTestPath = fmt.Sprintf("%sabcd", testPath)
						})

						It("should return ErrNotFound", AssertErrorOfType(&vaultkv.ErrNotFound{}))
					})
				})

				Describe("Delete", func() {
					var deleteTestPath string
					JustBeforeEach(func() {
						err = vault.Delete(deleteTestPath)
					})

					When("the key exists", func() {
						BeforeEach(func() {
							deleteTestPath = testPath
							getAssertionPath = testPath
						})

						It("should not return an error", AssertNoError())
						Specify("Get should not find the key", AssertExists(false))
					})

					When("the key doesn't exist", func() {
						BeforeEach(func() {
							deleteTestPath = fmt.Sprintf("%sabcd", testPath)
						})

						It("should not return an error", AssertNoError())
					})
				})

				Describe("Adding another key with multiple parts", func() {
					var secondTestPath string
					var secondTestValue map[string]string
					BeforeEach(func() {
						secondTestPath = "secret/foo/bar"
						secondTestValue = map[string]string{
							"werealljustlittlebabybirds": "peckingourwayoutofourshells",
						}
					})

					JustBeforeEach(func() {
						err = vault.Set(secondTestPath, secondTestValue)
						AssertNoError()()
					})

					Describe("List", func() {
						var listTestPath string
						var listTestOutput []string

						JustBeforeEach(func() {
							listTestOutput, err = vault.List(listTestPath)
						})

						//Order doesn't matter
						var AssertListEquals = func(expected []string) func() {
							return func() {
								Expect(listTestOutput).ToNot(BeNil())
								Expect(expected).ToNot(BeNil())
								sort.Strings(expected)
								sort.Strings(listTestOutput)
								Expect(listTestOutput).To(Equal(expected))
							}
						}

						Context("on `secret'", func() {
							BeforeEach(func() {
								listTestPath = "secret"
							})

							It("should not return an error", AssertNoError())
							It("should return the correct list of paths", AssertListEquals([]string{"foo", "foo/"}))
						})

						Context("on the dir of the nested key", func() {
							BeforeEach(func() {
								listTestPath = "secret/foo"
							})

							It("should not return an error", AssertNoError())
							It("should return the correct list of paths", AssertListEquals([]string{"bar"}))
						})

						When("the path doesn't exist", func() {
							BeforeEach(func() {
								listTestPath = "secret/boo/hiss"
							})

							It("should return an ErrNotFound", AssertErrorOfType(&vaultkv.ErrNotFound{}))
						})
					})

					Describe("Get", func() {
						var getTestPath string
						var getOutputValue map[string]string

						BeforeEach(func() {
							getOutputValue = make(map[string]string)
						})

						JustBeforeEach(func() {
							err = vault.Get(getTestPath, &getOutputValue)
						})
						Context("on the nested key", func() {
							BeforeEach(func() {
								getTestPath = secondTestPath
							})

							It("should not return an error", AssertNoError())
							It("should return the correct value", func() {
								Expect(getOutputValue).To(Equal(secondTestValue))
							})
						})
					})

					Describe("Delete", func() {
						var deleteTestPath string
						JustBeforeEach(func() {
							err = vault.Delete(deleteTestPath)
						})
						Context("on the nested key", func() {
							BeforeEach(func() {
								deleteTestPath = secondTestPath
								getAssertionPath = deleteTestPath
							})

							It("should not return an error", AssertNoError())
							Specify("Get should not find the key", AssertExists(false))
						})
					})
				})
			})
		})
	})
})
