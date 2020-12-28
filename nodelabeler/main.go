/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"net"
	"os"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/prometheus/common/log"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	logger, _ := zap.NewProduction()
	log := logger.Named("nodelabeler")

	nodeName, ok := os.LookupEnv("NODE")
	if !ok {
		log.Fatal("Environment variable is not defined",
			zap.String("var", "NODE"),
		)
	}

	log.Info("Starting service",
		zap.String("node", nodeName),
	)

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err.Error())
	}

	for {
		network, err := net.InterfaceByName(os.Getenv("VM_BRIDGE_INTERFACE"))
		if err != nil {
			log.Fatal(err.Error())
		}
		addrs, err := network.Addrs()
		if err != nil {
			log.Fatal(err.Error())
		}
		ifaceAddr := strings.Split(addrs[0].String(), "/")[0]
		labels := map[string]string{
			"airshipit.org/vino.nodebridgegw":  ifaceAddr,
		}

		node, err := clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
		if err != nil {
			log.Fatal(err.Error())
		}

		for label, value := range labels {
			err = addLabelToNode(clientset, node, label, value)
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		time.Sleep(600 * time.Second)
	}
}

func addLabelToNode(clientset *kubernetes.Clientset, node *v1.Node, key string, value string) error {
	log.Info("Applying node label",
		zap.String(key, value),
	)

	originalNode, err := json.Marshal(node)
	if err != nil {
		return err
	}

	node.ObjectMeta.Labels[key] = value

	newNode, err := json.Marshal(node)
	if err != nil {
		return err
	}

	patch, err := jsonpatch.CreateMergePatch(originalNode, newNode)
	if err != nil {
		return err
	}

	log.Info("Patching Node resource",
		zap.String("node", node.ObjectMeta.Name),
		zap.String("patch", string(patch)),
	)

	_, err = clientset.CoreV1().Nodes().Patch(node.Name, types.MergePatchType, patch)
	if err != nil {
		return err
	}

	return nil
}
