package mutators

import (
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/api/resource"

	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	sidecacheAnnotationInjectKey = "sidecache.trendyol.com/inject"

	limitMemory, _   = resource.ParseQuantity("1Gi")
	limitCPU, _      = resource.ParseQuantity("1")
	requestCPU, _    = resource.ParseQuantity("30m")
	requestMemory, _ = resource.ParseQuantity("30Mi")
)

func InjectSidecache(req admission.Request, deploymentObject v1beta1.Deployment) admission.Response {
	if deploymentObject.Spec.Template.ObjectMeta.Annotations[sidecacheAnnotationInjectKey] != "true" {
		for i, c := range deploymentObject.Spec.Template.Spec.Containers {
			if c.Name == "sidecache" {
				deploymentObject.Spec.Template.Spec.Containers = append(deploymentObject.Spec.Template.Spec.Containers[:i], deploymentObject.Spec.Template.Spec.Containers[i+1:]...)
				fmt.Println("Removing sidecache...")
			}
		}
	} else {
		alreadyExists := false
		for _, c := range deploymentObject.Spec.Template.Spec.Containers {
			if c.Name == "sidecache" {
				alreadyExists = true
			}
		}

		if !alreadyExists {
			currentAnnotations := deploymentObject.Spec.Template.ObjectMeta.Annotations
			sidecache := createSidecacheContainer(currentAnnotations)
			deploymentObject.Spec.Template.Spec.Containers = append(deploymentObject.Spec.Template.Spec.Containers, sidecache)
			fmt.Println("Injecting sidecache...")
		}
	}

	marshaledObj, err := json.Marshal(deploymentObject)
	if err != nil {
		fmt.Println(err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	fmt.Println("Sending kubeapiserver admissionResponse...")
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledObj)
}

func createSidecacheContainer(currentAnnotations map[string]string) corev1.Container {
	sidecacheImage := os.Getenv("SIDECACHE_IMAGE")

	return corev1.Container{
		Name:  "sidecache",
		Image: sidecacheImage,
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    limitCPU,
				corev1.ResourceMemory: limitMemory,
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    requestCPU,
				corev1.ResourceMemory: requestMemory,
			},
		},
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name:  "EXAMPLE_ENV_VAR",
				Value: "example_env_var",
			},
		},
	}
}
