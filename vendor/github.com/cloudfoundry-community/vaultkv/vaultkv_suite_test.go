package vaultkv_test

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/cloudfoundry-community/vaultkv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVaultkv(t *testing.T) {

	BeforeEach(func() {
		StartVault(currentVaultVersion)
		vault = NewTestClient()
	})

	AfterEach(StopVault)
	RegisterFailHandler(Fail)
	for i, version := range vaultVersions {
		currentVaultVersion = version
		RunSpecs(t, fmt.Sprintf("Vaultkv - Vault Version %s", currentVaultVersion))
		if i != len(vaultVersions)-1 {
			fmt.Println("")
			fmt.Println("")
			fmt.Println("========================================================")
			fmt.Println(`|/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\|`)
			fmt.Println("========================================================")
			fmt.Println("")
		}
	}
}

func init() {
	vaultVersions = []string{
		"0.6.5",
		"0.7.3",
		"0.8.3",
		"0.9.6",
	}

	if os.Getenv("VAULTKV_TEST_ONLY_LATEST") != "" {
		vaultVersions = vaultVersions[len(vaultVersions)-1:]
	}
}

//The current vault client used by each spec
var vault *vaultkv.Client
var err error

var vaultVersions []string
var currentVaultVersion string

var currentVaultProcess *os.Process
var processChan = make(chan *os.ProcessState)
var processOutputWriter, processOutputReader *os.File

var (
	vaultProcessLocation string
	configLocation       string
	certLocation         string
	keyLocation          string
)

var vaultURI *url.URL

func buildVaultPath(version string) string {
	return fmt.Sprintf("/tmp/testvaults/vault-%s-%s", runtime.GOOS, version)
}

func downloadVault(version string) error {
	fmt.Printf("Downloading Vault version %s... ", version)
	_, err := os.Stat(filepath.Dir(buildVaultPath(version)))
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(filepath.Dir(buildVaultPath(version)), 0755)
			if err != nil {
				return fmt.Errorf("Could not create dir `%s': %s", filepath.Dir(buildVaultPath(version)), err.Error())
			}
		}

		if err != nil {
			return fmt.Errorf("Could not stat `%s': %s", filepath.Dir(buildVaultPath(version)), err.Error())
		}
	}

	vaultZipFile, err := os.OpenFile(fmt.Sprintf("%s.zip", buildVaultPath(version)),
		os.O_RDWR|os.O_CREATE|os.O_EXCL|os.O_TRUNC,
		0755)
	if err != nil {
		return fmt.Errorf("Could not open Vault target zip file `%s.zip' for writing: %s", buildVaultPath(version), err.Error())
	}

	vaultDownloadURL := fmt.Sprintf("https://releases.hashicorp.com/vault/%[1]s/vault_%[1]s_%[2]s_%[3]s.zip",
		version,
		runtime.GOOS,
		runtime.GOARCH,
	)
	resp, err := http.Get(vaultDownloadURL)

	if err != nil {
		return fmt.Errorf("Could not download Vault from URL `%s': %s", vaultDownloadURL, err.Error())
	}

	bytesRead, err := io.Copy(vaultZipFile, resp.Body)
	if err != nil {
		return fmt.Errorf("Error when reading response body: %s", err.Error())
	}
	if bytesRead == 0 {
		return fmt.Errorf("No Vault binary was recieved from the remote")
	}

	zipReader, err := zip.NewReader(vaultZipFile, bytesRead)
	if err != nil {
		return fmt.Errorf("Could not prepare `%s' for zip decompression: %s", vaultZipFile.Name(), err.Error())
	}

	zipFile, err := zipReader.File[0].Open()
	if err != nil {
		return fmt.Errorf("Could not open first (and hopefully only) file in Vault zip archive: %s", err.Error())
	}

	vaultFile, err := os.OpenFile(buildVaultPath(version),
		os.O_RDWR|os.O_CREATE|os.O_EXCL|os.O_TRUNC,
		0755)
	if err != nil {
		return fmt.Errorf("Could not open Vault target file `%s' for writing: %s", buildVaultPath(version), err.Error())
	}

	_, err = io.Copy(vaultFile, zipFile)
	if err != nil {
		return fmt.Errorf("Could not unzip vault binary: %s", err.Error())
	}

	err = vaultZipFile.Close()
	if err != nil {
		return fmt.Errorf("Could not close vault zip file")
	}
	err = os.Remove(vaultZipFile.Name())
	if err != nil {
		return fmt.Errorf("Could not clean up vault zip file")
	}

	err = vaultFile.Close()
	if err != nil {
		return fmt.Errorf("Could not close vault file")
	}

	fmt.Printf("Successfully downloaded Vault version %s\n", version)
	return nil
}

var _ = BeforeSuite(func() {
	var err error
	const uriStr = "https://127.0.0.1:8201"
	vaultURI, err = url.Parse(uriStr)
	if err != nil {
		panic(fmt.Sprintf("Could not parse Vault URI: %s", uriStr))
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(fmt.Sprintf("Could not generate private key: %s", err))
	}

	templateCert := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Second),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:         true,
	}
	cert, err := x509.CreateCertificate(
		rand.Reader,
		&templateCert,
		&templateCert,
		privateKey.Public(),
		privateKey)
	if err != nil {
		panic(fmt.Sprintf("Could not generate certificate: %s", err))
	}

	certFile, err := ioutil.TempFile(os.TempDir(), "vaultkv-test-cert")
	if err != nil {
		panic(fmt.Sprintf("Could not make temp file for cert: %s", err))
	}
	certLocation = certFile.Name()

	err = pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
	if err != nil {
		panic(fmt.Sprintf("Could not write test certificate to file: %s", err))
	}

	keyFile, err := ioutil.TempFile(os.TempDir(), "vaultkv-test-key")
	if err != nil {
		panic(fmt.Sprintf("Could not make temp file for key: %s", err))
	}

	keyLocation = keyFile.Name()
	err = pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		panic(fmt.Sprintf("Could not write test key to file: %s", err))
	}

	configFile, err := ioutil.TempFile(os.TempDir(), "vaultkv-test-config")
	if err != nil {
		panic(fmt.Sprintf("Could not make temp file for cert: %s", err))
	}
	configLocation = configFile.Name()
	var vaultConfig = fmt.Sprintf(`
backend "inmem" {}

disable_mlock = true

listener "tcp" {
  address = "%s"
  tls_cert_file = "%s"
  tls_key_file = "%s"
}
`, vaultURI.Host, certLocation, keyLocation)
	_, err = configFile.WriteString(vaultConfig)
	if err != nil {
		panic(fmt.Sprintf("Could not write test config to file: %s", err))
	}

})

var _ = AfterSuite(func() {
	if configLocation != "" {
		os.Remove(configLocation)
	}

	if keyLocation != "" {
		os.Remove(keyLocation)
	}

	if certLocation != "" {
		os.Remove(certLocation)
	}

	if currentVaultProcess != nil {
		StopVault()
	}
})

func StartVault(version string) {
	if currentVaultProcess != nil {
		panic("Clean up your vault process")
	}

	var err error
	_, err = os.Stat(buildVaultPath(version))
	if err != nil {
		if !os.IsNotExist(err) {
			panic(fmt.Sprintf("Could not lookup Vault path `%s': %s", buildVaultPath(version), err.Error()))
		}

		err = downloadVault(version)
		if err != nil {
			panic(fmt.Sprintf("When downloading Vault version `%s': %s", version, err.Error()))
		}
	}

	//Gotta get that IPC from Vault in case we want to report errors
	processOutputReader, processOutputWriter, err = os.Pipe()
	if err != nil {
		panic("Could not set up IPC file descriptors")
	}

	loggingBuffer := &bytes.Buffer{}

	go io.Copy(loggingBuffer, processOutputReader)
	defer func() {
		if currentVaultProcess == nil {
			io.Copy(GinkgoWriter, loggingBuffer)
		}
	}()

	process, err := os.StartProcess(
		buildVaultPath(version), []string{buildVaultPath(version), "server", "-config", configLocation},
		&os.ProcAttr{
			Files: []*os.File{
				nil,                 //STDIN
				processOutputWriter, //STDOUT
				processOutputWriter, //STDERR
			},
		},
	)
	if err != nil {
		panic(fmt.Sprintf("Could not start Vault process: %s", err))
	}

	go func() {
		pState, err := process.Wait()
		if err != nil {
			panic(fmt.Sprintf("Err encountered while waiting on vault process: %s", err))
		}

		processChan <- pState
	}()

	startTime := time.Now()
	nextWarning := 5 * time.Second
	everySoOften := time.NewTicker(100 * time.Millisecond)
	client := NewTestClient()

	for {
		select {
		case <-everySoOften.C:
			err = client.Health(true)
			if err != nil {
				if _, isUninitialized := err.(*vaultkv.ErrUninitialized); isUninitialized {
					goto getMeOuttaHere
				}
			}

			if time.Since(startTime) > nextWarning {
				fmt.Printf("Been waiting for Vault server to start for %d seconds...\n", int64(nextWarning/time.Second))
				fmt.Println(err)
				nextWarning += 1 * time.Second
			}

		case <-processChan:
			panic("Vault exited prematurely")
		}
	}
getMeOuttaHere:

	currentVaultProcess = process
	everySoOften.Stop()
}

func StopVault() {
	if currentVaultProcess == nil {
		panic("No vault process to end")
	}

	err := currentVaultProcess.Signal(os.Interrupt)
	if err != nil {
		panic(fmt.Sprintf("Could not send interrupt signal to Vault process: %s", err))
	}

	_ = <-processChan
	processOutputWriter.Close()
	processOutputReader.Close()

	currentVaultProcess = nil
}

func NewTestClient() *vaultkv.Client {
	return &vaultkv.Client{
		VaultURL: vaultURI,
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		Trace: GinkgoWriter,
	}
}

func AssertNoError() func() {
	return func() {
		Expect(err).NotTo(HaveOccurred())
	}
}

func AssertErrorOfType(t interface{}) func() {
	return func() {
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(t))
	}
}
