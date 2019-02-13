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
	"time"

	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// Watcher watches for changes in kubernetes services and triggers a sync when required.
type Watcher struct {
	client *kube.Clientset
	syncer *Syncer

	running     bool
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

// NewWatcher creates a new watcher.
func NewWatcher(config *rest.Config, serviceNamespace string, serviceName string) (*Watcher, error) {
	client, err := kube.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &Watcher{
		client: client,
		syncer: NewSyncer(client, serviceNamespace, serviceName),
	}, nil
}

// Start starts the watcher.
func (w *Watcher) Start() error {
	if w.running {
		return ErrAlreadyRunning
	}
	if err := w.syncer.Start(); err != nil {
		return err
	}

	watchlist := cache.NewListWatchFromClient(
		w.client.CoreV1().RESTClient(),
		string(v1.ResourceServices),
		v1.NamespaceAll,
		fields.Everything(),
	)
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Service{},
		15*time.Minute, // resync every 15 minutes
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.syncer.TriggerSync()
			},
			DeleteFunc: func(obj interface{}) {
				w.syncer.TriggerSync()
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				w.syncer.TriggerSync()
			},
		},
	)

	w.running = true
	w.stopChan = make(chan struct{})
	w.stoppedChan = make(chan struct{})
	defer close(w.stoppedChan)
	go controller.Run(w.stopChan)
	return nil
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	if !w.running {
		return ErrNotRunning
	}
	if err := w.syncer.Stop(); err != nil {
		return err
	}

	close(w.stopChan)
	<-w.stoppedChan
	return nil
}

// IsRunning returns true if the watcher is running.
func (w *Watcher) IsRunning() bool {
	return w.running
}
