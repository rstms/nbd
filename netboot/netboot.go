package netboot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
var PKG_PATTERN = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})-package\.tgz$`)

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

func UploadPackageHandler(w http.ResponseWriter, r *http.Request) {

	err := r.ParseMultipartForm(256 << 20) // limit file size to 256MB
	if err != nil {
		http.Error(w, fmt.Sprintf("failed parsing form: %v", err), http.StatusBadRequest)
		return
	}

	uploadFile, fileHeader, err := r.FormFile("uploadFile")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed retreiving upload file: %v", err), http.StatusBadRequest)
		return
	}
	defer uploadFile.Close()

	packageFilename := fileHeader.Filename

	if !PKG_PATTERN.MatchString(packageFilename) {
		http.Error(w, fmt.Sprintf("illegal filename: %s", packageFilename), http.StatusBadRequest)
		return
	}

	packageFile, err := os.Create(filepath.Join(netbootDir, packageFilename))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer packageFile.Close()
	fileBytes, err := io.Copy(packageFile, uploadFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var out = Response{Message: fmt.Sprintf("%v bytes written", fileBytes)}
	json.NewEncoder(w).Encode(out)
}

func AddHostHandler(w http.ResponseWriter, r *http.Request) {

	var in Config
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !MAC_PATTERN.MatchString(in.Address) {
		http.Error(w, "invalid MAC address", http.StatusBadRequest)
		return
	}
	var osMenuPathname string
	var responsePathname string
	switch in.OS {
	case "debian":
		osMenuPathname = filepath.Join(netbootDir, "netboot-debian.ipxe")
		responsePathname = filepath.Join(netbootDir, fmt.Sprintf("%s-preseed.conf", in.Address))
	case "openbsd":
		osMenuPathname = filepath.Join(netbootDir, "netboot-openbsd.ipxe")
		responsePathname = filepath.Join(netbootDir, fmt.Sprintf("%s-install.conf", in.Address))
	default:
		http.Error(w, "unrecognized OS", http.StatusBadRequest)
		return
	}

	hostMenuPathname := filepath.Join(netbootDir, fmt.Sprintf("%s.ipxe", in.Address))

	err = copyFile(hostMenuPathname, osMenuPathname)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(in.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = os.WriteFile(responsePathname, decodedBytes, 0660)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	outputLines := []string{}

	if in.OS == "debian" {
		cmd := exec.Command("/usr/bin/doas", "/usr/local/bin/mkinitrd.debian", in.Address)
		output, err := cmd.CombinedOutput()
		if err != nil {
			http.Error(w, fmt.Sprintf("mkinitrd.debian: %v", err), http.StatusBadRequest)
			return
		}
		if len(output) > 0 {
			outputLines = strings.Split(string(output), "\n")
		}
	}

	var out = AddResponse{Message: fmt.Sprintf("%s configured", in.Address), Output: outputLines}
	json.NewEncoder(w).Encode(out)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !MAC_PATTERN.MatchString(in.Address) {
		http.Error(w, "invalid MAC address", http.StatusBadRequest)
		return
	}

	addresses, err := hostAddresses()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, address := range addresses {
		if strings.ToLower(in.Address) == strings.ToLower(address) {
			files, err := deleteHostFiles(address)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var response = DeleteResponse{Message: fmt.Sprintf("deleted: %d", len(files)), Files: files}
			json.NewEncoder(w).Encode(response)
			return
		}
	}
	http.Error(w, "host address not found", http.StatusNotFound)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var out = HostListResponse{Message: fmt.Sprintf("config count: %d", len(addresses)), Addresses: addresses}
	json.NewEncoder(w).Encode(out)
}
