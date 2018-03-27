// Package scp provides a simple interface to copying files over a
// go.crypto/ssh session.
package scp

import (
	"fmt"
	"io"
	"os"
	"path"
	"bufio"
	shellquote "github.com/kballard/go-shellquote"
	"gopkg.in/cheggaaa/pb.v1"
	"golang.org/x/crypto/ssh"
)

func Copy(size int64, mode os.FileMode, fileName string, contents io.Reader, destinationPath string, session *ssh.Session) error {
	return copy(size, mode, fileName, contents, destinationPath, session)
}

func CopyPath(filePath, destinationPath string, session *ssh.Session) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil {
		return err
	}
	return copy(s.Size(), s.Mode().Perm(), path.Base(filePath), f, destinationPath, session)
}

func GetPath(size int, src, dest string, session *ssh.Session) error {
	return get(size, src, dest, session)
}

func copy(size int64, mode os.FileMode, fileName string, contents io.Reader, destination string, session *ssh.Session) error {
	defer session.Close()
	w, err := session.StdinPipe()

	if err != nil {
		return err
	}

	cmd := shellquote.Join("scp", "-t", destination)
	if err := session.Start(cmd); err != nil {
		w.Close()
		return err
	}

	errors := make(chan error)

	go func() {
		errors <- session.Wait()
	}()

	fmt.Fprintf(w, "C%#o %d %s\n", mode, size, fileName)
	io.Copy(w, contents)
	fmt.Fprint(w, "\x00")
	w.Close()

	return <-errors
}

func get(size int, src string, dest string, session *ssh.Session) error {
	defer session.Close()
	r, err := session.StdoutPipe()

	if err != nil {
		return err
	}

	bar := pb.New(size).SetUnits(pb.U_BYTES)
	bar.Start()

	reader := bar.NewProxyReader(r)

	cmd := shellquote.Join("cat", src)
	if err := session.Start(cmd); err != nil {
//		r.Close()
		return err
	}

	errors := make(chan error)

	go func() {
		errors <- session.Wait()
	}()

	f, err := os.Create(dest)

	w := bufio.NewWriter(f)
	io.Copy(w, reader)
	w.Flush()
	bar.Finish()
	return <-errors
}
