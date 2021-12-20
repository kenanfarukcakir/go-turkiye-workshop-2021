package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"sidecache-injector/pkg/mutators"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
	version   string
)

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1.AddToScheme(runtimeScheme)
	_ = v1.AddToScheme(runtimeScheme)
}

func main() {

	webhookTlsCert, _ := tls.LoadX509KeyPair("/tmp/tls-secret/tls.crt", "/tmp/tls-secret/tls.key")

	server := &http.Server{
		Addr:      ":8080",
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{webhookTlsCert}},
	}

	sidecacheWebHookPath := "/mutate"

	sidecacheMutatingHook := &admission.Webhook{
		Handler: admission.HandlerFunc(func(ctx context.Context, req admission.Request) admission.Response {
			fmt.Println("Webhook mutate endpoint...")
			decoder, _ := admission.NewDecoder(runtimeScheme)
			var deploymentObject v1beta1.Deployment
			err := decoder.Decode(req, &deploymentObject)
			if err != nil {
				fmt.Println(err)
				return admission.Errored(http.StatusInternalServerError, err)
			}

			fmt.Println("Decoded deployment object...")

			return mutators.InjectSidecache(req, deploymentObject)
		}),
	}

	sidecacheHookHandler, _ := admission.StandaloneWebhook(sidecacheMutatingHook, admission.StandaloneOptions{})

	http.Handle(sidecacheWebHookPath, sidecacheHookHandler)

	server.ListenAndServeTLS("", "")
}
