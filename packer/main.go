package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"plugin"
	plugins "skynet/plugin"
	"skynet/sn"
	"strings"
)

func zipFolder(inst *sn.SNPluginInfo, source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()
	zipfile.Chmod(0755)

	archive := zip.NewWriter(zipfile)
	defer archive.Close()
	m, err := json.Marshal(inst)
	if err != nil {
		return err
	}
	archive.SetComment(string(m))

	filepath.Walk(source, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rpath := path.Clean(strings.TrimPrefix(p, source))
		if rpath == "." { // folder itself
			return nil
		}
		if info.IsDir() {
			rpath += "/"
			archive.Create(rpath)
			log.Println("Add ", rpath)
			return nil
		}
		log.Println("Add ", rpath)
		writer, err := archive.Create(rpath)
		if err != nil {
			return err
		}

		file, err := os.Open(p)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Skynet plugin packer\nUsage: %v [folder] [fileprefix]\n", os.Args[0])
		return
	}
	srcFolder := path.Clean(os.Args[1])
	destFile := os.Args[2] + ".sp"

	stat, err := os.Stat(srcFolder)
	if err != nil {
		log.Fatal(err)
	}
	if !stat.IsDir() {
		log.Fatal("Path is not a folder")
	}

	files, err := ioutil.ReadDir(srcFolder)
	if err != nil {
		log.Fatal(err)
	}
	ok := false
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".so") {
			pPlugin, err := plugin.Open(path.Join(srcFolder, f.Name()))
			if err != nil {
				log.Fatal(err)
			}
			pSymbol, err := pPlugin.Lookup("NewPlugin")
			if err != nil {
				log.Fatal(err)
			}
			pInterface := pSymbol.(func() plugins.PluginInterface)()
			if err := zipFolder(pInterface.Instance(), srcFolder, destFile); err != nil {
				log.Fatal(err)
			}
			ok = true
			break
		}
	}
	if ok {
		log.Println("Plugin generated ", destFile)
	} else {
		log.Println("No .so file found")
	}
}
