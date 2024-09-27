package netboot

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const netbootDir = "/var/www/netboot"

type Config struct {
	Address string `json:"address"`
	OS      string `json:"os"`
	Version string `json:"version"`
	Serial  string `json:"serial"`
	Config  string `json:"config"`
}

type Host struct {
	Address string `json:"address"`
}

type Response struct {
	Message string `json:"message"`
}

type AddResponse struct {
	Message string   `json:"message"`
	Output  []string `json:"output"`
}

type HostListResponse struct {
	Message   string   `json:"message"`
	Addresses []string `json:"addresses"`
}

type DeleteResponse struct {
	Message string   `json:"message"`
	Files   []string `json:"files"`
}

var MAC_PATTERN = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$`)
var IPXE_PATTERN = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})\.ipxe$`)
var PKG_PATTERN = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})\.tgz$`)

func copyFile(dstPath, srcPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}

func fail(w http.ResponseWriter, message string, status int) {
	log.Printf("  [%d] %s", status, message)
	http.Error(w, message, status)
}

func respond(w http.ResponseWriter, response any) {
	log.Printf("  [200] %v", response)
	json.NewEncoder(w).Encode(response)
}

func UploadPackageHandler(w http.ResponseWriter, r *http.Request) {

	err := r.ParseMultipartForm(256 << 20) // limit file size to 256MB
	if err != nil {
		fail(w, fmt.Sprintf("failed parsing form: %v", err), http.StatusBadRequest)
		return
	}

	uploadFile, fileHeader, err := r.FormFile("uploadFile")
	if err != nil {
		fail(w, fmt.Sprintf("failed retreiving upload file: %v", err), http.StatusBadRequest)
		return
	}
	defer uploadFile.Close()

	packageFilename := fileHeader.Filename

	if !PKG_PATTERN.MatchString(packageFilename) {
		fail(w, fmt.Sprintf("illegal filename: %s", packageFilename), http.StatusBadRequest)
		return
	}

	packageFile, err := os.Create(filepath.Join(netbootDir, packageFilename))
	if err != nil {
		fail(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer packageFile.Close()
	fileBytes, err := io.Copy(packageFile, uploadFile)
	if err != nil {
		fail(w, err.Error(), http.StatusBadRequest)
		return
	}

	respond(w, Response{Message: fmt.Sprintf("%v bytes written", fileBytes)})
}

func AddHostHandler(w http.ResponseWriter, r *http.Request) {

	var in Config
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		fail(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !MAC_PATTERN.MatchString(in.Address) {
		fail(w, "invalid MAC address", http.StatusBadRequest)
		return
	}

	switch in.OS {
	case "debian":
	case "openbsd":
	default:
		fail(w, "unrecognized OS", http.StatusBadRequest)
		return
	}

	osMenuPathname := filepath.Join(netbootDir, fmt.Sprintf("netboot-%s.ipxe", in.OS))
	responsePathname := filepath.Join(netbootDir, fmt.Sprintf("%s.conf", in.Address))
	hostMenuPathname := filepath.Join(netbootDir, fmt.Sprintf("%s.ipxe", in.Address))

	err = copyFile(hostMenuPathname, osMenuPathname)
	if err != nil {
		fail(w, err.Error(), http.StatusBadRequest)
		return
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(in.Config)
	if err != nil {
		fail(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = os.WriteFile(responsePathname, decodedBytes, 0660)
	if err != nil {
		fail(w, err.Error(), http.StatusBadRequest)
		return
	}

	script := fmt.Sprintf("/root/mkboot.%s", in.OS)
	args := []string{script, in.Address}
	if in.Serial != "" {
		args = append(args, in.Serial)
	}
	cmd := exec.Command("/usr/bin/doas", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		log.Printf("script %s failed: %v\n", script, err)
		for _, eline := range strings.Split(stderr.String(), "\n") {
			log.Printf("stderr: %s\n", eline)
		}
		fail(w, fmt.Sprintf("%s: %v", script, err), http.StatusBadRequest)
		return
	}
	if cmd.ProcessState.ExitCode() != 0 {
		log.Fatalf("uncaught process failure")
	}
	outputLines := strings.Split(stdout.String(), "\n")

	respond(w, AddResponse{Message: fmt.Sprintf("%s configured", in.Address), Output: outputLines})
}

func deleteHostFiles(address string) ([]string, error) {
	deletedFiles := []string{}
	files, err := ioutil.ReadDir(netbootDir)
	if err != nil {
		return []string{}, err
	}
	pattern, err := regexp.Compile(fmt.Sprintf("^%s.*$", strings.ToLower(address)))
	if err != nil {
		return []string{}, err
	}

	for _, file := range files {
		filename := file.Name()
		if pattern.MatchString(strings.ToLower(filename)) {
			err := os.Remove(filepath.Join(netbootDir, filename))
			if err != nil {
				return []string{}, err
			}
			deletedFiles = append(deletedFiles, filename)
		}
	}
	return deletedFiles, nil
}

func DeleteHostHandler(w http.ResponseWriter, r *http.Request) {
	var in Host
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		fail(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !MAC_PATTERN.MatchString(in.Address) {
		fail(w, "invalid MAC address", http.StatusBadRequest)
		return
	}

	deleteAddressFiles(in.Address, w)
}

func deleteAddressFiles(inAddress string, w http.ResponseWriter) {
	addresses, err := hostAddresses()
	if err != nil {
		fail(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, address := range addresses {
		if strings.ToLower(inAddress) == strings.ToLower(address) {
			files, err := deleteHostFiles(address)
			if err != nil {
				fail(w, err.Error(), http.StatusBadRequest)
				return
			}
			respond(w, DeleteResponse{Message: fmt.Sprintf("deleted: %d", len(files)), Files: files})
			return
		}
	}
	fail(w, "host address not found", http.StatusNotFound)
}

func HostBootedHandler(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(r.URL.Path, "/")
	if len(segments) > 3 {
		address := segments[3]
		deleteAddressFiles(address, w)
		return
	}
	fail(w, "invalid path", http.StatusBadRequest)
}

func hostAddresses() ([]string, error) {

	addresses := []string{}
	files, err := ioutil.ReadDir(netbootDir)
	if err != nil {
		return addresses, err
	}
	for _, file := range files {
		filename := file.Name()
		if IPXE_PATTERN.MatchString(filename) {
			fields := strings.Split(filename, ".")
			addresses = append(addresses, fields[0])
		}
	}
	return addresses, nil
}

func ListHostsHandler(w http.ResponseWriter, r *http.Request) {
	addresses, err := hostAddresses()
	if err != nil {
		fail(w, err.Error(), http.StatusBadRequest)
		return
	}

	respond(w, HostListResponse{Message: fmt.Sprintf("config count: %d", len(addresses)), Addresses: addresses})
}
