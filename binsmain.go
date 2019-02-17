package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

var nameFlg string
var myValue string

// kofi run command arguments
// go run binsmain.go run command arguments
func main() {

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	case "init":
		nameFlg := flag.NewFlagSet("", flag.ExitOnError)
		nameFlg.StringVar(&myValue, "name", "", "container name")
		nameFlg.Parse(os.Args[2:])
		fmt.Println("name:", myValue)
		dlandunpack()

	}

}

type containerData struct {
	ContainerName string `json: msgationName`
}

type msg struct {
	msgone string `json: msgone`
	msgtwo string `json: msgtwo`
}

func containerHandler(w http.ResponseWriter, r *http.Request) {
	msgation := msg{}
	log.Println(r.Method)
	jsn, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal("Error reading", err)
	}
	err = json.Unmarshal(jsn, &msgation)

	log.Printf("Received: %v\n", msgation)

	x, err := ioutil.ReadFile("/tmp/dat1")
	if err != nil {
		panic(err)
	}

	s := string(x)

	container := containerData{
		ContainerName: s,
	}
	containerJson, err := json.Marshal(container)
	if err != nil {
		fmt.Fprintf(w, "Error: %s", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(containerJson)

}
func server() {
	http.HandleFunc("/", containerHandler)
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

	//INSERT READFILE

	r, err := ioutil.ReadFile("/tmp/dat1")
	if err != nil {
		panic(err)
	}

	s := r
	var a []byte
	copy(a[:], s)

	must(syscall.Chroot("ubuntufs"))
	must(os.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	syscall.Sethostname(r)
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

	w := []byte(myValue)
	z := ioutil.WriteFile("/tmp/dat1", w, 0644)
	if z != nil {
		panic(z)
	}

	os.Mkdir(filepath.Join("ubuntufs"), 0777)
	fmt.Println("Created FS")
	os.Chdir("./ubuntufs")
	fmt.Println("Downloading ubuntu-fs")
	fileUrl := "https://partner-images.canonical.com/core/trusty/current/ubuntu-trusty-core-cloudimg-amd64-root.tar.gz"

	err := DownloadFile("ubuntu.tar.gz", fileUrl)
	if err != nil {
		panic(err)
	}
	fmt.Println("Finished downloading")
	// Extracts the archive in the newly created folder
	cmd := exec.Command("tar", "-xvzf", "ubuntu.tar.gz")
	fmt.Println("Finished Extracting")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Run())
	fmt.Println("Your container is ready for use")

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
