package execextra

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sync"
	"testing"
)

func TestExtraInOutPipe(t *testing.T) {
	c := Command("dd")
	in, inFd, inErr := c.ExtraInPipe()
	if inErr != nil {
		t.Fatal(inErr)
	}
	out, outFd, outErr := c.ExtraOutPipe()
	if outErr != nil {
		t.Fatal(outErr)
	}
	c.Args = append(
		c.Args,
		[]string{fmt.Sprintf("if=/dev/fd/%d", inFd), fmt.Sprintf("of=/dev/fd/%d", outFd)}...,
	)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}

	expectedBytes := []byte("hello")
	var actualBytes []byte
	go func() {
		n, err := in.Write(expectedBytes)
		expectedN := len(expectedBytes)
		if n != expectedN {
			t.Errorf("expected write %d bytes, got %d", expectedN, n)
		}
		if err != nil {
			t.Error(err)
		}
		err = in.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	b := &bytes.Buffer{}
	n, err := io.Copy(b, out)
	expectedN := len(expectedBytes)
	if n != int64(expectedN) {
		t.Errorf("expected read %d bytes, got %d", expectedN, n)
	}
	if err != nil {
		t.Error(err)
	}
	actualBytes = b.Bytes()

	// wait should be done after all reads are complete
	c.Wait()

	if bytes.Compare(expectedBytes, actualBytes) != 0 {
		t.Errorf("expected bytes %v got %v", expectedBytes, actualBytes)
	}
}

func TestPipe(t *testing.T) {
	expectedBytes := []byte("hello")
	inBuffer := bytes.NewBuffer(expectedBytes)
	outBuffer := &bytes.Buffer{}

	var csWG sync.WaitGroup
	var cs []*Cmd
	var prevOut io.Reader = inBuffer
	nchilds := 5
	csWG.Add(nchilds)
	for i := 0; i < nchilds; i++ {
		c := Command("dd")
		in, inFd, inErr := c.ExtraInPipe()
		if inErr != nil {
			t.Fatal(inErr)
		}
		out, outFd, outErr := c.ExtraOutPipe()
		if outErr != nil {
			t.Fatal(outErr)
		}
		c.Args = append(
			c.Args,
			[]string{fmt.Sprintf("if=/dev/fd/%d", inFd), fmt.Sprintf("of=/dev/fd/%d", outFd)}...,
		)
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}

		go func(w io.WriteCloser, r io.Reader, c *Cmd) {
			io.Copy(w, r)
			log.Println("copy1")
			w.Close()
			log.Println("close1")
			c.Wait()
			log.Println("wait")
			csWG.Done()
		}(in, prevOut, c)

		prevOut = out
		cs = append(cs, c)
	}

	var actualBytes []byte
	go func() {
		n, err := io.Copy(outBuffer, prevOut)
		expectedN := len(expectedBytes)
		if n != int64(expectedN) {
			t.Errorf("expected read %d bytes, got %d", expectedN, n)
		}
		if err != nil {
			t.Error(err)
		}
		actualBytes = outBuffer.Bytes()
	}()

	csWG.Wait()

	// wait should be done after all reads are complete
	// for _, c := range cs {
	// 	log.Println("wait")
	// 	go c.Wait()
	// }

	if bytes.Compare(expectedBytes, actualBytes) != 0 {
		t.Errorf("expected bytes %v got %v", expectedBytes, actualBytes)
	}
}

func TestExtraInOut(t *testing.T) {
	expectedBytes := []byte("hello")

	c := Command("dd")
	inFd, inErr := c.ExtraIn(bytes.NewBuffer(expectedBytes))
	if inErr != nil {
		t.Fatal(inErr)
	}
	actualBuffer := &bytes.Buffer{}
	outFd, outErr := c.ExtraOut(actualBuffer)
	if outErr != nil {
		t.Fatal(outErr)
	}
	c.Args = append(
		c.Args,
		[]string{fmt.Sprintf("if=/dev/fd/%d", inFd), fmt.Sprintf("of=/dev/fd/%d", outFd)}...,
	)
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}

	actualBytes := actualBuffer.Bytes()

	if bytes.Compare(expectedBytes, actualBytes) != 0 {
		t.Errorf("expected bytes %v got %v", expectedBytes, actualBytes)
	}
}

func TestExtraReadWriter2(t *testing.T) {
	expectedBytes := []byte("hello")

	pr, pw := io.Pipe()

	c := Command("dd")
	cInFD, err := c.ExtraIn(bytes.NewBuffer(expectedBytes))
	if err != nil {
		t.Fatal(err)
	}
	cOutFD, err := c.ExtraOut(pw)
	if err != nil {
		t.Fatal(err)
	}
	c.Args = append(
		c.Args,
		[]string{fmt.Sprintf("if=/dev/fd/%d", cInFD), fmt.Sprintf("of=/dev/fd/%d", cOutFD)}...,
	)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}

	d := Command("dd")
	dInFD, err := d.ExtraIn(pr)
	if err != nil {
		t.Fatal(err)
	}
	actualBuffer := &bytes.Buffer{}
	dOutFD, err := d.ExtraOut(actualBuffer)
	if err != nil {
		t.Fatal(err)
	}
	d.Args = append(
		d.Args,
		[]string{fmt.Sprintf("if=/dev/fd/%d", dInFD), fmt.Sprintf("of=/dev/fd/%d", dOutFD)}...,
	)
	if err := d.Start(); err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		c.Wait()
		pw.Close()
		wg.Done()
	}()
	go func() {
		d.Wait()
		wg.Done()
	}()
	wg.Wait()

	actualBytes := actualBuffer.Bytes()

	if bytes.Compare(expectedBytes, actualBytes) != 0 {
		t.Errorf("expected bytes %v got %v", expectedBytes, actualBytes)
	}
}
