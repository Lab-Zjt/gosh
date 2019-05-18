package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// Args is too few to build the target or file is not a Go file.
	if len(os.Args) < 2 || !strings.HasSuffix(os.Args[1], ".go") {
		fmt.Println("No Go Source File Found.")
		os.Exit(-1)
	}
	// Create temporary directory.
	td := CreateTempDir()
	// Open Go File
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalln(err)
		os.Exit(-1)
	}
	defer f.Close()
	// Check if it is need to rebuild.
	if exist, name := CheckMetaInfo(f, td); exist {
		Run(name)
	} else {
		Run(Compile(name))
	}
}

// CreateTempDir create the dir in /tmp to save meta file and exec file.
func CreateTempDir() string {
	dir := filepath.Dir(os.Args[1])
	abs, _ := filepath.Abs(dir)
	tmp := filepath.Join("/tmp", abs)
	os.MkdirAll(tmp, os.ModePerm)
	return tmp
}

// GetExecName get the exec name from go file.
func GetExecName(name string) string {
	return name[:strings.LastIndex(name, ".go")]
}

// Compile compile the file and return the name of exec file.
func Compile(name string) string {
	execName := GetExecName(name)
	cmd := exec.Command("go", "build", "-o", execName, name)
	buf, err := cmd.CombinedOutput()
	os.Stdout.Write(buf)
	if err != nil {
		log.Fatalln("Compile Error:", err)
	}
	return execName
}

// Run run the exec file with os.Args[2:].
func Run(execName string) {
	cmd := exec.Command(execName, os.Args[2:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}
}

// CheckMetaInfo check the meta file is exist or expire, if the metafile
// is not exist or is expire, return false,metafilename and create new metafile.
// if the metafile is exist, return true,executable file name.
func CheckMetaInfo(f *os.File, td string) (bool, string) {
	st, _ := f.Stat()
	name := f.Name()
	metaname := fmt.Sprintf("%d.%s", st.ModTime().Unix(), name[strings.LastIndex(name, "/")+1:])
	metapath := filepath.Join(td, metaname)
	_, err := os.Stat(metapath)
	// File exist
	if err == nil {
		execName := GetExecName(metapath)
		_, err = os.Stat(execName)
		if err == nil {
			return true, GetExecName(metapath)
		}
	}
	// File not exist
	CleanTempDir(td)
	m, err := os.OpenFile(metapath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	defer m.Close()
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalln(err)
	}
	// Trim shebang line if exist
	if buf[0] == '#' && buf[1] == '!' {
		buf = buf[bytes.Index(buf, []byte("\n"))+1:]
	}
	m.Write(buf)
	return false, metapath
}

// CleanTempDir clean the temporary dir before it start build
func CleanTempDir(dir string) {
	filepath.Walk(dir, func(name string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return err
		}
		return os.Remove(name)
	})
}
