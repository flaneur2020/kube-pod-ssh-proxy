package main

import (
	"io"
	"log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/gliderlabs/ssh"
)

type ptyTerminal struct {
	session              ssh.Session
	kubeClient           *kubernetes.Clientset
	kubeRestClientConfig *restclient.Config

	user          string
	namespace     string
	podName       string
	containerName string
}

func newPtyTerminal(session ssh.Session) *ptyTerminal {
	return &ptyTerminal{
		session: session,
		user:    session.User(),
	}
}

func (pty *ptyTerminal) execKubePty(
	namespace string,
	podName string,
	containerName string,
	command string,
) error {
	exec, err := pty.kubeRemoteExecutor()
	if err != nil {
		return err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  pty.Session,
		Stdout: pty.Session,
		Stderr: pty.Session,
	})
	if err != nil {
		return err
	}

	return nil
}

func (pty *ptyTerminal) kubeRemoteExecutor() (remotecommand.Executor, error) {
	option := &v1.PodExecOptions{
		Container: containerName,
		Command:   []string{command},
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}

	req := pty.kubeClient.
		CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", container).
		VersionedParams(option, scheme.ParameterCodec)

	return remotecommand.NewSPDYExecutor(pty.kubeRestClientConfig, "POST", req.URL())
}

func main() {
	ssh.Handle(func(s ssh.Session) {
		io.WriteString(s, "Hello world\n")
	})

	log.Fatal(ssh.ListenAndServe(":2222", nil))
}
