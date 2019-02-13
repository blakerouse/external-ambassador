// Copyright (c) 2019 Blake Rouse.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS
// OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
// IN THE SOFTWARE.

package sync

import (
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "k8s.io/client-go/kubernetes"
)

// Syncer actually performs the syncing of annotations.
type Syncer struct {
	client *kube.Clientset

	serviceNamespace string
	serviceName      string

	running     bool
	needsSync   bool
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

// NewSyncer creates and syncer.
func NewSyncer(client *kube.Clientset, serviceNamespace string, serviceName string) *Syncer {
	return &Syncer{
		client:           client,
		serviceNamespace: serviceNamespace,
		serviceName:      serviceName,
	}
}

// Start starts the syncer.
func (s *Syncer) Start() error {
	if s.running {
		return ErrAlreadyRunning
	}

	s.running = true
	s.stopChan = make(chan struct{})
	s.stoppedChan = make(chan struct{})
	go s.loop()
	return nil
}

// Stop stops the syncer.
func (s *Syncer) Stop() error {
	if !s.running {
		return ErrNotRunning
	}

	close(s.stopChan)
	<-s.stoppedChan
	s.running = false
	return nil
}

// IsRunning returns true if the syncer is running.
func (s *Syncer) IsRunning() bool {
	return s.running
}

// TriggerSync triggers a sync to occur.
func (s *Syncer) TriggerSync() {
	s.needsSync = true
}

// Sync performs the sync operation.
func (s *Syncer) Sync() ([]string, error) {
	var config AmbassadorConfig
	var ambassadorService *v1.Service

	hosts := make([]string, 0)
	namespaces, err := s.client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return hosts, err
	}
	for _, namespace := range namespaces.Items {
		services, err := s.client.CoreV1().Services(namespace.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, service := range services.Items {
			if namespace.Name == s.serviceNamespace && service.Name == s.serviceName {
				ambassadorService = &service
			}
			for key, value := range service.Annotations {
				if key == AmbassadorAnnotationKey {
					err := yaml.Unmarshal([]byte(value), &config)
					if err != nil {
						log.WithFields(log.Fields{
							"error": err,
						}).Error("Failed to parse ambassador annotation config")
						continue
					}
					if config.IsMapping() {
						if config.Host != "" {
							log.Debugf("Service %s.%s has host: %s", namespace.Name, service.Name, config.Host)
							if !funk.ContainsString(hosts, config.Host) {
								hosts = append(hosts, config.Host)
							}
						} else {
							log.Debugf("Service %s.%s doesn't host defined on ambassador mapping annotation", namespace.Name, service.Name)
						}
					} else {
						log.Debugf("Service %s.%s doesn't have a ambassador mapping annotation", namespace.Name, service.Name)
					}
				}
			}
		}
	}

	dnsValue := ""
	if len(hosts) > 0 {
		sort.Strings(hosts)
		dnsValue = strings.Join(hosts, ",")
		log.Debugf("Found the following hosts to update external-dns annotation: %s", dnsValue)
	} else {
		log.Debugf("Found zero hosts to set for external dns, annotation will be removed")
	}

	if ambassadorService == nil {
		return hosts, fmt.Errorf("Failed to find ambassador service to update external-dns annotation on")
	}

	needsUpdate := false
	foundDNS := false
	for key, value := range ambassadorService.Annotations {
		if key == ExternalDNSAnnotationKey {
			foundDNS = true
			if value != dnsValue {
				ambassadorService.Annotations[key] = dnsValue
				needsUpdate = true
			}
		}
	}
	if dnsValue != "" && !foundDNS {
		ambassadorService.Annotations[ExternalDNSAnnotationKey] = dnsValue
		needsUpdate = true
	} else if dnsValue == "" && foundDNS {
		delete(ambassadorService.Annotations, ExternalDNSAnnotationKey)
		needsUpdate = true
	}
	if needsUpdate {
		_, err := s.client.CoreV1().Services(s.serviceNamespace).Update(ambassadorService)
		if err != nil {
			return hosts, err
		}
		log.Debugf("Service %s.%s was external-dns annotation was updated", s.serviceNamespace, s.serviceName)
	} else {
		log.Debugf("Service %s.%s was external-dns annotation not updated", s.serviceNamespace, s.serviceName)
	}

	return hosts, nil
}

// loop loops and performs sync.
func (s *Syncer) loop() {
	defer close(s.stoppedChan)

	for {
		select {
		case <-s.stopChan:
			return
		case <-time.After(4 * time.Second):
			if s.needsSync {
				log.Info("Performing sync between annotations")
				hosts, err := s.Sync()
				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Error("Failed to perform sync")
				} else {
					s.needsSync = false
					if len(hosts) > 0 {
						log.Infof("Performed sync between annotations for hosts: %s", strings.Join(hosts, ", "))
					} else {
						log.Warnf("No hosts found; external-dns annotation removed")
					}
				}
			}
		}
	}
}
