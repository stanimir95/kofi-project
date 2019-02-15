package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// kofi run command arguments
// go run binsmain.go run command arguments
func main() {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
		//
	case "init":
		dlandunpack()
	default:
		panic("what?")
	}
}

type weatherData struct {
	ContainerName string `json: locationName`
}

type loc struct {
	Lat string `json: lat`
	Lon string `json: lon`
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	location := loc{}

	log.Println(r.Method)
	jsn, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal("Error reading the body", err)
	}
	err = json.Unmarshal(jsn, &location)
	if err != nil {
		log.Fatal("Decoding error: ", err)
	}
	log.Printf("Received: %v\n", location)
	weather := weatherData{
		ContainerName: "Zzyzx",
	}
	weatherJson, err := json.Marshal(weather)
	if err != nil {
		fmt.Fprintf(w, "Error: %s", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(weatherJson)

}
func server() {
	http.HandleFunc("/", weatherHandler)
	http.ListenAndServe(":8080", nil)
}

func run() {
	go server()
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS, //currently working on network separation
	}
	must(cmd.Run())
}

func child() {
	fmt.Printf("running %v as PID %d\n", os.Args[2:], os.Getpid()) // Expected output: [<for instance bin/bash> started as PID 1]
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Random container name
	a := 1000
	b := 2000
	rand.Seed(time.Now().UnixNano())
	n := a + rand.Intn(b-a+1)
	ns := []byte(strconv.Itoa(n))
	//-------------------------------
	//------------------------------
	must(syscall.Chroot("ubuntufs"))
	must(os.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	must(syscall.Sethostname(ns)) //Name of container
	must(cmd.Run())

	must(syscall.Unmount("proc", 0)) // Unmounting proc is a must, otherwise after exiting from the container, the host
	// os will still use the newly created namespaces and will be unusable
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

//from here is the "init" command)
// dlunpack creates a folder and downloads an archive in the directory that "kofi init" has been run
func dlandunpack() {
	os.Mkdir(filepath.Join("ubuntufs"), 0777)
	os.Chdir("./ubuntufs")
	fileUrl := "https://partner-images.canonical.com/core/trusty/current/ubuntu-trusty-core-cloudimg-amd64-root.tar.gz"

	err := DownloadFile("ubuntu.tar.gz", fileUrl)
	if err != nil {
		panic(err)
	}
	// Extracts the archive in the newly created folder
	cmd := exec.Command("tar", "-xvzf", "ubuntu.tar.gz")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Run())

}

func DownloadFile(filepath string, url string) error {

	// Create the archive file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Download
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
