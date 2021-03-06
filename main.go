package main

import (
	"flag"
	"io"
	"log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/gliderlabs/ssh"
)

type podPTY struct {
	session          ssh.Session
	kubeClient       *kubernetes.Clientset
	kubeClientConfig *restclient.Config

	user          string
	namespace     string
	podName       string
	containerName string
}

func newPodPTY(
	session ssh.Session,
	kubeClient *kubernetes.Clientset,
	kubeClientConfig *restclient.Config,
) *podPTY {
	return &podPTY{
		session:          session,
		user:             session.User(),
		kubeClient:       kubeClient,
		kubeClientConfig: kubeClientConfig,
	}
}

func (pty *podPTY) Exec(
	namespace string,
	podName string,
	containerName string,
	command string,
) error {
	executor, err := pty.kubeRemoteExecutor(namespace, podName, containerName, command)
	if err != nil {
		return err
	}

	err = executor.Stream(remotecommand.StreamOptions{
		Stdin:  pty.session,
		Stdout: pty.session,
		Stderr: pty.session,
	})
	if err != nil {
		return err
	}

	return nil
}

func (pty *podPTY) kubeRemoteExecutor(namespace, podName, containerName, command string) (remotecommand.Executor, error) {
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
		Param("container", containerName).
		VersionedParams(option, scheme.ParameterCodec)

	return remotecommand.NewSPDYExecutor(pty.kubeClientConfig, "POST", req.URL())
}

func main() {
	var (
		confPath      string
		namespace     string
		containerName string
	)

	flag.StringVar(&confPath, "conf", "", "k8s config")
	flag.StringVar(&namespace, "namespace", "default", "the namespace which we are serving")
	flag.StringVar(&containerName, "container", "", "the container name")

	flag.Parse()

	kubeClientConfig, err := clientcmd.BuildConfigFromFlags("", confPath)
	if err != nil {
		log.Fatalf(err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		log.Fatalf(err.Error())
	}

	ssh.Handle(func(session ssh.Session) {
		io.WriteString(session, "Welcome!\n")

		podName := session.User()
		pty := newPodPTY(session, kubeClient, kubeClientConfig)

		err := pty.Exec(namespace, podName, containerName, "/bin/sh")
		if err != nil {
			io.WriteString(session, err.Error()+"\n")
			log.Printf("session.closed: %s", err.Error())
		}
	})

	log.Printf("listening :2222")
	log.Fatal(ssh.ListenAndServe(":2222", nil))
}
