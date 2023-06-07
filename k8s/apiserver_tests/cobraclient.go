package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctlruntime "sigs.k8s.io/controller-runtime"
)

var count int
var concurrent int

func NewCliCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cobraClient",
		Short: "KubeBlocks CLI.",
		Long:  `just for test`,

		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.PersistentFlags().IntVarP(&count, "count", "", 10, "count of request")
	cmd.PersistentFlags().IntVarP(&concurrent, "concurrent", "", 1, "count of concurrent")
	return cmd
}

func NewEventCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event",
		Short: "KubeBlocks CLI.",
		Long:  `just for test`,

		Run: func(cmd *cobra.Command, args []string) {
			//gv := v1.SchemeGroupVersion
			//config.GroupVersion = &gv
			//config.APIPath = "/api"
			//config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
			restConfig := ctlruntime.GetConfigOrDie()

			gv := corev1.SchemeGroupVersion
			restConfig.GroupVersion = &gv
			restConfig.APIPath = "/api"
			restConfig.QPS = 10000
			restConfig.Burst = 10000
			restConfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
			var wg sync.WaitGroup
			for i := 0; i < concurrent; i++ {
				wg.Add(1)
				go sendEvent(restConfig, &wg)
			}
			wg.Wait()
		},
	}
	return cmd
}

func sendEvent(restConfig *rest.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	client, err1 := rest.RESTClientFor(restConfig)
	if err1 != nil {
		fmt.Printf("err: %v\n", err1)
	}
	i := 0
	result := &corev1.Event{}
	for ; i < count; i++ {
		event := createEvent()
		err := client.Post().
			Namespace("default").
			Resource("events").
			VersionedParams(&metav1.GetOptions{}, scheme.ParameterCodec).
			Body(event).
			Do(context.TODO()).
			Into(result)
		if err != nil {
			klog.V(0).Infof("error: %v", err)
			break
		}
		klog.V(1).Infof("event: %v", result)
	}
	klog.V(0).Infof("send %d events", i)
}

func createEvent() *corev1.Event {
	eventTmpl := `
apiVersion: v1
kind: Event
metadata:
  name: PodName
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{sqlchannel}
  kind: Pod
  name: PodName
reason: RoleChanged
type: Normal
source:
  component: sqlchannel
`

	// get pod object
	podName := os.Getenv("KB_POD_NAME")
	podUID := os.Getenv("KB_POD_UID")
	nodeName := os.Getenv("KB_NODENAME")
	//namespace := os.Getenv("KB_NAMESPACE")
	seq := rand.String(16)
	event := &corev1.Event{}
	_, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(eventTmpl), nil, event)
	if err != nil {
		fmt.Printf("event error: %v", err)
		return nil
	}
	event.Message = "for test"
	event.InvolvedObject.UID = types.UID(podUID)
	event.Source.Host = nodeName
	event.Reason = "test"
	event.Name = podName + "test" + seq

	return event
}

func main() {
	cmd := NewCliCmd()
	cmd.AddCommand(NewEventCmd())
	cmd.Execute()
}
